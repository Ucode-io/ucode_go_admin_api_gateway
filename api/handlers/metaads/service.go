package metaads

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"sync"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"golang.org/x/sync/errgroup"
)

type graphDataClient interface {
	fetchAccount(ctx context.Context) (graphAccount, error)
	fetchCampaigns(ctx context.Context) ([]graphCampaign, error)
	fetchAdSets(ctx context.Context) ([]graphAdSet, error)
	fetchAds(ctx context.Context) ([]graphAd, error)
	fetchInsights(ctx context.Context, query dashboardQuery, level string, daily bool, breakdowns []string, attributionWindows []string) ([]graphInsight, error)
}

type service struct {
	client             graphDataClient
	cache              dashboardCache
	cacheTTL           time.Duration
	accountID          string
	leadActionTypes    map[string]struct{}
	attributionWindows []string
	log                logger.LoggerI
}

func newService(client graphDataClient, cache dashboardCache, cacheTTL time.Duration, accountID string, leadActionTypes, attributionWindows []string, log logger.LoggerI) *service {
	leads := make(map[string]struct{}, len(leadActionTypes))
	for _, actionType := range leadActionTypes {
		if actionType != "" {
			leads[actionType] = struct{}{}
		}
	}
	return &service{
		client:             client,
		cache:              cache,
		cacheTTL:           cacheTTL,
		accountID:          accountID,
		leadActionTypes:    leads,
		attributionWindows: append([]string(nil), attributionWindows...),
		log:                log,
	}
}

func (s *service) dashboard(ctx context.Context, query dashboardQuery, preferCache bool) (models.MetaAdsDashboardResponse, error) {
	startedAt := time.Now()
	key := dashboardCacheKey(s.accountID, query)
	cached, cacheFound, cacheErr := s.cache.Get(ctx, key)
	if cacheErr != nil {
		s.log.Warn("meta ads: read fallback cache failed", logger.Error(cacheErr))
	}
	if preferCache && cacheFound {
		if response, ok := cachedDashboard(cached, false); ok {
			return response, nil
		}
	}

	fresh, err := s.fetchDashboard(ctx, query)
	if err == nil {
		payload, marshalErr := json.Marshal(fresh)
		if marshalErr == nil {
			if setErr := s.cache.Set(ctx, key, payload, s.cacheTTL); setErr != nil {
				s.log.Warn("meta ads: update fallback cache failed", logger.Error(setErr))
			}
		}
		s.log.Info("meta ads: dashboard refreshed",
			logger.String("since", query.Since.Format("2006-01-02")),
			logger.String("until", query.Until.Format("2006-01-02")),
			logger.Int("campaign_count", len(fresh.Campaigns)),
			logger.Int("breakdown_count", len(fresh.Breakdowns)),
			logger.String("latency", time.Since(startedAt).String()),
		)
		return fresh, nil
	}
	if cacheFound {
		if fallback, ok := cachedDashboard(cached, true); ok {
			s.log.Warn("meta ads: serving stale fallback", logger.Error(err))
			return fallback, nil
		}
	}
	return models.MetaAdsDashboardResponse{}, err
}

func cachedDashboard(payload []byte, stale bool) (models.MetaAdsDashboardResponse, bool) {
	var response models.MetaAdsDashboardResponse
	if json.Unmarshal(payload, &response) != nil {
		return models.MetaAdsDashboardResponse{}, false
	}
	response.Source = "cache"
	response.Stale = stale
	response.Warning = ""
	if stale {
		response.Warning = "Meta API is temporarily unavailable; returning the last successful response"
	}
	return response, true
}

func (s *service) account(ctx context.Context) (models.MetaAdsAccount, error) {
	account, err := s.client.fetchAccount(ctx)
	if err != nil {
		return models.MetaAdsAccount{}, err
	}
	return accountModel(account), nil
}

