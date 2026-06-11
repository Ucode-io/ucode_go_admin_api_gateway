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
	driveFolderMimeType     = "application/vnd.google-apps.folder"
	driveViewLinkURLPattern = "https://drive.google.com/file/d/%s/view"
	driveFolderURLPattern   = "https://drive.google.com/drive/folders/%s"
)

type googleDriveClient interface {
	Upload(ctx context.Context, credentials *pb.GoogleDriveCredentials, req GoogleDriveUploadRequest) (*GoogleDriveUploadResult, error)
	CreateFolder(ctx context.Context, credentials *pb.GoogleDriveCredentials, req GoogleDriveCreateFolderRequest) (*GoogleDriveFolderResult, error)
}

type GoogleDriveUploader struct {
	resources      resourceService
	client         googleDriveClient
	platformConfig GoogleDriveConfig
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

type GoogleDriveConfig struct {
	ClientID           string
	ClientSecret       string
	RedirectURI        string
	ServiceAccountJSON string
	ParentFolderID     string
	Visibility         string
}

type GoogleDriveCreateFolderRequest struct {
	Name           string
	ParentFolderID string
}

type GoogleDriveFolderRequest struct {
	ProjectID   string
	ProjectName string
}

type GoogleDriveFolderResult struct {
	FolderID string
	Link     string
	Name     string
}

func NewGoogleDriveUploader(resources resourceService, platformConfig GoogleDriveConfig) *GoogleDriveUploader {
	return &GoogleDriveUploader{
		resources:      resources,
		client:         googleDriveAPIClient{},
		platformConfig: platformConfig,
	}
}

func NewGoogleDriveUploaderWithClient(resources resourceService, platformConfig GoogleDriveConfig, client googleDriveClient) *GoogleDriveUploader {
	return &GoogleDriveUploader{
		resources:      resources,
		client:         client,
		platformConfig: platformConfig,
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

	resourceSettings := settings.GetGoogleDrive()
	if strings.TrimSpace(resourceSettings.GetFolderId()) == "" {
		return nil, true, errors.New("google drive resource folder_id is empty")
	}

	credentials, err := u.credentialsForResource(resourceSettings)
	if err != nil {
		return nil, true, err
	}

	result, err := u.client.Upload(ctx, credentials, req)
	if err != nil {
		return nil, true, err
	}

	return result, true, nil
}

func (u *GoogleDriveUploader) ProvisionProjectFolder(ctx context.Context, req GoogleDriveFolderRequest) (*GoogleDriveFolderResult, error) {
	credentials, err := u.platformCredentials()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(u.platformConfig.ParentFolderID) == "" {
		return nil, errors.New("GOOGLE_DRIVE_PARENT_FOLDER_ID is empty")
	}

	folderName := ProjectFolderName(req.ProjectName, req.ProjectID)
	if folderName == "" {
		return nil, errors.New("google drive folder name is empty")
	}

	return u.client.CreateFolder(ctx, credentials, GoogleDriveCreateFolderRequest{
		Name:           folderName,
		ParentFolderID: strings.TrimSpace(u.platformConfig.ParentFolderID),
	})
}

func (u *GoogleDriveUploader) ProvisionFolder(ctx context.Context, credentials *pb.GoogleDriveCredentials, req GoogleDriveCreateFolderRequest) (*GoogleDriveFolderResult, error) {
	return u.client.CreateFolder(ctx, credentials, req)
}

func (u *GoogleDriveUploader) ResourceSettingsForFolder(folder *GoogleDriveFolderResult) *pb.Settings {
	visibility := strings.TrimSpace(u.platformConfig.Visibility)
	if visibility == "" {
		visibility = driveVisibilityPrivate
	}

	return &pb.Settings{
		GoogleDrive: &pb.GoogleDriveCredentials{
			AuthType:   driveAuthTypeService,
			FolderId:   folder.GetFolderID(),
			Visibility: visibility,
		},
	}
}

func (u *GoogleDriveUploader) platformCredentials() (*pb.GoogleDriveCredentials, error) {
	if strings.TrimSpace(u.platformConfig.ServiceAccountJSON) == "" {
		return nil, errors.New("GOOGLE_DRIVE_SERVICE_ACCOUNT_JSON is empty")
	}

	visibility := strings.TrimSpace(u.platformConfig.Visibility)
	if visibility == "" {
		visibility = driveVisibilityPrivate
	}

	return &pb.GoogleDriveCredentials{
		AuthType:           driveAuthTypeService,
		ServiceAccountJson: strings.TrimSpace(u.platformConfig.ServiceAccountJSON),
		Visibility:         visibility,
	}, nil
}

func (u *GoogleDriveUploader) credentialsForResource(resourceSettings *pb.GoogleDriveCredentials) (*pb.GoogleDriveCredentials, error) {
	if resourceSettings == nil {
		return nil, errors.New("google drive resource settings are empty")
	}

	authType := strings.ToLower(strings.TrimSpace(resourceSettings.GetAuthType()))
	if authType == "" {
		authType = driveAuthTypeOAuth
	}

	visibility := strings.TrimSpace(resourceSettings.GetVisibility())
	if visibility == "" {
		visibility = strings.TrimSpace(u.platformConfig.Visibility)
	}
	if visibility == "" {
		visibility = driveVisibilityPrivate
	}

	switch authType {
	case driveAuthTypeOAuth:
		clientID := strings.TrimSpace(resourceSettings.GetClientId())
		if clientID == "" {
			clientID = strings.TrimSpace(u.platformConfig.ClientID)
		}
		clientSecret := strings.TrimSpace(resourceSettings.GetClientSecret())
		if clientSecret == "" {
			clientSecret = strings.TrimSpace(u.platformConfig.ClientSecret)
		}
		if clientID == "" || clientSecret == "" {
			return nil, errors.New("google drive oauth client config is empty")
		}
		if strings.TrimSpace(resourceSettings.GetRefreshToken()) == "" {
			return nil, errors.New("google drive oauth refresh_token is empty")
		}

		return &pb.GoogleDriveCredentials{
			AuthType:     driveAuthTypeOAuth,
			FolderId:     strings.TrimSpace(resourceSettings.GetFolderId()),
			Visibility:   visibility,
			ClientId:     clientID,
			ClientSecret: clientSecret,
			RefreshToken: strings.TrimSpace(resourceSettings.GetRefreshToken()),
		}, nil

	case driveAuthTypeService:
		serviceAccountJSON := strings.TrimSpace(resourceSettings.GetServiceAccountJson())
		if serviceAccountJSON == "" {
			serviceAccountJSON = strings.TrimSpace(u.platformConfig.ServiceAccountJSON)
		}
		if serviceAccountJSON == "" {
			return nil, errors.New("google drive service account json is empty")
		}

		return &pb.GoogleDriveCredentials{
			AuthType:           driveAuthTypeService,
			FolderId:           strings.TrimSpace(resourceSettings.GetFolderId()),
			Visibility:         visibility,
			ServiceAccountJson: serviceAccountJSON,
		}, nil

	default:
		return nil, errors.New("google drive auth_type must be oauth or service_account")
	}
}

func NewOAuthConfig(config GoogleDriveConfig) (*oauth2.Config, error) {
	if strings.TrimSpace(config.ClientID) == "" || strings.TrimSpace(config.ClientSecret) == "" || strings.TrimSpace(config.RedirectURI) == "" {
		return nil, errors.New("google drive oauth config is incomplete")
	}

	return &oauth2.Config{
		ClientID:     strings.TrimSpace(config.ClientID),
		ClientSecret: strings.TrimSpace(config.ClientSecret),
		RedirectURL:  strings.TrimSpace(config.RedirectURI),
		Endpoint:     google.Endpoint,
		Scopes:       []string{drive.DriveFileScope},
	}, nil
}

func OAuthCredentialsFromRefreshToken(config GoogleDriveConfig, refreshToken string) (*pb.GoogleDriveCredentials, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return nil, errors.New("google drive oauth refresh_token is empty")
	}
	if strings.TrimSpace(config.ClientID) == "" || strings.TrimSpace(config.ClientSecret) == "" {
		return nil, errors.New("google drive oauth client config is empty")
	}

	visibility := strings.TrimSpace(config.Visibility)
	if visibility == "" {
		visibility = driveVisibilityPrivate
	}

	return &pb.GoogleDriveCredentials{
		AuthType:     driveAuthTypeOAuth,
		ClientId:     strings.TrimSpace(config.ClientID),
		ClientSecret: strings.TrimSpace(config.ClientSecret),
		RefreshToken: strings.TrimSpace(refreshToken),
		Visibility:   visibility,
	}, nil
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

func (googleDriveAPIClient) CreateFolder(ctx context.Context, credentials *pb.GoogleDriveCredentials, req GoogleDriveCreateFolderRequest) (*GoogleDriveFolderResult, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("google drive folder name is empty")
	}

	service, err := newDriveService(ctx, credentials)
	if err != nil {
		return nil, err
	}

	fileMeta := &drive.File{
		Name:     strings.TrimSpace(req.Name),
		MimeType: driveFolderMimeType,
	}
	if strings.TrimSpace(req.ParentFolderID) != "" {
		fileMeta.Parents = []string{strings.TrimSpace(req.ParentFolderID)}
	}

	created, err := service.Files.Create(fileMeta).
		SupportsAllDrives(true).
		Fields("id", "webViewLink", "name").
		Context(ctx).
		Do()
	if err != nil {
		return nil, err
	}

	link := created.WebViewLink
	if link == "" {
		link = fmt.Sprintf(driveFolderURLPattern, created.Id)
	}

	return &GoogleDriveFolderResult{
		FolderID: created.Id,
		Link:     link,
		Name:     created.Name,
	}, nil
}

func newDriveService(ctx context.Context, credentials *pb.GoogleDriveCredentials) (*drive.Service, error) {
	if credentials == nil {
		return nil, errors.New("google drive credentials are empty")
	}

	authType := strings.ToLower(strings.TrimSpace(credentials.GetAuthType()))
	if authType == "" {
		authType = driveAuthTypeService
	}

	switch authType {
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

func ProjectFolderName(projectName, projectID string) string {
	name := strings.TrimSpace(projectName)
	if name == "" {
		name = strings.TrimSpace(projectID)
	}
	if name == "" {
		return ""
	}

	replacer := strings.NewReplacer("/", "-", "\\", "-", "\x00", "")
	name = replacer.Replace(name)

	projectID = strings.TrimSpace(projectID)
	if projectID != "" && !strings.Contains(name, projectID) {
		shortID := projectID
		if len(shortID) > 8 {
			shortID = shortID[:8]
		}
		name = fmt.Sprintf("%s - %s", name, shortID)
	}

	if len(name) > 200 {
		name = strings.TrimSpace(name[:200])
	}

	return name
}

func (r *GoogleDriveFolderResult) GetFolderID() string {
	if r != nil {
		return r.FolderID
	}
	return ""
}
