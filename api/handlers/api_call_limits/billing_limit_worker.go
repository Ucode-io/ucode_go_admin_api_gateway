package api_call_limits

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/services"

	"github.com/go-redis/redis/v8"
)

type BillingLimitWorker struct {
	rdb            *redis.Client
	serviceNodes   services.ServiceNodesI
	companyService services.CompanyServiceI
	ucodeNamespace string
	flushInterval  time.Duration
	wg             sync.WaitGroup
	cancelFn       context.CancelFunc
}

func NewBillingLimitWorker(
	rdb *redis.Client,
	serviceNodes services.ServiceNodesI,
	companyService services.CompanyServiceI,
	ucodeNamespace string,
	flushInterval time.Duration,
) *BillingLimitWorker {
	return &BillingLimitWorker{
		rdb:            rdb,
		serviceNodes:   serviceNodes,
		companyService: companyService,
		ucodeNamespace: ucodeNamespace,
		flushInterval:  flushInterval,
	}
}

func (w *BillingLimitWorker) Start(ctx context.Context) {
	innerCtx, cancel := context.WithCancel(ctx)
	w.cancelFn = cancel

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.run(innerCtx)
	}()

	log.Printf("[BillingLimitWorker] started: refresh_interval=%v", w.flushInterval)
}

func (w *BillingLimitWorker) Stop() {
	if w.cancelFn != nil {
		w.cancelFn()
	}
	w.wg.Wait()
	log.Println("[BillingLimitWorker] stopped")
}

func (w *BillingLimitWorker) run(ctx context.Context) {
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.refreshAll(ctx)
		}
	}
}

// billingCtxEntry mirrors billing.billingCacheCtx — same JSON tags, same fields.
// Kept local to avoid a package-level import cycle.
type billingCtxEntry struct {
	EnvId    string `json:"e"`
	FareId   string `json:"f"`
	NodeType string `json:"n"`
}

func (w *BillingLimitWorker) refreshAll(ctx context.Context) {
	var cursor uint64
	for {
		keys, next, err := w.rdb.Scan(ctx, cursor, config.KeyBillingDbLimitPattern, 100).Result()
		if err != nil {
			log.Printf("[BillingLimitWorker] SCAN error: %v", err)
			return
		}

		for _, key := range keys {
			projectId := key[len(config.KeyBillingDbLimitPrefix):]
			w.refreshProject(ctx, projectId)
		}

		cursor = next
		if cursor == 0 {
			break
		}
	}
}

func (w *BillingLimitWorker) refreshProject(ctx context.Context, projectId string) {
	ctxKey := fmt.Sprintf(config.KeyBillingDbCtx, projectId)

	raw, err := w.rdb.Get(ctx, ctxKey).Result()
	if err != nil {
		// Context not yet written (first live check not done) — skip until it is.
		return
	}

	var entry billingCtxEntry
	if err = json.Unmarshal([]byte(raw), &entry); err != nil || entry.EnvId == "" || entry.FareId == "" {
		return
	}

	// Resolve the right service for this project's node type.
	namespace := w.ucodeNamespace
	if entry.NodeType == config.ENTER_PRICE_TYPE {
		namespace = projectId
	}

	srvc, err := w.serviceNodes.Get(namespace)
	if err != nil {
		log.Printf("[BillingLimitWorker] service not found namespace=%s project=%s: %v", namespace, projectId, err)
		return
	}

	usage, err := srvc.GoObjectBuilderService().ObjectBuilder().GetResourceUsage(
		ctx, &nb.GetResourceUsageRequest{ProjectId: entry.EnvId},
	)
	if err != nil {
		log.Printf("[BillingLimitWorker] GetResourceUsage project=%s: %v", projectId, err)
		return
	}

	dbSizeMB := int32(usage.GetDatabaseSize() / 1024 / 1024)

	limitResp, err := w.companyService.Billing().CompareFunction(ctx, &pb.CompareFunctionRequest{
		Type:   config.FARE_DATABASE_SIZE,
		FareId: entry.FareId,
		Count:  dbSizeMB,
	})
	if err != nil {
		log.Printf("[BillingLimitWorker] CompareFunction project=%s: %v", projectId, err)
		return
	}

	val := "1"
	if !limitResp.GetHasAccess() {
		val = "0"
	}

	limitKey := fmt.Sprintf(config.KeyBillingDbLimit, projectId)
	if err = w.rdb.Set(ctx, limitKey, val, 15*time.Minute).Err(); err != nil {
		log.Printf("[BillingLimitWorker] Redis SET project=%s: %v", projectId, err)
	}
}
