package models

type Panel struct {
	ID            string         `json:"id"`
	Query         string         `json:"query"`
	Coordinates   []int32        `json:"coordinates"`
	Attributes    map[string]any `json:"attributes"`
	Title         string         `json:"title"`
	DashboardID   string         `json:"dashboard_id"`
	HasPagination bool           `json:"has_pagination"`
}

type CreatePanelsRequest struct {
	ID            string         `json:"id"`
	Query         string         `json:"query"`
	Coordinates   []int32        `json:"coordinates"`
	Attributes    map[string]any `json:"attributes"`
	Title         string         `json:"title"`
	DashboardID   string         `json:"dashboard_id"`
	HasPagination bool           `json:"has_pagination"`
}

type CreatePanelRequest struct {
	ID            string         `json:"id"`
	Query         string         `json:"query"`
	Coordinates   []int32        `json:"coordinates"`
	Attributes    map[string]any `json:"attributes"`
	Title         string         `json:"title"`
	DashboardID   string         `json:"dashboard_id"`
	HasPagination bool           `json:"has_pagination"`
}

type GetAllPanelsResponse struct {
	Panels []Panel `json:"panels"`
	Count  int32   `json:"count"`
}

type GetAllPanelsRequest struct {
	Title string `json:"title"`
}
