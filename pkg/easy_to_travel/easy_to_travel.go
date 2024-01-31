package easy_to_travel

var AgentApiPath = map[string]interface{}{
	"/api/v1/agent/airport": map[string]interface{}{
		"paths":         []string{"easy-to-travel-get-airports", "5c72a398-7c33-4c89-8a54-a639e6e8f6d5"},
		"is_cache":      true,
		"function_name": nil,
	},
	"/api/v1/agent/features": map[string]interface{}{
		"paths":         []string{"easy-to-travel-get-features", "95bdcf6b-60e7-43ee-8c59-d57258cdc866"},
		"is_cache":      true,
		"function_name": nil,
	},
	"/api/v1/agent/products": map[string]interface{}{
		"paths":         []string{"easy-to-travel-get-products-agent-swagger", "b693cc12-8551-475f-91d5-4913c1739df4"},
		"is_cache":      true,
		"function_name": AgentApiGetProduct,
		"delete_params": []string{"startTime", "endTime"},
	},
	"/api/v1/agent/contracts": map[string]interface{}{
		"paths":         []string{"easy-to-travel-get-agent-contracts", "eccfbf65-9d5d-470b-adeb-5b8254aafbca"},
		"is_cache":      true,
		"function_name": nil,
	},
	"/api/v1/agent/order": map[string]interface{}{
		"paths":         []string{"easy-to-travel-order-with-contractid", "c15fa3bf-600b-46d3-8f87-963f5d980619"},
		"is_cache":      false,
		"function_name": nil,
	},
}
