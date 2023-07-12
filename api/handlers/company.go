package handlers

import (
	"context"
	"errors"
	"fmt"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/auth_service"
	"ucode/ucode_go_api_gateway/genproto/company_service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateCompany godoc
// @Security ApiKeyAuth
// @ID create_company
// @Router /v1/company [POST]
// @Summary Create Company
// @Description Create Company
// @Tags Company
// @Accept json
// @Produce json
// @Param Company body models.CompanyCreateRequest true "CompanyCreateRequest"
// @Success 201 {object} status_http.Response{data=company_service.Company} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateCompany(c *gin.Context) {
	var company models.CompanyCreateRequest

	err := c.ShouldBindJSON(&company)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	authInfo, _ := h.GetAuthInfo(c)
	companyPKey, err := h.companyServices.CompanyService().Company().Create(context.Background(), &company_service.CreateCompanyRequest{
		Title:       company.Name,
		Logo:        "",
		Description: "",
		OwnerId:     authInfo.GetUserId(),
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	project, err := h.companyServices.CompanyService().Project().Create(context.Background(), &company_service.CreateProjectRequest{
		CompanyId:    companyPKey.GetId(),
		K8SNamespace: "cp-region-type-id",
		Title:        company.Name,
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	_, err = h.companyServices.CompanyService().Environment().Create(
		context.Background(),
		&company_service.CreateEnvironmentRequest{
			ProjectId:    project.GetProjectId(),
			Name:         "Production",
			DisplayColor: "#00FF00",
			Description:  "Production Environment",
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	_, err = h.authService.User().AddUserToProject(context.Background(), &auth_service.AddUserToProjectReq{
		CompanyId: companyPKey.GetId(),
		ProjectId: project.GetProjectId(),
		UserId:    authInfo.GetUserId(),
		RoleId:    authInfo.GetRoleId(),
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, &company_service.Company{
		Id:      companyPKey.GetId(),
		Name:    company.Name,
		OwnerId: authInfo.GetUserId(),
	})
}

// GetCompanyByID godoc
// @Security ApiKeyAuth
// @ID get_company_by_id
// @Router /v1/company/{company_id} [GET]
// @Summary Get Company by id
// @Description Get Company by id
// @Tags Company
// @Accept json
// @Produce json
// @Param company_id path string true "company_id"
// @Success 200 {object} status_http.Response{data=company_service.GetCompanyByIdResponse} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetCompanyByID(c *gin.Context) {
	companyId := c.Param("company_id")
	resp, err := h.companyServices.CompanyService().Company().GetById(
		context.Background(),
		&company_service.GetCompanyByIdRequest{
			Id: companyId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
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
// @Success 200 {object} status_http.Response{data=company_service.GetComanyListResponse} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetCompanyList(c *gin.Context) {

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().Company().GetList(
		context.Background(),
		&company_service.GetCompanyListRequest{
			Limit:    int32(limit),
			Offset:   int32(offset),
			Search:   c.DefaultQuery("search", ""),
			ComanyId: c.DefaultQuery("company_id", ""),
			OwnerId:  c.DefaultQuery("owner_id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
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
// @Success 200 {object} status_http.Response{data=company_service.GetListWithProjectsResponse} "Company datWithProjectsa"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetCompanyListWithProjects(c *gin.Context) {

	limit, err := h.getLimitParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	offset, err := h.getOffsetParam(c)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().Company().GetListWithProjects(
		context.Background(),
		&company_service.GetListWithProjectsRequest{
			Limit:    int32(limit),
			Offset:   int32(offset),
			Search:   c.DefaultQuery("search", ""),
			ComanyId: c.DefaultQuery("company_id", ""),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
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
// @Success 200 {object} status_http.Response{data=company_service.Company} "Company data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) UpdateCompany(c *gin.Context) {
	companyId := c.Param("company_id")

	fmt.Println("company_id", companyId)

	_, err := uuid.Parse(companyId)
	if err != nil {

		h.handleResponse(c, status_http.BadRequest, errors.New("uuid invalid!!! : "+companyId))
		return
	}
	var company models.CompanyCreateRequest

	err = c.ShouldBindJSON(&company)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	_, err = h.authService.Company().Update(
		c.Request.Context(),
		&auth_service.UpdateCompanyRequest{
			Id:   companyId,
			Name: company.Name,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().Company().Update(
		context.Background(),
		&company_service.Company{
			Id:          companyId,
			Name:        company.Name,
			Logo:        company.Logo,
			Description: company.Description,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, resp)
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
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) DeleteCompany(c *gin.Context) {
	companyId := c.Param("company_id")

	_, err := h.authService.Company().Remove(
		c.Request.Context(),
		&auth_service.CompanyPrimaryKey{Id: companyId},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().Company().Delete(
		context.Background(),
		&company_service.DeleteCompanyRequest{
			Id: companyId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, resp)
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
// @Success 201 {object} status_http.Response{data=models.CompanyProjectCreateResponse} "Project data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateCompanyProject(c *gin.Context) {
	var project company_service.CreateProjectRequest

	err := c.ShouldBindJSON(&project)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	resp, err := h.companyServices.CompanyService().Project().Create(
		context.Background(),
		&company_service.CreateProjectRequest{
			Title:        project.Title,
			K8SNamespace: project.K8SNamespace,
			CompanyId:    project.CompanyId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	authInfo, err := h.adminAuthInfo(c)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	_, err = h.authService.User().AddUserToProject(
		c.Request.Context(),
		&auth_service.AddUserToProjectReq{
			UserId:    authInfo.GetUserId(),
			ProjectId: resp.GetProjectId(),
			CompanyId: project.GetCompanyId(),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	_, err = h.companyServices.CompanyService().Environment().Create(
		c.Request.Context(),
		&company_service.CreateEnvironmentRequest{
			ProjectId:    resp.ProjectId,
			Name:         "Production",
			DisplayColor: "#00FF00",
			Description:  "Production Environment",
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}
