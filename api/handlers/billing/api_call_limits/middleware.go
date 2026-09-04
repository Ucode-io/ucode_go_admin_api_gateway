package apilimits

import (
	"context"
	"fmt"
	"net/http"

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
				method:     c.Request.Method,
				route:      c.FullPath(),
				collection: c.Param("collection"),
			})
		}
		c.Next()
	}
}
