package v2

import (
	_ "ucode/ucode_go_api_gateway/api/models"
	_ "ucode/ucode_go_api_gateway/api/status_http"

	"github.com/gin-gonic/gin"
)

// InvokeFunctionByPath godoc
// @Security ApiKeyAuth
// @Param function-path path string true "function-path"
// @ID v2_invoke_function_by_path
// @Router /v2/invoke_function/{function-path} [POST]
// @Summary Invoke Function By Path
// @Description Invoke Function By Path
// @Tags Function
// @Accept json
// @Produce json
// @Param InvokeFunctionByPathRequest body models.CommonMessage true "InvokeFunctionByPathRequest"
// @Success 201 {object} status_http.Response{data=models.InvokeFunctionRequest} "Function data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) InvokeFunctionByPath(c *gin.Context) {
	_ = h.MakeProxy(c, h.baseConf.GoFunctionServiceHost+h.baseConf.GoFunctionServiceHTTPPort, c.Request.URL.Path)
}
