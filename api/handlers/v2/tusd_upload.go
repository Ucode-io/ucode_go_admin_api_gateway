package v2

import (
	"log"

	//"ucode/ucode_go_api_gateway/api/models"

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
	ResourceEnvironmentId := "75fd774f-f048-4658-9244-4be214ce293c"
	//defaultBucket := "ucode"
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
		h.log.Error("error while starting movie upload handler")
	}

	s3Store := s3store.New(ResourceEnvironmentId, s3.New(sess))

	composer := tusd.NewStoreComposer()
	s3Store.UseIn(composer)

	handler, err := tusd.NewHandler(tusd.Config{
		// BasePath:                "http://localhost:8000/v2/upload-file/",
		BasePath:                h.baseConf.HTTPBaseURL + "/v2/upload-file/",
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
		NotifyUploadProgress:    true,
		NotifyTerminatedUploads: true,
		RespectForwardedHeaders: true,
	})

	if err != nil {
		h.log.Error("err while tusd new handler")
	}

	go h.eventHandler(handler, "handler")
	return handler
}

func (h *HandlerV2) eventHandler(handler *tusd.Handler, s string) {
	go func() {
		for event := range handler.CompleteUploads {
			log.Printf("Upload %s finished\n", event.Upload.ID)
			log.Printf("status:  >>>>>>>>>> %s \n", s)
		}
	}()
}
