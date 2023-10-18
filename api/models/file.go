package models

type FileCreateRequest struct {
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Tags             []string `json:"tags"`
	Storage          string   `json:"storage"`
	FileNameDisk     string   `json:"file_name_disk"`
	FileNameDownload string   `json:"file_name_download"`
	Link             string   `json:"link"`
	FileSize         string   `json:"file_size"`
}

type FileDelete struct {
	ObjectName string `json:"object_name"`
	ObjectId   string `json:"object_id"`
}

type FileDeleteRequest struct {
	Objects []FileDelete `json:"objects"`
}

type UpdateFileRequest struct {
	Id               string   `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Tags             []string `json:"tags"`
	FileNameDownload string   `json:"file_name_download"`
}