func (s *service) fetchDashboard(ctx context.Context, query dashboardQuery) (models.MetaAdsDashboardResponse, error) {
	var (
		account          graphAccount
		campaigns        []graphCampaign
		adSets           []graphAdSet
		ads              []graphAd
		accountInsights  []graphInsight
		dailyInsights    []graphInsight
		campaignInsights []graphInsight
		adSetInsights    []graphInsight
		adInsights       []graphInsight
	)

	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(4)
	group.Go(func() error { var err error; account, err = s.client.fetchAccount(groupCtx); return err })
	group.Go(func() error { var err error; campaigns, err = s.client.fetchCampaigns(groupCtx); return err })
	group.Go(func() error { var err error; adSets, err = s.client.fetchAdSets(groupCtx); return err })
	group.Go(func() error { var err error; ads, err = s.client.fetchAds(groupCtx); return err })
	group.Go(func() error {
		var err error
		accountInsights, err = s.client.fetchInsights(groupCtx, query, "account", false, nil, s.attributionWindows)
		return err
	})
	group.Go(func() error {
		var err error
		dailyInsights, err = s.client.fetchInsights(groupCtx, query, "account", true, nil, s.attributionWindows)
		return err
	})
	group.Go(func() error {
		var err error
		campaignInsights, err = s.client.fetchInsights(groupCtx, query, "campaign", false, nil, s.attributionWindows)
		return err
	})
	group.Go(func() error {
		var err error
		adSetInsights, err = s.client.fetchInsights(groupCtx, query, "adset", false, nil, s.attributionWindows)
		return err
	})
	group.Go(func() error {
		var err error
		adInsights, err = s.client.fetchInsights(groupCtx, query, "ad", false, nil, s.attributionWindows)
		return err
	})
	if err := group.Wait(); err != nil {
		return models.MetaAdsDashboardResponse{}, err
	}

	breakdowns, unavailableBreakdowns := s.fetchBreakdowns(ctx, query)
	response := models.MetaAdsDashboardResponse{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Source:      "meta",
		Stale:       false,
		DateRange: models.MetaAdsDateRange{
			Since: query.Since.Format("2006-01-02"),
			Until: query.Until.Format("2006-01-02"),
		},
		Account: accountModel(account),
		Attribution: models.MetaAdsAttribution{
			UseAccountSetting: true,
			Windows:           append([]string(nil), s.attributionWindows...),
		},
		KPIs:        aggregateMetrics(accountInsights, s.leadActionTypes),
		Trends:      s.trends(dailyInsights),
		Campaigns:   s.hierarchy(campaigns, adSets, ads, campaignInsights, adSetInsights, adInsights),
		Breakdowns:  breakdowns,
		ActionTypes: observedActionTypes(accountInsights, dailyInsights, campaignInsights, adSetInsights, adInsights),
	}
	if len(unavailableBreakdowns) > 0 {
		response.Warning = "Unavailable breakdowns: " + strings.Join(unavailableBreakdowns, ", ")
	}
	return response, nil
}

func (s *service) fetchBreakdowns(ctx context.Context, query dashboardQuery) ([]models.MetaAdsBreakdownGroup, []string) {
	supported := supportedBreakdowns()
	result := make([]models.MetaAdsBreakdownGroup, len(query.Breakdowns))
	unavailable := make([]string, 0)
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(3)
	var lock sync.Mutex
	for index, name := range query.Breakdowns {
		index, name := index, name
		group.Go(func() error {
			insights, err := s.client.fetchInsights(groupCtx, query, "account", false, supported[name], s.attributionWindows)
			if err != nil {
				s.log.Warn("meta ads: breakdown unavailable", logger.String("breakdown", name), logger.Error(err))
				lock.Lock()
				unavailable = append(unavailable, name)
				lock.Unlock()
				return nil
			}
			rows := make([]models.MetaAdsBreakdownRow, 0, len(insights))
			for _, insight := range insights {
				rows = append(rows, models.MetaAdsBreakdownRow{
					Dimensions: breakdownDimensions(insight, supported[name]),
					Metrics:    aggregateMetrics([]graphInsight{insight}, s.leadActionTypes),
				})
			}
			lock.Lock()
			result[index] = models.MetaAdsBreakdownGroup{Name: name, Rows: rows}
			lock.Unlock()
			return nil
		})
	}
	_ = group.Wait()
	filtered := result[:0]
	for _, item := range result {
		if item.Name != "" {
			filtered = append(filtered, item)
		}
	}
	sort.Strings(unavailable)
	return filtered, unavailable
}
