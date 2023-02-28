package handlers

import (
	"context"
	"errors"
	"fmt"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/genproto/integration_service_v2"

	"github.com/gin-gonic/gin"
)

// generate-payze-link godoc
// @ID generate-payze-link
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @Router /generate-payze-link [POST]
// @Summary Generate IntegrationPayze
// @Description Generate IntegrationPayze
// @Tags IntegrationPayze
// @Accept json
// @Produce json
// @Param Integration body integration_service_v2.PayzeLinkRequest true "PayzeLinkRequestBody"
// @Success 201 {object} status_http.Response{data=integration_service_v2.PayzeLinkResponse} "Generate Card Integration data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GeneratePayzeLink(c *gin.Context) {
	var payze integration_service_v2.PayzeLinkRequest

	fmt.Println("GeneratePayzeLink")

	err := c.ShouldBindJSON(&payze)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")

		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	payze.ProjectId = resourceEnvironment.GetId()

	resp, err := services.IntegrationPayzeService().IntegrationPayzeService().GeneratePayzeLink(
		c.Request.Context(),
		&payze,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// @Security ApiKeyAuth
// PayzeSaveCard godoc
// @ID PayzeSaveCard
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @Router /payze-save-card [POST]
// @Summary SaveCard IntegrationPayze
// @Description SaveCard IntegrationPayze
// @Tags IntegrationPayze
// @Accept json
// @Produce json
// @Param Integration body integration_service_v2.PayzeLinkRequest true "PayzeLinkRequestBody"
// @Success 201 {object} status_http.Response{data=integration_service_v2.PayzeLinkResponseSaveCard} "Save Card Integration data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) PayzeSaveCard(c *gin.Context) {
	var payze integration_service_v2.PayzeLinkRequest

	err := c.ShouldBindJSON(&payze)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	resourceId, ok := c.Get("resource_id")
	if !ok {
		err = errors.New("error getting resource id")
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok {
		err = errors.New("error getting environment id")
		h.handleResponse(c, status_http.BadRequest, errors.New("cant get environment_id"))
		return
	}

	resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
		context.Background(),
		&company_service.GetResEnvByResIdEnvIdRequest{
			EnvironmentId: environmentId.(string),
			ResourceId:    resourceId.(string),
		},
	)
	if err != nil {
		err = errors.New("error getting resource environment id")
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	payze.ProjectId = resourceEnvironment.GetId()

	resp, err := services.IntegrationPayzeService().IntegrationPayzeService().GeneratePayzeLink(
		c.Request.Context(),
		&payze,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}
