package v1

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	nb "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Upload godoc
// @ID create_file
// @Security ApiKeyAuth
// @Router /v1/files/folder_upload [POST]
// @Summary Upload Folder
// @Description Upload Folder
// @Tags Files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "file"
// @Param folder_name query string true "folder_name"
// @Success 200 {object} status_http.Response{data=Path} "Path"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UploadToFolder(c *gin.Context) {
	var (
		file File
	)

	if file.File != nil {
		h.handleResponse(c, status_http.BadRequest, "file is empty")
		return
	}

	folder_name := c.DefaultQuery("folder_name", "Media")

	err := c.ShouldBind(&file)
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

	var title string = file.File.Filename

	fName, _ := uuid.NewRandom()
	file.File.Filename = strings.ReplaceAll(file.File.Filename, " ", "")
	file.File.Filename = fmt.Sprintf("%s_%s", fName.String(), file.File.Filename)
	object, err := file.File.Open()
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}
	defer object.Close()

	minioClient, err := minio.New(h.baseConf.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(h.baseConf.MinioAccessKeyID, h.baseConf.MinioSecretAccessKey, ""),
		Secure: h.baseConf.MinioProtocol,
	})

	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	_, err = minioClient.PutObject(
		context.Background(),
		resource.ResourceEnvironmentId,
		folder_name+"/"+file.File.Filename,
		object,
		file.File.Size,
		minio.PutObjectOptions{ContentType: file.File.Header["Content-Type"][0]},
	)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		// err = os.Remove(dst + "/" + file.File.Filename)
		// if err != nil {
		// 	h.log.Error("cant remove file")
		// }
		return
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).File().Create(context.Background(), &obs.CreateFileRequest{
			Id:               fName.String(),
			Title:            title,
			Storage:          folder_name,
			FileNameDisk:     file.File.Filename,
			FileNameDownload: title,
			Link:             resource.ResourceEnvironmentId + "/" + folder_name + "/" + file.File.Filename,
			FileSize:         file.File.Size,
			ProjectId:        resource.ResourceEnvironmentId,
		})
		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		h.handleResponse(c, status_http.Created, resp)
		return
	case pb.ResourceType_POSTGRESQL:

		resp, err := services.GoObjectBuilderService().File().Create(context.Background(), &nb.CreateFileRequest{
			Id:               fName.String(),
			Title:            title,
			Storage:          folder_name,
			FileNameDisk:     file.File.Filename,
			FileNameDownload: title,
			Link:             resource.ResourceEnvironmentId + "/" + folder_name + "/" + file.File.Filename,
			FileSize:         file.File.Size,
			ProjectId:        resource.ResourceEnvironmentId,
		})

		if err != nil {
			h.handleResponse(c, status_http.BadRequest, err.Error())
			return
		}

		h.handleResponse(c, status_http.Created, resp)
		return
	}

	// err = os.Remove(dst + "/" + file.File.Filename)
	// if err != nil {
	// 	h.log.Error("cant remove file")
	// }

}

// GetSingleFile godoc
// @Security ApiKeyAuth
// @ID get_file_by_id
// @Router /v1/files/{id} [GET]
// @Summary Get single variable
// @Description Get single variable
// @Tags Files
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 200 {object} status_http.Response{data=obs.File} "FileBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetSingleFile(c *gin.Context) {
	fileID := c.Param("id")

	if !util.IsValidUUID(fileID) {
		h.handleResponse(c, status_http.InvalidArgument, "variable id is an invalid uuid")
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).File().GetSingle(
			context.Background(),
			&obs.FilePrimaryKey{
				ProjectId: resource.ResourceEnvironmentId,
				Id:        fileID,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().File().GetSingle(
			context.Background(),
			&nb.FilePrimaryKey{
				ProjectId: resource.ResourceEnvironmentId,
				Id:        fileID,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)

	}

}

// UpdateFile godoc
// @Security ApiKeyAuth
// @ID update_file
// @Router /v1/files [PUT]
// @Summary Update file
// @Description Update file
// @Tags Files
// @Accept json
// @Produce json
// @Param variable body models.UpdateFileRequest true "UpdateFileRequestBody"
// @Success 200 {object} status_http.Response{data=obs.File} "File data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) UpdateFile(c *gin.Context) {
	var file models.UpdateFileRequest

	err := c.ShouldBindJSON(&file)
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).File().Update(
			context.Background(),
			&obs.File{
				Id:               file.Id,
				Title:            file.Title,
				Description:      file.Description,
				Tags:             file.Tags,
				FileNameDownload: file.FileNameDownload,
				ProjectId:        resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().File().Update(
			context.Background(),
			&nb.File{
				Id:               file.Id,
				Title:            file.Title,
				Description:      file.Description,
				Tags:             file.Tags,
				FileNameDownload: file.FileNameDownload,
				ProjectId:        resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}

}

// DeleteFile godoc
// @Security ApiKeyAuth
// @ID delete_file
// @Router /v1/files/{id} [DELETE]
// @Summary Delete file
// @Description Delete file
// @Tags Files
// @Accept json
// @Produce json
// @Param id path string true "id"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteFile(c *gin.Context) {
	path := c.Param("id")
	if path == "" {
		h.handleResponse(c, status_http.BadRequest, "id is empty")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 20*time.Second)
	defer cancel()

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

	resource, err := h.companyServices.ServiceResource().GetSingle(ctx,
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

	minioClient, err := minio.New(h.baseConf.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(h.baseConf.MinioAccessKeyID, h.baseConf.MinioSecretAccessKey, ""),
		Secure: true,
	})
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	err = minioClient.RemoveObject(ctx, resource.ResourceEnvironmentId, path, minio.RemoveObjectOptions{})
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.NoContent, "Successfully deleted")
}

