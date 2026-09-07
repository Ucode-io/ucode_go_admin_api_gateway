package callquality

import (
	"context"

	"ucode/ucode_go_api_gateway/api/handlers/ai"
	"ucode/ucode_go_api_gateway/api/handlers/ai/openai"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/logger"

	"github.com/gin-gonic/gin"
)

type model interface {
	Complete(context.Context, ai.CompletionRequest) (*ai.CompletionResult, error)
}

type Handler struct {
	evaluator *Evaluator
	log       logger.LoggerI
}

func NewHandler(conf config.BaseConfig, log logger.LoggerI) Handler {
	return Handler{
		evaluator: NewEvaluator(openai.NewOpenAIChatModel(conf), conf.OpenAIAgents.Inspector),
		log:       log,
	}
}

func (h Handler) Evaluate(c *gin.Context) {
	var req models.CallQualityEvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respond(c, status_http.BadRequest, "invalid call quality request")
		return
	}
	if err := validateRequest(req); err != nil {
		respond(c, status_http.BadRequest, err.Error())
		return
	}

	result, err := h.evaluator.Evaluate(c.Request.Context(), req)
	if err != nil {
		h.log.Error("call quality evaluation failed", logger.Error(err), logger.String("call_id", req.CallID))
		respond(c, status_http.InternalServerError, "call quality evaluation failed")
		return
	}
	respond(c, status_http.OK, result)
}

func respond(c *gin.Context, status status_http.Status, data any) {
	c.JSON(status.Code, status_http.Response{
		Status: status.Status, Description: status.Description, Data: data,
	})
}
