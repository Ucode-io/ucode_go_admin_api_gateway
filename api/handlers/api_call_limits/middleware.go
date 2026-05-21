package api_call_limits

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"

	"github.com/gin-gonic/gin"
)

func (t *Tracker) BillingLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.GetString("project_id")
		if projectID == "" {
			c.Next()
			return
		}

		if config.RateLimitSkipFiles[c.Param("collection")] {
			c.Next()
			return
		}

		limitKey := fmt.Sprintf(config.KeyBillingApiLimit, projectID)
		if val, err := t.rdb.Get(context.Background(), limitKey).Result(); err == nil && val == "0" {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, status_http.Response{
				Status:      status_http.PaymentRequired.Status,
				Description: "Monthly API call limit exceeded. Please upgrade your plan.",
				Data: models.PaymentRequiredData{
					Type: "payment_required",
					Code: "api_call_limit",
					Unit: "requests",
				},
			})
			return
		}

		c.Next()
	}
}

func (t *Tracker) ApiCallCountMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.GetString("project_id")

		if projectID != "" {
			counter, _ := t.l1Cache.LoadOrStore(projectID, &atomic.Int64{})
			switch v := counter.(type) {
			case *atomic.Int64:
				v.Add(1)
			default:
				log.Printf("[ApiCallLimits] Warning: Invalid counter type in L1 cache for project %s", projectID)
			}
		}

		c.Next()
	}
}
