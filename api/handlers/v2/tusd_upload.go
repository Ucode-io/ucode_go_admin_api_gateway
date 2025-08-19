package v2

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	tusd "github.com/tus/tusd/pkg/handler"
	"github.com/tus/tusd/pkg/s3store"
)

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
			case event := <-handler.CompleteUploads:
				log.Printf("-------------------UPLOAD FINISHED--------------- %s\n", event.Upload.ID)
			case event := <-handler.TerminatedUploads:
				log.Printf("---------------UPLOAD TERMINATED--------------- %s\n", event.Upload.ID)
			}
		}
	}()

	return handler
}
