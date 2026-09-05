package apilimits

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"ucode/ucode_go_api_gateway/config"

	"github.com/go-redis/redis/v8"
)

// usageKey is one dimensioned slice of a project's API call count. It is
// comparable, so it can be used directly as a map key.
type usageKey struct {
	projectID  string
	source     string // client | admin
	authType   string // api_key | bearer | "" when the caller sent neither
	actorID    string // api_keys record id, or user id for bearer — never a credential
	actorName  string // display copy, e.g. "Mobile App"; empty for bearer
	method     string
	route      string // gin route template, e.g. /v2/items/:collection
	collection string // table slug, empty when the route has none
}

// usageBucket is the resolution the breakdown is stored at. The bucket is
// stamped here, on the 10-second flush, rather than when the counts are drained
// to Postgres ten minutes later — otherwise a request could be filed under an
// interval it did not happen in.
const usageBucket = 15 * time.Minute

// bucketLayout is what Postgres parses back into a timestamp.
const bucketLayout = "2006-01-02 15:04:05"

// field encodes the key as a Redis hash field:
// "d|bucket|source|authType|method|route|collection". The project id is already
// in the hash key itself, so it is not repeated here.
func (k usageKey) field(bucket string) string {
	return config.KeyUsageDetailPrefix + strings.Join(
		[]string{
			bucket,
			sanitize(k.source), sanitize(k.authType),
			sanitize(k.actorID), sanitize(k.actorName),
			sanitize(k.method), sanitize(k.route), sanitize(k.collection),
		},
		config.KeyUsageDetailSep,
	)
}

// usageField is one decoded detail field.
type usageField struct {
	bucket     string
	source     string
	authType   string
	actorID    string
	actorName  string
	method     string
	route      string
	collection string
}

// parseUsageField reverses field(). Returns ok=false for the "total" field and
// for anything else that is not a detail field.
func parseUsageField(f string) (usageField, bool) {
	if !strings.HasPrefix(f, config.KeyUsageDetailPrefix) {
		return usageField{}, false
	}
	parts := strings.Split(strings.TrimPrefix(f, config.KeyUsageDetailPrefix), config.KeyUsageDetailSep)
	if len(parts) != 8 {
		return usageField{}, false
	}
	return usageField{
		bucket:     parts[0],
		source:     parts[1],
		authType:   parts[2],
		actorID:    parts[3],
		actorName:  parts[4],
		method:     parts[5],
		route:      parts[6],
		collection: parts[7],
	}, true
}

// sanitize keeps the separator unambiguous. Slugs and route templates never
// legitimately contain a pipe.
func sanitize(s string) string {
	return strings.ReplaceAll(s, config.KeyUsageDetailSep, "_")
}

// Tracker counts incoming API calls in an L1 in-memory map and flushes totals to
// Redis on a regular interval. Safe for concurrent middleware use.
//
// The map is swapped out wholesale on every flush rather than being mutated in
// place, which keeps memory bounded without a delete race that could drop counts.
type Tracker struct {
	worker
	flushInterval time.Duration
	rdb           *redis.Client

	mu     sync.Mutex
	counts map[usageKey]int64
}

func NewTracker(rdb *redis.Client, flushInterval time.Duration) *Tracker {
	return &Tracker{
		flushInterval: flushInterval,
		rdb:           rdb,
		counts:        make(map[usageKey]int64),
	}
}

func (t *Tracker) Start(ctx context.Context) {
	t.spawn(ctx, t.run)
	log.Printf("[Tracker] started flush_interval=%v", t.flushInterval)
}

func (t *Tracker) Stop() {
	t.shutdown()
	t.flush() // final drain on graceful shutdown
	log.Println("[Tracker] stopped")
}

func (t *Tracker) run(ctx context.Context) {
	ticker := time.NewTicker(t.flushInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.flush()
		}
	}
}

// add records one request. When a pod accumulates more distinct keys than the
// cap between two flushes, the excess is folded into a single "other" bucket per
// project so that customer-chosen collection names can never grow the map without
// bound. The map is emptied every flush, so this is not a permanent degradation.
func (t *Tracker) add(k usageKey) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, seen := t.counts[k]; !seen && len(t.counts) >= config.UsageDetailMaxKeys {
		k = usageKey{projectID: k.projectID, source: k.source, route: config.UsageDetailOverflowRoute}
	}
	t.counts[k]++
}

// take swaps the accumulated counts out under the lock and hands them back.
func (t *Tracker) take() map[usageKey]int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.counts) == 0 {
		return nil
	}
	batch := t.counts
	t.counts = make(map[usageKey]int64, len(batch))
	return batch
}

func (t *Tracker) flush() {
	batch := t.take()
	if batch == nil {
		return
	}

	ctx := context.Background()
	now := time.Now()
	bucket := now.UTC().Truncate(usageBucket).Format(bucketLayout)
	pipe := t.rdb.Pipeline()

	// Project totals drive billing; the per-dimension fields only slice them up.
	totals := make(map[string]int64, len(batch))
	for k, delta := range batch {
		if delta == 0 {
			continue
		}
		totals[k.projectID] += delta
		pipe.HIncrBy(ctx, fmt.Sprintf(config.KeyUsagePending, k.projectID), k.field(bucket), delta)
	}

	for projectID, delta := range totals {
		minKey := fmt.Sprintf(config.KeyRateMin, projectID, now.Format("2006-01-02-15-04"))
		hourKey := fmt.Sprintf(config.KeyRateHour, projectID, now.Format("2006-01-02-15"))
		dayKey := fmt.Sprintf(config.KeyRateDay, projectID, now.Format("2006-01-02"))

		pipe.IncrBy(ctx, minKey, delta)
		pipe.Expire(ctx, minKey, 15*time.Minute)
		pipe.IncrBy(ctx, hourKey, delta)
		pipe.Expire(ctx, hourKey, 24*time.Hour)
		pipe.IncrBy(ctx, dayKey, delta)
		pipe.Expire(ctx, dayKey, 48*time.Hour)
		pipe.HIncrBy(ctx, fmt.Sprintf(config.KeyUsagePending, projectID), config.KeyUsageTotalField, delta)
	}

	if len(totals) > 0 {
		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("[Tracker] flush error: %v", err)
		}
	}
}
