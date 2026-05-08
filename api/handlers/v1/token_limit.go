package v1

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"ucode/ucode_go_api_gateway/api/models"

	pb "ucode/ucode_go_api_gateway/genproto/company_service"

	"github.com/spf13/cast"
	"golang.org/x/sync/errgroup"
)

type TokenLimitError struct {
	Period string
	Used   int64
	Limit  int64
}

func (e *TokenLimitError) Error() string {
	return fmt.Sprintf("token limit exceeded (%s): used %d / %d", e.Period, e.Used, e.Limit)
}

func (p *ChatProcessor) initTokenBudget(ctx context.Context) {
	if p.mcpUcodeProjectId == "" {
		log.Printf("[TOKEN BUDGET] skipped: no project_id")
		return
	}

	var (
		limitsResp  *pb.GetPricingLimitsResponse
		metricsResp *pb.GetAiTokenUsageMetricsResponse
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		var err error
		limitsResp, err = p.h.companyServices.Billing().GetPricingLimits(gCtx, &pb.GetPricingLimitsRequest{ProjectId: p.mcpUcodeProjectId})
		return err
	})
	g.Go(func() error {
		var err error
		metricsResp, err = p.h.companyServices.Billing().GetAiTokenUsageMetrics(gCtx, &pb.GetAiTokenUsageMetricsRequest{ProjectId: p.mcpUcodeProjectId})
		return err
	})
	if err := g.Wait(); err != nil {
		log.Printf("[TOKEN BUDGET] billing unavailable, skipping limit check: %v", err)
		return
	}

	var snap models.TokenBudgetSnapshot
	for _, l := range limitsResp.GetLimits() {
		log.Printf("[TOKEN BUDGET] limit type=%q name=%q value=%q", l.GetType(), l.GetName(), l.GetValue())
		switch l.GetType() {
		case "tokens_day":
			snap.DayLimit = cast.ToInt64(l.GetValue())
		case "tokens_month":
			snap.MonthLimit = cast.ToInt64(l.GetValue())
		}
	}

	if snap.DayLimit == 0 && snap.MonthLimit == 0 {
		log.Printf("[TOKEN BUDGET] skipped: no token limits configured for project_id=%s", p.mcpUcodeProjectId)
		return
	}

	snap.DayUsed = metricsResp.GetTodayInputTokens() + metricsResp.GetTodayOutputTokens()
	snap.MonthUsed = metricsResp.GetMonthlyInputTokens() + metricsResp.GetMonthlyOutputTokens()

	remain := int64(-1)
	if snap.DayLimit > 0 {
		r := snap.DayLimit - snap.DayUsed
		if remain == -1 || r < remain {
			remain = r
		}
	}
	if snap.MonthLimit > 0 {
		r := snap.MonthLimit - snap.MonthUsed
		if remain == -1 || r < remain {
			remain = r
		}
	}

	p.tokenBudgetEnabled = true
	p.tokenBudgetSnap = snap
	atomic.StoreInt64(&p.tokenBudgetRemain, remain)

	log.Printf("[TOKEN BUDGET] initialized: remain=%d (day %d/%d, month %d/%d)",
		remain, snap.DayUsed, snap.DayLimit, snap.MonthUsed, snap.MonthLimit)
}

func (p *ChatProcessor) checkTokenBudget() error {
	if !p.tokenBudgetEnabled {
		return nil
	}
	if atomic.LoadInt64(&p.tokenBudgetRemain) <= 0 {
		return p.buildTokenLimitError()
	}
	return nil
}

func (p *ChatProcessor) deductTokenBudget(tokens int64) {
	if !p.tokenBudgetEnabled || tokens <= 0 {
		return
	}
	atomic.AddInt64(&p.tokenBudgetRemain, -tokens)
}

func (p *ChatProcessor) buildTokenLimitError() *TokenLimitError {
	snap := p.tokenBudgetSnap
	spent := snap.DayLimit - atomic.LoadInt64(&p.tokenBudgetRemain) - snap.DayUsed
	if spent < 0 {
		spent = 0
	}

	DayUsedNow := snap.DayUsed + spent
	MonthUsedNow := snap.MonthUsed + spent

	// determine which limit is the bottleneck
	period := "day"
	used := DayUsedNow
	limit := snap.DayLimit

	if snap.MonthLimit > 0 && snap.DayLimit > 0 {
		dayRemain := snap.DayLimit - DayUsedNow
		monthRemain := snap.MonthLimit - MonthUsedNow
		if monthRemain < dayRemain {
			period, used, limit = "month", MonthUsedNow, snap.MonthLimit
		}
	} else if snap.MonthLimit > 0 {
		period, used, limit = "month", MonthUsedNow, snap.MonthLimit
	}

	return &TokenLimitError{Period: period, Used: used, Limit: limit}
}

func (p *ChatProcessor) tokenLimitData(err *TokenLimitError) models.TokenLimitData {
	snap := p.tokenBudgetSnap
	spent := snap.DayLimit - atomic.LoadInt64(&p.tokenBudgetRemain) - snap.DayUsed
	if spent < 0 {
		spent = 0
	}
	return models.TokenLimitData{
		Type:       "token_limit_exceeded",
		Period:     err.Period,
		Used:       err.Used,
		Limit:      err.Limit,
		Unit:       "tokens",
		DayUsed:    snap.DayUsed + spent,
		DayLimit:   snap.DayLimit,
		MonthUsed:  snap.MonthUsed + spent,
		MonthLimit: snap.MonthLimit,
	}
}
