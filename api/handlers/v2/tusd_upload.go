package v2

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	tusd "github.com/tus/tusd/pkg/handler"
	"github.com/tus/tusd/pkg/s3store"
)

// UploadFile godoc
// @Security ApiKeyAuth
// @ID v2_upload_file
// @Router /v2/upload-file/{collection}/{id} [POST]
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
func (h *HandlerV2) MovieUpload() *tusd.Handler {
	// var (
	// 	file File
	// )
	// err := c.ShouldBind(&file)
	// if err != nil {
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }

	// fileNameForObjectBuilder := file.File.Filename

	// fName, _ := uuid.NewRandom()
	// file.File.Filename = strings.ReplaceAll(file.File.Filename, " ", "")
	// file.File.Filename = fmt.Sprintf("%s_%s", fName.String(), file.File.Filename)
	// dst, _ := os.Getwd()
	fmt.Println("boshlandi")
	defaultBucket := "ucode"
	s3Config := aws.NewConfig().
		WithRegion("us-east-1"). // Change to your region
		WithCredentials(awscredentials.NewStaticCredentials(
			h.baseConf.MinioAccessKeyID,
			h.baseConf.MinioSecretAccessKey,
			"")).
		WithEndpoint(h.baseConf.MinioEndpoint).
		WithS3ForcePathStyle(true)

	sess, err := session.NewSession(s3Config)
	if err != nil {
		h.log.Fatal("error while starting movie upload handler")
	}

	s3Store := s3store.New(defaultBucket, s3.New(sess))
	fmt.Println("\ns3Store: ", s3Store)

	composer := tusd.NewStoreComposer()
	s3Store.UseIn(composer)
	fmt.Println("wopirwoutnvopwutnwpotvnu")
	handler, err := tusd.NewHandler(tusd.Config{
		BasePath:                h.baseConf.HTTPBaseURL + "/v2/upload-file/",
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
		NotifyUploadProgress:    true,
		NotifyTerminatedUploads: true,
		RespectForwardedHeaders: true,
	})

	if err != nil {
		log.Printf("convert map", err.Error())
		return nil
	}

	// minioClient, err := minio.New(h.baseConf.MinioEndpoint, &minio.Options{
	// 	Creds:  credentials.NewStaticV4(h.baseConf.MinioAccessKeyID, h.baseConf.MinioSecretAccessKey, ""),
	// 	Secure: h.baseConf.MinioProtocol,
	// })

	// err = c.SaveUploadedFile(file.File, dst+"/"+file.File.Filename)
	// if err != nil {
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }
	// fileLink := "https://" + h.baseConf.MinioEndpoint + "/docs/" + file.File.Filename
	// splitedFileName := strings.Split(fileNameForObjectBuilder, ".")
	// f, err := os.Stat(dst + "/" + file.File.Filename)
	// if err != nil {
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	return
	// }
	// ContentTypeOfFile := file.File.Header["Content-Type"][0]

	// _, err = minioClient.FPutObject(
	// 	context.Background(),
	// 	"docs",
	// 	file.File.Filename,
	// 	dst+"/"+file.File.Filename,
	// 	minio.PutObjectOptions{ContentType: ContentTypeOfFile},
	// )
	// if err != nil {
	// 	h.handleResponse(c, status_http.BadRequest, err.Error())
	// 	err = os.Remove(dst + "/" + file.File.Filename)
	// 	if err != nil {
	// 		h.log.Error("cant remove file")
	// 	}

	// 	return
	// }
	// err = os.Remove(dst + "/" + file.File.Filename)
	// if err != nil {
	// 	h.log.Error("cant remove file")
	// }

	// var tags []string
	// if c.Query("tags") != "" {
	// 	tags = strings.Split(c.Query("tags"), ",")
	// }
	// var requestMap = make(map[string]interface{})
	// requestMap["table_slug"] = c.Param("collection")
	// requestMap["object_id"] = c.Param("id")
	// requestMap["date"] = time.Now().Format(time.RFC3339)
	// //requestMap["tags"] = tags
	// //requestMap["size"] = int32(f.Size())
	// //requestMap["type"] = splitedFileName[len(splitedFileName)-1]
	// //requestMap["file_link"] = fileLink
	// //requestMap["name"] = fileNameForObjectBuilder
	// structData, err := helper.ConvertMapToStruct(requestMap)
	// if err != nil {
	// 	log.Printf("convert map", err.Error())
	// }

	// projectId, ok := c.Get("project_id")
	// if !ok || !util.IsValidUUID(projectId.(string)) {
	// 	h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
	// }

	// environmentId, ok := c.Get("environment_id")
	// if !ok || !util.IsValidUUID(environmentId.(string)) {
	// 	err = errors.New("error getting environment id | not valid")
	// 	h.handleResponse(c, status_http.BadRequest, err)
	// }

	// resource, err := h.companyServices.ServiceResource().GetSingle(
	// 	c.Request.Context(),
	// 	&pb.GetSingleServiceResourceReq{
	// 		ProjectId:     projectId.(string),
	// 		EnvironmentId: environmentId.(string),
	// 		ServiceType:   pb.ServiceType_BUILDER_SERVICE,
	// 	},
	// )
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, err.Error())
	// }

	// services, err := h.GetProjectSrvc(
	// 	c.Request.Context(),
	// 	resource.GetProjectId(),
	// 	resource.NodeType,
	// )
	// if err != nil {
	// 	h.handleResponse(c, status_http.GRPCError, err.Error())
	// }

	// _, err = services.GetBuilderServiceByType(resource.NodeType).ObjectBuilder().Create(
	// 	context.Background(),
	// 	&obs.CommonMessage{
	// 		TableSlug: "file",
	// 		Data:      structData,
	// 		ProjectId: resource.ResourceEnvironmentId,
	// 	},
	// )
	// if err != nil {
	// 	h.handleResponse(c, status_http.InternalServerError, err.Error())
	// }

	// // h.handleResponse(c, status_http.Created, Path{
	// // 	Filename: file.File.Filename,
	// // 	Hash:     fName.String(),
	// // })

	go h.eventHandler(handler, "handler")
	return handler
}

