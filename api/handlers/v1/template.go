package v1

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	tmp "ucode/ucode_go_api_gateway/genproto/template_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/spf13/cast"
)

// CreateTemplate godoc
// @Security ApiKeyAuth
// @ID create_template
// @Router /v1/template [POST]
// @Summary Create template
// @Description Create template
// @Tags Template
// @Accept json
// @Produce json
// @Param template body tmp.CreateTemplateReq true "CreateTemplateReq"
// @Success 201 {object} status_http.Response{data=tmp.Template} "Template data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) CreateTemplate(c *gin.Context) {
	var template tmp.CreateTemplateReq

	if err := c.ShouldBindJSON(&template); err != nil {
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
		h.handleResponse(c, status_http.BadRequest, "error getting environment id | not valid")
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

	template.ProjectId = projectId.(string)
	template.ResourceId = resource.ResourceEnvironmentId

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	template.CommitId = uuID.String()
	template.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"

	res, err := services.TemplateService().Template().CreateTemplate(
		context.Background(),
		&template,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, res)
}

// GetSingleTemplate godoc
// @Security ApiKeyAuth
// @ID get_single_template
// @Router /v1/template/{template-id} [GET]
// @Summary Get single template
// @Description Get single template
// @Tags Template
// @Accept json
// @Produce json
// @Param template-id path string true "template-id"
// @Success 200 {object} status_http.Response{data=tmp.Template} "TemplateBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetSingleTemplate(c *gin.Context) {
	templateId := c.Param("template-id")

	if !util.IsValidUUID(templateId) {
		h.handleResponse(c, status_http.InvalidArgument, "folder id is an invalid uuid")
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

	res, err := services.TemplateService().Template().GetSingleTemplate(
		context.Background(),
		&tmp.GetSingleTemplateReq{
			Id:         templateId,
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

// UpdateTemplate godoc
// @Security ApiKeyAuth
// @ID update_template
// @Router /v1/template [PUT]
// @Summary Update template
// @Description Update template
// @Tags Template
// @Accept json
// @Produce json
// @Param template body tmp.UpdateTemplateReq true "UpdateTemplateReqBody"
// @Success 200 {object} status_http.Response{data=tmp.Template} "Template data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateTemplate(c *gin.Context) {
	var template tmp.UpdateTemplateReq

	if err := c.ShouldBindJSON(&template); err != nil {
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
		h.handleResponse(c, status_http.BadRequest, "error getting environment id | not valid")
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

	template.ProjectId = projectId.(string)
	template.ResourceId = resource.ResourceEnvironmentId

	uuID, err := uuid.NewRandom()
	if err != nil {
		err = errors.New("error generating new id")
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	template.CommitId = uuID.String()
	template.VersionId = "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88"

	res, err := services.TemplateService().Template().UpdateTemplate(
		context.Background(),
		&template,
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// DeleteTemplate godoc
// @Security ApiKeyAuth
// @ID delete_template
// @Router /v1/template/{template-id} [DELETE]
// @Summary Delete template
// @Description Delete template
// @Tags Template
// @Accept json
// @Produce json
// @Param template-id path string true "template-id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteTemplate(c *gin.Context) {
	templateId := c.Param("template-id")

	if !util.IsValidUUID(templateId) {
		h.handleResponse(c, status_http.InvalidArgument, "view id is an invalid uuid")
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

	res, err := services.TemplateService().Template().DeleteTemplate(
		context.Background(),
		&tmp.DeleteTemplateReq{
			Id:         templateId,
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

// GetListTemplate godoc
// @Security ApiKeyAuth
// @ID get_list_template
// @Router /v1/template [GET]
// @Summary Get List template
// @Description Get List template
// @Tags Template
// @Accept json
// @Produce json
// @Param folder-id query string true "folder-id"
// @Param limit query string false "limit"
// @Param offset query string false "offset"
// @Success 200 {object} status_http.Response{data=tmp.GetListFolderRes} "FolderBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetListTemplate(c *gin.Context) {
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

	res, err := services.TemplateService().Template().GetListTemplate(
		context.Background(),
		&tmp.GetListTemplateReq{
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
			VersionId:  "0bc85bb1-9b72-4614-8e5f-6f5fa92aaa88",
			FolderId:   c.DefaultQuery("folder-id", ""),
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

// ConvertHtmlToPdfV2 godoc
// @Security ApiKeyAuth
// @ID convert_html_to_pdfV2
// @Router /v1/html-to-pdfV2 [POST]
// @Summary Convert html to pdf
// @Description Convert html to pdf
// @Tags Template
// @Accept json
// @Produce json
// @Param template body models.HtmlBody true "HtmlBody"
// @Success 201 {object} status_http.Response{data=tmp.PdfBody} "PdfBody data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) ConvertHtmlToPdfV2(c *gin.Context) {
	var html models.HtmlBody

	if err := c.ShouldBindJSON(&html); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(html.Data)
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

	resp, err := services.TemplateService().Template().ConvertHtmlToPdf(
		context.Background(),
		&tmp.HtmlBody{
			Data:       structData,
			Html:       html.Html,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

func (h *HandlerV1) ConvertTemplateToHtmlV3(c *gin.Context) {
	var html models.HtmlBody

	if err := c.ShouldBindJSON(&html); err != nil {
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

	htmlUrl := cast.ToString(html.Data["html_url"])

	if htmlUrl == "" {
		h.handleResponse(c, status_http.BadRequest, "html_url is required")
		return
	}

	currentDir, err := os.Getwd()
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	htmlFileName := fmt.Sprintf("%s_temp.html", uuid.New().String())
	htmlPath := filepath.Join(currentDir, htmlFileName)

	if err := h.downloadFileFromURL(htmlUrl, htmlPath); err != nil {
		h.handleResponse(c, status_http.InternalServerError, fmt.Sprintf("Failed to download HTML file: %v", err))
		return
	}

	// Clean up downloaded HTML file after processing
	defer func() {
		if err := os.Remove(htmlPath); err != nil {
			log.Printf("Warning: Failed to remove temporary HTML file %s: %v", htmlPath, err)
		}
	}()

	htmlFile, err := os.Open(htmlPath)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	defer htmlFile.Close()

	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	page := wkhtmltopdf.NewPageReader(htmlFile)
	page.EnableLocalFileAccess.Set(true)

	pdfg.AddPage(page)

	pdfg.Dpi.Set(300)
	pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait)
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)

	if err := pdfg.Create(); err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	pdfFileName := fmt.Sprintf("%s.pdf", uuid.New().String())
	outputPDFPath := filepath.Join(currentDir, pdfFileName)

	// Write PDF to local file
	if err := pdfg.WriteFile(outputPDFPath); err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
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

	file, err := os.Open(outputPDFPath)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	_, err = minioClient.PutObject(
		c.Request.Context(),
		resource.ResourceEnvironmentId,
		pdfFileName,
		file,
		fileInfo.Size(),
		minio.PutObjectOptions{ContentType: "application/pdf"},
	)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	if err := os.Remove(outputPDFPath); err != nil {
		log.Printf("Warning: Failed to remove temporary PDF file %s: %v", outputPDFPath, err)
	}

	link := fmt.Sprintf("%s/%s/%s", h.baseConf.MinioEndpoint, resource.ResourceEnvironmentId, pdfFileName)

	response := map[string]any{
		"pdf_url": link,
	}

	h.handleResponse(c, status_http.OK, response)
}

// ConvertTemplateToHtmlV2 godoc
// @Security ApiKeyAuth
// @ID convert_template_to_htmlV2
// @Router /v1/template-to-htmlV2 [POST]
// @Summary Convert template to html
// @Description Convert template to html
// @Tags Template
// @Accept json
// @Produce json
// @Param view body models.HtmlBody true "TemplateBody"
// @Success 201 {object} status_http.Response{data=models.HtmlBody} "HtmlBody data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) ConvertTemplateToHtmlV2(c *gin.Context) {
	var html models.HtmlBody

	if err := c.ShouldBindJSON(&html); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(html.Data)
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

	resp, err := services.TemplateService().Template().ConvertTemplateToHtml(
		context.Background(),
		&tmp.HtmlBody{
			Data:       structData,
			Html:       html.Html,
			ProjectId:  projectId.(string),
			ResourceId: resource.ResourceEnvironmentId,
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

func (h *HandlerV1) downloadFileFromURL(url, filepath string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create GET request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Add headers if needed (for authentication, etc.)
	req.Header.Set("User-Agent", "HTML-to-PDF-Service/1.0")

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response is successful
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: HTTP %d", resp.StatusCode)
	}

	// Create the output file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer out.Close()

	// Copy the response body to the file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	return nil
}
