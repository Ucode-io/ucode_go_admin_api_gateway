package v2

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
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
// @Param template body nb.CreateDocxTemplateRequest true "CreateDocxTemplateReq"
// @Success 201 {object} status_http.Response{data=nb.DocxTemplate} "DocxTemplate data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) CreateDocxTemplate(c *gin.Context) {
	var (
		docxTemplate nb.CreateDocxTemplateRequest
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

	docxTemplate.ResourceId = resource.GetResourceEnvironmentId()

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

	fileUUID := uuid.New().String()
	docxFileName := fileUUID + ".docx"
	pdfFileName := fileUUID + ".pdf"

	if docxTemplate.FileUrl != "" {
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
			h.handleResponse(c, status_http.GRPCError, "error getting docx")
			return
		}

		out, err := os.Create(docxFileName)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}
		defer out.Close()

		if _, err = io.Copy(out, resp.Body); err != nil {
			h.handleResponse(c, status_http.GrpcStatusToHTTP["Internal"], err.Error())
		}

		dst, _ := os.Getwd()

		fileData, err := ioutil.ReadFile(dst + "/" + docxFileName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading saved document"})
			return
		}
		base64FileData := base64.StdEncoding.EncodeToString(fileData)

		payload := map[string]interface{}{
			"Parameters": []map[string]interface{}{
				{
					"Name": "File",
					"FileValue": map[string]interface{}{
						"Name": "output.docx",
						"Data": base64FileData,
					},
				},
				{
					"Name":  "StoreFile",
					"Value": true,
				},
			},
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error marshaling payload"})
			return
		}

		convertResp, err := http.Post(
			config.ConvertDocxToPdfUrl+config.ConvertDocxToPdfSecret,
			"application/json",
			bytes.NewBuffer(payloadBytes),
		)
		if err != nil || convertResp.StatusCode != http.StatusOK {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error converting document to PDF error: %v", err)})
			return
		}
		defer convertResp.Body.Close()

		var convertApiResponse models.ConvertAPIResponse
		if err = json.NewDecoder(convertResp.Body).Decode(&convertApiResponse); err != nil || len(convertApiResponse.Files) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing conversion response"})
			return
		}

		pdfUrl := ""
		if len(convertApiResponse.Files) > 0 {
			pdfUrl = convertApiResponse.Files[0].Url
		}

		req, err = http.NewRequest("GET", pdfUrl, nil)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36")

		resp, err = client.Do(req)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			h.handleResponse(c, status_http.BadRequest, "error getting pdf")
			return
		}

		pdfOut, err := os.Create(pdfFileName)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		if _, err = io.Copy(pdfOut, resp.Body); err != nil {
			h.handleResponse(c, status_http.GrpcStatusToHTTP["Internal"], err.Error())
			return
		}

		if err = pdfOut.Close(); err != nil {
			h.handleResponse(c, status_http.GrpcStatusToHTTP["Internal"], err.Error())
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

		if _, err = minioClient.FPutObject(context.Background(), defaultBucket, docxFileName, dst+"/"+docxFileName, minio.PutObjectOptions{}); err != nil {
			err = os.Remove(dst + "/" + docxFileName)
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		if err = os.Remove(dst + "/" + docxFileName); err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
		}

		if _, err = minioClient.FPutObject(context.Background(), defaultBucket, pdfFileName, dst+"/"+pdfFileName, minio.PutObjectOptions{}); err != nil {
			err = os.Remove(dst + "/" + pdfFileName)
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		if err = os.Remove(dst + "/" + pdfFileName); err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
		}

		docxTemplate.FileUrl = h.baseConf.MinioEndpoint + "/" + defaultBucket + "/" + docxFileName
		docxTemplate.PdfUrl = h.baseConf.MinioEndpoint + "/" + defaultBucket + "/" + pdfFileName
	}

	res, err := services.GoObjectBuilderService().DocxTemplate().Create(
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
// @Success 200 {object} status_http.Response{data=nb.DocxTemplate} "DocxTemplateBody"
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

	res, err := services.GoObjectBuilderService().DocxTemplate().GetByID(
		context.Background(),
		&nb.DocxTemplatePrimaryKey{
			Id:         docxTemplateId,
			ProjectId:  projectId.(string),
			ResourceId: resource.GetResourceEnvironmentId(),
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
// @Param docx_template body nb.DocxTemplate true "UpdateDocxTemplateReqBody"
// @Success 200 {object} status_http.Response{data=nb.DocxTemplate} "DocxTemplate data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UpdateDocxTemplate(c *gin.Context) {
	var (
		docxTemplate nb.DocxTemplate
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

	docxTemplate.ResourceId = resource.GetResourceEnvironmentId()

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

	fileUUID := uuid.New().String()
	docxFileName := fileUUID + ".docx"
	pdfFileName := fileUUID + ".pdf"

	if docxTemplate.FileUrl != "" {
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
			h.handleResponse(c, status_http.BadRequest, "error getting docx")
			return
		}

		out, err := os.Create(docxFileName)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}
		defer out.Close()

		if _, err = io.Copy(out, resp.Body); err != nil {
			h.handleResponse(c, status_http.GrpcStatusToHTTP["Internal"], err.Error())
			return
		}

		dst, _ := os.Getwd()

		fileData, err := ioutil.ReadFile(dst + "/" + docxFileName)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reading saved document"})
			return
		}
		base64FileData := base64.StdEncoding.EncodeToString(fileData)

		payload := map[string]interface{}{
			"Parameters": []map[string]interface{}{
				{
					"Name": "File",
					"FileValue": map[string]interface{}{
						"Name": "output.docx",
						"Data": base64FileData,
					},
				},
				{
					"Name":  "StoreFile",
					"Value": true,
				},
			},
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error marshaling payload"})
			return
		}

		convertResp, err := http.Post(
			config.ConvertDocxToPdfUrl+config.ConvertDocxToPdfSecret,
			"application/json",
			bytes.NewBuffer(payloadBytes),
		)
		if err != nil || convertResp.StatusCode != http.StatusOK {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Error converting document to PDF error: %v", err)})
			return
		}
		defer convertResp.Body.Close()

		var convertApiResponse models.ConvertAPIResponse
		if err = json.NewDecoder(convertResp.Body).Decode(&convertApiResponse); err != nil || len(convertApiResponse.Files) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error parsing conversion response"})
			return
		}

		pdfUrl := ""
		if len(convertApiResponse.Files) > 0 {
			pdfUrl = convertApiResponse.Files[0].Url
		}

		req, err = http.NewRequest("GET", pdfUrl, nil)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.36")

		resp, err = client.Do(req)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			h.handleResponse(c, status_http.BadRequest, "error getting pdf")
			return
		}

		pdfOut, err := os.Create(pdfFileName)
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		if _, err = io.Copy(pdfOut, resp.Body); err != nil {
			h.handleResponse(c, status_http.GrpcStatusToHTTP["Internal"], err.Error())
			return
		}

		if err = pdfOut.Close(); err != nil {
			h.handleResponse(c, status_http.GrpcStatusToHTTP["Internal"], err.Error())
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

		if _, err = minioClient.FPutObject(context.Background(), defaultBucket, docxFileName, dst+"/"+docxFileName, minio.PutObjectOptions{}); err != nil {
			err = os.Remove(dst + "/" + docxFileName)
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		if err = os.Remove(dst + "/" + docxFileName); err != nil {
			h.log.Error("Error removing file", logger.Error(err))
		}

		if _, err = minioClient.FPutObject(context.Background(), defaultBucket, pdfFileName, dst+"/"+pdfFileName, minio.PutObjectOptions{}); err != nil {
			err = os.Remove(dst + "/" + pdfFileName)
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		if err = os.Remove(dst + "/" + pdfFileName); err != nil {
			h.log.Error("Error removing file", logger.Error(err))
		}

		docxTemplate.FileUrl = h.baseConf.MinioEndpoint + "/" + defaultBucket + "/" + docxFileName
		docxTemplate.PdfUrl = h.baseConf.MinioEndpoint + "/" + defaultBucket + "/" + pdfFileName
	}

	res, err := services.GoObjectBuilderService().DocxTemplate().Update(
		context.Background(),
		&docxTemplate,
	)

	if err != nil {
		h.log.Error("error in update docx template", logger.Error(err))
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

	res, err := services.GoObjectBuilderService().DocxTemplate().Delete(
		context.Background(),
		&nb.DocxTemplatePrimaryKey{
			Id:         docxTemplateId,
			ProjectId:  projectId.(string),
			ResourceId: resource.GetResourceEnvironmentId(),
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
// @Success 200 {object} status_http.Response{data=nb.GetAllDocxTemplateResponse} "DocxTemplateBody"
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

	res, err := services.GoObjectBuilderService().DocxTemplate().GetAll(
		context.Background(),
		&nb.GetAllDocxTemplateRequest{
			ProjectId:  projectId.(string),
			TableSlug:  c.DefaultQuery("table-slug", ""),
			Limit:      int32(limit),
			Offset:     int32(offset),
			ResourceId: resource.GetResourceEnvironmentId(),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}

// GenerateDocxToPdf godoc
// @Security ApiKeyAuth
// @ID generate_docx_to_pdf
// @Router /v2/docx-template/convert/pdf [POST]
// @Summary Generate PDF from docx template
// @Description Generate PDF from docx template
// @Tags Template
// @Accept json
// @Produce json
// @Param link query string true "link"
// @Param request body models.DocxTemplateVariables true "Variables"
// @Success 200 {object} status_http.Response{data=string} "Success"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GenerateDocxToPdf(c *gin.Context) {
	link := c.Query("link")
	if link == "" {
		h.handleResponse(c, status_http.InvalidArgument, "link is required")
		return
	}

	request := models.DocxTemplateVariables{
		Data: make(map[string]interface{}),
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		h.handleResponse(c, status_http.BadRequest, "invalid body data")
		return
	}

	additionalFields := make(map[string]interface{})

	for key, value := range request.Data {
		if strings.Contains(key, "_id") {
			additionalFields[key] = value
		}
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

	res, err := services.GoObjectBuilderService().Relation().GetAll(c.Request.Context(), &nb.GetAllRelationsRequest{
		TableSlug: request.TableSlug,
		Limit:     100,
		Offset:    0,
		ProjectId: resource.ResourceEnvironmentId,
	})
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	var (
		tableSlugs = make([]string, 0)
	)

	for _, relation := range res.GetRelations() {

		if relation.GetTableTo().GetSlug() != request.TableSlug {
			tableSlugs = append(tableSlugs, relation.GetTableTo().GetSlug())
		} else if relation.GetTableFrom().GetSlug() != request.TableSlug {
			tableSlugs = append(tableSlugs, relation.GetTableFrom().GetSlug())
		}
	}

	objectRequest := models.CommonMessage{
		Data: map[string]interface{}{
			"limit":                                 10,
			fmt.Sprintf("%s_id", request.TableSlug): request.ID,
			"table_slugs":                           tableSlugs,
			"with_relations":                        true,
			"additional_fields":                     additionalFields,
		},
	}

	structData, err := helper.ConvertMapToStruct(objectRequest.Data)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	respGetAll, err := services.GoObjectBuilderService().ObjectBuilder().GetAllForDocx(
		context.Background(),
		&nb.CommonMessage{
			TableSlug: request.TableSlug,
			Data:      structData,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	mapV2, err := helper.ConvertStructToMap(respGetAll.Data)
	if err != nil {
		h.log.Error("error converting struct to map resp to respNew", logger.Error(err))
	}

	jsM, _ := json.Marshal(mapV2)

	req, err := http.NewRequest(http.MethodPost, config.NodeDocxConvertToPdfServiceUrl, nil)
	if err != nil {
		h.log.Error("error in 1 docx gen", logger.Error(err))
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	query := req.URL.Query()
	query.Set("link", link)
	query.Set("data", string(jsM))
	//query.Set("relations", string(jsRelations))
	req.URL.RawQuery = query.Encode()

	client := http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		h.log.Error("error in 2 docx gen", logger.Error(err))
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		js, _ := json.Marshal(resp.Body)
		h.log.Error("error in 3 docx gen", logger.Error(err), logger.Int("resp status", resp.StatusCode), logger.Any("resp", string(js)))
		h.handleResponse(c, status_http.InternalServerError, err)
		return
	}

	c.Header("Content-Disposition", "attachment; filename=file.pdf")
	c.Header("Content-Type", resp.Header.Get("Content-Type"))

	_, err = io.Copy(c.Writer, resp.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to send file")
		return
	}

	c.Status(http.StatusOK)
}

// GetAllFieldsDocxTemplate godoc
// @Security ApiKeyAuth
// @ID get_all_fields_docx_template
// @Router /v2/docx-template/fields/list [GET]
// @Summary Get All fields docx template
// @Description Get All fields docx template
// @Tags Template
// @Accept json
// @Produce json
// @Param table-slug query string true "table-slug"
// @Success 200 {object} status_http.Response{data=nb.CommonMessage} "DocxTemplateBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) GetAllFieldsDocxTemplate(c *gin.Context) {
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

	res, err := services.GoObjectBuilderService().ObjectBuilder().GetAllFieldsForDocx(
		context.Background(),
		&nb.CommonMessage{
			TableSlug: c.DefaultQuery("table-slug", ""),
			ProjectId: projectId.(string),
		},
	)

	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.handleResponse(c, status_http.OK, res)
}
