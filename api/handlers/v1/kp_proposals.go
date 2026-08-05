package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// kpAgentTimeout bounds the synchronous KP generation call. It must stay under
// the ingress proxy timeout in front of the gateway (see deploy notes) and
// comfortably above a typical HTML-only generation.
const kpAgentTimeout = 120 * time.Second

// kpProposalRequest is what the Professio frontend sends. The endpoint is a thin
// pass-through: only prompt/locale/themeTokens reach the agent. Brand/web-research
// behavior is intentionally left at the agent's defaults.
type kpProposalRequest struct {
	Prompt      string          `json:"prompt"`
	Locale      string          `json:"locale"`
	ThemeTokens json.RawMessage `json:"themeTokens,omitempty"`
	DealID      string          `json:"dealId,omitempty"`
}

type kpAgentRequest struct {
	Prompt                 string          `json:"prompt"`
	Locale                 string          `json:"locale,omitempty"`
	ThemeTokens            json.RawMessage `json:"themeTokens,omitempty"`
	PrototypePublicBaseURL string          `json:"prototypePublicBaseUrl,omitempty"`
}

type kpAgentError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type kpAgentResponse struct {
	Ok           bool              `json:"ok"`
	RequestID    string            `json:"requestId"`
	Title        string            `json:"title"`
	HTML         string            `json:"html"`
	PageCount    int               `json:"pageCount"`
	HasPDF       bool              `json:"hasPdf"`
	PrototypeURL string            `json:"prototypeUrl"`
	Prototype    *kpAgentPrototype `json:"prototype"`
	Error        *kpAgentError     `json:"error"`
}

type kpAgentPrototype struct {
	URL             string `json:"url"`
	QAStatus        string `json:"qaStatus,omitempty"`
	ScreenCount     int    `json:"screenCount,omitempty"`
	RendererVersion string `json:"rendererVersion,omitempty"`
}

// kpPdfCacheEntry is the tenant-isolation record stored in centralRedis under
// config.KpProposalPdfCachePrefix+requestId when a proposal has a downloadable
// PDF. It lets GET /v1/kp-proposals/:requestId/pdf (kp_proposal_pdf.go) verify
// the caller's project/environment owns requestId, across gateway replicas.
type kpPdfCacheEntry struct {
	ProjectID     string `json:"projectId"`
	EnvironmentID string `json:"environmentId"`
	Title         string `json:"title"`
}

// GenerateKpProposal godoc
// @Security ApiKeyAuth
// @ID v1_generate_kp_proposal
// @Router /v1/kp-proposals [POST]
// @Summary Generate a commercial proposal (KP) as HTML
// @Description Synchronously generates a KP via the standalone kp-generator-agent and returns the HTML inline.
// @Tags KP
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Param GenerateKpProposalRequest body kpProposalRequest true "GenerateKpProposalRequest"
// @Success 200 {object} status_http.Response "KP HTML"
// @Failure 400 {object} status_http.Response "Bad Request"
// @Failure 500 {object} status_http.Response "Server Error"
func (h *HandlerV1) GenerateKpProposal(c *gin.Context) {
	// AuthMiddleware has already validated the JWT and populated the context.
	// A valid, project-scoped token establishes project-level access.
	projectID, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(cast.ToString(projectID)) {
		h.HandleResponse(c, status_http.InvalidArgument, config.ErrProjectIdValid)
		return
	}
	environmentID, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(cast.ToString(environmentID)) {
		h.HandleResponse(c, status_http.InvalidArgument, config.ErrEnvironmentIdValid)
		return
	}

	if strings.TrimSpace(h.baseConf.KpAgentURL) == "" {
		h.HandleResponse(c, status_http.InternalServerError, kpError("KP_AGENT_UNAVAILABLE", "KP agent is not configured"))
		return
	}

	var req kpProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.HandleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}
	if strings.TrimSpace(req.Prompt) == "" {
		h.HandleResponse(c, status_http.InvalidArgument, "prompt is required")
		return
	}

	agentResp, err := h.callKpAgent(c.Request.Context(), kpAgentRequest{
		Prompt:                 req.Prompt,
		Locale:                 mapKpLocale(req.Locale),
		ThemeTokens:            req.ThemeTokens,
		PrototypePublicBaseURL: kpPrototypePublicBaseURL(c),
	})
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, kpError("KP_AGENT_UNAVAILABLE", err.Error()))
		return
	}
	if !agentResp.Ok {
		code, message := "KP_AGENT_FAILED", "KP generation failed"
		if agentResp.Error != nil {
			if agentResp.Error.Code != "" {
				code = agentResp.Error.Code
			}
			if agentResp.Error.Message != "" {
				message = agentResp.Error.Message
			}
		}
		h.HandleResponse(c, status_http.InvalidArgument, kpError(code, message))
		return
	}
	if strings.TrimSpace(agentResp.HTML) == "" {
		h.HandleResponse(c, status_http.InternalServerError, kpError("KP_ARTIFACT_HTML_MISSING", "agent returned empty HTML"))
		return
	}
	prototypeURL := strings.TrimSpace(agentResp.PrototypeURL)
	if prototypeURL == "" && agentResp.Prototype != nil {
		prototypeURL = strings.TrimSpace(agentResp.Prototype.URL)
	}
	if prototypeURL == "" {
		h.HandleResponse(c, status_http.InternalServerError, kpError("KP_ARTIFACT_PROTOTYPE_MISSING", "agent returned no prototype URL"))
		return
	}
	prototype := gin.H{"url": prototypeURL}
	if agentResp.Prototype != nil {
		prototype["qaStatus"] = agentResp.Prototype.QAStatus
		prototype["screenCount"] = agentResp.Prototype.ScreenCount
		prototype["rendererVersion"] = agentResp.Prototype.RendererVersion
	}

	var pdfURL string
	if agentResp.HasPDF && h.centralRedis != nil {
		entry, err := json.Marshal(kpPdfCacheEntry{
			ProjectID:     cast.ToString(projectID),
			EnvironmentID: cast.ToString(environmentID),
			Title:         agentResp.Title,
		})
		if err != nil {
			h.log.Error("kp-proposal: failed to marshal pdf cache entry", logger.Error(err))
		} else if err := h.centralRedis.Set(c.Request.Context(), config.KpProposalPdfCachePrefix+agentResp.RequestID, entry, config.KpProposalPdfCacheTTL).Err(); err != nil {
			h.log.Error("kp-proposal: failed to cache pdf tenant mapping", logger.Error(err))
		} else {
			pdfURL = "/v1/kp-proposals/" + agentResp.RequestID + "/pdf"
		}
	}

	h.HandleResponse(c, status_http.OK, gin.H{
		"ok":           true,
		"status":       "completed",
		"requestId":    agentResp.RequestID,
		"title":        agentResp.Title,
		"html":         agentResp.HTML,
		"pageCount":    agentResp.PageCount,
		"prototypeUrl": prototypeURL,
		"prototype":    prototype,
		"pdfUrl":       pdfURL,
	})
}

