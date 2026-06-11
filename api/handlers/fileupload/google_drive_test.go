package fileupload

import (
	"context"
	"errors"
	"strings"
	"testing"

	pb "ucode/ucode_go_api_gateway/genproto/company_service"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
)

type fakeResourceService struct {
	response *pb.ListProjectResource
	err      error
	request  *pb.GetProjectResourceListRequest
}

func (f *fakeResourceService) GetProjectResourceList(ctx context.Context, in *pb.GetProjectResourceListRequest, opts ...grpc.CallOption) (*pb.ListProjectResource, error) {
	f.request = in
	return f.response, f.err
}

type fakeDriveClient struct {
	result             *GoogleDriveUploadResult
	err                error
	called             bool
	req                GoogleDriveUploadRequest
	creds              *pb.GoogleDriveCredentials
	folderResult       *GoogleDriveFolderResult
	folderErr          error
	createFolderCalled bool
	createFolderReq    GoogleDriveCreateFolderRequest
	createFolderCreds  *pb.GoogleDriveCredentials
}

func (f *fakeDriveClient) Upload(ctx context.Context, credentials *pb.GoogleDriveCredentials, req GoogleDriveUploadRequest) (*GoogleDriveUploadResult, error) {
	f.called = true
	f.req = req
	f.creds = credentials
	return f.result, f.err
}

func (f *fakeDriveClient) CreateFolder(ctx context.Context, credentials *pb.GoogleDriveCredentials, req GoogleDriveCreateFolderRequest) (*GoogleDriveFolderResult, error) {
	f.createFolderCalled = true
	f.createFolderReq = req
	f.createFolderCreds = credentials
	return f.folderResult, f.folderErr
}

func TestGoogleDriveUploaderUploadIfConfiguredFallbackWhenResourceMissing(t *testing.T) {
	resources := &fakeResourceService{
		response: &pb.ListProjectResource{},
	}
	client := &fakeDriveClient{}
	uploader := NewGoogleDriveUploaderWithClient(resources, GoogleDriveConfig{}, client)

	result, configured, err := uploader.UploadIfConfigured(context.Background(), GoogleDriveUploadRequest{
		ProjectID:     "project-id",
		EnvironmentID: "environment-id",
		FileName:      "file.txt",
		Reader:        strings.NewReader("file"),
	})

	require.NoError(t, err)
	require.False(t, configured)
	require.Nil(t, result)
	require.False(t, client.called)
	require.Equal(t, pb.ResourceType_GOOGLE_DRIVE, resources.request.GetType())
}

func TestGoogleDriveUploaderUploadIfConfiguredRejectsMultipleResources(t *testing.T) {
	resources := &fakeResourceService{
		response: &pb.ListProjectResource{
			Resources: []*pb.ProjectResource{
				{Id: "one"},
				{Id: "two"},
			},
		},
	}
	uploader := NewGoogleDriveUploaderWithClient(resources, GoogleDriveConfig{}, &fakeDriveClient{})

	result, configured, err := uploader.UploadIfConfigured(context.Background(), GoogleDriveUploadRequest{
		ProjectID:     "project-id",
		EnvironmentID: "environment-id",
		FileName:      "file.txt",
		Reader:        strings.NewReader("file"),
	})

	require.Error(t, err)
	require.True(t, configured)
	require.Nil(t, result)
	require.Contains(t, err.Error(), "multiple google drive resources")
}

func TestGoogleDriveUploaderUploadIfConfiguredUploadsWithSettings(t *testing.T) {
	resources := &fakeResourceService{
		response: &pb.ListProjectResource{
			Resources: []*pb.ProjectResource{
				{
					Id: "drive-resource",
					Settings: &pb.Settings{
						GoogleDrive: &pb.GoogleDriveCredentials{
							AuthType:     "oauth",
							FolderId:     "folder-id",
							Visibility:   "private",
							RefreshToken: "refresh-token",
						},
					},
				},
			},
		},
	}
	client := &fakeDriveClient{
		result: &GoogleDriveUploadResult{
			FileID:       "drive-file-id",
			Link:         "https://drive.google.com/file/d/drive-file-id/view",
			FileNameDisk: "drive-file-id",
			Storage:      DriveStorageName,
		},
	}
	uploader := NewGoogleDriveUploaderWithClient(resources, GoogleDriveConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		Visibility:   "anyone_with_link",
	}, client)

	result, configured, err := uploader.UploadIfConfigured(context.Background(), GoogleDriveUploadRequest{
		ProjectID:     "project-id",
		EnvironmentID: "environment-id",
		FileName:      "file.txt",
		ContentType:   "text/plain",
		Reader:        strings.NewReader("file"),
	})

	require.NoError(t, err)
	require.True(t, configured)
	require.True(t, client.called)
	require.Equal(t, "oauth", client.creds.GetAuthType())
	require.Equal(t, "client-id", client.creds.GetClientId())
	require.Equal(t, "client-secret", client.creds.GetClientSecret())
	require.Equal(t, "refresh-token", client.creds.GetRefreshToken())
	require.Equal(t, "folder-id", client.creds.GetFolderId())
	require.Equal(t, "private", client.creds.GetVisibility())
	require.Equal(t, "file.txt", client.req.FileName)
	require.Equal(t, "drive-file-id", result.FileID)
	require.Equal(t, DriveStorageName, result.Storage)
}

