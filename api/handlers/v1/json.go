package v1

import (
	"context"
	"errors"
	"fmt"
	"math"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// LanguageJson godoc
// @Security ApiKeyAuth
// @ID language_json
// @Router /v2/language-json [GET]
// @Summary Language json
// @Description Language json
// @Tags language
// @Accept json
// @Produce json
// @Success 200 {object} status_http.Response{data=models.CommonMessage} "Response"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetLanguageJson(c *gin.Context) {
	var (
		objectRequest models.CommonMessage
		resp          *obs.CommonMessage
		statusHttp    = status_http.GrpcStatusToHTTP["Ok"]
		object        = make(map[string]interface{})
		languages     = make(map[string]interface{})
	)
	tokenInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}
	if tokenInfo != nil {
		if tokenInfo.Tables != nil {
			object["tables"] = tokenInfo.GetTables()
		}
		object["user_id_from_token"] = tokenInfo.GetUserId()
		object["role_id_from_token"] = tokenInfo.GetRoleId()
		object["client_type_id_from_token"] = tokenInfo.GetClientTypeId()
	}

	object["limit"] = math.MaxInt16

	objectRequest.Data = object
	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
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

	resp, err = services.GetBuilderServiceByType(resource.NodeType).ObjectBuilder().GetList(
		context.Background(),
		&obs.CommonMessage{
			TableSlug: "file",
			Data:      structData,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	languages, err = helper.ConvertStructToResponse(resp.GetData())
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	fmt.Print("\n\n::::::::::::::LANGUAGES:::::::::\n\n", languages, "\n\n")
	statusHttp.CustomMessage = resp.GetCustomMessage()
	h.handleResponse(c, statusHttp, resp)
}