// GetKpPrototype proxies a published prototype from the internal agent. The
// route is public so links embedded in downloaded PDFs work without an app JWT.
func (h *HandlerV1) GetKpPrototype(c *gin.Context) {
	publicID := strings.TrimSpace(c.Param("publicId"))
	if !isValidKpPrototypeID(publicID) {
		c.JSON(http.StatusNotFound, kpError("KP_PROTOTYPE_NOT_FOUND", "prototype not found"))
		return
	}
	if strings.TrimSpace(h.baseConf.KpAgentURL) == "" {
		c.JSON(http.StatusServiceUnavailable, kpError("KP_AGENT_UNAVAILABLE", "KP agent is not configured"))
		return
	}

	url := strings.TrimRight(h.baseConf.KpAgentURL, "/") + "/p/" + publicID + "/"
	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, kpError("KP_PROTOTYPE_FAILED", err.Error()))
		return
	}
	if h.baseConf.KpAgentAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+h.baseConf.KpAgentAPIKey)
	}

	res, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, kpError("KP_AGENT_UNAVAILABLE", err.Error()))
		return
	}
	defer res.Body.Close()

	for _, name := range []string{
		"Cache-Control",
		"Content-Security-Policy",
		"Referrer-Policy",
		"X-Content-Type-Options",
		"X-Robots-Tag",
	} {
		if value := res.Header.Get(name); value != "" {
			c.Header(name, value)
		}
	}
	contentType := res.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "text/html; charset=utf-8"
	}
	c.DataFromReader(res.StatusCode, res.ContentLength, contentType, res.Body, nil)
}

func (h *HandlerV1) callKpAgent(ctx context.Context, body kpAgentRequest) (*kpAgentResponse, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := strings.TrimRight(h.baseConf.KpAgentURL, "/") + "/v1/proposals"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if h.baseConf.KpAgentAPIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+h.baseConf.KpAgentAPIKey)
	}

	client := &http.Client{Timeout: kpAgentTimeout}
	res, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var parsed kpAgentResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("kp agent returned non-JSON (status %d): %s", res.StatusCode, truncate(string(raw), 300))
	}
	return &parsed, nil
}

// mapKpLocale maps the app locale to the agent's accepted locale set
// (uz-Latn, ru-RU, en); unknown/empty defaults to ru-RU.
func mapKpLocale(locale string) string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "uz", "uz-latn", "uz_latn":
		return "uz-Latn"
	case "en", "en-us":
		return "en"
	default:
		return "ru-RU"
	}
}

func kpPrototypePublicBaseURL(c *gin.Context) string {
	scheme := firstForwardedValue(c.GetHeader("X-Forwarded-Proto"))
	if scheme != "http" && scheme != "https" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := firstForwardedValue(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	return scheme + "://" + host + "/v1/kp-prototypes"
}

func firstForwardedValue(value string) string {
	return strings.ToLower(strings.TrimSpace(strings.Split(value, ",")[0]))
}

func isValidKpPrototypeID(value string) bool {
	if len(value) < 8 || len(value) > 64 {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '_' || character == '-' {
			continue
		}
		return false
	}
	return true
}

func kpError(code, message string) gin.H {
	return gin.H{"ok": false, "error": gin.H{"code": code, "message": message}}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
