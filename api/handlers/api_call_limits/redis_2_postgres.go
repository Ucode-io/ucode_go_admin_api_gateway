package api_call_limits

import (
	"context"
	"log"
	"strconv"
	"sync"
	"time"

	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/services"

	"github.com/go-redis/redis/v8"
)

type MetricsConsumer struct {
	rdb            *redis.Client
	companyService services.CompanyServiceI
	flushInterval  time.Duration
	wg             sync.WaitGroup
	cancelFn       context.CancelFunc
}

func NewMetricsConsumer(rdb *redis.Client, companyService services.CompanyServiceI, flushInterval time.Duration) *MetricsConsumer {
	return &MetricsConsumer{
		rdb:            rdb,
		companyService: companyService,
		flushInterval:  flushInterval,
	}
}

func (c *MetricsConsumer) Start(ctx context.Context) {
	innerCtx, cancel := context.WithCancel(ctx)
	c.cancelFn = cancel

	c.wg.Add(1)
	go func() { defer c.wg.Done(); c.dbFlusher(innerCtx) }()

	log.Printf("[Consumer] started: db_flush_interval=%v", c.flushInterval)
}

func (c *MetricsConsumer) Stop() {
	if c.cancelFn != nil {
		c.cancelFn()
	}
	c.wg.Wait()
	log.Println("[Consumer] stopped")
}

func (c *MetricsConsumer) dbFlusher(ctx context.Context) {
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
		keys, next, err := c.rdb.Scan(ctx, cursor, config.KeyUsagePendingPattern, 100).Result()
		if err != nil {
			log.Printf("[Consumer] usage SCAN error: %v", err)
			return
		}

		if len(keys) == 0 {
			return
		}

		for _, key := range keys {
			projectID := key[len(config.KeyUsagePendingPrefix):]

			countStr, err := c.rdb.HGet(ctx, key, config.KeyUsageTotalField).Result()
			if err != nil {
				continue
			}

			count, err := strconv.ParseInt(countStr, 10, 64)
			if err != nil || count <= 0 {
				continue
			}

			_, err = c.companyService.Billing().LogUsage(ctx, &pb.LogUsageRequest{
				ProjectId: projectID,
				Count:     count,
				TimeRange: 3600,
				DateTime:  time.Now().Truncate(time.Hour).Format("2006-01-02 15:00:00"),
			})

			if err != nil {
				log.Printf("[MetricsConsumer] ERROR sending LogUsage for project %s: %v", projectID, err)
				continue
			}

			if err := c.rdb.HIncrBy(ctx, key, config.KeyUsageTotalField, -count).Err(); err != nil {
				log.Printf("[Consumer] HIncrBy error key=%s: %v", key, err)
			}
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}
}
