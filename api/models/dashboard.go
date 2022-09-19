package models

type Dashboard struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type CreateDashboardsRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type CreateDashboardRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
}

type GetAllDashboardsResponse struct {
	Dashboards []Dashboard `json:"dashboards"`
	Count      int32       `json:"count"`
}

type GetAllDashboardsRequest struct {
	Name string `json:"name"`
}


