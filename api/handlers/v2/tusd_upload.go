package v2

import (
	"log"
	"net/http"

	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/aws/aws-sdk-go/aws"
	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-gonic/gin"
	tusd "github.com/tus/tusd/pkg/handler"
	"github.com/tus/tusd/pkg/s3store"
)

func (h *HandlerV2) MovieUpload(c *gin.Context) {
	projectID, ok := c.Get("project_id")
	projectIDString, isString := projectID.(string)
	if !ok || !isString || !util.IsValidUUID(projectIDString) {
		h.HandleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentID, ok := c.Get("environment_id")
	environmentIDString, isString := environmentID.(string)
	if !ok || !isString || !util.IsValidUUID(environmentIDString) {
		h.HandleResponse(c, status_http.InvalidArgument, "environment id is an invalid uuid")
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectIDString,
			EnvironmentId: environmentIDString,
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	bucketName := resource.GetResourceEnvironmentId()
	if !util.IsValidUUID(bucketName) {
		h.HandleResponse(c, status_http.InternalServerError, "builder resource environment id is invalid")
		return
	}

	handler, err := h.movieUploadHandler(bucketName)
	if err != nil {
		h.log.Error("error while starting movie upload handler")
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	http.StripPrefix("/v2/upload-file/", handler).ServeHTTP(c.Writer, c.Request)
}

func (h *HandlerV2) movieUploadHandler(bucketName string) (*tusd.Handler, error) {
	h.tusdMu.Lock()
	defer h.tusdMu.Unlock()

	if handler, ok := h.tusdHandlers[bucketName]; ok {
		return handler, nil
	}
	if h.tusdHandlers == nil {
		h.tusdHandlers = make(map[string]*tusd.Handler)
	}

	s3Config := aws.NewConfig().
		WithRegion("us-east-1").
		WithCredentials(awscredentials.NewStaticCredentials(
			h.baseConf.MinioAccessKeyID,
			h.baseConf.MinioSecretAccessKey,
			"")).
		WithEndpoint(h.baseConf.MinioEndpoint).
		WithS3ForcePathStyle(true)

	sess, err := session.NewSession(s3Config)
	if err != nil {
		return nil, err
	}

	s3Store := s3store.New(bucketName, s3.New(sess))

	composer := tusd.NewStoreComposer()
	s3Store.UseIn(composer)

	handler, err := tusd.NewHandler(tusd.Config{
		BasePath:                h.baseConf.HTTPBaseURL + "/v2/upload-file/",
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
		NotifyUploadProgress:    true,
		NotifyTerminatedUploads: true,
		RespectForwardedHeaders: true,
	})

	if err != nil {
		return nil, err
	}

	h.tusdHandlers[bucketName] = handler
	go h.eventHandler(handler, "handler")
	return handler, nil
}

func (h *HandlerV2) eventHandler(handler *tusd.Handler, s string) {
	go func() {
		for event := range handler.CompleteUploads {
			log.Printf("Upload %s finished\n", event.Upload.ID)
			log.Printf("status:  >>>>>>>>>> %s \n", s)
		}
	}()
}

func (h *HandlerV2) Tusd() *tusd.Handler {
	ResourceEnvironmentId := "b8199457-9a0e-4260-bcca-75b7bc55c1f9"
	s3Config := aws.NewConfig().
		WithRegion("us-east-1").
		WithCredentials(awscredentials.NewStaticCredentials(
			h.baseConf.MinioAccessKeyID,
			h.baseConf.MinioSecretAccessKey,
			"")).
		WithEndpoint(h.baseConf.MinioEndpoint).
		WithS3ForcePathStyle(true)

	sess, err := session.NewSession(s3Config)
	if err != nil {
		h.log.Error("error while starting movie upload handler")
		return nil
	}
	s3Store := s3store.New(ResourceEnvironmentId, s3.New(sess))

	composer := tusd.NewStoreComposer()
	s3Store.UseIn(composer)

	handler, err := tusd.NewHandler(tusd.Config{
		BasePath:                "/v1/tusd/",
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
		NotifyUploadProgress:    true,
		NotifyTerminatedUploads: true,
		RespectForwardedHeaders: true,
	})

	if err != nil {
		h.log.Error("err while tusd new handler")
		return nil
	}

	go func() {
		for {
			select {
			case event := <-handler.UploadProgress:
				log.Printf("-------------------UPLOAD FINISHED--------------- %s\n", event.Upload.ID)
			case event := <-handler.CompleteUploads:
				log.Printf("-------------------UPLOAD FINISHED--------------- %s\n", event.Upload.ID)
			case event := <-handler.TerminatedUploads:
				log.Printf("---------------UPLOAD TERMINATED--------------- %s\n", event.Upload.ID)
			}
		}
	}()

	return handler
}
