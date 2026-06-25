package models

// Facebook (Meta) Lead Ads webhook payload models.
// Reference: https://developers.facebook.com/docs/graph-api/webhooks/reference/page/#leadgen

type (
	FacebookLeadWebhookEvent struct {
		Object string                 `json:"object"` // always "page" for Lead Ads
		Entry  []FacebookWebhookEntry `json:"entry"`
	}

	FacebookWebhookEntry struct {
		ID      string                  `json:"id"`   // Page ID
		Time    int64                   `json:"time"` // event time, unix seconds
		Changes []FacebookWebhookChange `json:"changes"`
	}

	FacebookWebhookChange struct {
		Field string                  `json:"field"` // "leadgen" for lead ads
		Value FacebookLeadChangeValue `json:"value"`
	}

	FacebookLeadChangeValue struct {
		LeadgenID   string `json:"leadgen_id"`
		PageID      string `json:"page_id"`
		FormID      string `json:"form_id"`
		AdgroupID   string `json:"adgroup_id"`
		AdID        string `json:"ad_id"`
		CreatedTime int64  `json:"created_time"` // unix seconds
	}
)
