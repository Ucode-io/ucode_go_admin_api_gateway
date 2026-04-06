package models

type PricingUsage struct {
	Current float64 `json:"current"`
	Limit   float64 `json:"limit"`
	Unit    string  `json:"unit"`
}

type AllPricingUsage struct {
	Functions     PricingUsage `json:"functions"`
	Microfrontend PricingUsage `json:"microfrontend"`
	AssetSize     PricingUsage `json:"asset_size"`
	DatabaseSize  PricingUsage `json:"database_size"`
	Users         PricingUsage `json:"users"`
}
