package v1

import (
	"errors"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

func (h *HandlerV1) Cache(c *gin.Context) {

	var request models.Cache

	err := c.ShouldBindJSON(&request)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	key := request.Key
	projectId := request.ProjectId
	value := request.Value

	var res string

	if request.Method == "GET" {
		data, err := h.redis.Get(c, key, projectId, "u-code")
		if err != nil && err != redis.Nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}

		if err == redis.Nil {
			err := h.redis.SetX(c, key, value, 0, projectId, "u-code")
			if err != nil {
				h.handleResponse(c, status_http.InternalServerError, err.Error())
				return
			}
			res = "Data stored in Redis"
		}
		res = data
	} else if request.Method == "SET" {
		err := h.redis.SetX(c, key, value, 0, projectId, "u-code")
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
	} else if request.Method == "DEL" {
		err := h.redis.Del(c, key, projectId, "u-code")
		if err != nil {
			h.handleResponse(c, status_http.InternalServerError, err.Error())
			return
		}
	} else {
		h.handleResponse(c, status_http.BadRequest, errors.New("invalid method").Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}
