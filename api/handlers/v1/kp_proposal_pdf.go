package v1

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// kpAgentPdfTimeout bounds the proxied PDF download call to the agent. Kept
// short relative to kpAgentTimeout (generation): the PDF was already rendered
// during POST /v1/kp-proposals, this call is just a file fetch.
const kpAgentPdfTimeout = 30 * time.Second

var kpRequestIDPattern = regexp.MustCompile(`^KP-[A-Za-z0-9_-]+$`)

const kpRequestIDMaxLen = 96

// isValidKpRequestID guards the path parameter before it reaches the tenant
// cache lookup or gets embedded in the upstream agent URL.
func isValidKpRequestID(requestID string) bool {
	return len(requestID) <= kpRequestIDMaxLen && kpRequestIDPattern.MatchString(requestID)
}

// DownloadKpProposalPDF godoc
// @Security ApiKeyAuth
// @ID v1_download_kp_proposal_pdf
// @Router /v1/kp-proposals/{requestId}/pdf [GET]
// @Summary Download the PDF rendition of a previously generated KP
// @Description Proxies GET {KP_AGENT_URL}/v1/proposals/{requestId}/pdf, enforcing that requestId belongs to the caller's project/environment. Streams a binary application/pdf response, no JSON envelope.
// @Tags KP
// @Param Authorization header string true "Bearer access token"
// @Param requestId path string true "KP request id"
// @Success 200 {file} binary "PDF file"
// @Failure 400 {object} status_http.Response "Bad Request"
// @Failure 403 {object} status_http.Response "Forbidden"
// @Failure 404 {object} status_http.Response "Not Found"
// @Failure 410 {object} status_http.Response "Gone"
// @Failure 502 {object} status_http.Response "Bad Gateway"
// @Failure 503 {object} status_http.Response "Service Unavailable"
func (h *HandlerV1) DownloadKpProposalPDF(c *gin.Context) {
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

	requestID := c.Param("requestId")
	if !isValidKpRequestID(requestID) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid requestId format")
		return
	}

	// requestId -> {projectId,environmentId} tenant-isolation lookup. This is
	// intentionally the source of "does this artifact exist" too: a request that
	// never had a PDF, or whose cache entry expired, looks identical to the agent
	// having nothing for it, and both map to KP_ARTIFACT_NOT_FOUND.
	if h.centralRedis == nil {
		h.HandleResponse(c, status_http.NotFound, kpError("KP_ARTIFACT_NOT_FOUND", "PDF artifact not found"))
		return
	}
	cached, err := h.centralRedis.Get(c.Request.Context(), config.KpProposalPdfCachePrefix+requestID).Bytes()
	if err != nil {
		h.HandleResponse(c, status_http.NotFound, kpError("KP_ARTIFACT_NOT_FOUND", "PDF artifact not found"))
		return
	}
	var entry kpPdfCacheEntry
	if err := json.Unmarshal(cached, &entry); err != nil {
		h.HandleResponse(c, status_http.NotFound, kpError("KP_ARTIFACT_NOT_FOUND", "PDF artifact not found"))
		return
	}
	if entry.ProjectID != cast.ToString(projectID) || entry.EnvironmentID != cast.ToString(environmentID) {
		h.HandleResponse(c, status_http.Forbidden, kpError("KP_ARTIFACT_FORBIDDEN", "requestId does not belong to the current project/environment"))
		return
	}

	if strings.TrimSpace(h.baseConf.KpAgentURL) == "" {
		h.HandleResponse(c, status_http.ServiceUnavailable, kpError("KP_AGENT_UNAVAILABLE", "KP agent is not configured"))
		return
	}

	agentURL := strings.TrimRight(h.baseConf.KpAgentURL, "/") + "/v1/proposals/" + requestID + "/pdf"
	httpReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, agentURL, nil)
	if err != nil {
		h.HandleResponse(c, status_http.InternalServerError, err.Error())
		return
	}
	if h.baseConf.KpAgentAPIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+h.baseConf.KpAgentAPIKey)
	}

	client := &http.Client{Timeout: kpAgentPdfTimeout}
	res, err := client.Do(httpReq)
	if err != nil {
		h.HandleResponse(c, status_http.BadGateway, kpError("KP_AGENT_UNAVAILABLE", err.Error()))
		return
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		// handled below
	case http.StatusNotFound:
		h.HandleResponse(c, status_http.NotFound, kpError("KP_ARTIFACT_NOT_FOUND", "PDF artifact not found"))
		return
	case http.StatusGone:
		h.HandleResponse(c, status_http.Gone, kpError("KP_ARTIFACT_GONE", "PDF artifact is no longer available"))
		return
	case http.StatusUnauthorized, http.StatusForbidden:
		h.HandleResponse(c, status_http.BadGateway, kpError("KP_AGENT_AUTH_FAILED", "KP agent rejected the service credential"))
		return
	default:
		h.HandleResponse(c, status_http.BadGateway, kpError("KP_AGENT_UNAVAILABLE", "KP agent returned an unexpected status: "+res.Status))
		return
	}

	if !strings.HasPrefix(res.Header.Get("Content-Type"), "application/pdf") {
		h.HandleResponse(c, status_http.BadGateway, kpError("KP_ARTIFACT_INVALID", "KP agent did not return a PDF"))
		return
	}

	if disposition := res.Header.Get("Content-Disposition"); disposition != "" {
		c.Header("Content-Disposition", disposition)
	} else {
		c.Header("Content-Disposition", `attachment; filename="proposal.pdf"`)
	}
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")

	c.DataFromReader(http.StatusOK, res.ContentLength, "application/pdf", res.Body, nil)
}
