package tests

import (
	"encoding/json"
	"testing"
	"time"

	sdk "github.com/golanguzb70/ucode-sdk"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Done
func TestCRUDPgPg(t *testing.T) {
	guid := uuid.New().String()

	_, _, err := UcodeApiForPg.CreateObject(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"guid":                             guid,
				"single_line_field":                fakeData.CompanyName(),
				"multi_line_field":                 fakeData.FirstName(),
				"email_field":                      fakeData.Email(),
				"internation_phone_field":          fakeData.PhoneNumber(),
				"date_time_field":                  time.Now(),
				"date_time_without_timezone_field": time.Now().UTC(),
				"date_field":                       time.Now(),
			},
		},
	})
	assert.NoError(t, err)

	_, _, err = UcodeApiForPg.UpdateObject(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"guid":             guid,
				"multi_line_field": fakeData.Name(),
				"checkbox_field":   true,
			},
		},
	})
	assert.NoError(t, err)

	slimResponse, _, err := UcodeApiForPg.GetSingleSlim(&sdk.Argument{
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

	itemResponse, _, err := UcodeApiForPg.GetSingle(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"guid": guid,
			},
		},
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, itemResponse.Data.Data.Response, "item response should not be empty")

	_, err = UcodeApiForPg.Delete(&sdk.Argument{
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
func TestGetListSlimPg(t *testing.T) {
	getProductResp, _, err := UcodeApiForPg.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug: "product",
		Request:   sdk.Request{Data: map[string]interface{}{}},
		Limit:     10,
		Page:      1,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, getProductResp.Data.Data.Response, "response not equal to limit")
}

// Done
func TestGetListSlimPaginationPg(t *testing.T) {
	getProductResp, _, err := UcodeApiForPg.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug: "product",
		Request:   sdk.Request{Data: map[string]interface{}{}},
		Limit:     5,
		Page:      4,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, getProductResp.Data.Data.Response, "response not equal to limit")

	getProductResp, _, err = UcodeApiForPg.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug:   "product",
		Request:     sdk.Request{Data: map[string]interface{}{}},
		Limit:       5,
		Page:        5,
		DisableFaas: true,
	})
	assert.NoError(t, err)

	assert.Empty(t, getProductResp.Data.Data.Response, "response should be emtpy")
}

// Done
func TestGetListSlimWithRelationPg(t *testing.T) {
	getProductResp, _, err := UcodeApiForPg.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{"with_relations": true},
		},
		Limit:       10,
		Page:        1,
		DisableFaas: true,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, getProductResp.Data.Data.Response, "wrong response")
}

