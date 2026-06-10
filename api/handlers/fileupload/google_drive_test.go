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
	result *GoogleDriveUploadResult
	err    error
	called bool
	req    GoogleDriveUploadRequest
	creds  *pb.GoogleDriveCredentials
}

func (f *fakeDriveClient) Upload(ctx context.Context, credentials *pb.GoogleDriveCredentials, req GoogleDriveUploadRequest) (*GoogleDriveUploadResult, error) {
	f.called = true
	f.req = req
	f.creds = credentials
	return f.result, f.err
}

func TestGoogleDriveUploaderUploadIfConfiguredFallbackWhenResourceMissing(t *testing.T) {
	resources := &fakeResourceService{
		response: &pb.ListProjectResource{},
	}
	client := &fakeDriveClient{}
	uploader := NewGoogleDriveUploaderWithClient(resources, client)

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
	uploader := NewGoogleDriveUploaderWithClient(resources, &fakeDriveClient{})

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
	creds := &pb.GoogleDriveCredentials{
		AuthType:     "oauth",
		ClientId:     "client-id",
		ClientSecret: "client-secret",
		RefreshToken: "refresh-token",
	}
	resources := &fakeResourceService{
		response: &pb.ListProjectResource{
			Resources: []*pb.ProjectResource{
				{
					Id: "drive-resource",
					Settings: &pb.Settings{
						GoogleDrive: creds,
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
	uploader := NewGoogleDriveUploaderWithClient(resources, client)

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
	require.Same(t, creds, client.creds)
	require.Equal(t, "file.txt", client.req.FileName)
	require.Equal(t, "drive-file-id", result.FileID)
	require.Equal(t, DriveStorageName, result.Storage)
}

func TestGoogleDriveUploaderUploadIfConfiguredReturnsDriveUploadError(t *testing.T) {
	driveErr := errors.New("drive failed")
	resources := &fakeResourceService{
		response: &pb.ListProjectResource{
			Resources: []*pb.ProjectResource{
				{
					Id: "drive-resource",
					Settings: &pb.Settings{
						GoogleDrive: &pb.GoogleDriveCredentials{AuthType: "oauth"},
					},
				},
			},
		},
	}
	client := &fakeDriveClient{err: driveErr}
	uploader := NewGoogleDriveUploaderWithClient(resources, client)

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
