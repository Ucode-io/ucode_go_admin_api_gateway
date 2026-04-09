package api_call_limits

import (
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

// Middleware перехватывает запросы и атомарно инкрементит счетчик в памяти (L1).
func (t *Tracker) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		projectID := c.GetString("project_id")

		if projectID != "" {
			// Атомарно увеличиваем счётчик. Работает за наносекунды.
			counter, _ := t.l1Cache.LoadOrStore(projectID, &atomic.Int64{})
			counter.(*atomic.Int64).Add(1)
		}

		c.Next()
	}
}
