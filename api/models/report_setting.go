package models

type AppReportSetting struct {
	Id             string           `json:"id"`
	MainTableLabel string           `json:"main_table_label"`
	MainTableSlug  string           `json:"main_table_slug"`
	Rows           []*TableSetting  `json:"rows"`
	Columns        []*TableSetting  `json:"columns"`
	Values         []*ValueSetting  `json:"values"`
	Filters        []*FilterSetting `json:"filters"`
}

type TableSetting struct {
	Id    string `json:"id"`
	Label string `json:"label"`
	Slug  string `json:"slug"`
}

type ValueSetting struct {
	Label  string    `json:"label"`
	Entity []*Entity `json:"entity"`
}

type Entity struct {
	Label     string `json:"label"`
	TableSlug string `json:"table_slug"`
	FieldSlug string `json:"field_slug"`
	FieldType string `json:"field_type"`
}

type FilterSetting struct {
	Label string `json:"label"`
	Slug  string `json:"slug"`
}

type Empty struct{}
