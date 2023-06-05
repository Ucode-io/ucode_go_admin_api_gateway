package models

type CreateMenuRequest struct {
	Label    string `json:"label"`
	Icon     string `json:"icon"`
	TableId  string `json:"table_id"`
	LayoutId string `json:"layout_id"`
	ParentId string `json:"parent_id"`
	Type     string `json:"type"`
}

type Menu struct {
	Id       string `json:"id"`
	Label    string `json:"label"`
	Icon     string `json:"icon"`
	TableId  string `json:"table_id"`
	LayoutId string `json:"layout_id"`
	ParentId string `json:"parent_id"`
	Type     string `json:"type"`
}
