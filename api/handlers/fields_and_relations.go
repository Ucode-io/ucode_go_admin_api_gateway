package handlers

import (
	"context"
	"errors"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"github.com/gin-gonic/gin"
)

// CreateField godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @Param Environment-Id header string true "Environment-Id"
// @ID create_field
// @Router /v1/field [POST]
// @Summary Create field
// @Description Create field
// @Tags Field
// @Accept json
// @Produce json
// @Param table body models.CreateFieldRequest true "CreateFieldRequestBody"
// @Success 201 {object} status_http.Response{data=models.Field} "Field data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateFieldsAndRelations(c *gin.Context) {

	var request obs.CreateFieldsAndRelationsRequest

	err := c.ShouldBindJSON(&request)
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

	resourceEnvironment, err := services.ResourceService().GetResEnvByResIdEnvId(
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

	// commitID, commitGuid, err := h.CreateAutoCommit(c, environmentId.(string), config.COMMIT_TYPE_FIELD)
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err))
	// 	return
	// }
	// fmt.Println("create table -- commit_id ---->>", commitID)

	// setting options
	request.Options = &obs.CreateFieldsAndRelationsRequest_Options{
		CommitId:   0,
		CommitGuid: "",
		ProjectId:  resourceEnvironment.GetId(),
	}

	// Creating Fields and relations
	resp, err := services.FieldAndRelationService().CreateFieldsAndRelations(c.Request.Context(), &request)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}