// MovieUpload .
// @Router /v1/upload-file/ [POST]
// @Summary Tusd file upload
// @Description API for upload a large files
// @Tags tusd-file-upload
// func (h *HandlerV2) MovieUpload() *tusd.Handler {
// 	h.log.Info("call MovieUpload func: *****************************")
// 	fmt.Println(h.cfg.CdnEndpoint)
// 	fmt.Println(h.cfg.CdnRegion)
// 	fmt.Println(h.cfg.CdnAccessKeyID)
// 	fmt.Println(h.cfg.CdnSecretAccessKey)

// 	s3Config := aws.NewConfig().
// 		WithRegion("us-east-1"). // Change to your region
// 		WithCredentials(awscredentials.NewStaticCredentials(
// 			h.cfg.CdnAccessKeyID,
// 			h.cfg.CdnSecretAccessKey,
// 			"")).
// 		WithEndpoint(h.cfg.CdnEndpoint).
// 		WithS3ForcePathStyle(true)

// 	sess, err := session.NewSession(s3Config)
// 	if err != nil {
// 		h.log.Fatal("error while starting movie upload handler")
// 	}

// 	s3Store := s3store.New(h.cfg.CdnMovieUploadBucketName, s3.New(sess))
// 	fmt.Println("\ns3Store: ", s3Store)

// 	composer := tusd.NewStoreComposer()
// 	s3Store.UseIn(composer)
// 	handler, err := tusd.NewHandler(tusd.Config{
// 		BasePath:                h.cfg.ServiceURl + "/v1/upload-file/",
// 		StoreComposer:           composer,
// 		NotifyCompleteUploads:   true,
// 		NotifyUploadProgress:    true,
// 		NotifyTerminatedUploads: true,
// 		RespectForwardedHeaders: true,
// 	})

// 	if err != nil {
// 		log.Printf("Unable to create handler: %s", err)
// 	}
// 	go h.eventHandler(handler, "handler")
// 	return handler
// }

func (h *HandlerV2) eventHandler(handler *tusd.Handler, s string) {
	go func() {
		for event := range handler.CompleteUploads {
			log.Printf("Upload %s finished\n", event.Upload.ID)
			log.Printf("status:  >>>>>>>>>> %s \n", s)
		}
	}()
}
