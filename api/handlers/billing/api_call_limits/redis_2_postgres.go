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

// logProjectUsage drains one project's pending hash: the "total" field, which is
// what billing is measured against, plus the per-dimension fields that slice it
// up. Both are sent in one call so company-service can write them in one
// transaction. Counts are only subtracted back out of Redis after a successful
// write, so a failed tick retries instead of losing data.
func (c *MetricsConsumer) logProjectUsage(ctx context.Context, key, projectID string) {
	fields, err := c.rdb.HGetAll(ctx, key).Result()
	if err != nil || len(fields) == 0 {
		return
	}

	total, _ := strconv.ParseInt(fields[config.KeyUsageTotalField], 10, 64)
	if total <= 0 {
		return
	}

	details, sent := collectDetails(fields)

	_, err = c.companyService.Billing().LogUsage(ctx, &pb.LogUsageRequest{
		ProjectId: projectID,
		Count:     total,
		TimeRange: usageLogTimeRangeSec,
		DateTime:  time.Now().Truncate(time.Hour).Format("2006-01-02 15:00:00"),
		Details:   details,
	})
	if err != nil {
		// Leave counts in Redis — will retry on next tick.
		log.Printf("[MetricsConsumer] LogUsage project=%s: %v", projectID, err)
		return
	}

	// Subtract exactly what was written. Zeroed fields are left in place on
	// purpose: deleting them would race with a Tracker flush and drop counts.
	pipe := c.rdb.Pipeline()
	pipe.HIncrBy(ctx, key, config.KeyUsageTotalField, -total)
	for field, count := range sent {
		pipe.HIncrBy(ctx, key, field, -count)
	}
	if _, err = pipe.Exec(ctx); err != nil {
		log.Printf("[MetricsConsumer] drain project=%s: %v", projectID, err)
	}
}

// collectDetails turns the raw hash fields into wire rows, keeping the biggest
// buckets and folding the tail into one "other" row. It returns both the rows to
// send and, keyed by hash field, exactly how much of each was consumed.
func collectDetails(fields map[string]string) ([]*pb.ApiUsageDetail, map[string]int64) {
	type entry struct {
		field  string
		detail *pb.ApiUsageDetail
	}

	entries := make([]entry, 0, len(fields))
	for field, raw := range fields {
		source, method, route, collection, ok := parseUsageField(field)
		if !ok {
			continue
		}
		count, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || count <= 0 {
			continue
		}
		entries = append(entries, entry{
			field: field,
			detail: &pb.ApiUsageDetail{
				Source:     source,
				Method:     method,
				Route:      route,
				Collection: collection,
				Count:      count,
			},
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].detail.Count > entries[j].detail.Count
	})

	var (
		details  = make([]*pb.ApiUsageDetail, 0, len(entries))
		sent     = make(map[string]int64, len(entries))
		overflow int64
	)
	for i, e := range entries {
		sent[e.field] = e.detail.Count
		if i < maxDetailRowsPerFlush {
			details = append(details, e.detail)
			continue
		}
		overflow += e.detail.Count
	}
	if overflow > 0 {
		details = append(details, &pb.ApiUsageDetail{
			Route: config.UsageDetailOverflowRoute,
			Count: overflow,
		})
	}

	return details, sent
}
