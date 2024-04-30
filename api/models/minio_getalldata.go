package models

type FindMinioObjectsRequest struct {
	FolderName string `json:"folder_name"`
}

type MinioObject struct {
	ObjectName string `json:"object_name"`
	ObjectSize int64  `json:"object_size"`
}

type FindMinioObjectsResponse struct {
	Objects []MinioObject `json:"objects"`
}

type DeleteObject struct {
	ObjectName string `json:"object_name"`
}

type DeleteObjectRequest struct {
	Objects []DeleteObject `json:"objects"`
}
