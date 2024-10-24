package tests

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

var FaaSNameForMongo = "integrationtestmongo-earth"

func TestOpenFaaS(t *testing.T) {
	_, err := UcodeApi.DoRequest(BaseUrl+"/v1/invoke_function/"+FaaSNameForMongo, http.MethodPost, map[string]interface{}{},
		map[string]string{
			"Resource-Id":    ResourceId,
			"Environment-Id": EnvironmentId,
			"X-API-KEY":      UcodeApi.Config().AppId,
			"Authorization":  "API-KEY",
		},
	)
	assert.NoError(t, err)
}