func TestGoogleDriveUploaderUploadIfConfiguredRequiresPlatformCredentials(t *testing.T) {
	resources := &fakeResourceService{
		response: &pb.ListProjectResource{
			Resources: []*pb.ProjectResource{
				{
					Id: "drive-resource",
					Settings: &pb.Settings{
						GoogleDrive: &pb.GoogleDriveCredentials{
							FolderId: "folder-id",
						},
					},
				},
			},
		},
	}
	client := &fakeDriveClient{}
	uploader := NewGoogleDriveUploaderWithClient(resources, GoogleDriveConfig{}, client)

	result, configured, err := uploader.UploadIfConfigured(context.Background(), GoogleDriveUploadRequest{
		ProjectID:     "project-id",
		EnvironmentID: "environment-id",
		FileName:      "file.txt",
		Reader:        strings.NewReader("file"),
	})

	require.Error(t, err)
	require.True(t, configured)
	require.Nil(t, result)
	require.False(t, client.called)
	require.Contains(t, err.Error(), "oauth client config")
}

func TestGoogleDriveUploaderUploadIfConfiguredReturnsDriveUploadError(t *testing.T) {
	driveErr := errors.New("drive failed")
	resources := &fakeResourceService{
		response: &pb.ListProjectResource{
			Resources: []*pb.ProjectResource{
				{
					Id: "drive-resource",
					Settings: &pb.Settings{
						GoogleDrive: &pb.GoogleDriveCredentials{
							AuthType:     "oauth",
							FolderId:     "folder-id",
							RefreshToken: "refresh-token",
						},
					},
				},
			},
		},
	}
	client := &fakeDriveClient{err: driveErr}
	uploader := NewGoogleDriveUploaderWithClient(resources, GoogleDriveConfig{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}, client)

	result, configured, err := uploader.UploadIfConfigured(context.Background(), GoogleDriveUploadRequest{
		ProjectID:     "project-id",
		EnvironmentID: "environment-id",
		FileName:      "file.txt",
		Reader:        strings.NewReader("file"),
	})

	require.ErrorIs(t, err, driveErr)
	require.True(t, configured)
	require.Nil(t, result)
}

func TestGoogleDriveUploaderUploadIfConfiguredSupportsServiceAccountSettings(t *testing.T) {
	resources := &fakeResourceService{
		response: &pb.ListProjectResource{
			Resources: []*pb.ProjectResource{
				{
					Id: "drive-resource",
					Settings: &pb.Settings{
						GoogleDrive: &pb.GoogleDriveCredentials{
							AuthType:   "service_account",
							FolderId:   "folder-id",
							Visibility: "private",
						},
					},
				},
			},
		},
	}
	client := &fakeDriveClient{
		result: &GoogleDriveUploadResult{
			FileID:       "drive-file-id",
			Link:         "https://drive.google.com/file/d/drive-file-id/view",
			FileNameDisk: "drive-file-id",
			Storage:      DriveStorageName,
		},
	}
	uploader := NewGoogleDriveUploaderWithClient(resources, GoogleDriveConfig{
		ServiceAccountJSON: `{"type":"service_account"}`,
	}, client)

	result, configured, err := uploader.UploadIfConfigured(context.Background(), GoogleDriveUploadRequest{
		ProjectID:     "project-id",
		EnvironmentID: "environment-id",
		FileName:      "file.txt",
		Reader:        strings.NewReader("file"),
	})

	require.NoError(t, err)
	require.True(t, configured)
	require.Equal(t, "service_account", client.creds.GetAuthType())
	require.Equal(t, `{"type":"service_account"}`, client.creds.GetServiceAccountJson())
	require.Equal(t, "folder-id", client.creds.GetFolderId())
	require.Equal(t, "drive-file-id", result.FileID)
}

func TestGoogleDriveUploaderProvisionProjectFolder(t *testing.T) {
	client := &fakeDriveClient{
		folderResult: &GoogleDriveFolderResult{
			FolderID: "folder-id",
			Link:     "https://drive.google.com/drive/folders/folder-id",
			Name:     "Project Name - project-",
		},
	}
	uploader := NewGoogleDriveUploaderWithClient(nil, GoogleDriveConfig{
		ServiceAccountJSON: `{"type":"service_account"}`,
		ParentFolderID:     "parent-folder-id",
		Visibility:         "anyone_with_link",
	}, client)

	folder, err := uploader.ProvisionProjectFolder(context.Background(), GoogleDriveFolderRequest{
		ProjectID:   "project-id-123",
		ProjectName: "Project/Name",
	})

	require.NoError(t, err)
	require.True(t, client.createFolderCalled)
	require.Equal(t, "Project-Name - project-", client.createFolderReq.Name)
	require.Equal(t, "parent-folder-id", client.createFolderReq.ParentFolderID)
	require.Equal(t, `{"type":"service_account"}`, client.createFolderCreds.GetServiceAccountJson())
	require.Equal(t, "folder-id", folder.FolderID)

	settings := uploader.ResourceSettingsForFolder(folder)
	require.Equal(t, "service_account", settings.GetGoogleDrive().GetAuthType())
	require.Equal(t, "folder-id", settings.GetGoogleDrive().GetFolderId())
	require.Equal(t, "anyone_with_link", settings.GetGoogleDrive().GetVisibility())
	require.Empty(t, settings.GetGoogleDrive().GetServiceAccountJson())
}
