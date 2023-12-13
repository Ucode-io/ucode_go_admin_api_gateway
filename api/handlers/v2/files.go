package v2

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"os"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"github.com/gin-gonic/gin"

	"github.com/minio/minio-go/v7"
)

type UploadResponse struct {
	Filename string `json:"filename"`
}

type File struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}

type Path struct {
	Filename string `json:"filename"`
	Hash     string `json:"hash"`
}

// Upload godoc
// @ID v2_upload_image
// @Security ApiKeyAuth
// @Param from-chat query string false "from-chat"
// @Router /v2/files/import [POST]
// @Summary Upload
// @Description Upload
// @Tags file
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "file"
// @Success 200 {object} status_http.Response{data=Path} "Path"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) Upload(c *gin.Context) {
	var (
		file          File
		defaultBucket = "ucode"
	)
	err := c.ShouldBind(&file)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	fName, _ := uuid.NewRandom()
	file.File.Filename = strings.ReplaceAll(file.File.Filename, " ", "")
	file.File.Filename = fmt.Sprintf("%s_%s", fName.String(), file.File.Filename)
	dst, _ := os.Getwd()

	minioClient, err := minio.New(h.baseConf.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(h.baseConf.MinioAccessKeyID, h.baseConf.MinioSecretAccessKey, ""),
		Secure: h.baseConf.MinioProtocol,
	})
	h.log.Info("info", logger.String("MinioEndpoint: ", h.baseConf.MinioEndpoint), logger.String("access_key: ",
		h.baseConf.MinioAccessKeyID), logger.String("access_secret: ", h.baseConf.MinioSecretAccessKey))

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	err = c.SaveUploadedFile(file.File, dst+"/"+file.File.Filename)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	splitedContentType := strings.Split(file.File.Header["Content-Type"][0], "/")
	if splitedContentType[0] != "image" && splitedContentType[0] != "video" {
		defaultBucket = "docs"
	}

	if c.Query("from-chat") == "to_telegram_bot" {
		defaultBucket = "telegram"
	}

	_, err = minioClient.FPutObject(
		context.Background(),
		defaultBucket,
		file.File.Filename,
		dst+"/"+file.File.Filename,
		minio.PutObjectOptions{
			ContentType: file.File.Header["Content-Type"][0]},
	)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		err = os.Remove(dst + "/" + file.File.Filename)
		if err != nil {
			h.log.Error("cant remove file")
		}
		return
	}

	err = os.Remove(dst + "/" + file.File.Filename)
	if err != nil {
		h.log.Error("cant remove file")
	}

	h.handleResponse(c, status_http.Created, Path{
		Filename: file.File.Filename,
		Hash:     fName.String(),
	})
}

// UploadFile godoc
// @Security ApiKeyAuth
// @ID v2_upload_file
// @Router /v1/upload-file/{collection}/{id} [POST]
// @Summary Upload file
// @Description Upload file
// @Tags file
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "file"
// @Param collection path string true "collection"
// @Param id path string true "id"
// @Param tags query string false "tags"
// @Success 200 {object} status_http.Response{data=Path} "Path"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UploadFile(c *gin.Context) {
	var (
		file File
	)
	err := c.ShouldBind(&file)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	fileNameForObjectBuilder := file.File.Filename

	fName, _ := uuid.NewRandom()
	file.File.Filename = strings.ReplaceAll(file.File.Filename, " ", "")
	file.File.Filename = fmt.Sprintf("%s_%s", fName.String(), file.File.Filename)
	dst, _ := os.Getwd()

	minioClient, err := minio.New(h.baseConf.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(h.baseConf.MinioAccessKeyID, h.baseConf.MinioSecretAccessKey, ""),
		Secure: h.baseConf.MinioProtocol,
	})
	h.log.Info("info", logger.String("access_key: ",
		h.baseConf.MinioAccessKeyID), logger.String("access_secret: ", h.baseConf.MinioSecretAccessKey))

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	err = c.SaveUploadedFile(file.File, dst+"/"+file.File.Filename)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	fileLink := "https://" + h.baseConf.MinioEndpoint + "/docs/" + file.File.Filename
	splitedFileName := strings.Split(fileNameForObjectBuilder, ".")
	f, err := os.Stat(dst + "/" + file.File.Filename)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	ContentTypeOfFile := file.File.Header["Content-Type"][0]

	_, err = minioClient.FPutObject(
		context.Background(),
		"docs",
		file.File.Filename,
		dst+"/"+file.File.Filename,
		minio.PutObjectOptions{ContentType: ContentTypeOfFile},
	)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		err = os.Remove(dst + "/" + file.File.Filename)
		if err != nil {
			h.log.Error("cant remove file")
		}

		return
	}
	err = os.Remove(dst + "/" + file.File.Filename)
	if err != nil {
		h.log.Error("cant remove file")
	}

	var tags []string
	if c.Query("tags") != "" {
		tags = strings.Split(c.Query("tags"), ",")
	}
	var requestMap = make(map[string]interface{})
	requestMap["table_slug"] = c.Param("collection")
	requestMap["object_id"] = c.Param("id")
	requestMap["date"] = time.Now().Format(time.RFC3339)
	requestMap["tags"] = tags
	requestMap["size"] = int32(f.Size())
	requestMap["type"] = splitedFileName[len(splitedFileName)-1]
	requestMap["file_link"] = fileLink
	requestMap["name"] = fileNameForObjectBuilder
	structData, err := helper.ConvertMapToStruct(requestMap)
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
		resource.GetProjectId(),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	_, err = services.GetBuilderServiceByType(resource.NodeType).ObjectBuilder().Create(
		context.Background(),
		&object_builder_service.CommonMessage{
			TableSlug: "file",
			Data:      structData,
			ProjectId: resource.ResourceEnvironmentId,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, Path{
		Filename: file.File.Filename,
		Hash:     fName.String(),
	})
}
