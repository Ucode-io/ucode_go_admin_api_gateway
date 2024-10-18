package tests

// func TestCRUD(t *testing.T) {
// 	guid := uuid.New().String()
// 	faker1, err := faker.New("en")
// 	assert.NoError(t, err, "faker error")

// 	_, _, err = UcodeApi.CreateObject(&sdk.Argument{
// 		TableSlug: "product",
// 		Request: sdk.Request{
// 			Data: map[string]interface{}{
// 				"guid":                             guid,
// 				"single_line_field":                faker1.CompanyName(),
// 				"multi_line_field":                 faker1.FirstName(),
// 				"email_field":                      faker1.Email(),
// 				"internation_phone_field":          faker1.PhoneNumber(),
// 				"date_time_field":                  time.Now(),
// 				"date_time_without_timezone_field": time.Now().UTC(),
// 				"date_field":                       time.Now(),
// 			},
// 		},
// 	})
// 	assert.NoError(t, err)

// 	_, _, err = UcodeApi.UpdateObject(&sdk.Argument{
// 		TableSlug: "product",
// 		Request: sdk.Request{
// 			Data: map[string]interface{}{
// 				"guid":             guid,
// 				"multi_line_field": faker1.Name(),
// 				"checkbox_field":   true,
// 			},
// 		},
// 	})
// 	assert.NoError(t, err)

// 	slimResponse, _, err := UcodeApi.GetSingleSlim(&sdk.Argument{
// 		TableSlug: "product",
// 		Request: sdk.Request{
// 			Data: map[string]interface{}{
// 				"guid":           guid,
// 				"with_relations": true,
// 			},
// 		},
// 	})
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, slimResponse.Data.Data.Response, "item response should not be empty")

// 	itemResponse, _, err := UcodeApi.GetSingle(&sdk.Argument{
// 		TableSlug: "product",
// 		Request: sdk.Request{
// 			Data: map[string]interface{}{
// 				"guid": guid,
// 			},
// 		},
// 	})
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, itemResponse.Data.Data.Response, "item response should not be empty")

// 	_, err = UcodeApi.Delete(&sdk.Argument{
// 		TableSlug: "product",
// 		Request: sdk.Request{
// 			Data: map[string]interface{}{
// 				"guid": guid,
// 			},
// 		},
// 	})
// 	assert.NoError(t, err)
// }

// func TestGetListSlim(t *testing.T) {
// 	getProductReq := sdk.Request{Data: map[string]interface{}{}}
// 	getProductResp, _, err := UcodeApi.GetListSlim(&sdk.ArgumentWithPegination{
// 		TableSlug: "product",
// 		Request:   getProductReq,
// 		Limit:     10,
// 		Page:      1,
// 	})
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, getProductResp.Data.Data.Response, "response not equal to limit")
// }

// func TestGetListSlimPagination(t *testing.T) {
// 	getProductReq := sdk.Request{Data: map[string]interface{}{}}
// 	getProductResp, _, err := UcodeApi.GetListSlim(&sdk.ArgumentWithPegination{
// 		TableSlug: "product",
// 		Request:   getProductReq,
// 		Limit:     5,
// 		Page:      4,
// 	})
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, getProductResp.Data.Data.Response, "response not equal to limit")

// 	getProductResp, _, err = UcodeApi.GetListSlim(&sdk.ArgumentWithPegination{
// 		TableSlug:   "product",
// 		Request:     getProductReq,
// 		Limit:       5,
// 		Page:        5,
// 		DisableFaas: true,
// 	})
// 	assert.NoError(t, err)

// 	assert.Empty(t, getProductResp.Data.Data.Response, "response should be emtpy")
// }

// func TestGetListSlimWithRelation(t *testing.T) {
// 	getProductReq := sdk.Request{
// 		Data: map[string]interface{}{"with_relations": true},
// 	}
// 	getProductResp, _, err := UcodeApi.GetListSlim(&sdk.ArgumentWithPegination{
// 		TableSlug:   "product",
// 		Request:     getProductReq,
// 		Limit:       10,
// 		Page:        1,
// 		DisableFaas: true,
// 	})
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, getProductResp.Data.Data.Response, "wrong response")
// }

// func TestGetListSlimWithDate(t *testing.T) { // $lt and $gte
// 	getProductReq := sdk.Request{
// 		Data: map[string]interface{}{
// 			"date_time_field": map[string]interface{}{
// 				"$gte": "2024-10-01T00:04:19.336Z",
// 			}},
// 	}

