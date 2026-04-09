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

type ConsumerConfig struct {
	DbFlushInterval time.Duration
}

type MetricsConsumer struct {
	rdb            *redis.Client
	companyService services.CompanyServiceI
	cfg            ConsumerConfig
	wg             sync.WaitGroup
	cancelFn       context.CancelFunc
}

func NewMetricsConsumer(rdb *redis.Client, companyService services.CompanyServiceI, cfg ConsumerConfig) *MetricsConsumer {
	return &MetricsConsumer{
		rdb:            rdb,
		companyService: companyService,
		cfg:            cfg,
	}
}

func (c *MetricsConsumer) Start(ctx context.Context) {
	innerCtx, cancel := context.WithCancel(ctx)
	c.cancelFn = cancel

	c.wg.Add(1)
	go func() { defer c.wg.Done(); c.dbFlusher(innerCtx) }()

	log.Printf("[Consumer] started: db_flush_interval=%v", c.cfg.DbFlushInterval)
}

func (c *MetricsConsumer) Stop() {
	if c.cancelFn != nil {
		c.cancelFn()
	}
	c.wg.Wait()
	log.Println("[Consumer] stopped")
}

func (c *MetricsConsumer) dbFlusher(ctx context.Context) {
	ticker := time.NewTicker(c.cfg.DbFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Consumer] dbFlusher stopped")
			return
		case <-ticker.C:
			c.flushApiUsage(ctx)
		}
	}
}

func (c *MetricsConsumer) flushApiUsage(ctx context.Context) {
	var cursor uint64
	for {
		// Ищем все ключи по паттерну "api_usage:pending:*"
		keys, next, err := c.rdb.Scan(ctx, cursor, config.KeyUsagePendingPattern, 100).Result()
		if err != nil {
			log.Printf("[Consumer] usage SCAN error: %v", err)
			return
		}

		if len(keys) == 0 {
			return
		}

		log.Printf("[MetricsConsumer] Found %d projects with pending usage to flush", len(keys))

		for _, key := range keys {
			// Вытаскиваем ProjectID из ключа
			projectID := key[len(config.KeyUsagePendingPrefix):]

			// Получаем накопленное значение
			countStr, err := c.rdb.HGet(ctx, key, config.KeyUsageTotalField).Result()
			if err != nil {
				continue
			}

			count, err := strconv.ParseInt(countStr, 10, 64)
			if err != nil || count <= 0 {
				continue
			}

			// Отправляем в PostgreSQL через микросервис
			_, err = c.companyService.Billing().LogUsage(ctx, &pb.LogUsageRequest{
				ProjectId: projectID,
				Count:     count,
				TimeRange: 3600,
				DateTime:  time.Now().UTC().Truncate(time.Hour).Format("2006-01-02 15:00:00"),
			})

			if err != nil {
				log.Printf("[MetricsConsumer] ERROR sending LogUsage for project %s: %v", projectID, err)
				continue
			} else {
				log.Printf("[MetricsConsumer] SUCCESS LogUsage for project %s: %d calls forwarded to DB", projectID, count)
			}

			// Минусуем только то количество, которое успешно отправили в БД
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
