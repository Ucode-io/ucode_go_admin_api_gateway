package handlers

import (
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
)

// CreateFieldsAndRelations godoc
// @Security ApiKeyAuth
// @ID create_fields_and_relations
// @Router /v1/fields-relations [POST]
// @Summary Create field
// @Description Create field
// @Tags Field
// @Accept json
// @Produce json
// @Param table body obs.CreateFieldsAndRelationsRequest true "Request Body"
// @Success 201 {object} status_http.Response{data=obs.CreateFieldsAndRelationsResponse} "Response Body"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateFieldsAndRelations(c *gin.Context) {

	var request obs.CreateFieldsAndRelationsRequest

	err := c.ShouldBindJSON(&request)
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

	request.ProjectId = resource.ResourceEnvironmentId

	// Creating Fields and relations
	resp, err := services.GetBuilderServiceByType(resource.NodeType).FieldsAndRelations().CreateFieldsAndRelations(c.Request.Context(), &request)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}