// 	getProductResp, _, err := UcodeApi.GetListSlim(&sdk.ArgumentWithPegination{
// 		TableSlug: "product",
// 		Request:   getProductReq,
// 		Limit:     10,
// 		Page:      1,
// 	})
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, getProductResp.Data.Data.Response, err)

// 	getProductReq = sdk.Request{
// 		Data: map[string]interface{}{
// 			"date_time_field": map[string]interface{}{
// 				"$gte": "2024-10-01T00:04:19.336Z",
// 				"$lt":  "2024-10-06T00:04:19.336Z",
// 			}},
// 	}

// 	getProductResp, _, err = UcodeApi.GetListSlim(&sdk.ArgumentWithPegination{
// 		TableSlug:   "product",
// 		Request:     getProductReq,
// 		Limit:       10,
// 		Page:        1,
// 		DisableFaas: true,
// 	})
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, getProductResp.Data.Data.Response, err)
// }

// func TestGetListSlimWithEq(t *testing.T) { // $eq
// 	getProductReq := sdk.Request{
// 		Data: map[string]interface{}{
// 			"increment_id_field": map[string]interface{}{
// 				"$eq": "T-000000020",
// 			}},
// 	}

// 	getProductResp, _, err := UcodeApi.GetListSlim(&sdk.ArgumentWithPegination{
// 		TableSlug:   "product",
// 		Request:     getProductReq,
// 		Limit:       10,
// 		Page:        1,
// 		DisableFaas: true,
// 	})
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, getProductResp.Data.Data.Response, err)
// }

// func TestGetListSlimWithIn(t *testing.T) { // $in
// 	getProductReq := sdk.Request{
// 		Data: map[string]interface{}{
// 			"increment_id_field": map[string]interface{}{
// 				"$in": []string{"T-000000020", "T-000000019", "T-000000018"},
// 			}},
// 	}

// 	getProductResp, _, err := UcodeApi.GetListSlim(&sdk.ArgumentWithPegination{
// 		TableSlug:   "product",
// 		Request:     getProductReq,
// 		Limit:       10,
// 		Page:        1,
// 		DisableFaas: true,
// 	})
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, getProductResp.Data.Data.Response, err)
// }

// func TestGetListSlimWithBoolean(t *testing.T) {
// 	getProductReq := sdk.Request{
// 		Data: map[string]interface{}{
// 			"switch_field": false},
// 	}
// 	getProductResp, _, err := UcodeApi.GetListSlim(&sdk.ArgumentWithPegination{
// 		TableSlug:   "product",
// 		Request:     getProductReq,
// 		Limit:       10,
// 		Page:        1,
// 		DisableFaas: true,
// 	})
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, getProductResp.Data.Data.Response, "Switch Does not work")
// }

// func TestGetListObject(t *testing.T) {
// 	getProductReq := sdk.Request{
// 		Data: map[string]interface{}{},
// 	}
// 	getProductResp, _, err := UcodeApi.GetList(&sdk.ArgumentWithPegination{
// 		TableSlug:   "product",
// 		Request:     getProductReq,
// 		Limit:       10,
// 		Page:        1,
// 		DisableFaas: true,
// 	})
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, getProductResp.Data.Data.Response, "Switch Does not work")
// }

// func TestGetListObjectSearch(t *testing.T) {
// 	getProductReq := sdk.Request{
// 		Data: map[string]interface{}{
// 			"search":      "+99894",
// 			"view_fields": []string{"internation_phone_field"},
// 		},
// 	}
// 	getProductResp, _, err := UcodeApi.GetList(&sdk.ArgumentWithPegination{
// 		TableSlug:   "product",
// 		Request:     getProductReq,
// 		Limit:       10,
// 		Page:        1,
// 		DisableFaas: true,
// 	})
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, getProductResp.Data.Data.Response, "Switch Does not work")

// 	getProductReq = sdk.Request{
// 		Data: map[string]interface{}{
// 			"search":      "+4",
// 			"view_fields": []string{"internation_phone_field"},
// 		},
// 	}
// 	getProductResp, _, err = UcodeApi.GetList(&sdk.ArgumentWithPegination{
// 		TableSlug: "product",
// 		Request:   getProductReq,
// 		Limit:     10,
// 		Page:      1,
// 	})
// 	assert.NoError(t, err)
// 	assert.Empty(t, getProductResp.Data.Data.Response, "Response should be empty")
// }

