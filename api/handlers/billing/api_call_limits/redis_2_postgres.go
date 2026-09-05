package apilimits

import (
	"context"
	"log"
	"sort"
	"strconv"
	"time"

	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/services"

	"github.com/go-redis/redis/v8"
)

// maxDetailRowsPerFlush caps how many dimension rows one LogUsage call carries.
// Anything past the cap is folded into a single "other" row, so the detail rows
// still add up to the project total no matter how wide the tail is.
const maxDetailRowsPerFlush = 500

// MetricsConsumer drains pending API usage counts from Redis and logs them to
// the billing service (PostgreSQL via gRPC) on a regular interval.
type MetricsConsumer struct {
	worker
	rdb            *redis.Client
	companyService services.CompanyServiceI
	flushInterval  time.Duration
}

func NewMetricsConsumer(rdb *redis.Client, companyService services.CompanyServiceI, flushInterval time.Duration) *MetricsConsumer {
	return &MetricsConsumer{
		rdb:            rdb,
		companyService: companyService,
		flushInterval:  flushInterval,
	}
}

func (c *MetricsConsumer) Start(ctx context.Context) {
	c.spawn(ctx, c.run)
	log.Printf("[MetricsConsumer] started db_flush_interval=%v", c.flushInterval)
}

func (c *MetricsConsumer) Stop() {
	c.shutdown()
	log.Println("[MetricsConsumer] stopped")
}

func (c *MetricsConsumer) run(ctx context.Context) {
	ticker := time.NewTicker(c.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.flushApiUsage(ctx)
		}
	}
}

func (c *MetricsConsumer) flushApiUsage(ctx context.Context) {
	var cursor uint64
	for {
		keys, next, err := c.rdb.Scan(ctx, cursor, config.KeyUsagePendingPattern, redisScanBatch).Result()
		if err != nil {
			log.Printf("[MetricsConsumer] SCAN error: %v", err)
			return
		}

		for _, key := range keys {
			projectID := key[len(config.KeyUsagePendingPrefix):]
			c.logProjectUsage(ctx, key, projectID)
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}
}

// logProjectUsage drains one project's pending hash: the "total" field, which
// is what billing is measured against, plus the per-dimension fields that slice
// it up. Fields are grouped by their time bucket and sent one call per bucket,
// so each row lands in the interval the requests actually happened in.
//
// Counts are only subtracted back out of Redis after a successful write, so a
// failed bucket retries on the next tick instead of losing data.
func (c *MetricsConsumer) logProjectUsage(ctx context.Context, key, projectID string) {
	fields, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil || len(fields) == 0 {
		return
	}

	total, _ := strconv.ParseInt(fields[config.KeyUsageTotalField], 10, 64)
	if total <= 0 {
		return
	}

	var accounted int64
	for bucket, entries := range groupByBucket(fields) {
		details, sent := detailRows(bucket, entries)

		var count int64
		for _, n := range sent {
			count += n
		}
		if count <= 0 {
			continue
		}

		if _, err := c.companyService.Billing().LogUsage(ctx, &pb.LogUsageRequest{
			ProjectId: projectID,
			Count:     count,
			TimeRange: usageLogTimeRangeSec,
			DateTime:  hourOf(bucket),
			Details:   details,
		}); err != nil {
			// Leave this bucket in Redis — it will retry on the next tick.
			log.Printf("[MetricsConsumer] LogUsage project=%s bucket=%s: %v", projectID, bucket, err)
			continue
		}
		accounted += count

		// Subtract exactly what was written. Zeroed fields are left in place on
		// purpose: deleting them would race with a Tracker flush and drop counts.
		pipe := c.rdb.Pipeline()
		for field, n := range sent {
			pipe.HIncrBy(ctx, key, field, -n)
		}
		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("[MetricsConsumer] drain project=%s bucket=%s: %v", projectID, bucket, err)
		}
	}

	// Counts carrying no dimension fields — a gateway pod from before this
	// rollout — would otherwise sit in the total forever. Settle them against
	// the current hour with no breakdown; the breakdown query reports them as
	// "other" rather than losing them.
	if remainder := total - accounted; remainder > 0 {
		if _, err := c.companyService.Billing().LogUsage(ctx, &pb.LogUsageRequest{
			ProjectId: projectID,
			Count:     remainder,
			TimeRange: usageLogTimeRangeSec,
			DateTime:  time.Now().UTC().Truncate(time.Hour).Format(bucketLayout),
		}); err != nil {
			log.Printf("[MetricsConsumer] LogUsage(untagged) project=%s: %v", projectID, err)
		} else {
			accounted += remainder
		}
	}

	if accounted > 0 {
		if err := c.rdb.HIncrBy(ctx, key, config.KeyUsageTotalField, -accounted).Err(); err != nil {
			log.Printf("[MetricsConsumer] drain total project=%s: %v", projectID, err)
		}
	}
}

// bucketEntry is one decoded hash field together with its count.
type bucketEntry struct {
	field string
	usageField
	count int64
}

// groupByBucket decodes the detail fields and buckets them by time interval.
func groupByBucket(fields map[string]string) map[string][]bucketEntry {
	grouped := make(map[string][]bucketEntry)

	for field, raw := range fields {
		parsed, ok := parseUsageField(field)
		if !ok {
			continue
		}
		count, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || count <= 0 {
			continue
		}
		grouped[parsed.bucket] = append(grouped[parsed.bucket], bucketEntry{
			field: field, usageField: parsed, count: count,
		})
	}

	return grouped
}

// detailRows turns one bucket's entries into wire rows, keeping the biggest
// buckets and folding the tail into a single "other" row so the rows still add
// up to the bucket total no matter how wide the tail is. It returns both the
// rows to send and, keyed by hash field, exactly how much of each was consumed.
func detailRows(bucket string, entries []bucketEntry) ([]*pb.ApiUsageDetail, map[string]int64) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].count > entries[j].count })

	var (
		details  = make([]*pb.ApiUsageDetail, 0, len(entries))
		sent     = make(map[string]int64, len(entries))
		overflow int64
	)

	for i, e := range entries {
		sent[e.field] = e.count
		if i >= maxDetailRowsPerFlush {
			overflow += e.count
			continue
		}
		details = append(details, &pb.ApiUsageDetail{
			Source:     e.source,
			AuthType:   e.authType,
			ActorId:    e.actorID,
			ActorName:  e.actorName,
			Method:     e.method,
			Route:      e.route,
			Collection: e.collection,
			Count:      e.count,
			Bucket:     bucket,
		})
	}

	if overflow > 0 {
		details = append(details, &pb.ApiUsageDetail{
			Route:  config.UsageDetailOverflowRoute,
			Count:  overflow,
			Bucket: bucket,
		})
	}

	return details, sent
}

// hourOf maps a 15-minute bucket onto the hour billing_usage is keyed by.
func hourOf(bucket string) string {
	t, err := time.Parse(bucketLayout, bucket)
	if err != nil {
		return time.Now().UTC().Truncate(time.Hour).Format(bucketLayout)
	}
	return t.Truncate(time.Hour).Format(bucketLayout)
}
