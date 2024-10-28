package tests

import (
	"encoding/json"
	"testing"
	"time"

	sdk "github.com/golanguzb70/ucode-sdk"
	"github.com/google/uuid"
	"github.com/manveru/faker"
	"github.com/stretchr/testify/assert"
)

// Done
func TestCRUD(t *testing.T) {
	guid := uuid.New().String()
	faker1, err := faker.New("en")
	assert.NoError(t, err, "faker error")

	_, _, err = UcodeApiForStaging.CreateObject(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"guid":                             guid,
				"single_line_field":                faker1.CompanyName(),
				"multi_line_field":                 faker1.FirstName(),
				"email_field":                      faker1.Email(),
				"internation_phone_field":          faker1.PhoneNumber(),
				"date_time_field":                  time.Now(),
				"date_time_without_timezone_field": time.Now().UTC(),
				"date_field":                       time.Now(),
			},
		},
	})
	assert.NoError(t, err)

	_, _, err = UcodeApiForStaging.UpdateObject(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"guid":             guid,
				"multi_line_field": faker1.Name(),
				"checkbox_field":   true,
			},
		},
	})
	assert.NoError(t, err)

	slimResponse, _, err := UcodeApiForStaging.GetSingleSlim(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"guid":           guid,
				"with_relations": true,
			},
		},
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, slimResponse.Data.Data.Response, "item response should not be empty")

	itemResponse, _, err := UcodeApiForStaging.GetSingle(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"guid": guid,
			},
		},
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, itemResponse.Data.Data.Response, "item response should not be empty")

	_, err = UcodeApiForStaging.Delete(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"guid": guid,
			},
		},
	})
	assert.NoError(t, err)
}

// Done
func TestGetListSlim(t *testing.T) {
	getProductReq := sdk.Request{Data: map[string]interface{}{}}
	getProductResp, _, err := UcodeApiForStaging.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug: "product",
		Request:   getProductReq,
		Limit:     10,
		Page:      1,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, getProductResp.Data.Data.Response, "response not equal to limit")
}

// Done
func TestGetListSlimPagination(t *testing.T) {
	getProductReq := sdk.Request{Data: map[string]interface{}{}}
	getProductResp, _, err := UcodeApiForStaging.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug: "product",
		Request:   getProductReq,
		Limit:     5,
		Page:      4,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, getProductResp.Data.Data.Response, "response not equal to limit")

	getProductResp, _, err = UcodeApi.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug:   "product",
		Request:     getProductReq,
		Limit:       5,
		Page:        5,
		DisableFaas: true,
	})
	assert.NoError(t, err)

	assert.Empty(t, getProductResp.Data.Data.Response, "response should be emtpy")
}

// Done
func TestGetListSlimWithRelation(t *testing.T) {
	getProductReq := sdk.Request{
		Data: map[string]interface{}{"with_relations": true},
	}
	getProductResp, _, err := UcodeApiForStaging.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug:   "product",
		Request:     getProductReq,
		Limit:       10,
		Page:        1,
		DisableFaas: true,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, getProductResp.Data.Data.Response, "wrong response")
}

// Done
func TestGetListSlimWithDate(t *testing.T) { // $lt and $gte
	getProductReq := sdk.Request{
		Data: map[string]interface{}{
			"date_time_field": map[string]interface{}{
				"$gte": "2024-10-01T00:04:19.336Z",
			}},
	}

	getProductResp, _, err := UcodeApiForStaging.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug: "product",
		Request:   getProductReq,
		Limit:     10,
		Page:      1,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, getProductResp.Data.Data.Response, err)

	getProductReq = sdk.Request{
		Data: map[string]interface{}{
			"date_time_field": map[string]interface{}{
				"$gte": "2024-10-01T00:04:19.336Z",
				"$lt":  "2024-10-06T00:04:19.336Z",
			}},
	}

	getProductResp, _, err = UcodeApi.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug:   "product",
		Request:     getProductReq,
		Limit:       10,
		Page:        1,
		DisableFaas: true,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, getProductResp.Data.Data.Response, err)
}

// Done
func TestGetListSlimWithEq(t *testing.T) { // $eq
	getProductReq := sdk.Request{
		Data: map[string]interface{}{
			"increment_id_field": map[string]interface{}{
				"$eq": "T-000000021",
			}},
	}

	getProductResp, _, err := UcodeApiForStaging.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug:   "product",
		Request:     getProductReq,
		Limit:       10,
		Page:        1,
		DisableFaas: true,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, getProductResp.Data.Data.Response, err)
}

// Done
func TestGetListSlimWithIn(t *testing.T) { // $in
	getProductReq := sdk.Request{
		Data: map[string]interface{}{
			"increment_id_field": map[string]interface{}{
				"$in": []string{"T-000000022", "T-000000023", "T-000000024"},
			}},
	}

	getProductResp, _, err := UcodeApiForStaging.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug:   "product",
		Request:     getProductReq,
		Limit:       10,
		Page:        1,
		DisableFaas: true,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, getProductResp.Data.Data.Response, err)
}

// Done
func TestGetListSlimWithBoolean(t *testing.T) {
	getProductReq := sdk.Request{
		Data: map[string]interface{}{
			"switch_field": false},
	}
	getProductResp, _, err := UcodeApiForStaging.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug:   "product",
		Request:     getProductReq,
		Limit:       10,
		Page:        1,
		DisableFaas: true,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, getProductResp.Data.Data.Response, "Switch Does not work")
}