// func TestGetAggregation(t *testing.T) {
// 	getProductReq := sdk.Request{Data: map[string]interface{}{"pipelines": []map[string]interface{}{{"$group": map[string]interface{}{"_id": "$single_line_field"}}}}}
// 	getProductResp, _, err := UcodeApi.GetListAggregation(&sdk.Argument{
// 		TableSlug:   "product",
// 		Request:     getProductReq,
// 		DisableFaas: true,
// 	})

// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, getProductResp.Data.Data.Data, "Switch Does not work")
// }

// func TestGetListAutoFilter(t *testing.T) {
// 	resp, err := Login()
// 	assert.NoError(t, err)

// 	body, err := UcodeApi.DoRequest(BaseUrl+"/v2/object/get-list/product", "POST", map[string]interface{}{
// 		"data": map[string]interface{}{},
// 	}, map[string]string{
// 		"Authorization": "Bearer " + resp,
// 	})
// 	assert.NoError(t, err)
// 	var response GetListApiResponse

// 	err = json.Unmarshal(body, &response)
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, response.Data.Data.Response)
// }

// func TestGetListExcel(t *testing.T) {
// 	body, err := UcodeApi.DoRequest(BaseUrl+"/v1/object/excel/product", "POST", ExcelReq, map[string]string{
// 		"authorization": "API-KEY",
// 		"X-API-KEY":     UcodeApi.Config().AppId,
// 	})
// 	assert.NoError(t, err)
// 	var response ExcelResponse
// 	err = json.Unmarshal(body, &response)
// 	assert.NoError(t, err)

// 	assert.NotEmpty(t, response.Data.Data.Link)
// }

// func TestGetListRBAC(t *testing.T) {
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

// func TestMultipleCRUD(t *testing.T) {
// 	var ids = []string{}
// 	var multipleInsert = []map[string]interface{}{}
// 	var multipleUpdate = []map[string]interface{}{}
// 	UcodeApi.Config().RequestTimeout = time.Second * 30
// 	faker1, err := faker.New("en")
// 	assert.NoError(t, err, "faker error")

// 	for i := 0; i < 10; i++ {
// 		guid := uuid.New().String()
// 		multipleInsert = append(multipleInsert, map[string]interface{}{
// 			"guid":                             guid,
// 			"single_line_field":                faker1.CompanyName(),
// 			"multi_line_field":                 faker1.FirstName(),
// 			"email_field":                      faker1.Email(),
// 			"internation_phone_field":          faker1.PhoneNumber(),
// 			"date_time_field":                  time.Now(),
// 			"date_time_without_timezone_field": time.Now().UTC(),
// 			"date_field":                       time.Now(),
// 			"is_new":                           true,
// 		})

// 		multipleUpdate = append(multipleUpdate, map[string]interface{}{
// 			"guid":                             guid,
// 			"single_line_field":                faker1.CompanyName(),
// 			"multi_line_field":                 faker1.FirstName(),
// 			"email_field":                      faker1.Email(),
// 			"internation_phone_field":          faker1.PhoneNumber(),
// 			"date_time_field":                  time.Now(),
// 			"date_time_without_timezone_field": time.Now().UTC(),
// 			"date_field":                       time.Now(),
// 			"is_new":                           false,
// 		})
// 		ids = append(ids, guid)
// 	}

// 	_, _, err = UcodeApi.MultipleUpdate(&sdk.Argument{
// 		TableSlug: "product",
// 		Request: sdk.Request{
// 			Data: map[string]interface{}{
// 				"objects": multipleInsert,
// 			},
// 		},
// 		DisableFaas: true,
// 	})
// 	assert.NoError(t, err)

// 	_, _, err = UcodeApi.MultipleUpdate(&sdk.Argument{
// 		TableSlug: "product",
// 		Request: sdk.Request{
// 			Data: map[string]interface{}{
// 				"objects": multipleUpdate,
// 			},
// 		},
// 		DisableFaas: true,
// 	})
// 	assert.NoError(t, err)

// 	_, err = UcodeApi.MultipleDelete(&sdk.Argument{
// 		TableSlug: "product",
// 		Request: sdk.Request{
// 			Data: map[string]interface{}{
// 				"ids": ids,
// 			},
// 		},
// 	})
// 	assert.NoError(t, err)
// }
