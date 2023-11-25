package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// DynamicReportFormula godoc
// @Security ApiKeyAuth
// @ID dynamicreportformula
// @Router /v1/dynamic-report-formula [GET]
// @Summary Dynamic Report Formula
// @Description Dynamic Report Formula
// @Tags Dynamic-Report
// @Accept json
// @Produce json
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DynamicReportFormula(c *gin.Context) {
	h.handleResponse(c, status_http.OK, struct {
		Data struct {
			Values []string `json:"values"`
		} `json:"data"`
	}{Data: struct {
		Values []string `json:"values"`
	}{Values: config.DynamicReportFormula}})
}

// DynamicReport godoc
// @Security ApiKeyAuth
// @ID dynamicreport
// @Router /v1/dynamic-report [POST]
// @Summary Dynamic Report
// @Description Dynamic Report
// @Tags Dynamic-Report
// @Accept json
// @Produce json
// @Param id query string true "id"
// @Param click_action query bool false "click_action"
// @Param object body models.CommonMessage true "GetListObjectRequestBody"
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "ObjectBody"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DynamicReport(c *gin.Context) {

	var (
		objectRequest    models.CommonMessage
		reportSetting    interface{}
		fromDate, toDate string
	)

	err := c.ShouldBindJSON(&objectRequest)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	if _, ok := objectRequest.Data["from_date"]; ok && cast.ToString(objectRequest.Data["from_date"]) != "" {
		formatFromDate, err := time.Parse("2006-01-02", cast.ToString(objectRequest.Data["from_date"]))
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		fromDate = formatFromDate.Format("2006-01-02")
	}

	if _, ok := objectRequest.Data["to_date"]; ok && cast.ToString(objectRequest.Data["to_date"]) != "" {
		formatToDate, err := time.Parse("2006-01-02", cast.ToString(objectRequest.Data["to_date"]))
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		toDate = formatToDate.Format("2006-01-02")
	}
	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	if c.Query("click_action") == "true" {
		reportSetting = objectRequest.Data["report_setting"]
	} else {
		reportSetting, err = services.GetBuilderServiceByType(resource.NodeType).ReportSetting().GetByIdPivotTemplate(
			context.Background(),
			&obs.PivotTemplatePrimaryKey{
				Id:                    c.Query("id"),
				FromDate:              fromDate,
				ToDate:                toDate,
				ResourceEnvironmentId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	objectRequest.Data["report_setting"] = reportSetting
	response, err := h.DynamicReportHelper(NewRequestBody{
		Data: objectRequest.Data,
	}, services, resource.ResourceEnvironmentId, resource.NodeType)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	} else if response.Status == "error" {
		h.handleResponse(c, status_http.GRPCError, response.Data["message"])
		return
	}

	h.handleResponse(c, status_http.OK, response.Data["response"]) // TODO response.Data["response"]
}

// GetByIdReportSetting godoc
// @Security ApiKeyAuth
// @ID get_by_id_report_setting
// @Router /v1/get-report-setting/{id} [GET]
// @Summary Get Report Setting
// @Description Get Report Setting
// @Tags Report-Setting
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=models.AppReportSetting} "AppReportSetting data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetByIdReportSetting(c *gin.Context) {

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.GetBuilderServiceByType(resource.NodeType).ReportSetting().GetByIdReportSetting(
		context.Background(),
		&obs.AppReportSettingPrimaryKey{
			Id:                    c.Param("id"),
			ResourceEnvironmentId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetListReportSetting godoc
// @Security ApiKeyAuth
// @ID get_list_report_setting
// @Router /v1/get-report-setting [GET]
// @Summary Get List Report Setting
// @Description Get List Report Setting
// @Tags Report-Setting
// @Accept json
// @Produce json
// @Success 200 {object} status_http.Response{data=models.AppReportSetting} "AppReportSetting data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListReportSetting(c *gin.Context) {

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.GetBuilderServiceByType(resource.NodeType).ReportSetting().GetListReportSetting(
		context.Background(),
		&obs.GetListReportSettingRequest{
			ResourceEnvironmentId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpsertReportSetting godoc
// @Security ApiKeyAuth
// @ID upsert_report_setting
// @Router /v1/upsert-report-setting [PUT]
// @Summary Upsert Report Setting
// @Description Upsert Report Setting
// @Tags Report-Setting
// @Accept json
// @Produce json
// @Param relation body object_builder_service.UpsertAppReportSettingRequest  true "UpsertAppReportSettingRequestBody"
// @Success 200 {object} status_http.Response{data=models.AppReportSetting} "AppReportSetting data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpsertReportSetting(c *gin.Context) {
	var (
		upsertReportSettingRequestObs obs.UpsertAppReportSettingRequest
	)

	err := c.ShouldBindJSON(&upsertReportSettingRequestObs)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	upsertReportSettingRequestObs.ResourceEnvironmentId = resource.ResourceEnvironmentId

	resp, err := services.GetBuilderServiceByType(resource.NodeType).ReportSetting().UpsertReportSetting(
		context.Background(),
		&upsertReportSettingRequestObs,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// DeleteReportSetting godoc
// @Security ApiKeyAuth
// @ID delete_report_setting
// @Router /v1/delete-report-setting/{id} [DELETE]
// @Summary Delete Report Setting
// @Description Delete Report Setting
// @Tags Report-Setting
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=models.AppReportSetting} "AppReportSetting data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteReportSetting(c *gin.Context) {

	if !util.IsValidUUID(c.Param("id")) {
		h.handleResponse(c, status_http.BadRequest, "invalid app id")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.GetBuilderServiceByType(resource.NodeType).ReportSetting().DeleteReportSetting(
		context.Background(),
		&obs.AppReportSettingPrimaryKey{
			Id:                    c.Param("id"),
			ResourceEnvironmentId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// SavePivotTemplate godoc
// @Security ApiKeyAuth
// @ID save_pivot_template
// @Router /v1/save-pivot-template [POST]
// @Summary Save Pivot Template
// @Description Save Pivot Template
// @Tags Dynamic-Report
// @Accept json
// @Produce json
// @Param relation body object_builder_service.SavePivotTemplateRequest true "SavePivotTemplateRequestBody"
// @Success 200 {object} status_http.Response{data=object_builder_service.PivotTemplateSetting} "Field data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) SavePivotTemplate(c *gin.Context) {
	var savePivotTemplateRequestObs obs.SavePivotTemplateRequest

	err := c.ShouldBindJSON(&savePivotTemplateRequestObs)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	savePivotTemplateRequestObs.ResourceEnvironmentId = resource.ResourceEnvironmentId
	resp, err := services.GetBuilderServiceByType(resource.NodeType).ReportSetting().SavePivotTemplate(
		context.Background(),
		&savePivotTemplateRequestObs,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetByIdPivotTemplate godoc
// @Security ApiKeyAuth
// @ID get_by_id_pivot_template_setting
// @Router /v1/get-pivot-template-setting/{id} [GET]
// @Summary Get Pivot Template Setting
// @Description Get Pivot Template Setting
// @Tags Dynamic-Report
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=object_builder_service.PivotTemplateSetting} "PivotTemplateSetting data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetByIdPivotTemplate(c *gin.Context) {

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.GetBuilderServiceByType(resource.NodeType).ReportSetting().GetByIdPivotTemplate(
		context.Background(),
		&obs.PivotTemplatePrimaryKey{
			Id:                    c.Param("id"),
			ResourceEnvironmentId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// GetByIdPivotTemplate godoc
// @Security ApiKeyAuth
// @ID get_list_pivot_template_setting
// @Router /v1/get-pivot-template-setting [GET]
// @Summary Get List Pivot Template Setting
// @Description Get List Pivot Template Setting
// @Tags Dynamic-Report
// @Accept json
// @Produce json
// @Param status query string false "status" Enums(SAVED,HISTORY)
// @Success 200 {object} status_http.Response{data=object_builder_service.GetListPivotTemplateResponse} "GetListPivotTemplateResponse data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetListPivotTemplate(c *gin.Context) {

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.GetBuilderServiceByType(resource.NodeType).ReportSetting().GetListPivotTemplate(
		context.Background(),
		&obs.GetListPivotTemplateRequest{
			Status:                c.DefaultQuery("status", "SAVED"),
			ResourceEnvironmentId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	fmt.Println("test 12")

	h.handleResponse(c, status_http.OK, resp)
}

// UpsertPivotTemplate godoc
// @Security ApiKeyAuth
// @ID upsert_pivot_template
// @Router /v1/upsert-pivot-template [PUT]
// @Summary Upsert Pivot Template
// @Description Upsert Pivot Template
// @Tags Dynamic-Report
// @Accept json
// @Produce json
// @Param relation body object_builder_service.PivotTemplateSetting true "UpsertPivotTemplateRequestBody"
// @Success 200 {object} status_http.Response{data=object_builder_service.PivotTemplateSetting} "Field data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpsertPivotTemplate(c *gin.Context) {
	var upsertTemplateRequestObs obs.PivotTemplateSetting

	err := c.ShouldBindJSON(&upsertTemplateRequestObs)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	upsertTemplateRequestObs.ResourceEnvironmentId = resource.ResourceEnvironmentId
	resp, err := services.GetBuilderServiceByType(resource.NodeType).ReportSetting().UpsertPivotTemplate(
		context.Background(),
		&upsertTemplateRequestObs,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}

// RemovePivotTemplate godoc
// @Security ApiKeyAuth
// @ID remove_pivot_template
// @Router /v1/remove-pivot-template/{id} [DELETE]
// @Summary Remove Pivot Template
// @Description Remove Pivot Template
// @Tags Dynamic-Report
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=models.Empty} "Empty data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) RemovePivotTemplate(c *gin.Context) {

	id := c.Param("id")
	if !util.IsValidUUID(id) {
		h.handleResponse(c, status_http.InvalidArgument, "id is an invalid uuid")
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := services.GetBuilderServiceByType(resource.NodeType).ReportSetting().RemovePivotTemplate(
		context.Background(),
		&obs.RemovePivotTemplateSettingRequest{
			Id:                    id,
			ResourceEnvironmentId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
}
