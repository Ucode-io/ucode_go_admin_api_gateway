package easy_to_travel

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
)

var AgentApiPath = map[string]interface{}{
	"/api/v1/agent/airport": map[string]interface{}{
		"paths":         []string{"easy-to-travel-get-airports", "5c72a398-7c33-4c89-8a54-a639e6e8f6d5"},
		"is_cache":      true,
		"function_name": nil,
	},
	"/api/v1/agent/features": map[string]interface{}{
		"paths":         []string{"easy-to-travel-get-features", "95bdcf6b-60e7-43ee-8c59-d57258cdc866"},
		"is_cache":      true,
		"function_name": AgentApiGetFeatures,
	},
	"/api/v1/agent/products": map[string]interface{}{
		"paths":         []string{"easy-to-travel-get-products-agent-swagger", "b693cc12-8551-475f-91d5-4913c1739df4"},
		"is_cache":      true,
		"function_name": nil,
		"delete_params": []string{"startTime", "endTime"},
		"continue":      true,
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

// Request structures
type (
	// Handle request body
	NewRequestBody struct {
		RequestData HttpRequest            `json:"request_data"`
		Auth        AuthData               `json:"auth"`
		Data        map[string]interface{} `json:"data"`
	}

	HttpRequest struct {
		Method  string      `json:"method"`
		Path    string      `json:"path"`
		Headers http.Header `json:"headers"`
		Params  url.Values  `json:"params"`
		Body    []byte      `json:"body"`
	}

	AuthData struct {
		Type string                 `json:"type"`
		Data map[string]interface{} `json:"data"`
	}

	// Function request body >>>>> GET_LIST, GET_LIST_SLIM, CREATE, UPDATE
	Request struct {
		Data map[string]interface{} `json:"data"`
	}

	// Multiple update function request body >>>>> MULTIPLE_UPDATE
	MultipleUpdateRequest struct {
		Data struct {
			Objects []map[string]interface{} `json:"objects"`
		} `json:"data"`
	}
)

// Response structures
type (
	// Create function response body >>>>> CREATE
	Datas struct {
		Data struct {
			Data struct {
				Data map[string]interface{} `json:"data"`
			} `json:"data"`
		} `json:"data"`
	}

	// ClientApiResponse This is get single api response >>>>> GET_SINGLE_BY_ID, GET_SLIM_BY_ID
	ClientApiResponse struct {
		Data ClientApiData `json:"data"`
	}

	ClientApiData struct {
		Data ClientApiResp `json:"data"`
	}

	ClientApiResp struct {
		Response map[string]interface{} `json:"response"`
	}

	Response struct {
		Status      string                 `json:"status"`
		Data        map[string]interface{} `json:"data"`
		Attributes  map[string]interface{} `json:"attributes"`
		ServerError string                 `json:"server_error"`
	}

	// GetListClientApiResponse This is get list api response >>>>> GET_LIST, GET_LIST_SLIM
	GetListClientApiResponse struct {
		Data GetListClientApiData `json:"data"`
	}

	GetListClientApiData struct {
		Data GetListClientApiResp `json:"data"`
	}

	GetListClientApiResp struct {
		Count    int                      `json:"count"`
		Response []map[string]interface{} `json:"response"`
	}

	// ClientApiUpdateResponse This is single update api response >>>>> UPDATE
	ClientApiUpdateResponse struct {
		Status      string `json:"status"`
		Description string `json:"description"`
		Data        struct {
			TableSlug string                 `json:"table_slug"`
			Data      map[string]interface{} `json:"data"`
		} `json:"data"`
	}

	// ClientApiMultipleUpdateResponse This is multiple update api response >>>>> MULTIPLE_UPDATE
	ClientApiMultipleUpdateResponse struct {
		Status      string `json:"status"`
		Description string `json:"description"`
		Data        struct {
			Data struct {
				Objects []map[string]interface{} `json:"objects"`
			} `json:"data"`
		} `json:"data"`
	}
)

func GetSlimV2(service obs.ObjectBuilderServiceClient, request map[string]interface{}, tableSlug, resourceEnvironmentId string) (GetListClientApiData, error) {
	structData, err := helper.ConvertMapToStruct(request)
	if err != nil {
		return GetListClientApiData{}, err
	}

	resp, err := service.GetListSlimV2(
		context.Background(),
		&obs.CommonMessage{
			TableSlug: tableSlug,
			Data:      structData,
			ProjectId: resourceEnvironmentId,
		},
	)
	if err != nil {
		return GetListClientApiData{}, err
	}

	respByte, err := json.Marshal(resp)
	if err != nil {
		return GetListClientApiData{}, err
	}

	var respData GetListClientApiData
	err = json.Unmarshal(respByte, &respData)
	if err != nil {
		return GetListClientApiData{}, err
	}

	return respData, nil
}
