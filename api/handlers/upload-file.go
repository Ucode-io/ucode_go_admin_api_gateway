package handlers

import (
	"context"
	"fmt"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/http"
	https "ucode/ucode_go_api_gateway/api/http"
	"ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"github.com/minio/minio-go/v7/pkg/credentials"

	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

type UploadResponse struct {
	Filename string `json:"filename"`
}

var (
	defaultBucket = "ucode"
)

type File struct {
	File *multipart.FileHeader `form:"file" binding:"required"`
}

type Path struct {
	Filename string `json:"filename"`
	Hash     string `json:"hash"`
}

// Upload godoc
// @ID upload_image
// @Security ApiKeyAuth
// @Router /v1/upload [POST]
// @Summary Upload
// @Description Upload
// @Tags file
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "file"
// @Success 200 {object} http.Response{data=Path} "Path"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) Upload(c *gin.Context) {
	var (
		file File
	)
	err := c.ShouldBind(&file)
	if err != nil {
		h.handleResponse(c, https.BadRequest, err.Error())
		return
	}

	fName, _ := uuid.NewRandom()
	file.File.Filename = strings.ReplaceAll(file.File.Filename, " ", "")
	file.File.Filename = fmt.Sprintf("%s_%s", fName.String(), file.File.Filename)
	dst, _ := os.Getwd()

	minioClient, err := minio.New(h.cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(h.cfg.MinioAccessKeyID, h.cfg.MinioSecretAccessKey, ""),
		Secure: h.cfg.MinioProtocol,
	})
	h.log.Info("info", logger.String("access_key: ",
		h.cfg.MinioAccessKeyID), logger.String("access_secret: ", h.cfg.MinioSecretAccessKey))

	if err != nil {
		h.handleResponse(c, https.BadRequest, err.Error())
		return
	}

	err = c.SaveUploadedFile(file.File, dst+"/"+file.File.Filename)
	if err != nil {
		h.handleResponse(c, https.BadRequest, err.Error())
		return
	}
	splitedContentType := strings.Split(file.File.Header["Content-Type"][0], "/")
	if splitedContentType[0] != "image" && splitedContentType[0] != "video" {
		defaultBucket = "docs"
	}

	_, err = minioClient.FPutObject(
		context.Background(),
		defaultBucket,
		file.File.Filename,
		dst+"/"+file.File.Filename,
		minio.PutObjectOptions{ContentType: file.File.Header["Content-Type"][0]},
	)
	if err != nil {
		h.handleResponse(c, https.BadRequest, err.Error())
		os.Remove(dst + "/" + file.File.Filename)

		return
	}

	os.Remove(dst + "/" + file.File.Filename)

	h.handleResponse(c, https.Created, Path{
		Filename: file.File.Filename,
		Hash:     fName.String(),
	})
}

// UploadFile godoc
// @Security ApiKeyAuth
// @ID upload_file
// @Router /v1/upload-file/{table_slug}/{object_id} [POST]
// @Summary Upload file
// @Description Upload file
// @Tags file
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "file"
// @Param table_slug path string true "table_slug"
// @Param object_id path string true "object_id"
// @Param tags query string false "tags"
// @Success 200 {object} http.Response{data=Path} "Path"
// @Response 400 {object} http.Response{data=string} "Bad Request"
// @Failure 500 {object} http.Response{data=string} "Server Error"
func (h *Handler) UploadFile(c *gin.Context) {
	var (
		file File
	)
	err := c.ShouldBind(&file)
	if err != nil {
		h.handleResponse(c, https.BadRequest, err.Error())
		return
	}

	fileNameForObjectBuilder := file.File.Filename

	fName, _ := uuid.NewRandom()
	file.File.Filename = strings.ReplaceAll(file.File.Filename, " ", "")
	file.File.Filename = fmt.Sprintf("%s_%s", fName.String(), file.File.Filename)
	dst, _ := os.Getwd()

	minioClient, err := minio.New(h.cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(h.cfg.MinioAccessKeyID, h.cfg.MinioSecretAccessKey, ""),
		Secure: h.cfg.MinioProtocol,
	})
	h.log.Info("info", logger.String("access_key: ",
		h.cfg.MinioAccessKeyID), logger.String("access_secret: ", h.cfg.MinioSecretAccessKey))

	if err != nil {
		h.handleResponse(c, https.BadRequest, err.Error())
		return
	}
	err = c.SaveUploadedFile(file.File, dst+"/"+file.File.Filename)
	if err != nil {
		h.handleResponse(c, https.BadRequest, err.Error())
		return
	}
	fileLink := "https://" + h.cfg.MinioEndpoint + "/docs/" + file.File.Filename
	splitedFileName := strings.Split(fileNameForObjectBuilder, ".")
	f, err := os.Stat(dst + "/" + file.File.Filename)
	if err != nil {
		h.handleResponse(c, https.BadRequest, err.Error())
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
		h.handleResponse(c, https.BadRequest, err.Error())
		os.Remove(dst + "/" + file.File.Filename)

		return
	}
	os.Remove(dst + "/" + file.File.Filename)

	tags := []string{}
	if c.Query("tags") != "" {
		tags = strings.Split(c.Query("tags"), ",")
	}
	var requestMap = make(map[string]interface{})
	requestMap["table_slug"] = c.Param("table_slug")
	requestMap["object_id"] = c.Param("object_id")
	requestMap["date"] = time.Now().Format(time.RFC3339)
	requestMap["tags"] = tags
	requestMap["size"] = int32(f.Size())
	requestMap["type"] = splitedFileName[len(splitedFileName)-1]
	requestMap["file_link"] = fileLink
	requestMap["name"] = fileNameForObjectBuilder
	structData, err := helper.ConvertMapToStruct(requestMap)
	if err != nil {
		h.handleResponse(c, https.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, http.Forbidden, err)
		return
	}

	authInfo := h.GetAuthInfo(c)

	_, err = services.ObjectBuilderService().Create(
		context.Background(),
		&object_builder_service.CommonMessage{
			TableSlug: "file",
			Data:      structData,
			ProjectId: authInfo.GetProjectId(),
		},
	)
	if err != nil {
		h.handleResponse(c, https.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, https.Created, Path{
		Filename: file.File.Filename,
		Hash:     fName.String(),
	})
}