// DeleteFiles godoc
// @Security ApiKeyAuth
// @ID delete_files
// @Router /v1/files [DELETE]
// @Summary Delete files
// @Description Delete files
// @Tags Files
// @Accept json
// @Produce json
// @Param file body models.FileDeleteRequest true "DeleteFilesRequestBody"
// @Success 204
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) DeleteFiles(c *gin.Context) {

	var file models.FileDeleteRequest
	res := obs.File{}

	err := c.ShouldBindJSON(&file)
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).File().GetSingle(
			context.Background(),
			&obs.FilePrimaryKey{
				ProjectId: resource.ResourceEnvironmentId,
				Id:        file.Objects[0].ObjectId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		res.Id = resp.Id
		res.Title = resp.Title
		res.Description = resp.Description
		res.Tags = resp.Tags
		res.Storage = resp.Storage
		res.FileNameDisk = resp.FileNameDisk
		res.FileNameDownload = resp.FileNameDownload
		res.Link = resp.Link
		res.FileSize = resp.FileSize
		res.ProjectId = resp.ProjectId

	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().File().GetSingle(
			context.Background(),
			&nb.FilePrimaryKey{
				ProjectId: resource.ResourceEnvironmentId,
				Id:        file.Objects[0].ObjectId,
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		res.Id = resp.Id
		res.Title = resp.Title
		res.Description = resp.Description
		res.Tags = resp.Tags
		res.Storage = resp.Storage
		res.FileNameDisk = resp.FileNameDisk
		res.FileNameDownload = resp.FileNameDownload
		res.Link = resp.Link
		res.FileSize = resp.FileSize
		res.ProjectId = resp.ProjectId

	}

	minioClient, err := minio.New(h.baseConf.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(h.baseConf.MinioAccessKeyID, h.baseConf.MinioSecretAccessKey, ""),
		Secure: true,
	})
	if err != nil {
		log.Println(err)
	}

	ctx := context.Background()

	var delete_request []string

	for _, val := range file.Objects {
		delete_request = append(delete_request, val.ObjectId)
		err = minioClient.RemoveObject(ctx, resource.ResourceEnvironmentId, res.Storage+"/"+val.ObjectName, minio.RemoveObjectOptions{})
		if err != nil {
			log.Println(err)
		}
	}

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).File().Delete(
			context.Background(),
			&obs.FileDeleteRequest{
				Ids:       delete_request,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.NoContent, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().File().Delete(
			context.Background(),
			&nb.FileDeleteRequest{
				Ids:       delete_request,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.NoContent, resp)
	}
}

// GetAllFiles godoc
// @Security ApiKeyAuth
// @ID get_file_list
// @Router /v1/files [GET]
// @Summary Get file list
// @Description Get file list
// @Tags Files
// @Accept json
// @Produce json
// @Param filters query obs.GetAllFilesRequest true "filters"
// @Success 200 {object} status_http.Response{data=obs.GetAllFilesRequest} "FileBody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) GetAllFiles(c *gin.Context) {

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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).File().GetList(
			context.Background(),
			&obs.GetAllFilesRequest{
				Search:     c.DefaultQuery("search", ""),
				Sort:       c.DefaultQuery("sort", ""),
				ProjectId:  resource.ResourceEnvironmentId,
				FolderName: c.DefaultQuery("folder_name", ""),
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)
	case pb.ResourceType_POSTGRESQL:
		resp, err := services.GoObjectBuilderService().File().GetList(
			context.Background(),
			&nb.GetAllFilesRequest{
				Search:     c.DefaultQuery("search", ""),
				Sort:       c.DefaultQuery("sort", ""),
				ProjectId:  resource.ResourceEnvironmentId,
				FolderName: c.DefaultQuery("folder_name", ""),
			},
		)

		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		h.handleResponse(c, status_http.OK, resp)

	}
}

// WordTemplate godoc
// @Security ApiKeyAuth
// @ID word_template
// @Router /v1/files/word-template [POST]
// @Summary Word template
// @Description Word template
// @Tags Files
// @Accept json
// @Produce json
// @Param variable body models.CommonMessage true "WordTemplateRequestBody"
// @Success 200 {object} status_http.Response{data=obs.WordTemplateResponse} "File data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV1) WordTemplate(c *gin.Context) {
	var file models.CommonMessage

	err := c.ShouldBindJSON(&file)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	structData, err := helper.ConvertMapToStruct(file.Data)
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

	switch resource.ResourceType {
	case pb.ResourceType_MONGODB:
		resp, err := services.GetBuilderServiceByType(resource.NodeType).File().WordTemplate(
			context.Background(),
			&obs.CommonMessage{
				Data:      structData,
				ProjectId: resource.ResourceEnvironmentId,
			},
		)
		if err != nil {
			h.handleResponse(c, status_http.GRPCError, err.Error())
			return
		}
		h.handleResponse(c, status_http.OK, resp)
	}

}
