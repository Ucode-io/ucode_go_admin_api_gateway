package handlers

import (
	"context"
	"errors"
	"fmt"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/types/known/emptypb"
)

// GetAllSections godoc
// @Security ApiKeyAuth
// @ID get_all_sections
// @Router /v1/section [GET]
// @Summary Get all sections
// @Description Get all sections
// @Tags Section
// @Accept json
// @Produce json
// @Param filters query obs.GetAllSectionsRequest true "filters"
// @Success 200 {object} status_http.Response{data=string} "FieldBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetAllSections(c *gin.Context) {

	//tokenInfo := h.GetAuthInfo
	var (
		resp *obs.GetAllSectionsResponse
	)

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	authInfo, err := h.GetAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err.Error())
		return
	}

	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	//resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
	//	context.Background(),
	//	&company_service.GetResEnvByResIdEnvIdRequest{
	//		EnvironmentId: environmentId.(string),
	//		ResourceId:    resourceId.(string),
	//	},
	//)
	//if err != nil {
	//	err = errors.New("error getting resource environment id")
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}

	fmt.Println("\n\n >>>>>>  auth", authInfo)
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		// resp, err = services.BuilderService().Section().GetAll(
		// 	context.Background(),
		// 	&obs.GetAllSectionsRequest{
		// 		TableId:   c.Query("table_id"),
		// 		TableSlug: c.Query("table_slug"),
		// 		RoleId:    authInfo.GetRoleId(),
		// 		ProjectId: resource.ResourceEnvironmentId,
		// 	},
		// )

		// if err != nil {
		// 	h.handleResponse(c, status_http.GRPCError, err.Error())
		// 	return
		// }
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Section().GetAll(
			context.Background(),
			&obs.GetAllSectionsRequest{
				TableId:   c.Query("table_id"),
				TableSlug: c.Query("table_slug"),
				RoleId:    authInfo.GetRoleId(),
				ProjectId: resource.ResourceEnvironmentId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
	}

	h.handleResponse(c, status_http.OK, resp)
}

// UpdateSection godoc
// @Security ApiKeyAuth
// @ID update_section
// @Router /v1/section [PUT]
// @Summary Update section
// @Description Update section
// @Tags Section
// @Accept json
// @Produce json
// @Param table body obs.UpdateSectionsRequest  true "UpdateSectionRequestBody"
// @Success 200 {object} status_http.Response{data=string} "Section data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateSection(c *gin.Context) {
	var (
		sections obs.UpdateSectionsRequest
		resp     *emptypb.Empty
	)

	err := c.ShouldBindJSON(&sections)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	//authInfo, err := h.GetAuthInfo(c)
	//if err != nil {
	//	h.handleResponse(c, status_http.Forbidden, err.Error())
	//	return
	//}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}
	//resourceId, ok := c.Get("resource_id")
	//if !ok {
	//	err = errors.New("error getting resource id")
	//	h.handleResponse(c, status_http.BadRequest, err.Error())
	//	return
	//}

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

	resource, err := services.CompanyService().ServiceResource().GetSingle(
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

	//resourceEnvironment, err := services.CompanyService().Resource().GetResEnvByResIdEnvId(
	//	context.Background(),
	//	&company_service.GetResEnvByResIdEnvIdRequest{
	//		EnvironmentId: environmentId.(string),
	//		ResourceId:    resourceId.(string),
	//	},
	//)
	//if err != nil {
	//	err = errors.New("error getting resource environment id")
	//	h.handleResponse(c, status_http.GRPCError, err.Error())
	//	return
	//}
	sections.ProjectId = resource.ResourceEnvironmentId
	fmt.Println("project id::", resource.ResourceEnvironmentId)

	// commitID, commitGuid, err := h.CreateAutoCommit(c, environmentId.(string), config.COMMIT_TYPE_SECTION)
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, fmt.Errorf("error creating commit: %w", err))
	// 	return
	// }

	// sections.CommitId = commitID
	// sections.CommitGuid = commitGuid
	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		// resp, err = services.BuilderService().Section().Update(
		// 	context.Background(),
		// 	&sections,
		// )

		// if err != nil {
		// 	h.handleResponse(c, status_http.GRPCError, err.Error())
		// 	return
		// }
	case pb.ResourceType_POSTGRESQL:
		resp, err = services.PostgresBuilderService().Section().Update(
			context.Background(),
			&sections,
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

	}

	h.handleResponse(c, status_http.OK, resp)
}
