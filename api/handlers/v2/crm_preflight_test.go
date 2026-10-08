package v2

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNativeUnsupportedShapesRejectBeforeAnyRPCOrHook(t *testing.T) {
	for _, tc := range []struct{ method, route, path, body string }{{"POST", "/v2/items/:collection/multiple-insert", "/v2/items/deals/multiple-insert", `{"data":{"name":"x"}}`}, {"POST", "/v2/items/:collection/board", "/v2/items/deals/board", `{}`}, {"PATCH", "/v2/items/:collection", "/v2/items/deals", `{"data":{"guid":"x"}}`}, {"DELETE", "/v2/items/:collection", "/v2/items/deals", `{"data":{"guid":"x"}}`}, {"PUT", "/v2/items/:collection/:id", "/v2/items/deals/a", `{"data":{"guid":"b"}}`}, {"POST", "/v2/items/:collection", "/v2/items/deals", `{"data":{"auth_guid":"x"}}`}, {"PUT", "/v2/items/:collection/:id", "/v2/items/deals/a", `{"data":{"pbx_branch":"andijon"}}`}} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			r := gin.New()
			r.Handle(tc.method, tc.route, func(c *gin.Context) {
				if err := (&HandlerV2{}).CRMNativePreflight(c); err == nil {
					t.Fatal("unsupported shape admitted")
				}
				c.Status(403)
			})
			w := httptest.NewRecorder()
			r.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body)))
			if w.Code != 403 {
				t.Fatal(w.Code)
			}
		})
	}
}
