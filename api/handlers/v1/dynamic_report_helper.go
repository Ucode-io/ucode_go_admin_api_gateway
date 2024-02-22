package v1

import (
	"context"
	"encoding/json"
	"strings"
	"time"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/helper"
	"ucode/ucode_go_api_gateway/pkg/util"
	"ucode/ucode_go_api_gateway/services"

	"github.com/spf13/cast"
)

const (
	SUM       = "SUM"
	COUNT     = "COUNT"
	AVERAGE   = "AVERAGE"
	MAX       = "MAX"
	MIN       = "MIN"
	FIRST     = "FIRST"
	LAST      = "LAST"
	END_FIRST = "END_FIRST"
	END_LAST  = "END_LAST"

	BY_DAYS  = "by_days"
	BY_WEEK  = "by_week"
	BY_MONTH = "by_month"
	BY_YEAR  = "by_year"

	LOOKUP   = "LOOKUP"
	DATETIME = "DATE_TIME"
)

type ClientApiResponse struct {
	Data ClientApiData `json:"data,omitempty"`
}

type ClientApiData struct {
	Data ClientApiResp `json:"data,omitempty"`
}

type ClientApiResp struct {
	Response map[string]interface{} `json:"response,omitempty"`
}

type Response struct {
	Status string                 `json:"status,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

type RequestBody struct {
	ObjectIDs []string `json:"object_ids,omitempty"`
}

type Request struct {
	Data map[string]interface{} `json:"data,omitempty"`
}

type GetListClientApiResponse struct {
	Data              GetListClientApiData   `json:"data,omitempty"`
	MatchTables       map[string]interface{} `json:"match_tables,omitempty"`
	TableSlug         string                 `json:"table_slug,omitempty"`
	OrderNumber       int                    `json:"order_number,omitempty"`
	RelationTableSlug string                 `json:"relation_table_slug"`
	ReportSetting     map[string]interface{} `json:"report_setting,omitempty"`
}

type GetListClientApiData struct {
	TableSlug string               `json:"table_slug,omitempty"`
	Data      GetListClientApiResp `json:"data,omitempty"`
}

type GetListClientApiResp struct {
	Count       int                      `json:"count,omitempty"`
	Rows        []map[string]interface{} `json:"rows,omitempty"`
	Columns     []map[string]interface{} `json:"columns,omitempty"`
	Values      []map[string]interface{} `json:"values,omitempty"`
	Value       map[string]interface{}   `json:"value,omitempty"`
	TotalValues []map[string]interface{} `json:"total_values,omitempty"`
}

type CreateClientApiResponse struct {
	Data CreateClientApiData `json:"data,omitempty"`
}

type CreateClientApiData struct {
	Data CreateClientApiResp `json:"data,omitempty"`
}

type CreateClientApiResp struct {
	Data map[string]interface{} `json:"data,omitempty"`
}

type NewRequestBody struct {
	Data map[string]interface{} `json:"data,omitempty"`
}

func (h *HandlerV1) DynamicReportHelper(requestData NewRequestBody, services services.ServiceManagerI, resourceEnvironmentId string, nodeType string) (Response, error) {

	var (
		response Response
		// requestData    NewRequestBody
		request        GetListClientApiResponse
		errorMessage   = make(map[string]interface{})
		successMessage = make(map[string]interface{})
	)

	object_data, err := json.Marshal(requestData.Data)
	if err != nil {
		response.Status = "error"
		errorMessage["message"] = err.Error()
		response.Data = errorMessage
		return response, err
	}

	err = json.Unmarshal(object_data, &request)
	if err != nil {
		response.Status = "error"
		errorMessage["message"] = err.Error()
		response.Data = errorMessage

		return response, err
	}
	if len(cast.ToString(request.ReportSetting["main_table_slug"])) <= 0 {
		successMessage["response"] = GetListClientApiResponse{
			Data: GetListClientApiData{
				TableSlug: request.Data.TableSlug,
				Data: GetListClientApiResp{
					Count: request.Data.Data.Count,
					Rows:  request.Data.Data.Rows,
				},
			},
		}
		response.Status = "done"
		response.Data = successMessage

		return response, err
	}

	var (
		// Mongo query variables
		rowsQuery               = map[string]interface{}{}
		rowsRelationQuery       = map[string]interface{}{}
		rowsInsideRelationQuery = map[string]interface{}{}
		rowsRelationNestedQuery = map[string]interface{}{}
		columnsQuery            = map[string]interface{}{}
		valuesQuery             = map[string]interface{}{}
		totalValuesQuery        = map[string]interface{}{}

		rowTableSlug                 string
		rowTableFields               []interface{}
		rowFieldOrderNumber          int
		rowMatchValues               = make([]map[string]interface{}, 0)
		rowMatchMapInterface         = make(map[string]interface{}, 0)
		rowRelationMatchMapInterface = make(map[string]interface{}, 0)
		rowMatchRecursive            func(map[string]interface{})
		rowExists                    bool

		rowRelationTableSlug    string
		rowRelationTables       []interface{}
		rowRelationTablesExists bool

		rowInsideRelationTableSlug   string
		rowInsideRelationTableFields []interface{}
		rowInsideRelationExists      bool

		rowRelationNestedTableSlug   string
		rowRelationNestedTableFields []interface{}
		rowRelationNestedExists      bool

		rowDateSlugs          []string
		rowLookupMapInterface = []interface{}{}

		columnTableSlug   string
		columnTableFields []interface{}
		columnExists      bool

		// columnDateSlugs          []string
		// columnLookupMapInterface = []interface{}{}

		valueObjects = make(map[string]interface{}, 0)

		defaultTableFields []interface{}

		// Field variable

		rowsFiltersTableFieldsMap               map[string]interface{} = make(map[string]interface{}, 0)
		rowsInsideRelationFiltersTableFieldsMap map[string]interface{} = make(map[string]interface{}, 0)
		columnsFiltersTableFieldsMap            map[string]interface{} = make(map[string]interface{}, 0)

		mainTableSlug = cast.ToString(request.ReportSetting["main_table_slug"])
		fromDate      = cast.ToString(request.ReportSetting["from_date"])
		toDate        = cast.ToString(request.ReportSetting["to_date"])
		rows          = cast.ToSlice(request.ReportSetting["rows"])
		rowsRelation  = cast.ToSlice(request.ReportSetting["rows_relation"])
		columns       = cast.ToSlice(request.ReportSetting["columns"])
		filters       = cast.ToSlice(request.ReportSetting["filters"])
		values        = cast.ToSlice(request.ReportSetting["values"])
		defaults      = cast.ToSlice(request.ReportSetting["defaults"])
	)
	if request.OrderNumber == 0 {
		rowFieldOrderNumber = 1
	} else {
		rowFieldOrderNumber = request.OrderNumber + 1
	}

	// Rows and Rows Relation and Rows Inside Relation and  Rows Relation Nested...
	{
		rowMatchRecursive = func(rowMatchObject map[string]interface{}) {
			if len(rowMatchObject) > 0 {
				childObject := rowMatchObject["child"]
				delete(rowMatchObject, "child")
				rowMatchValues = append(rowMatchValues, cast.ToStringMap(rowMatchObject))
				rowMatchObject = cast.ToStringMap(childObject)
				rowMatchRecursive(rowMatchObject)
			}
		}
		rowMatchRecursive(request.MatchTables)

		if len(rowMatchValues) > 0 {
			rowMatchMapInterface = map[string]interface{}{"$and": rowMatchValues}
		}

		for _, row := range rows {
			var (
				rowMap         = cast.ToStringMap(row)
				rowOrderNumber = cast.ToInt(rowMap["order_number"])
			)

			if rowOrderNumber == rowFieldOrderNumber && len(request.RelationTableSlug) > 0 {
				rowRelationNestedTableSlug = cast.ToString(rowMap["slug"])
				rowRelationNestedTableFields = cast.ToSlice(cast.ToStringMap(row)["table_field_settings"])
				rowRelationNestedExists = true
				break
			} else if rowOrderNumber == rowFieldOrderNumber {
				rowTableSlug = cast.ToString(rowMap["slug"])
				rowTableFields = cast.ToSlice(cast.ToStringMap(row)["table_field_settings"])
				rowExists = true
				break
			}
		}

		if !rowExists {
			for _, rowRelation := range rowsRelation {
				var (
					rowRelationMap         = cast.ToStringMap(rowRelation)
					rowRelationOrderNumber = cast.ToInt(rowRelationMap["order_number"])
				)

				if rowRelationOrderNumber == rowFieldOrderNumber && len(request.RelationTableSlug) > 0 {
					rowRelationTables = cast.ToSlice(rowRelationMap["objects"])
					for _, rowRelationTable := range rowRelationTables {
						var rowRelationTableMap = cast.ToStringMap(rowRelationTable)
						rowRelationTableSlug = cast.ToString(rowRelationTableMap["slug"])

						if rowRelationTableSlug == request.RelationTableSlug {
							rowRelationTableSlug = cast.ToString(rowRelationTableMap["slug"])
							rowInsideRelationTableSlug = cast.ToString(rowRelationTableMap["inside_relation_table_slug"])
							rowInsideRelationTableFields = cast.ToSlice(rowRelationTableMap["table_field_settings"])
							rowInsideRelationExists = true
							break
						}
					}
					break
				} else if rowRelationOrderNumber == rowFieldOrderNumber {
					rowRelationTables = cast.ToSlice(rowRelationMap["objects"])
					rowRelationTablesExists = true
					break
				}
			}
		}

		// Tables settings parse match, project and sort query
		if rowExists {
			rowsQuery = map[string]interface{}{
				"row_table_slug": rowTableSlug,
				"match":          map[string]interface{}{"$match": rowMatchMapInterface},
				"group":          map[string]interface{}{"$group": map[string]interface{}{"_id": "$" + rowTableSlug + "_id"}},
			}

			var (
				rowProjectMapInterface = map[string]interface{}{"_id": 0, "guid": 1, "table_slug": 1}
				rowSortMapInterface    = map[string]interface{}{}
			)

			for ind, val := range rowTableFields {
				var (
					fieldSlug        = cast.ToString(cast.ToStringMap(val)["field_slug"])
					fieldType        = cast.ToString(cast.ToStringMap(val)["field_type"])
					lookupViewFields = cast.ToSlice(cast.ToStringMap(val)["view_fields"])
				)
				rowProjectMapInterface[fieldSlug] = 1

				if ind < 1 {
					rowSortMapInterface[fieldSlug] = 1
				}

				if fieldType == LOOKUP {
					var (
						concatString    []interface{}
						projectLookup   = map[string]interface{}{}
						lookupTableSlug = cast.ToString(cast.ToStringMap(cast.ToStringMap(val)["table_to"])["slug"])
					)

					for index, lookupViewField := range lookupViewFields {
						var lookupViewSlug = cast.ToString(cast.ToStringMap(lookupViewField)["slug"])
						projectLookup[lookupViewSlug] = 1
						concatString = append(concatString, map[string]interface{}{"$arrayElemAt": []interface{}{"$" + lookupTableSlug + "_data." + lookupViewSlug, 0}})
						if len(lookupViewFields)-1 > index {
							concatString = append(concatString, " ")
						}
					}
					joinTable := util.PluralizeWord(lookupTableSlug)
					rowLookupMapInterface = append(rowLookupMapInterface, map[string]interface{}{
						"$lookup": map[string]interface{}{
							"from": joinTable,
							"let":  map[string]interface{}{fieldSlug: "$" + fieldSlug},
							"pipeline": []interface{}{
								map[string]interface{}{"$match": map[string]interface{}{"$expr": map[string]interface{}{"$eq": []string{"$guid", "$$" + fieldSlug}}}},
								map[string]interface{}{"$project": projectLookup},
							},
							"as": lookupTableSlug + "_data",
						},
					})

					rowLookupMapInterface = append(rowLookupMapInterface, map[string]interface{}{"$set": map[string]interface{}{fieldSlug: map[string]interface{}{"$concat": concatString}}})
				} else if fieldType == DATETIME {
					rowDateSlugs = append(rowDateSlugs, fieldSlug)
				}
			}

			if rowProjectMapInterface != nil && rowSortMapInterface != nil {
				rowsQuery["project"] = rowProjectMapInterface
				rowsQuery["sort"] = rowSortMapInterface
			}
			rowsQuery["lookups"] = rowLookupMapInterface
		} else if rowRelationTablesExists {
			if len(rowMatchValues) > 0 {
				var rowRelationTablesMatchValues = []map[string]interface{}{}
				for _, rowMatchValue := range rowMatchValues {
					for rowMatchKey, rowMatchVal := range rowMatchValue {
						rowRelationTablesMatchValues = append(rowRelationTablesMatchValues, map[string]interface{}{rowMatchKey: map[string]interface{}{"$in": []interface{}{nil, rowMatchVal}}})
					}
				}
				rowMatchMapInterface = map[string]interface{}{"$and": rowRelationTablesMatchValues}
			}

			var rowInsideRelationTableSlugs []string
			for _, rowRelationTable := range rowRelationTables {
				var rowRelationTableMap = cast.ToStringMap(rowRelationTable)
				rowInsideRelationTableSlugs = append(rowInsideRelationTableSlugs, cast.ToString(rowRelationTableMap["inside_relation_table_slug"]))
			}

			rowsRelationQuery = map[string]interface{}{"match": map[string]interface{}{"$match": rowMatchMapInterface}, "inside_relation_tables": rowInsideRelationTableSlugs}
		} else if rowInsideRelationExists {
			if len(rowMatchValues) > 0 {
				var rowInsideRelationMatchValues = []map[string]interface{}{}
				for _, rowMatchValue := range rowMatchValues {
					for rowMatchKey, rowMatchVal := range rowMatchValue {
						rowInsideRelationMatchValues = append(rowInsideRelationMatchValues, map[string]interface{}{rowMatchKey: map[string]interface{}{"$in": []interface{}{nil, rowMatchVal}}})
					}
				}
				rowMatchMapInterface = map[string]interface{}{"$and": rowInsideRelationMatchValues}
			}

			rowsInsideRelationQuery = map[string]interface{}{
				"row_relation_table_slug":        rowRelationTableSlug,
				"row_inside_relation_table_slug": rowInsideRelationTableSlug,
				"match":                          map[string]interface{}{"$match": rowMatchMapInterface},
			}

			var (
				rowRelationProjectMapInterface = map[string]interface{}{"_id": 0, "guid": 1, "table_slug": 1}
				rowRelationSortMapInterface    = map[string]interface{}{}
			)

			for ind, val := range rowInsideRelationTableFields {
				var (
					fieldSlug        = cast.ToString(cast.ToStringMap(val)["field_slug"])
					fieldType        = cast.ToString(cast.ToStringMap(val)["field_type"])
					lookupViewFields = cast.ToSlice(cast.ToStringMap(val)["view_fields"])
				)

				rowRelationProjectMapInterface[fieldSlug] = 1
				if ind < 1 {
					rowRelationSortMapInterface[fieldSlug] = 1
				}

				if fieldType == LOOKUP {
					var (
						concatString    []interface{}
						projectLookup   = map[string]interface{}{}
						lookupTableSlug = cast.ToString(cast.ToStringMap(cast.ToStringMap(val)["table_to"])["slug"])
					)

					for index, lookupViewField := range lookupViewFields {
						var lookupViewSlug = cast.ToString(cast.ToStringMap(lookupViewField)["slug"])
						projectLookup[lookupViewSlug] = 1
						concatString = append(concatString, map[string]interface{}{"$arrayElemAt": []interface{}{"$" + lookupTableSlug + "_data." + lookupViewSlug, 0}})
						if len(lookupViewFields)-1 > index {
							concatString = append(concatString, " ")
						}
					}
					joinTable := util.PluralizeWord(lookupTableSlug)

					rowLookupMapInterface = append(rowLookupMapInterface, map[string]interface{}{
						"$lookup": map[string]interface{}{
							"from": joinTable,
							"let":  map[string]interface{}{fieldSlug: "$" + fieldSlug},
							"pipeline": []interface{}{
								map[string]interface{}{"$match": map[string]interface{}{"$expr": map[string]interface{}{"$eq": []string{"$guid", "$$" + fieldSlug}}}},
								map[string]interface{}{"$project": projectLookup},
							},
							"as": lookupTableSlug + "_data",
						},
					})

					rowLookupMapInterface = append(rowLookupMapInterface, map[string]interface{}{"$set": map[string]interface{}{fieldSlug: map[string]interface{}{"$concat": concatString}}})
				} else if fieldType == DATETIME {
					rowDateSlugs = append(rowDateSlugs, fieldSlug)
				}
			}

			if rowRelationProjectMapInterface != nil && rowRelationSortMapInterface != nil {
				rowsInsideRelationQuery["project"] = rowRelationProjectMapInterface
				rowsInsideRelationQuery["sort"] = rowRelationSortMapInterface
			}
			rowsInsideRelationQuery["lookups"] = rowLookupMapInterface
		} else if rowRelationNestedExists {
			for _, rowRelation := range rowsRelation {
				var rowRelationMap = cast.ToStringMap(rowRelation)
				rowRelationTables = cast.ToSlice(rowRelationMap["objects"])
				for _, rowRelationTable := range rowRelationTables {
					var (
						rowRelationTableMap     = cast.ToStringMap(rowRelationTable)
						rowRelationOneTableSlug = cast.ToString(rowRelationTableMap["slug"])
					)

					if rowRelationOneTableSlug == request.RelationTableSlug {
						rowRelationTableSlug = cast.ToString(rowRelationTableMap["slug"])
						rowInsideRelationTableSlug = cast.ToString(rowRelationTableMap["inside_relation_table_slug"])
						break
					}
				}
			}

			if len(rowMatchValues) > 0 {
				var (
					findRowRelation              bool
					rowRelationMatchValues       = []map[string]interface{}{}
					rowInsideRelationMatchValues = []map[string]interface{}{}
				)

				for _, rowMatchValue := range rowMatchValues {
					for rowMatchKey, rowMatchVal := range rowMatchValue {
						if rowMatchKey == rowRelationTableSlug+"_id" {
							rowRelationMatchValues = append(rowRelationMatchValues, map[string]interface{}{"guid": map[string]interface{}{"$in": []interface{}{nil, rowMatchVal}}})
							rowInsideRelationMatchValues = append(rowInsideRelationMatchValues, map[string]interface{}{rowMatchKey: map[string]interface{}{"$in": []interface{}{nil, rowMatchVal}}})
							findRowRelation = true
						} else if findRowRelation {
							rowRelationMatchValues = append(rowRelationMatchValues, map[string]interface{}{rowMatchKey: map[string]interface{}{"$in": []interface{}{nil, rowMatchVal}}})
							rowInsideRelationMatchValues = append(rowInsideRelationMatchValues, map[string]interface{}{rowMatchKey: map[string]interface{}{"$in": []interface{}{nil, rowMatchVal}}})
						}
					}
				}

				rowRelationMatchValues = append(rowRelationMatchValues, map[string]interface{}{rowRelationNestedTableSlug + "_id": map[string]interface{}{"$exists": true}})
				rowInsideRelationMatchValues = append(rowInsideRelationMatchValues, map[string]interface{}{rowRelationNestedTableSlug + "_id": map[string]interface{}{"$exists": true}})

				rowRelationMatchMapInterface = map[string]interface{}{"$and": rowRelationMatchValues}
				rowMatchMapInterface = map[string]interface{}{"$and": rowInsideRelationMatchValues}
			}

			rowsRelationNestedQuery = map[string]interface{}{
				"row_relation_table_slug":        rowRelationTableSlug,
				"row_inside_relation_table_slug": rowInsideRelationTableSlug,
				"row_relation_nested_table_slug": rowRelationNestedTableSlug,
				"match_relation_table":           map[string]interface{}{"$match": rowRelationMatchMapInterface},
				"match_inside_relation_table":    map[string]interface{}{"$match": rowMatchMapInterface},
			}

			var (
				rowRelationNestedProjectMapInterface = map[string]interface{}{"_id": 0, "guid": 1, "table_slug": 1}
				rowRelationNestedSortMapInterface    = map[string]interface{}{}
			)

			for ind, val := range rowRelationNestedTableFields {
				var fieldSlug = cast.ToString(cast.ToStringMap(val)["field_slug"])
				rowRelationNestedProjectMapInterface[fieldSlug] = 1
				if ind < 1 {
					rowRelationNestedSortMapInterface[fieldSlug] = 1
				}
			}

			if rowRelationNestedProjectMapInterface != nil && rowRelationNestedSortMapInterface != nil {
				rowsRelationNestedQuery["project"] = rowRelationNestedProjectMapInterface
				rowsRelationNestedQuery["sort"] = rowRelationNestedSortMapInterface
			}
		}
	}

	// Columns...
	{
		for _, column := range columns {
			var columnMap = cast.ToStringMap(column)
			columnTableSlug = cast.ToString(columnMap["slug"])
			columnTableFields = cast.ToSlice(cast.ToStringMap(column)["table_field_settings"])
			columnExists = true
			break
		}

		if len(columnTableFields) > 0 {
			columnsQuery = map[string]interface{}{
				"column_table_slug": columnTableSlug,
				"match":             map[string]interface{}{"$match": map[string]interface{}{}},
				"group":             map[string]interface{}{"$group": map[string]interface{}{"_id": "$" + columnTableSlug + "_id"}},
			}

			var (
				columnProjectMapInterface = map[string]interface{}{"_id": 0, "guid": 1}
				columnSortMapInterface    = map[string]interface{}{}
			)

			for ind, val := range columnTableFields {
				var fieldSlug = cast.ToString(cast.ToStringMap(val)["field_slug"])
				columnProjectMapInterface[fieldSlug] = 1

				if ind < 1 {
					columnSortMapInterface[fieldSlug] = 1
				}
			}

			if columnProjectMapInterface != nil && columnSortMapInterface != nil {
				columnsQuery["project"] = columnProjectMapInterface
				columnsQuery["sort"] = columnSortMapInterface
			}
		}
	}

	// Filters...
	{
		for _, filter := range filters {
			var (
				filterTableSlug = cast.ToString(cast.ToStringMap(filter)["slug"]) + "_id"
				tableGuids      = cast.ToSlice(cast.ToStringMap(filter)["table_guids"])
			)

			if len(tableGuids) > 0 {
				tableGuids = append(tableGuids, nil)
				rowsFiltersTableFieldsMap[filterTableSlug] = map[string]interface{}{"$in": tableGuids}
				rowsInsideRelationFiltersTableFieldsMap[filterTableSlug] = map[string]interface{}{"$in": tableGuids}
				columnsFiltersTableFieldsMap[filterTableSlug] = map[string]interface{}{"$in": tableGuids}
			}
		}

		if rowExists {
			var matchRowAnd = cast.ToStringMap(cast.ToStringMap(rowsQuery["match"])["$match"])
			if _, ok := matchRowAnd["$and"]; ok {
				rowsFiltersTableFieldsMap["$and"] = matchRowAnd["$and"]
			}
			rowsQuery["match"] = map[string]interface{}{"$match": rowsFiltersTableFieldsMap}
		}

		if rowInsideRelationExists {
			var matchRowInsideAnd = cast.ToStringMap(cast.ToStringMap(rowsInsideRelationQuery["match"])["$match"])
			if _, ok := matchRowInsideAnd["$and"]; ok {
				rowsInsideRelationFiltersTableFieldsMap["$and"] = matchRowInsideAnd["$and"]
			}
			rowsInsideRelationQuery["inside_relation_match"] = map[string]interface{}{"$match": rowsInsideRelationFiltersTableFieldsMap}
		}

		if columnExists {
			columnsQuery["match"] = map[string]interface{}{"$match": columnsFiltersTableFieldsMap}
		}
	}

	// Values...
	{
		var (
			slug               string
			dateFieldSlug      string
			objects            []interface{}
			tableFieldSettings []interface{}
		)

		for _, value := range values {
			objects = cast.ToSlice(cast.ToStringMap(value)["objects"])

			for _, object := range objects {
				slug = cast.ToString(cast.ToStringMap(object)["slug"])
				dateFieldSlug = cast.ToString(cast.ToStringMap(object)["date_field_slug"])
				tableFieldSettings = cast.ToSlice(cast.ToStringMap(object)["table_field_settings"])

				var (
					fieldSlugsGroupMapInterface = cast.ToStringMap(cast.ToStringMap(cast.ToStringMap(valuesQuery[slug])["group"])["$group"])
					valueAggregationQuery       = make(map[string]interface{}, 0)
					fieldGroup                  = make(map[string]string, 0)
				)

				if rowExists {
					fieldSlugsGroupMapInterface["row_id"] = map[string]interface{}{"$first": "$" + rowTableSlug + "_id"}
					valueAggregationQuery["match_row_guid"] = rowTableSlug + "_id"
					fieldGroup[rowTableSlug+"_id"] = "$" + rowTableSlug + "_id"
				}

				if rowInsideRelationExists {
					fieldSlugsGroupMapInterface["row_relatoin_id"] = map[string]interface{}{"$first": "$" + rowRelationTableSlug + "_id"}
					valueAggregationQuery["match_row_relation_guid"] = rowRelationTableSlug + "_id"
					fieldGroup[rowRelationTableSlug+"_id"] = "$" + rowRelationTableSlug + "_id"
				}

				if columnExists {
					fieldSlugsGroupMapInterface["column_id"] = map[string]interface{}{"$first": "$" + columnTableSlug + "_id"}
					valueAggregationQuery["match_column_guid"] = columnTableSlug + "_id"
					fieldGroup[columnTableSlug+"_id"] = "$" + columnTableSlug + "_id"
				}

				if len(fromDate) > 0 && len(toDate) > 0 && len(dateFieldSlug) > 0 {
					formatFromDate, _ := time.Parse("2006-01-02", fromDate)
					formatToDate, _ := time.Parse("2006-01-02", toDate)
					valueAggregationQuery["match_date_field"] = dateFieldSlug
					valueAggregationQuery["match_from_date"] = formatFromDate.Format("2006-01-02")
					valueAggregationQuery["match_to_date"] = formatToDate.AddDate(0, 0, 1).Format("2006-01-02")
				}

				fieldSlugsGroupMapInterface["_id"] = fieldGroup
				for _, tableFieldSetting := range tableFieldSettings {
					fieldSlug := cast.ToString(cast.ToStringMap(tableFieldSetting)["field_slug"])
					aggregateFormula := cast.ToString(cast.ToStringMap(tableFieldSetting)["aggregate_formula"])
					if aggregateFormula != "" {
						fieldSlugsGroupMapInterface[fieldSlug] = map[string]interface{}{MongoAggregation(DetermineFormula(len(rows), rowFieldOrderNumber, aggregateFormula)): "$" + fieldSlug}
					} else {
						fieldSlugsGroupMapInterface[fieldSlug] = map[string]interface{}{MongoAggregation(SUM): "$" + fieldSlug}
					}
				}

				valueAggregationQuery["group"] = map[string]interface{}{"$group": fieldSlugsGroupMapInterface}
				valuesQuery[slug] = valueAggregationQuery
			}
		}
	}

	// Total Values...
	{
		var (
			slug               string
			dateFieldSlug      string
			objects            []interface{}
			tableFieldSettings []interface{}
		)

		for _, value := range values {
			objects = cast.ToSlice(cast.ToStringMap(value)["objects"])

			for _, object := range objects {
				slug = cast.ToString(cast.ToStringMap(object)["slug"])
				dateFieldSlug = cast.ToString(cast.ToStringMap(object)["date_field_slug"])
				tableFieldSettings = cast.ToSlice(cast.ToStringMap(object)["table_field_settings"])

				var (
					fieldSlugsGroupMapInterface = cast.ToStringMap(cast.ToStringMap(cast.ToStringMap(totalValuesQuery[slug])["group"])["$group"])
					totalValueAggregationQuery  = make(map[string]interface{}, 0)
					fieldGroup                  = make(map[string]string, 0)
				)

				if rowExists {
					fieldSlugsGroupMapInterface["row_id"] = map[string]interface{}{"$first": "$" + rowTableSlug + "_id"}
					totalValueAggregationQuery["match_row_guid"] = rowTableSlug + "_id"
					fieldGroup[rowTableSlug+"_id"] = "$" + rowTableSlug + "_id"
				}

				if rowInsideRelationExists {
					fieldSlugsGroupMapInterface["row_relatoin_id"] = map[string]interface{}{"$first": "$" + rowRelationTableSlug + "_id"}
					totalValueAggregationQuery["match_row_relation_guid"] = rowRelationTableSlug + "_id"
					fieldGroup[rowRelationTableSlug+"_id"] = "$" + rowRelationTableSlug + "_id"
				}

				if len(fromDate) > 0 && len(toDate) > 0 && len(dateFieldSlug) > 0 {
					formatFromDate, _ := time.Parse("2006-01-02", fromDate)
					formatToDate, _ := time.Parse("2006-01-02", toDate)
					totalValueAggregationQuery["match_date_field"] = dateFieldSlug
					totalValueAggregationQuery["match_from_date"] = formatFromDate.Format("2006-01-02")
					totalValueAggregationQuery["match_to_date"] = formatToDate.AddDate(0, 0, 1).Format("2006-01-02")
				}

				fieldSlugsGroupMapInterface["_id"] = fieldGroup
				for _, tableFieldSetting := range tableFieldSettings {
					fieldSlug := cast.ToString(cast.ToStringMap(tableFieldSetting)["field_slug"])
					aggregateFormula := cast.ToString(cast.ToStringMap(tableFieldSetting)["aggregate_formula"])
					if aggregateFormula != "" {
						fieldSlugsGroupMapInterface[fieldSlug] = map[string]interface{}{MongoAggregation(DetermineFormula(len(rows), rowFieldOrderNumber, aggregateFormula)): "$" + fieldSlug}
					} else {
						fieldSlugsGroupMapInterface[fieldSlug] = map[string]interface{}{MongoAggregation(SUM): "$" + fieldSlug}
					}
				}

				totalValueAggregationQuery["group"] = map[string]interface{}{"$group": fieldSlugsGroupMapInterface}
				totalValuesQuery[slug] = totalValueAggregationQuery
			}
		}
	}

	// Default...
	{
		for _, defaultValue := range defaults {
			var defaultTableSlug = cast.ToString(cast.ToStringMap(defaultValue)["slug"])
			if rowTableSlug == defaultTableSlug || rowInsideRelationTableSlug == defaultTableSlug || rowRelationNestedTableSlug == defaultTableSlug {
				defaultTableFields = cast.ToSlice(cast.ToStringMap(defaultValue)["table_field_settings"])
				break
			}
		}

		var defualtSlugs []string
		for _, value := range defaultTableFields {
			var defaultFieldSlug = cast.ToString(cast.ToStringMap(value)["field_slug"])
			defualtSlugs = append(defualtSlugs, defaultFieldSlug)
		}

		if rowExists {
			var rowProject = cast.ToStringMap(rowsQuery["project"])
			for _, defaultSlug := range defualtSlugs {
				rowProject[defaultSlug] = 1
			}
			rowsQuery["project"] = rowProject
		} else if rowInsideRelationExists {
			var rowRelatoinProject = cast.ToStringMap(rowsInsideRelationQuery["project"])
			for _, defaultSlug := range defualtSlugs {
				rowRelatoinProject[defaultSlug] = 1
			}
			rowsInsideRelationQuery["project"] = rowRelatoinProject
		} else if rowRelationNestedExists {
			var rowRelationNestedProject = cast.ToStringMap(rowsRelationNestedQuery["project"])
			for _, defaultSlug := range defualtSlugs {
				rowRelationNestedProject[defaultSlug] = 1
			}
			rowsRelationNestedQuery["project"] = rowRelationNestedProject
		}
	}
	var (
		tableGetFilterResp GetListClientApiResponse
	)

	structData, err := helper.ConvertMapToStruct(map[string]interface{}{
		"rows":                 rowsQuery,
		"rows_relation":        rowsRelationQuery,
		"rows_inside_relation": rowsInsideRelationQuery,
		"rows_relation_nested": rowsRelationNestedQuery,
		"columns":              columnsQuery,
		"values":               valuesQuery,
		"total_values":         totalValuesQuery,
	})
	if err != nil {
		response.Status = "error"
		errorMessage["message"] = "error convert map to struct [tableGetGroupByFieldURL]: "
		response.Data = errorMessage

		return response, err
	}

	responseBuilderReport, err := services.BuilderService().ObjectBuilder().GetGroupReportTables(context.Background(), &obs.CommonMessage{
		TableSlug: mainTableSlug,
		Data:      structData,
		ProjectId: resourceEnvironmentId,
	})
	if err != nil {
		response.Status = "error"
		errorMessage["message"] = "error get group report tables [tableGetGroupByFieldURL]: "
		response.Data = errorMessage
		return response, err
	}

	var (
		rowMatchParentValue string
		rowMatchParentIds   = []string{}
	)
	tableGetFilterResp.Data = GetListClientApiData{
		TableSlug: responseBuilderReport.TableSlug,
		Data: GetListClientApiResp{
			Count:       0,
			Rows:        []map[string]interface{}{},
			Columns:     []map[string]interface{}{},
			Values:      []map[string]interface{}{},
			Value:       map[string]interface{}{},
			TotalValues: []map[string]interface{}{},
		},
	}
	body, _ := json.Marshal(responseBuilderReport.Data.AsMap())
	filteredData := GetListClientApiResp{}
	err = json.Unmarshal(body, &filteredData)
	if err != nil {
		response.Status = "error"
		errorMessage["message"] = "error unmarshalling report data"
		response.Data = errorMessage
		return response, err
	}
	tableGetFilterResp.Data.Data = filteredData

	if len(rowMatchValues) > 0 {
		for index, val := range rowMatchValues {
			for _, insideVal := range cast.ToStringMap(val) {
				rowMatchParentIds = append(rowMatchParentIds, cast.ToString(insideVal))
				if len(rowMatchValues)-1 == index {
					rowMatchParentValue = cast.ToString(insideVal)
				}
			}
		}
	}

	// Dynamic report response processing...
	if rowExists || rowInsideRelationExists || rowRelationNestedExists {
		for index := range tableGetFilterResp.Data.Data.Rows {
			tableGetFilterResp.Data.Data.Rows[index]["is_tree"] = true
			tableGetFilterResp.Data.Data.Rows[index]["parent_ids"] = rowMatchParentIds
			tableGetFilterResp.Data.Data.Rows[index]["parent_value"] = rowMatchParentValue

			if len(rows)+len(rowsRelation) == rowFieldOrderNumber {
				tableGetFilterResp.Data.Data.Rows[index]["is_tree"] = false
			}

			for _, dateSlug := range rowDateSlugs {
				if _, ok := tableGetFilterResp.Data.Data.Rows[index][dateSlug]; ok && cast.ToString(tableGetFilterResp.Data.Data.Rows[index][dateSlug]) != "" {
					if !strings.Contains(cast.ToString(tableGetFilterResp.Data.Data.Rows[index][dateSlug]), "Z") && !strings.Contains(cast.ToString(tableGetFilterResp.Data.Data.Rows[index][dateSlug]), "+") {
						tableGetFilterResp.Data.Data.Rows[index][dateSlug] = cast.ToString(tableGetFilterResp.Data.Data.Rows[index][dateSlug]) + "Z"
					}

					dateTime, _ := time.Parse(time.RFC3339, cast.ToString(tableGetFilterResp.Data.Data.Rows[index][dateSlug]))
					_, offset := dateTime.Zone()
					if offset > 0 {
						tableGetFilterResp.Data.Data.Rows[index][dateSlug] = dateTime.Format("2006-01-02 15:04:05")
					} else {
						tableGetFilterResp.Data.Data.Rows[index][dateSlug] = dateTime.Add(time.Hour * 5).Format("2006-01-02 15:04:05")
					}
				}
			}

			if rowExists {
				for _, values := range tableGetFilterResp.Data.Data.Values {
					for table_slug, row_values := range cast.ToStringMap(values) {
						var row_value, row_id = cast.ToStringMap(row_values), cast.ToString(tableGetFilterResp.Data.Data.Rows[index]["guid"])
						if _, ok := row_value[row_id]; ok {
							tableGetFilterResp.Data.Data.Rows[index][table_slug] = row_value[row_id]
						}
					}
				}
			} else if rowInsideRelationExists {
				for _, values := range tableGetFilterResp.Data.Data.Values {
					for table_slug, row_values := range cast.ToStringMap(values) {
						var row_relation_value, row_relation_id = cast.ToStringMap(row_values), cast.ToString(tableGetFilterResp.Data.Data.Rows[index]["guid"])
						if _, ok := row_relation_value[row_relation_id]; ok {
							tableGetFilterResp.Data.Data.Rows[index][table_slug] = row_relation_value[row_relation_id]
						}
					}
				}
			}
		}
	} else if rowRelationTablesExists {
		var rowRelationTableExists = map[string]interface{}{}
		if len(cast.ToSlice(tableGetFilterResp.Data.Data.Rows)) > 0 {
			rowRelationTableExists = cast.ToStringMap(cast.ToSlice(tableGetFilterResp.Data.Data.Rows)[0])
		}
		tableGetFilterResp.Data.Data.Rows = []map[string]interface{}{}

		for _, rowRelationTable := range rowRelationTables {
			var (
				rowRelationTableMap           = cast.ToStringMap(rowRelationTable)
				rowRelationTableSettings      = cast.ToSlice(rowRelationTableMap["table_field_settings"])
				rowRelationTableSettingExists = false
				rowRelationTableResp          = map[string]interface{}{
					"guid":         rowRelationTableMap["slug"],
					"title":        rowRelationTableMap["label"],
					"table_slug":   rowRelationTableMap["slug"],
					"slug_type":    "RELATION",
					"is_tree":      true,
					"parent_ids":   rowMatchParentIds,
					"parent_value": rowMatchParentValue,
				}
			)

			for _, rowRelationTableSetting := range rowRelationTableSettings {
				var (
					rowRelationTableSettingMap     = cast.ToStringMap(rowRelationTableSetting)
					rowRelationTableSettingChecked = cast.ToBool(rowRelationTableSettingMap["checked"])
				)

				if rowRelationTableSettingChecked {
					rowRelationTableSettingExists = true
					break
				}
			}

			if !rowRelationTableSettingExists {
				continue
			}

			for _, values := range tableGetFilterResp.Data.Data.Values {
				var valueSlug = cast.ToString(rowRelationTableMap["inside_relation_table_slug"])
				if _, ok := cast.ToStringMap(values)[valueSlug]; ok {
					rowRelationTableResp[valueSlug] = cast.ToStringMap(values)[valueSlug]
				}
			}

			if len(rowRelationTableExists) > 0 {
				var insideRelationTableSlug = cast.ToString(rowRelationTableMap["inside_relation_table_slug"])
				if _, ok := rowRelationTableExists[insideRelationTableSlug]; !ok {
					continue
				}
			}

			tableGetFilterResp.Data.Data.Rows = append(tableGetFilterResp.Data.Data.Rows, rowRelationTableResp)
		}
	} else if columnExists {
		for index := range tableGetFilterResp.Data.Data.Columns {
			for _, values := range tableGetFilterResp.Data.Data.Values {
				for table_slug, column_values := range cast.ToStringMap(values) {
					var column_value, column_id = cast.ToStringMap(column_values), cast.ToString(tableGetFilterResp.Data.Data.Columns[index]["guid"])
					if _, ok := column_value[column_id]; ok {
						tableGetFilterResp.Data.Data.Columns[index][table_slug] = column_value[column_id]
					}
				}
			}
		}
	} else {
		for _, valueObjs := range tableGetFilterResp.Data.Data.Values {
			for table_slug, value_data := range cast.ToStringMap(valueObjs) {
				valueObjects[table_slug] = value_data
			}
		}
	}

	if response.Status == "" {
		successMessage["response"] = GetListClientApiResponse{
			Data: GetListClientApiData{
				TableSlug: rowTableSlug,
				Data: GetListClientApiResp{
					Count:       tableGetFilterResp.Data.Data.Count,
					Rows:        tableGetFilterResp.Data.Data.Rows,
					Columns:     tableGetFilterResp.Data.Data.Columns,
					Value:       valueObjects,
					TotalValues: tableGetFilterResp.Data.Data.TotalValues,
				},
			},
		}
		response.Status = "done"
		response.Data = successMessage
	}

	return response, err
}

func DetermineFormula(rowsLength, orderNumber int, aggregateFormula string) string {
	if rowsLength == orderNumber {
		switch aggregateFormula {
		case END_FIRST:
			return FIRST
		case END_LAST:
			return LAST
		}
	} else {
		switch aggregateFormula {
		case END_FIRST:
			return SUM
		case END_LAST:
			return SUM
		}
	}
	return aggregateFormula
}

func MongoAggregation(valueAs string) string {

	switch valueAs {
	case SUM:
		return "$sum"
	case AVERAGE:
		return "$avg"
	case MIN:
		return "$min"
	case MAX:
		return "$max"
	case FIRST:
		return "$first"
	case LAST:
		return "$last"
	}

	return "$sum"
}