// Done
func TestGetListObject(t *testing.T) {
	getProductReq := sdk.Request{
		Data: map[string]interface{}{},
	}
	getProductResp, _, err := UcodeApiForStaging.GetList(&sdk.ArgumentWithPegination{
		TableSlug:   "product",
		Request:     getProductReq,
		Limit:       10,
		Page:        1,
		DisableFaas: true,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, getProductResp.Data.Data.Response, "Switch Does not work")
}

// Done
func TestGetListObjectSearch(t *testing.T) {
	getProductReq := sdk.Request{
		Data: map[string]interface{}{
			"search":      "+99894",
			"view_fields": []string{"internation_phone_field"},
		},
	}
	getProductResp, _, err := UcodeApiForStaging.GetList(&sdk.ArgumentWithPegination{
		TableSlug:   "product",
		Request:     getProductReq,
		Limit:       10,
		Page:        1,
		DisableFaas: true,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, getProductResp.Data.Data.Response, "Switch Does not work")

	getProductReq = sdk.Request{
		Data: map[string]interface{}{
			"search":      "+4",
			"view_fields": []string{"internation_phone_field"},
		},
	}
	getProductResp, _, err = UcodeApi.GetList(&sdk.ArgumentWithPegination{
		TableSlug: "product",
		Request:   getProductReq,
		Limit:     10,
		Page:      1,
	})
	assert.NoError(t, err)
	assert.Empty(t, getProductResp.Data.Data.Response, "Response should be empty")
}

// Done
func TestGetAggregation(t *testing.T) {
	getProductReq := sdk.Request{Data: map[string]interface{}{"pipelines": []map[string]interface{}{{"$group": map[string]interface{}{"_id": "$single_line_field"}}}}}
	getProductResp, _, err := UcodeApiForStaging.GetListAggregation(&sdk.Argument{
		TableSlug:   "product",
		Request:     getProductReq,
		DisableFaas: true,
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, getProductResp.Data.Data.Data, "Switch Does not work")
}

// Done
func TestGetListAutoFilter(t *testing.T) {
	resp, err := Login()
	assert.NoError(t, err)

	body, err := UcodeApiForStaging.DoRequest(BaseUrlStaging+"/v2/object/get-list/product", "POST", map[string]interface{}{
		"data": map[string]interface{}{},
	}, map[string]string{
		"Authorization": "Bearer " + resp,
	})
	assert.NoError(t, err)
	var response GetListApiResponse

	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.NotEmpty(t, response.Data.Data.Response)
}

// Done
func TestGetListExcel(t *testing.T) {
	body, err := UcodeApi.DoRequest(BaseUrlStaging+"/v1/object/excel/product", "POST", ExcelReq, map[string]string{
		"authorization": "API-KEY",
		"X-API-KEY":     UcodeApiForStaging.Config().AppId,
	})
	assert.NoError(t, err)
	var response ExcelResponse
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.NotEmpty(t, response.Data.Data.Link)
}

// Done
func TestGetListRBAC(t *testing.T) {
	resp, err := Login()
	assert.NoError(t, err)

	body, err := UcodeApi.DoRequest(BaseUrlStaging+"/v2/object/get-list/company", "POST", map[string]interface{}{
		"data": map[string]interface{}{},
	}, map[string]string{
		"Authorization": "Bearer " + resp,
	})
	assert.NoError(t, err)
	var response GetListApiResponse

	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.NotEmpty(t, response.Data.Data.Response)
}

// Done
func TestMultipleCRUD(t *testing.T) {
	var ids = []string{}
	var multipleInsert = []map[string]interface{}{}
	var multipleUpdate = []map[string]interface{}{}
	UcodeApiForStaging.Config().RequestTimeout = time.Second * 30
	faker1, err := faker.New("en")
	assert.NoError(t, err, "faker error")

	for i := 0; i < 10; i++ {
		guid := uuid.New().String()
		multipleInsert = append(multipleInsert, map[string]interface{}{
			"guid":                             guid,
			"single_line_field":                faker1.CompanyName(),
			"multi_line_field":                 faker1.FirstName(),
			"email_field":                      faker1.Email(),
			"internation_phone_field":          faker1.PhoneNumber(),
			"date_time_field":                  time.Now(),
			"date_time_without_timezone_field": time.Now().UTC(),
			"date_field":                       time.Now(),
			"is_new":                           true,
		})

		multipleUpdate = append(multipleUpdate, map[string]interface{}{
			"guid":                             guid,
			"single_line_field":                faker1.CompanyName(),
			"multi_line_field":                 faker1.FirstName(),
			"email_field":                      faker1.Email(),
			"internation_phone_field":          faker1.PhoneNumber(),
			"date_time_field":                  time.Now(),
			"date_time_without_timezone_field": time.Now().UTC(),
			"date_field":                       time.Now(),
			"is_new":                           false,
		})
		ids = append(ids, guid)
	}

	_, _, err = UcodeApiForStaging.MultipleUpdate(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"objects": multipleInsert,
			},
		},
		DisableFaas: true,
	})
	assert.NoError(t, err)

	_, _, err = UcodeApiForStaging.MultipleUpdate(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"objects": multipleUpdate,
			},
		},
		DisableFaas: true,
	})
	assert.NoError(t, err)

	_, err = UcodeApiForStaging.MultipleDelete(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"ids": ids,
			},
		},
	})
	assert.NoError(t, err)
}
