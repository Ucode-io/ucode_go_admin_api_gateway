package models

type CommonMessage struct {
	Data     map[string]interface{} `json:"data"`
	IsCached bool                   `json:"is_cached"`
}

type Wayll struct {
	OrderID             string `json:"orderId"`
	BindingID           string `json:"bindingId"`
	OrderNumber         string `json:"orderNumber"`
	OperationType       string `json:"operationType"`
	OperationState      string `json:"operationState"`
	OperationId         string `json:"operationId"`
	MerchantOperationId string `json:"merchantOperationId"`
	Rrn                 string `json:"rrn"`
}

type HtmlBody struct {
	Data map[string]interface{} `json:"data"`
	Html string                 `json:"html"`
}

type GetListRequest struct {
	TableSlug string `json:"table_slug"`
	Search    string `json:"search"`
	Limit     int32  `json:"limit"`
	Offset    int32  `json:"offset"`
}

type UpsertCommonMessage struct {
	Data          map[string]interface{} `json:"data"`
	UpdatedFields []string               `json:"updated_fields"`
}

type Ids struct {
	Ids []string `json:"ids"`
}

type MultipleUpdateItems struct {
	Ids  []string               `json:"ids"`
	Data map[string]interface{} `json:"data"`
}

type MultipleInsertItems struct {
	Items []map[string]interface{} `json:"items"`
}

type ObjectsResponse struct {
	Data struct{} `json:"data"`
}
