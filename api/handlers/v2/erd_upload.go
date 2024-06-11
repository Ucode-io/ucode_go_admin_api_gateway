package v2

import (
	"os"
	"strings"
	"ucode/ucode_go_api_gateway/api/status_http"

	sdk "github.com/baxromumarov/ucode-sdk/erd_reader"
	"github.com/gin-gonic/gin"
)

// UploadFile godoc
// @Security ApiKeyAuth
// @ID erd_upload
// @Router /v2/erd [POST]
// @Summary Upload erd
// @Description Upload erd
// @Tags V2Files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "file"
// @Success 200 {object} status_http.Response{data=Path} "Path"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) UploadERD(c *gin.Context) {
	// just in case. custom erd reader might panic
	defer func() {
		if err := recover(); err != nil {
			h.handleResponse(c, status_http.InternalServerError, " erd upload paniced")
			return
		}
	}()

	var (
		file File
	)

	if err := c.ShouldBind(&file); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	file.File.Filename = strings.ReplaceAll(file.File.Filename, " ", "")
	dir, _ := os.Getwd()

	// Create a new file in the uploads directory
	dst, err := os.Create(dir + "/" + file.File.Filename)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}
	defer dst.Close()

	bearerToken := c.GetHeader("Authorization")

	err = sdk.Reader(dst, bearerToken)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, "ERD successfully transfered to Ucode")
}
