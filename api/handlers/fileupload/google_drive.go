package fileupload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	pb "ucode/ucode_go_api_gateway/genproto/company_service"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

const (
	DriveStorageName        = "google_drive"
	driveAuthTypeOAuth      = "oauth"
	driveAuthTypeService    = "service_account"
	driveVisibilityPrivate  = "private"
	drivePublicRole         = "reader"
	drivePublicType         = "anyone"
	driveViewLinkURLPattern = "https://drive.google.com/file/d/%s/view"
)

type googleDriveClient interface {
	Upload(ctx context.Context, credentials *pb.GoogleDriveCredentials, req GoogleDriveUploadRequest) (*GoogleDriveUploadResult, error)
}

type GoogleDriveUploader struct {
	resources resourceService
	client    googleDriveClient
}

type resourceService interface {
	GetProjectResourceList(ctx context.Context, in *pb.GetProjectResourceListRequest, opts ...grpc.CallOption) (*pb.ListProjectResource, error)
}

type GoogleDriveUploadRequest struct {
	ProjectID     string
	EnvironmentID string
	FileName      string
	ContentType   string
	Size          int64
	Reader        io.Reader
}

type GoogleDriveUploadResult struct {
	FileID       string
	Link         string
	FileNameDisk string
	Storage      string
}

func NewGoogleDriveUploader(resources resourceService) *GoogleDriveUploader {
	return &GoogleDriveUploader{
		resources: resources,
		client:    googleDriveAPIClient{},
	}
}

func NewGoogleDriveUploaderWithClient(resources resourceService, client googleDriveClient) *GoogleDriveUploader {
	return &GoogleDriveUploader{
		resources: resources,
		client:    client,
	}
}

func (u *GoogleDriveUploader) UploadIfConfigured(ctx context.Context, req GoogleDriveUploadRequest) (*GoogleDriveUploadResult, bool, error) {
	if u == nil || u.resources == nil {
		return nil, false, errors.New("google drive resource client is not configured")
	}

	list, err := u.resources.GetProjectResourceList(ctx, &pb.GetProjectResourceListRequest{
		ProjectId:     req.ProjectID,
		EnvironmentId: req.EnvironmentID,
		Type:          pb.ResourceType_GOOGLE_DRIVE,
	})
	if err != nil {
		return nil, false, err
	}

	if len(list.GetResources()) == 0 {
		return nil, false, nil
	}
	if len(list.GetResources()) > 1 {
		return nil, true, errors.New("multiple google drive resources configured for project environment")
	}

	settings := list.GetResources()[0].GetSettings()
	if settings == nil || settings.GetGoogleDrive() == nil {
		return nil, true, errors.New("google drive resource settings are empty")
	}

	result, err := u.client.Upload(ctx, settings.GetGoogleDrive(), req)
	if err != nil {
		return nil, true, err
	}

	return result, true, nil
}

type googleDriveAPIClient struct{}

func (googleDriveAPIClient) Upload(ctx context.Context, credentials *pb.GoogleDriveCredentials, req GoogleDriveUploadRequest) (*GoogleDriveUploadResult, error) {
	if req.Reader == nil {
		return nil, errors.New("google drive upload reader is empty")
	}
	if req.FileName == "" {
		return nil, errors.New("google drive upload file name is empty")
	}

	service, err := newDriveService(ctx, credentials)
	if err != nil {
		return nil, err
	}

	fileMeta := &drive.File{
		Name: req.FileName,
	}
	if strings.TrimSpace(credentials.GetFolderId()) != "" {
		fileMeta.Parents = []string{strings.TrimSpace(credentials.GetFolderId())}
	}

	created, err := service.Files.Create(fileMeta).
		Media(req.Reader, googleapi.ContentType(req.ContentType)).
		SupportsAllDrives(true).
		Fields("id", "webViewLink").
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
	}

	if shouldMakePublic(credentials.GetVisibility()) {
		_, err = service.Permissions.Create(created.Id, &drive.Permission{
			Role: drivePublicRole,
			Type: drivePublicType,
		}).
			SupportsAllDrives(true).
			Fields("id").
			Context(ctx).
			Do()
		if err != nil {
			return nil, err
		}
	}

	link := created.WebViewLink
	if link == "" {
		link = fmt.Sprintf(driveViewLinkURLPattern, created.Id)
	}

	return &GoogleDriveUploadResult{
		FileID:       created.Id,
		Link:         link,
		FileNameDisk: created.Id,
		Storage:      DriveStorageName,
	}, nil
}

func newDriveService(ctx context.Context, credentials *pb.GoogleDriveCredentials) (*drive.Service, error) {
	if credentials == nil {
		return nil, errors.New("google drive credentials are empty")
	}

	switch strings.ToLower(strings.TrimSpace(credentials.GetAuthType())) {
	case driveAuthTypeOAuth:
		if credentials.GetClientId() == "" || credentials.GetClientSecret() == "" || credentials.GetRefreshToken() == "" {
			return nil, errors.New("google drive oauth credentials are incomplete")
		}

		oauthConfig := &oauth2.Config{
			ClientID:     credentials.GetClientId(),
			ClientSecret: credentials.GetClientSecret(),
			Endpoint:     google.Endpoint,
			Scopes:       []string{drive.DriveFileScope},
		}
		httpClient := oauthConfig.Client(ctx, &oauth2.Token{
			RefreshToken: credentials.GetRefreshToken(),
		})
		return drive.NewService(ctx, option.WithHTTPClient(httpClient))

	case driveAuthTypeService:
		if credentials.GetServiceAccountJson() == "" {
			return nil, errors.New("google drive service account json is empty")
		}

		return drive.NewService(
			ctx,
			option.WithCredentialsJSON([]byte(credentials.GetServiceAccountJson())),
			option.WithScopes(drive.DriveFileScope),
		)

	default:
		return nil, errors.New("google drive auth_type must be oauth or service_account")
	}
}

func shouldMakePublic(visibility string) bool {
	return strings.ToLower(strings.TrimSpace(visibility)) != driveVisibilityPrivate
}