// Done
func TestGetListSlimWithDatePg(t *testing.T) { // $lt and $gte
	getProductReq := sdk.Request{
		Data: map[string]interface{}{
			"date_time_field": map[string]interface{}{
				"$gte": "2024-10-01T00:04:19.336Z",
			}},
	}

	getProductResp, _, err := UcodeApiForPg.GetListSlim(&sdk.ArgumentWithPegination{
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

	getProductResp, _, err = UcodeApiForPg.GetListSlim(&sdk.ArgumentWithPegination{
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
func TestGetListSlimWithInPg(t *testing.T) { // $in
	getProductReq := sdk.Request{
		Data: map[string]interface{}{
			"increment_id_field": map[string]interface{}{
				"$in": []string{"T-000000021", "T-000000023", "T-000000026"},
			}},
	}

	getProductResp, _, err := UcodeApiForPg.GetListSlim(&sdk.ArgumentWithPegination{
		TableSlug:   "product",
		Request:     getProductReq,
		Limit:       10,
		Page:        1,
		DisableFaas: true,
	})
	assert.NoError(t, err)

	assert.NotEmpty(t, getProductResp.Data.Data.Response, err)
}

// Does Not Work
// func TestGetListSlimWithBooleanPg(t *testing.T) {
// 	getProductResp, _, err := UcodeApiForPg.GetListSlim(&sdk.ArgumentWithPegination{
// 		TableSlug:   "product",
// 		Request:     sdk.Request{Data: map[string]interface{}{"switch_field": false}},
// 		Limit:       10,
// 		Page:        1,
// 		DisableFaas: true,
// 	})
// 	fmt.Println("ERROR", err)
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, getProductResp.Data.Data.Response, "Switch Does not work")
// }

// Done
func TestGetListObjectPg(t *testing.T) {
	getProductResp, _, err := UcodeApiForPg.GetList(&sdk.ArgumentWithPegination{
		TableSlug:   "product",
		Request:     sdk.Request{Data: map[string]interface{}{}},
		Limit:       10,
		Page:        1,
		DisableFaas: true,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, getProductResp.Data.Data.Response, "Switch Does not work")
}

// Done
func TestGetListObjectSearchPg(t *testing.T) {
	getProductReq := sdk.Request{
		Data: map[string]interface{}{"search": "+99894", "view_fields": []string{"internation_phone_field"}},
	}
	getProductResp, _, err := UcodeApiForPg.GetList(&sdk.ArgumentWithPegination{
		TableSlug:   "product",
		Request:     getProductReq,
		Limit:       10,
		Page:        1,
		DisableFaas: true,
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, getProductResp.Data.Data.Response, "Switch Does not work")

	getProductReq = sdk.Request{
		Data: map[string]interface{}{"search": "+4", "view_fields": []string{"internation_phone_field"}},
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
func TestGetListAutoFilterPg(t *testing.T) {
	resp, err := LoginPg()
	assert.NoError(t, err)

	body, err := UcodeApi.DoRequest(BaseUrlStaging+"/v2/object/get-list/product", "POST", map[string]interface{}{
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
func TestGetListExcelPg(t *testing.T) {
	body, err := UcodeApi.DoRequest(BaseUrlStaging+"/v1/object/excel/product", "POST", ExcelReqPg, map[string]string{
		"authorization": "API-KEY",
		"X-API-KEY":     UcodeApiForPg.Config().AppId,
	})
	assert.NoError(t, err)
	var response ExcelResponse
	err = json.Unmarshal(body, &response)
	assert.NoError(t, err)

	assert.NotEmpty(t, response.Data.Data.Link)
}

// NOT DONE
// func TestGetListRBACPg(t *testing.T) {
// 	resp, err := Login()
// 	assert.NoError(t, err)

// 	body, err := UcodeApi.DoRequest(BaseUrl+"/v2/object/get-list/company", "POST", map[string]interface{}{
// 		"data": map[string]interface{}{},
// 	}, map[string]string{
// 		"Authorization": "Bearer " + resp,
// 	})
// 	assert.NoError(t, err)
// 	var response GetListApiResponse
// 	fmt.Println("GGGG", string(body))
// 	// fmt.Println(response.Data.Data.Response)
// 	err = json.Unmarshal(body, &response)
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, response.Data.Data.Response)
// }

// Done
func TestMultipleCRUDPg(t *testing.T) {
	var ids = []string{}
	var multipleInsert = []map[string]interface{}{}
	var multipleUpdate = []map[string]interface{}{}
	UcodeApiForPg.Config().RequestTimeout = time.Second * 30

	for i := 0; i < 10; i++ {
		guid := uuid.New().String()
		multipleInsert = append(multipleInsert, map[string]interface{}{
			"guid":                             guid,
			"single_line_field":                fakeData.CompanyName(),
			"multi_line_field":                 fakeData.FirstName(),
			"email_field":                      fakeData.Email(),
			"internation_phone_field":          fakeData.PhoneNumber(),
			"date_time_field":                  time.Now(),
			"date_time_without_timezone_field": time.Now().UTC(),
			"date_field":                       time.Now(),
			"is_new":                           true,
		})

		multipleUpdate = append(multipleUpdate, map[string]interface{}{
			"guid":                             guid,
			"single_line_field":                fakeData.CompanyName(),
			"multi_line_field":                 fakeData.FirstName(),
			"email_field":                      fakeData.Email(),
			"internation_phone_field":          fakeData.PhoneNumber(),
			"date_time_field":                  time.Now(),
			"date_time_without_timezone_field": time.Now().UTC(),
			"date_field":                       time.Now(),
			"is_new":                           false,
		})
		ids = append(ids, guid)
	}

	_, _, err := UcodeApiForPg.MultipleUpdate(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"objects": multipleInsert,
			},
		},
		DisableFaas: true,
	})
	assert.NoError(t, err)

	_, _, err = UcodeApiForPg.MultipleUpdate(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"objects": multipleUpdate,
			},
		},
		DisableFaas: true,
	})
	assert.NoError(t, err)

	_, err = UcodeApiForPg.MultipleDelete(&sdk.Argument{
		TableSlug: "product",
		Request: sdk.Request{
			Data: map[string]interface{}{
				"ids": ids,
			},
		},
	})
	assert.NoError(t, err)
}
