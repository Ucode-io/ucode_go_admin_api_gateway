package api_call_limits

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"ucode/ucode_go_api_gateway/config"

	"github.com/go-redis/redis/v8"
)

type TrackerConfig struct {
	MetricsFlushInterval time.Duration
}

func LoadTrackerConfig(uConf config.Config) TrackerConfig {
	return TrackerConfig{
		MetricsFlushInterval: time.Duration(uConf.AuditMetricsFlushInterval) * time.Second,
	}
}

type Tracker struct {
	cfg     TrackerConfig
	rdb     *redis.Client
	l1Cache sync.Map

	wg       sync.WaitGroup
	cancelFn context.CancelFunc
}

func NewTracker(rdb *redis.Client, cfg TrackerConfig) *Tracker {
	return &Tracker{
		cfg: cfg,
		rdb: rdb,
	}
}

func (t *Tracker) Start(ctx context.Context) {
	innerCtx, cancel := context.WithCancel(ctx)
	t.cancelFn = cancel

	t.wg.Add(1)
	go func() { defer t.wg.Done(); t.metricsWorker(innerCtx) }()

	log.Printf("[Tracker] started: flush_interval=%v", t.cfg.MetricsFlushInterval)
}

func (t *Tracker) Stop() {
	if t.cancelFn != nil {
		t.cancelFn()
	}
	t.wg.Wait()
	t.flushMetricsToRedis()
	log.Println("[Tracker] stopped")
}

func (t *Tracker) metricsWorker(ctx context.Context) {
	ticker := time.NewTicker(t.cfg.MetricsFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.flushMetricsToRedis()
		}
	}
}

func (t *Tracker) flushMetricsToRedis() {
	ctx := context.Background()
	now := time.Now()
	pipe := t.rdb.Pipeline()
	var hasData bool

	t.l1Cache.Range(func(k, v any) bool {
		projectID := k.(string)
		// Swap(0) забирает накопленное и сразу ставит 0 без блокировок
		delta := v.(*atomic.Int64).Swap(0)

		if delta > 0 {
			hasData = true

			// 1. Ключи для real-time статистики (Форматируем по твоим константам)
			minKey := fmt.Sprintf(config.KeyRateMin, projectID, now.Format("2006-01-02-15-04"))
			hourKey := fmt.Sprintf(config.KeyRateHour, projectID, now.Format("2006-01-02-15"))
			dayKey := fmt.Sprintf(config.KeyRateDay, projectID, now.Format("2006-01-02"))

			// Инкремент и TTL (время жизни)
			pipe.IncrBy(ctx, minKey, delta)
			pipe.Expire(ctx, minKey, 15*time.Minute)

			pipe.IncrBy(ctx, hourKey, delta)
			pipe.Expire(ctx, hourKey, 24*time.Hour)

			pipe.IncrBy(ctx, dayKey, delta)
			pipe.Expire(ctx, dayKey, 48*time.Hour)

			// 2. Накопительный ключ для Consumer'а (который пойдет в PostgreSQL)
			usageKey := fmt.Sprintf(config.KeyUsagePending, projectID)
			pipe.HIncrBy(ctx, usageKey, config.KeyUsageTotalField, delta)
		}
		return true
	})

	if hasData {
		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("[Tracker] redis flush error: %v", err)
		}
	}
}
