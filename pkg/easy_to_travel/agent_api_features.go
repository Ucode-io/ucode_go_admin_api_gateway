package easy_to_travel

import (
	"encoding/json"
	"strconv"
	"sync"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/services"

	"github.com/spf13/cast"
)

type FinalResponse struct {
	Metadata   Metadata               `json:"metadata"`
	Results    []Result               `json:"results"`
	Attributes map[string]interface{} `json:"attributes"`
}

type Metadata struct {
	Count  int `json:"count"`
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
	Totals int `json:"total"`
}

type Result struct {
	Id    string `json:"id"`
	Title string `json:"title"`
}

func AgentApiGetFeatures(services services.ServiceManagerI, req []byte) string {

	var (
		response  Response = Response{Attributes: map[string]interface{}{"is_own_data": true}}
		request   NewRequestBody
		returnErr = func(code int, message string, errMess string) string {
			response.Status = "done"
			response.ServerError = errMess
			response.Data = map[string]interface{}{"code": code, "message": message}
			marshaledResponse, err := json.Marshal(response)
			if err != nil {
				return err.Error()
			}
			return string(marshaledResponse)
		}
	)

	err := json.Unmarshal(req, &request)
	if err != nil {
		return returnErr(400, "The JSON is not valid.", err.Error())
	}

	if request.RequestData.Method != "GET" {
		return returnErr(405, "Method not allowed.", "")
	}

	if request.Data["app_id"] == nil {
		return returnErr(401, "The request requires an user authentication.", "")
	}

	var (
		appId                 = cast.ToString(request.Data["app_id"])
		nodeType              = cast.ToString(request.Data["node_type"])
		resourceEnvironmentId = cast.ToString(request.Data["resource_environment_id"])

		apiKeyRequest                = map[string]interface{}{"api_key": appId, "limit": 1}
		offset                   int = 0
		limit                    int = 20
		apiResponse, getListResp GetListClientApiData
		wg                       sync.WaitGroup
		globalErr                error
	)

	if v, ok := request.RequestData.Params["offset"]; ok {
		offset, err = strconv.Atoi(v[0])
		if err != nil {
			return returnErr(404, "Wrong pagination parameters.", err.Error()+"offset")
		}
	}
	if v, ok := request.RequestData.Params["limit"]; ok {
		limit, err = strconv.Atoi(v[0])
		if err != nil {
			return returnErr(404, "Wrong pagination parameters.", err.Error()+"offset")
		}
	}

	service := services.GetBuilderServiceByType(nodeType).ObjectBuilder()
	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		apiResponse, err = GetSlimV2(service, apiKeyRequest, "api_key", resourceEnvironmentId)
		if err != nil {
			globalErr = err
			return
		}
	}(&wg)
	wg.Add(1)

	go func(wg *sync.WaitGroup) {
		defer wg.Done()
		var getFeatureRequest = map[string]interface{}{"offset": offset, "limit": limit}
		getListResp, err = GetSlimV2(service, getFeatureRequest, "place_feature", resourceEnvironmentId)
		if err != nil {
			globalErr = err
			return
		}
	}(&wg)

	wg.Add(1)
	wg.Wait()

	if globalErr != nil {
		return returnErr(500, "Internal server error.", globalErr.Error())
	}

	if len(apiResponse.Data.Response) <= 0 {
		return returnErr(401, "The request requires an user authentication.", "")
	}

	if len(getListResp.Data.Response) == 0 {
		return returnErr(404, "Wrong pagination parameters.", "")
	}

	var status = cast.ToStringSlice(cast.ToStringMap(apiResponse.Data.Response[0])["status"])
	if !helper.Contains(status, "active") {
		return returnErr(403, "The access is not allowed.", "")
	}

	finalRespone := FinalResponse{Attributes: map[string]interface{}{"is_own_data": true}}
	finalRespone.Metadata.Count = len(getListResp.Data.Response)
	finalRespone.Metadata.Offset = offset
	finalRespone.Metadata.Limit = limit
	finalRespone.Metadata.Totals = getListResp.Data.Count
	for _, v := range getListResp.Data.Response {
		var result Result
		result.Id = cast.ToString(v["guid"])
		result.Title = cast.ToString(v["title"])
		finalRespone.Results = append(finalRespone.Results, result)
	}

	response.Data = map[string]interface{}{"metadata": finalRespone.Metadata, "results": finalRespone.Results}
	response.Status = "done"
	responseByte, _ := json.Marshal(response)

	return string(responseByte)
}
