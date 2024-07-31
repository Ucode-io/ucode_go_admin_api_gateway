package v2

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	tmp "ucode/ucode_go_api_gateway/genproto/template_service"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"
)

// CreateDocxTemplate godoc
// @Security ApiKeyAuth
// @ID create_docx_template
// @Router /v2/docx-template [POST]
// @Summary Create docx template
// @Description Create docx template
// @Tags Template
// @Accept json
// @Produce json
// @Param template body tmp.CreateDocxTemplateReq true "CreateDocxTemplateReq"
// @Success 201 {object} status_http.Response{data=tmp.DocxTemplate} "DocxTemplate data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) CreateDocxTemplate(c *gin.Context) {
	var (
		docxTemplate tmp.CreateDocxTemplateReq
	)

	if err := c.ShouldBindJSON(&docxTemplate); err != nil {
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
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	docxTemplate.ProjectId = projectId.(string)
	docxTemplate.ResourceId = resource.ResourceEnvironmentId
	docxTemplate.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"

	{
		fileName := uuid.New().String() + ".docx"
		f, err := os.Create(fileName)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		if _, err = f.WriteString(""); err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		if err = f.Close(); err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		minioClient, err := minio.New(h.baseConf.MinioEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(h.baseConf.MinioAccessKeyID, h.baseConf.MinioSecretAccessKey, ""),
			Secure: h.baseConf.MinioProtocol,
		})
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		defaultBucket := "docs"
		dst, _ := os.Getwd()

		if _, err = minioClient.FPutObject(context.Background(), defaultBucket, fileName, dst+"/"+fileName, minio.PutObjectOptions{}); err != nil {
			err = os.Remove(dst + "/" + fileName)
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		if err = os.Remove(dst + "/" + fileName); err != nil {
			h.log.Error("Error removing file", logger.Error(err))
		}

		docxTemplate.FileUrl = h.baseConf.MinioEndpoint + "/" + defaultBucket + "/" + fileName
	}

	res, err := services.TemplateService().DocxTemplate().CreateDocxTemplate(
		context.Background(),
		&docxTemplate,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}

// GetSingleDocxTemplate godoc
// @Security ApiKeyAuth
// @ID get_single_docx_template
// @Router /v2/docx-template/{docx-template-id} [GET]
// @Summary Get single docx template
// @Description Get single docx template
// @Tags Template
// @Accept json
// @Produce json
// @Param docx-template-id path string true "docx-template-id"
// @Success 200 {object} status_http.Response{data=tmp.DocxTemplate} "DocxTemplateBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetSingleDocxTemplate(c *gin.Context) {
	docxTemplateId := c.Param("docx-template-id")

	if !util.IsValidUUID(docxTemplateId) {
		h.handleResponse(c, status_http.InvalidArgument, "docx template id is an invalid uuid")
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	res, err := services.TemplateService().DocxTemplate().GetSingleDocxTemplate(
		context.Background(),
		&tmp.GetSingleDocxTemplateReq{
			Id:         docxTemplateId,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// UpdateDocxTemplate godoc
// @Security ApiKeyAuth
// @ID update_docx_template
// @Router /v2/docx-template [PUT]
// @Summary Update docx template
// @Description Update docx template
// @Tags Template
// @Accept json
// @Produce json
// @Param docx_template body tmp.UpdateDocxTemplateReq true "UpdateDocxTemplateReqBody"
// @Success 200 {object} status_http.Response{data=tmp.DocxTemplate} "DocxTemplate data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateDocxTemplate(c *gin.Context) {
	var (
		docxTemplate tmp.UpdateDocxTemplateReq
	)

	if err := c.ShouldBindJSON(&docxTemplate); err != nil {
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
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	docxTemplate.ProjectId = projectId.(string)
	docxTemplate.ResourceId = resource.ResourceEnvironmentId
	docxTemplate.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"

	url := "http://37.27.196.85:8084/cache/files/data/C7xTx2EUiG55llVrRdOIrWNsdfsd5zQ0mCaTY_3451/output.docx/Example%20Document%20Title.docx?md5=SZ0U0Ef1YUF8GqloFLzLfw&expires=1722436016&shardkey=C7xTx2EUiG55llVrRdOIrWNsdfsd5zQ0mCaTY&filename=Example%20Document%20Title.docx"
	docxTemplate.FileUrl = url

	{
		client := &http.Client{}

		req, err := http.NewRequest("GET", docxTemplate.FileUrl, nil)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Fatalf("Failed to download file: status code %d", resp.StatusCode)
		}

		fileName := uuid.New().String() + ".docx"

		out, err := os.Create(fileName)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}
		defer out.Close()

		if _, err = io.Copy(out, resp.Body); err != nil {
			log.Fatalf("Failed to write to file: %v", err)
		}

		fmt.Println("file name in docx", fileName)

		minioClient, err := minio.New(h.baseConf.MinioEndpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(h.baseConf.MinioAccessKeyID, h.baseConf.MinioSecretAccessKey, ""),
			Secure: h.baseConf.MinioProtocol,
		})
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		defaultBucket := "docs"
		dst, _ := os.Getwd()

		if _, err = minioClient.FPutObject(context.Background(), defaultBucket, fileName, dst+"/"+fileName, minio.PutObjectOptions{}); err != nil {
			err = os.Remove(dst + "/" + fileName)
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		if err = os.Remove(dst + "/" + fileName); err != nil {
			h.log.Error("Error removing file", logger.Error(err))
		}

		docxTemplate.FileUrl = h.baseConf.MinioEndpoint + "/" + defaultBucket + "/" + fileName
	}

	res, err := services.TemplateService().DocxTemplate().UpdateDocxTemplate(
		context.Background(),
		&docxTemplate,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// DeleteDocxTemplate godoc
// @Security ApiKeyAuth
// @ID delete_docx_template
// @Router /v2/docx-template/{docx-template-id} [DELETE]
// @Summary Delete docx template
// @Description Delete docx template
// @Tags Template
// @Accept json
// @Produce json
// @Param docx-template-id path string true "docx-template-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) DeleteDocxTemplate(c *gin.Context) {
	docxTemplateId := c.Param("docx-template-id")

	if !util.IsValidUUID(docxTemplateId) {
		h.handleResponse(c, status_http.InvalidArgument, "docx template id is an invalid uuid")
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	res, err := services.TemplateService().DocxTemplate().DeleteDocxTemplate(
		context.Background(),
		&tmp.DeleteDocxTemplateReq{
			Id:         docxTemplateId,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, res)
}

// GetListDocxTemplate godoc
// @Security ApiKeyAuth
// @ID get_list_docx_template
// @Router /v2/docx-template [GET]
// @Summary Get List docx template
// @Description Get List docx template
// @Tags Template
// @Accept json
// @Produce json
// @Param table-slug query string true "table-slug"
// @Param limit query string false "limit"
// @Param offset query string false "offset"
// @Success 200 {object} status_http.Response{data=tmp.GetListDocxTemplateRes} "DocxTemplateBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetListDocxTemplate(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err)
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
			ServiceType:   pb.ServiceType_TEMPLATE_SERVICE,
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

	fmt.Println("resource.ResourceEnvironmentId docx", resource.ResourceEnvironmentId, resource)

	res, err := services.TemplateService().DocxTemplate().GetListDocxTemplate(
		context.Background(),
		&tmp.GetListDocxTemplateReq{
			ProjectId:  resource.ResourceEnvironmentId,
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
			TableSlug:  c.DefaultQuery("table-slug", ""),
			Limit:      int32(limit),
			Offset:     int32(offset),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}
