package v1

import (
	"encoding/json"
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

// Cache godoc
// @Security ApiKeyAuth
// @ID Cache
// @Router /v1/cache [POST]
// @Summary Cache
// @Description Cache
// @Tags Cache
// @Accept json
// @Produce json
// @Param cache body models.CacheRequest true "Cache body"
// @Success 200 {object} status_http.Response{data=models.CacheResponse} "Response body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) Cache(c *gin.Context) {
	//t := time.Now()
	defer func() {
	}()

	var request models.CacheRequest

	err := c.ShouldBindJSON(&request)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	key := request.Key
	projectId := request.ProjectId
	value, err := json.Marshal(request.Value)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	var res = make(map[string]interface{})

	if request.Method == "GET" {
		data, err := h.redis.Get(c, key, "", h.baseConf.UcodeNamespace)
		if err == redis.Nil {
			_ = h.redis.Set(c, key, value, 0, projectId, h.baseConf.UcodeNamespace)
			res["value"] = "Successfully set"
			h.handleResponse(c, status_http.Created, res)
			return
		}
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}

		err = json.Unmarshal([]byte(data), &res)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
		// res["value"] = data
	} else if request.Method == "SET" {
		err := h.redis.Set(c, key, value, 0, projectId, h.baseConf.UcodeNamespace)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
		res["value"] = "Successfully set"
	} else if request.Method == "DEL" {
		err := h.redis.Del(c, key, projectId, h.baseConf.UcodeNamespace)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
		res["value"] = "Successfully deleted"
	} else if request.Method == "DELMANY" {
		err := h.redis.DelMany(c, request.Keys, projectId, h.baseConf.UcodeNamespace)
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
		res["value"] = "Successfully deleted"
	} else {
		h.handleResponse(c, status_http.BadRequest, errors.New("invalid method").Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}
