package metaads

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"github.com/stretchr/testify/require"
)

type fakeGraphClient struct {
	lock         sync.Mutex
	accountCalls int
	err          error
}

func (c *fakeGraphClient) fetchAccount(context.Context) (graphAccount, error) {
	c.lock.Lock()
	c.accountCalls++
	c.lock.Unlock()
	return graphAccount{ID: "act_1", Name: "Ads", Currency: "USD", TimezoneName: "Asia/Tashkent"}, c.err
}

func (c *fakeGraphClient) fetchCampaigns(context.Context) ([]graphCampaign, error) {
	return []graphCampaign{{ID: "campaign-1", Name: "Campaign"}}, c.err
}

func (c *fakeGraphClient) fetchAdSets(context.Context) ([]graphAdSet, error) {
	return []graphAdSet{{ID: "adset-1", CampaignID: "campaign-1", Name: "Ad set"}}, c.err
}

func (c *fakeGraphClient) fetchAds(context.Context) ([]graphAd, error) {
	return []graphAd{{ID: "ad-1", AdSetID: "adset-1", CampaignID: "campaign-1", Name: "Ad"}}, c.err
}

func (c *fakeGraphClient) fetchInsights(_ context.Context, _ dashboardQuery, level string, daily bool, breakdowns []string, _ []string) ([]graphInsight, error) {
	if c.err != nil {
		return nil, c.err
	}
	insight := graphInsight{
		CampaignID:       "campaign-1",
		CampaignName:     "Campaign",
		AdSetID:          "adset-1",
		AdSetName:        "Ad set",
		AdID:             "ad-1",
		AdName:           "Ad",
		Spend:            "10",
		Impressions:      "1000",
		Reach:            "800",
		Clicks:           "50",
		InlineLinkClicks: "40",
		DateStart:        "2026-08-01",
		Actions:          []graphAction{{ActionType: "lead", Value: "5"}},
	}
	if len(breakdowns) > 0 {
		insight.Age = "25-34"
		insight.Gender = "male"
	}
	if daily || level != "" {
		return []graphInsight{insight}, nil
	}
	return nil, nil
}

type memoryDashboardCache struct {
	lock     sync.Mutex
	value    []byte
	found    bool
	setCalls int
}

func (c *memoryDashboardCache) Get(context.Context, string) ([]byte, bool, error) {
	c.lock.Lock()
	defer c.lock.Unlock()
	return append([]byte(nil), c.value...), c.found, nil
}

func (c *memoryDashboardCache) Set(_ context.Context, _ string, value []byte, _ time.Duration) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.value = append([]byte(nil), value...)
	c.found = true
	c.setCalls++
	return nil
}

func TestDashboardAlwaysFetchesFreshDataAndUpdatesFallback(t *testing.T) {
	client := &fakeGraphClient{}
	cache := &memoryDashboardCache{found: true, value: []byte(`{"source":"cache","stale":true}`)}
	service := newService(client, cache, time.Minute, "1", []string{"lead"}, nil, logger.NewLogger("test", logger.LevelError))
	query := dashboardQuery{
		Since:      time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Until:      time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC),
		Breakdowns: []string{"age_gender"},
	}

	first, err := service.dashboard(context.Background(), query, false)
	require.NoError(t, err)
	second, err := service.dashboard(context.Background(), query, false)
	require.NoError(t, err)

	require.Equal(t, "meta", first.Source)
	require.False(t, first.Stale)
	require.Equal(t, int64(5), first.KPIs.Leads)
	require.Len(t, first.Campaigns, 1)
	require.Equal(t, "meta", second.Source)
	require.Equal(t, 2, client.accountCalls)
	require.Equal(t, 2, cache.setCalls)
}

func TestDashboardReturnsCacheBeforeRefreshWhenPreferred(t *testing.T) {
	cached := models.MetaAdsDashboardResponse{
		GeneratedAt: "2026-08-01T00:00:00Z",
		Source:      "meta",
		Stale:       true,
		Warning:     "old warning",
		KPIs:        models.MetaAdsMetrics{Spend: 15},
	}
	payload, err := json.Marshal(cached)
	require.NoError(t, err)
	client := &fakeGraphClient{}
	cache := &memoryDashboardCache{found: true, value: payload}
	service := newService(client, cache, time.Minute, "1", []string{"lead"}, nil, logger.NewLogger("test", logger.LevelError))

	response, err := service.dashboard(context.Background(), dashboardQuery{
		Since: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Until: time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC),
	}, true)

	require.NoError(t, err)
	require.Equal(t, "cache", response.Source)
	require.False(t, response.Stale)
	require.Empty(t, response.Warning)
	require.Equal(t, 15.0, response.KPIs.Spend)
	require.Zero(t, client.accountCalls)
	require.Zero(t, cache.setCalls)
}

func TestDashboardReturnsCachedResponseWhenMetaFails(t *testing.T) {
	cached := models.MetaAdsDashboardResponse{GeneratedAt: "2026-08-01T00:00:00Z", Source: "meta", KPIs: models.MetaAdsMetrics{Spend: 15}}
	payload, err := json.Marshal(cached)
	require.NoError(t, err)
	client := &fakeGraphClient{err: errors.New("meta unavailable")}
	cache := &memoryDashboardCache{found: true, value: payload}
	service := newService(client, cache, time.Minute, "1", []string{"lead"}, nil, logger.NewLogger("test", logger.LevelError))

	response, err := service.dashboard(context.Background(), dashboardQuery{
		Since: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Until: time.Date(2026, 8, 7, 0, 0, 0, 0, time.UTC),
	}, false)

	require.NoError(t, err)
	require.Equal(t, "cache", response.Source)
	require.True(t, response.Stale)
	require.Equal(t, 15.0, response.KPIs.Spend)
	require.NotEmpty(t, response.Warning)
}
