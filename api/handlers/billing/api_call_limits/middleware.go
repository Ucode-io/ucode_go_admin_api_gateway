package apilimits

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"

	"github.com/gin-gonic/gin"
)

// BillingLimitMiddleware aborts requests when a project's monthly API call
// limit flag is set to "0" in Redis by BillingLimitWorker.
func (t *Tracker) BillingLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.GetString("project_id")
		if projectID == "" || config.RateLimitSkipFiles[c.Param("collection")] {
			c.Next()
			return
		}

		limitKey := fmt.Sprintf(config.KeyBillingApiLimit, projectID)
		if val, err := t.rdb.Get(context.Background(), limitKey).Result(); err == nil && val == "0" {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, status_http.Response{
				Status:      status_http.PaymentRequired.Status,
				Description: "Monthly API call limit exceeded. Please upgrade your plan.",
				Data:        models.PaymentApiCallLimit,
			})
			return
		}

		c.Next()
	}
}

// ApiCallCountMiddleware increments the per-project L1 counter on every request,
// sliced by where the call came from and which route it hit. Counts are flushed
// to Redis by Tracker on its flush interval.
//
// source is fixed per route group at registration time: admin/builder traffic is
// counted but never blocked, client traffic is both counted and blocked, and
// telling them apart is what stops a customer's own builder usage from
// dominating their breakdown.
func (t *Tracker) ApiCallCountMiddleware(source string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if projectID := c.GetString("project_id"); projectID != "" {
			t.add(usageKey{
				projectID:  projectID,
				source:     source,
				authType:   authKind(c),
				actorID:    actorID(c),
				actorName:  c.GetString("actor_name"),
				method:     c.Request.Method,
				route:      c.FullPath(),
				collection: c.Param("collection"),
			})
		}
		c.Next()
	}
}

// authKind reports how the caller authenticated. It reads the header directly
// rather than the "auth" context value, which only some middlewares populate
// and which never carries the bearer case on the client routes.
func authKind(c *gin.Context) string {
	switch scheme := c.GetHeader("Authorization"); {
	case strings.HasPrefix(scheme, "Bearer"):
		return config.UsageAuthBearer
	case strings.HasPrefix(scheme, "API-KEY"):
		return config.UsageAuthApiKey
	}
	if c.GetHeader("X-API-KEY") != "" {
		return config.UsageAuthApiKey
	}
	return ""
}

// actorID reports who signed the request: the api_keys record id when a key was
// used, the user id when a token was. Never the credential itself — the value in
// X-API-KEY is the secret, and these rows are read back by a project-facing
// endpoint.
func actorID(c *gin.Context) string {
	if id := c.GetString("actor_id"); id != "" {
		return id
	}
	return c.GetString("user_id")
}
