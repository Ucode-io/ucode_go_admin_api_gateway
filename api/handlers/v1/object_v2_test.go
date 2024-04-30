package v1

import (
	"testing"
	"ucode/ucode_go_api_gateway/config"
	"ucode/ucode_go_api_gateway/pkg/caching"
	"ucode/ucode_go_api_gateway/pkg/logger"
	"ucode/ucode_go_api_gateway/services"
	"ucode/ucode_go_api_gateway/storage"

	"github.com/gin-gonic/gin"
)

func TestHandlerV1_GetListV2(t *testing.T) {
	type fields struct {
		baseConf        config.BaseConfig
		projectConfs    map[string]config.Config
		log             logger.LoggerI
		services        services.ServiceNodesI
		companyServices services.CompanyServiceI
		authService     services.AuthServiceManagerI
		redis           storage.RedisStorageI
		cache           *caching.ExpiringLRUCache
	}
	type args struct {
		c *gin.Context
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HandlerV1{
				baseConf:        tt.fields.baseConf,
				projectConfs:    tt.fields.projectConfs,
				log:             tt.fields.log,
				services:        tt.fields.services,
				companyServices: tt.fields.companyServices,
				authService:     tt.fields.authService,
				redis:           tt.fields.redis,
				cache:           tt.fields.cache,
			}
			h.GetListV2(tt.args.c)
		})
	}
}
