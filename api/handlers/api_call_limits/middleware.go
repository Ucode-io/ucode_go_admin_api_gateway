package api_call_limits

import (
	"log"
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
			switch v := counter.(type) {
			case *atomic.Int64:
				v.Add(1)
				// Разгружаем логи, чтобы не спамило при тысячах RPS (если нужен детальный дебаг - раскомментируй)
				// log.Printf("[ApiCallLimits] Tracked +1 request for projectID: %s", projectID)
			default:
				log.Printf("[ApiCallLimits] Warning: Invalid counter type in L1 cache for project %s", projectID)
			}
		}

		c.Next()
	}
}
