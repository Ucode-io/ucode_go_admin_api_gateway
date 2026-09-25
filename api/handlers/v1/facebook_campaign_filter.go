package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"ucode/ucode_go_api_gateway/api/models"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
)

const facebookAdCampaignCacheTTL = 30 * 24 * time.Hour

// A missing selection keeps existing integrations working. Once a selection
// has been saved, an empty list means no campaigns are allowed.
func (h *HandlerV1) facebookCampaignAllowed(ctx context.Context, resource *pb.ProjectResource, pipeline, adID string) (bool, error) {
	if h.centralRedis == nil {
		return false, fmt.Errorf("campaign selection storage is unavailable")
	}
	key := metaAdsPipelineCampaignsKey(resource.GetProjectId(), resource.GetEnvironmentId(), pipeline)
	body, err := h.centralRedis.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("load campaign selection: %w", err)
	}
	var selected []string
	if err := json.Unmarshal(body, &selected); err != nil {
		return false, fmt.Errorf("parse campaign selection: %w", err)
	}
	if len(selected) == 0 || strings.TrimSpace(adID) == "" {
		return false, nil
	}
	cacheKey := "crm:meta-ads:ad-campaign:" + resource.GetProjectId() + ":" + resource.GetEnvironmentId() + ":" + url.QueryEscape(adID)
	campaignID, err := h.centralRedis.Get(ctx, cacheKey).Result()
	if err == redis.Nil {
		state := models.FacebookOAuthState{ProjectId: resource.GetProjectId(), EnvironmentId: resource.GetEnvironmentId()}
		token, tokenErr := h.getFacebookUserToken(ctx, state)
		if tokenErr != nil {
			return false, fmt.Errorf("resolve campaign token: %w", tokenErr)
		}
		var ad struct {
			CampaignID string `json:"campaign_id"`
		}
		if graphErr := h.facebookGraphGet(ctx, adID, url.Values{"fields": {"campaign_id"}, "access_token": {token}}, &ad); graphErr != nil {
			return false, fmt.Errorf("resolve ad campaign: %w", graphErr)
		}
		campaignID = ad.CampaignID
		if campaignID == "" {
			return false, fmt.Errorf("ad %s has no campaign", adID)
		}
		_ = h.centralRedis.Set(ctx, cacheKey, campaignID, facebookAdCampaignCacheTTL).Err()
	} else if err != nil {
		return false, fmt.Errorf("load ad campaign cache: %w", err)
	}
	for _, id := range selected {
		if id == campaignID {
			return true, nil
		}
	}
	return false, nil
}
