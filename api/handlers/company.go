package handlers

import (
	"context"
	"errors"
	"fmt"
	"ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// // CreateCompany godoc
// // @Security ApiKeyAuth
// // @ID create_company
// // @Router /v1/company [POST]
// // @Summary Create Company
// // @Description Create Company
// // @Tags Company
// // @Accept json
// // @Produce json
// // @Param Company body company_service.CreateCompanyRequest true "CompanyCreateRequest"
// // @Success 201 {object} http.Response{data=company_service.CreateCompanyResponse} "Company data"
// // @Response 400 {object} http.Response{data=string} "Bad Request"
// // @Failure 500 {object} http.Response{data=string} "Server Error"
// func (h *Handler) CreateCompany(c *gin.Context) {
// 	var company company_service.CreateCompanyRequest

// 	err := c.ShouldBindJSON(&company)
// 	if err != nil {
// 		h.handleResponse(c, http.BadRequest, err.Error())
// 		return
// 	}

// 	resp, err := h.companyServices.CompanyService().CreateCompany(
// 		context.Background(),
// 		&company_service.CreateCompanyRequest{
// 			Title:       company.Title,
// 			Logo:        company.Logo,
// 			Description: company.Description,
// 		},
// 	)

// 	if err != nil {
// 		h.handleResponse(c, http.GRPCError, err.Error())
// 		return
// 	}

// 	h.handleResponse(c, http.Created, resp)
// }

// GetCompanyById godoc
// @Security ApiKeyAuth
// @ID get_company_by_id
// @Router /v1/company/{company_id} [GET]
// @Summary Get Company by id
// @Description Get Company by id
// @Tags Company
// @Accept json
// @Produce json
// @Param company_id path string true "company_id"
// @Success 200 {object} http.Response{data=company_service.GetCompanyByIdResponse} "Company data"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetCompanyByID(c *gin.Context) {
	companyId := c.Param("company_id")
	resp, err := h.companyServices.CompanyService().GetById(
		context.Background(),
		&company_service.GetCompanyByIdRequest{
			Id: companyId,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetCompanyList godoc
// @Security ApiKeyAuth
// @ID get_company_list
// @Router /v1/company [GET]
// @Summary Get all companies
// @Description Get all companies
// @Tags Company
// @Accept json
// @Produce json
// @Param filters query company_service.GetProjectListRequest true "filters"
// @Success 200 {object} http.Response{data=company_service.GetComanyListResponse} "Company data"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetCompanyList(c *gin.Context) {

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().GetList(
		context.Background(),
		&company_service.GetCompanyListRequest{
			Limit:    int32(limit),
			Offset:   int32(offset),
			Search:   c.DefaultQuery("search", ""),
			ComanyId: c.DefaultQuery("company_id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetCompanyListWithProjects godoc
// @Security ApiKeyAuth
// @ID get_company_list
// @Router /v1/company [GET]
// @Summary Get all companies
// @Description Get all companies
// @Tags Company
// WithProjects@Accept json
// @Produce json
// @Param filters query company_service.GetListWithProjectsRequest true "filters"
// @Success 200 {object} http.Response{data=company_service.GetListWithProjectsResponse} "Company datWithProjectsa"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetCompanyListWithProjects(c *gin.Context) {

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().GetListWithProjects(
		context.Background(),
		&company_service.GetListWithProjectsRequest{
			Limit:    int32(limit),
			Offset:   int32(offset),
			Search:   c.DefaultQuery("search", ""),
			ComanyId: c.DefaultQuery("company_id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateCompany godoc
// @Security ApiKeyAuth
// @ID update_company
// @Router /v1/company/{company_id} [PUT]
// @Summary Update company
// @Description Update company
// @Tags Company
// @Accept json
// @Produce json
// @Param company_id path string true "company_id"
// @Param Company body models.CompanyCreateRequest  true "CompanyCreateRequest"
// @Success 200 {object} http.Response{data=company_service.Company} "Company data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateCompany(c *gin.Context) {
	company_id := c.Param("company_id")

	fmt.Println("company_id", company_id)

	_, err := uuid.Parse(company_id)
	if err != nil {

		h.handleResponse(c, http.BadRequest, errors.New("uuid invalid!!! : "+company_id))
		return
	}
	var company models.CompanyCreateRequest

	err = c.ShouldBindJSON(&company)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	_, err = h.authService.CompanyService().Update(
		c.Request.Context(),
		&auth_service.UpdateCompanyRequest{
			Id:   company_id,
			Name: company.Name,
		},
	)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().Update(
		context.Background(),
		&company_service.Company{
			Id:          company_id,
			Name:        company.Name,
			Logo:        company.Logo,
			Description: company.Description,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteCompany godoc
// @Security ApiKeyAuth
// @ID delete_company
// @Router /v1/company/{company_id} [DELETE]
// @Summary Delete Company
// @Description Delete Company
// @Tags Company
// @Accept json
// @Produce json
// @Param company_id path string true "company_id"
// @Success 204
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteCompany(c *gin.Context) {
	company_id := c.Param("company_id")

	_, err := h.authService.CompanyService().Remove(
		c.Request.Context(),
		&auth_service.CompanyPrimaryKey{Id: company_id},
	)
	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().Delete(
		context.Background(),
		&company_service.DeleteCompanyRequest{
			Id: company_id,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// CreateCompanyProject godoc
// @Security ApiKeyAuth
// @ID create_project
// @Router /v1/company-project [POST]
// @Summary Create Company
// @Description Create Company
// @Tags Company Project
// @Accept json
// @Produce json
// @Param Project body company_service.CreateProjectRequest true "CompanyProjectCreateRequest"
// @Success 201 {object} http.Response{data=models.CompanyProjectCreateResponse} "Project data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) CreateCompanyProject(c *gin.Context) {
	var project company_service.CreateProjectRequest

	err := c.ShouldBindJSON(&project)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ProjectService().Create(
		context.Background(),
		&company_service.CreateProjectRequest{
			Title:        project.Title,
			K8SNamespace: project.K8SNamespace,
			CompanyId:    project.CompanyId,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetPorjectById godoc
// @Security ApiKeyAuth
// @ID get_company_project_id
// @Router /v1/company-project/{project_id} [GET]
// @Summary Get Project By Id
// @Description Get Project By Id
// @Tags Company Project
// @Accept json
// @Produce json
// @Param project_id path string true "project_id"
// @Param company_id query string false "company_id"
// @Success 200 {object} http.Response{data=company_service.Project} "Company data"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetCompanyProjectById(c *gin.Context) {
	projectId := c.Param("project_id")

	resp, err := h.companyServices.ProjectService().GetById(
		context.Background(),
		&company_service.GetProjectByIdRequest{
			ProjectId: projectId,
			CompanyId: c.DefaultQuery("company_id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetProjectList godoc
// @Security ApiKeyAuth
// @ID get_project_list
// @Router /v1/company-project [GET]
// @Summary Get all projects
// @Description Get all projects
// @Tags Company Project
// @Accept json
// @Produce json
// @Param filters query company_service.GetProjectListRequest true "filters"
// @Success 200 {object} http.Response{data=company_service.GetProjectListResponse} "Company data"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetCompanyProjectList(c *gin.Context) {

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.companyServices.ProjectService().GetList(
		context.Background(),
		&company_service.GetProjectListRequest{
			Limit:     int32(limit),
			Offset:    int32(offset),
			Search:    c.DefaultQuery("search", ""),
			CompanyId: c.DefaultQuery("company_id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// UpdateProject godoc
// @Security ApiKeyAuth
// @ID update_project
// @Router /v1/company-project/{project_id} [PUT]
// @Summary Update Project
// @Description Update Project
// @Tags Company Project
// @Accept json
// @Produce json
// @Param project_id path string true "project_id"
// @Param Company body company_service.Project  true "CompanyProjectCreateRequest"
// @Success 200 {object} http.Response{data=company_service.Project} "Company data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UpdateCompanyProject(c *gin.Context) {
	project_id := c.Param("project_id")
	var project company_service.Project

	err := c.ShouldBindJSON(&project)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ProjectService().Update(
		context.Background(),
		&company_service.Project{
			ProjectId:    project_id,
			CompanyId:    project.CompanyId,
			Title:        project.Title,
			K8SNamespace: project.K8SNamespace,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// DeleteCompanyProject godoc
// @Security ApiKeyAuth
// @ID delete_project
// @Router /v1/company-project/{project_id} [DELETE]
// @Summary Delete Project
// @Description Delete Project
// @Tags Company Project
// @Accept json
// @Produce json
// @Param project_id path string true "project_id"
// @Success 204 {object} http.Response{data=string} "Data"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) DeleteCompanyProject(c *gin.Context) {
	project_id := c.Param("project_id")

	resp, err := h.companyServices.ProjectService().Delete(
		context.Background(),
		&company_service.DeleteProjectRequest{
			ProjectId: project_id,
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.NoContent, resp)
}

// AddProjectResource godoc
// @Security ApiKeyAuth
// @ID add_project_resource
// @Router /v1/company/project/resource [POST]
// @Summary Add ProjectResource
// @Description Add ProjectResource
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param ProjectResource body company_service.AddResourceRequest true "ProjectResourceAddRequest"
// @Success 201 {object} http.Response{data=company_service.AddResourceResponse} "ProjectResource data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) AddProjectResource(c *gin.Context) {
	var company company_service.AddResourceRequest

	err := c.ShouldBindJSON(&company)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ProjectService().AddResource(
		c.Request.Context(),
		&company,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// RemoveProjectResource godoc
// @Security ApiKeyAuth
// @ID remove_project_resource
// @Router /v1/company/project/resource [DELETE]
// @Summary Remove ProjectResource
// @Description Remove ProjectResource
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param ProjectResource body company_service.RemoveResourceRequest true "ProjectResourceRemoveRequest"
// @Success 201 {object} http.Response{data=company_service.EmptyProto} "ProjectResource data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) RemoveProjectResource(c *gin.Context) {
	var company company_service.RemoveResourceRequest

	err := c.ShouldBindJSON(&company)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ProjectService().RemoveResource(
		c.Request.Context(),
		&company,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}

// GetResourceById godoc
// @Security ApiKeyAuth
// @ID get_resource_id
// @Router /v1/company/project/resource/{resource_id} [GET]
// @Summary Get Resource by id
// @Description Get Resource by id
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param resource_id path string true "resource_id"
// @Success 200 {object} http.Response{data=company_service.ResourceWithoutPassword} "Resource data"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetResource(c *gin.Context) {

	resp, err := h.companyServices.ProjectService().GetResource(
		context.Background(),
		&company_service.GetResourceRequest{
			Id: c.Param("resource_id"),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// GetResourceList godoc
// @Security ApiKeyAuth
// @ID get_resource_list
// @Router /v1/company/project/resource [GET]
// @Summary Get all companies
// @Description Get all companies
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param filters query company_service.GetReourceListRequest true "filters"
// @Success 200 {object} http.Response{data=company_service.GetReourceListResponse} "Resource data"
// @Response 400 {object} http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) GetResourceList(c *gin.Context) {

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.companyServices.ProjectService().GetReourceList(
		context.Background(),
		&company_service.GetReourceListRequest{
			Limit:     int32(limit),
			Offset:    int32(offset),
			Search:    c.DefaultQuery("search", ""),
			ProjectId: c.DefaultQuery("project_id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.OK, resp)
}

// ReconnectProjectResource godoc
// @Security ApiKeyAuth
// @ID reconnect_project_resource
// @Router /v1/company/project/resource/reconnect [POST]
// @Summary Reconnect ProjectResource
// @Description Reconnect ProjectResource
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param ProjectResource body company_service.ReconnectResourceRequest true "ProjectResourceReconnectRequest"
// @Success 201 {object} http.Response{data=company_service.EmptyProto} "ProjectResource data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) ReconnectProjectResource(c *gin.Context) {
	var company company_service.ReconnectResourceRequest

	fmt.Println("REQUEST HERE")
	err := c.ShouldBindJSON(&company)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ProjectService().ReconnectResource(
		c.Request.Context(),
		&company,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	fmt.Println("REQUEST PASSED")

	h.handleResponse(c, http.Created, resp)
}


// AddProjectResource godoc
// @Security ApiKeyAuth
// @ID add_project_resource_in_ucode
// @Router /v1/company/project/ucode-resource [POST]
// @Summary Add ProjectResource In Ucode Cluster
// @Description Add ProjectResource In Ucode Cluster
// @Tags Company Resource
// @Accept json
// @Produce json
// @Param ProjectResource body company_service.AddResourceInUcodeRequest true "ProjectResourceAddRequest"
// @Success 201 {object} http.Response{data=company_service.AddResourceResponse} "ProjectResource data"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) AddProjectResourceInUcodeCluster(c *gin.Context) {
	var resource company_service.AddResourceInUcodeRequest

	err := c.ShouldBindJSON(&resource)
	if err != nil {
		h.handleResponse(c, http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.ProjectService().AddResourceInUcode(
		c.Request.Context(),
		&resource,
	)

	if err != nil {
		h.handleResponse(c, http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, http.Created, resp)
}