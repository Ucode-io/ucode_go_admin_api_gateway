package models

import "encoding/xml"

type CreatePatientInfo struct {
	MisId      string `json:"mis_id" xml:"mis_id"`
	Code1      string `json:"code_1" xml:"code_1"`
	Code2      string `json:"code_2" xml:"code_2"`
	LastName   string `json:"last_name" xml:"last_name"`
	FirstName  string `json:"first_name" xml:"first_name"`
	MiddleName string `json:"middle_name" xml:"middle_name"`
	Gender     int    `json:"gender" xml:"gender"`
}

type DirectionsRequest struct {
	Patient  Patient  `json:"patient"`
	Referral Referral `json:"referral"`
}

type CreateDirections struct {
	Patient  Patient  `json:"patient"`
	Referral Referral `json:"referral"`
	Assays   Assays   `json:"assays"`
}

type Patient struct {
	MisId           string `json:"mis_id" xml:"MisId"`
	LastName        string `json:"last_name" xml:"LastName"`
	FirstName       string `json:"first_name" xml:"FirstName"`
	MiddleName      string `json:"middle_name" xml:"MiddleName"`
	Gender          string `json:"gender" xml:"Gender"`
	PayCategoryId   string `json:"pay_category_id" xml:"PayCategoryId"`
	BirthDate       string `json:"birth_date" xml:"BirthDate"`
	Email           string `json:"email" xml:"Email"`
	Phone           string `json:"phone" xml:"Phone"`
	DocumentNumber  string `json:"document_number" xml:"DocumentNumber"`
	BodyTemperature string `json:"body_temperature" xml:"BodyTemperature"`
}

type Referral struct {
	MisId            string `json:"mis_id" xml:"MisId"`
	Nr               string `json:"nr" xml:"Nr"`
	Date             string `json:"date" xml:"Date"`
	SamplingDate     string `json:"sampling_date" xml:"SamplingDate"`
	DeliveryDate     string `json:"delivery_date" xml:"DeliveryDate"`
	DepartmentName   string `json:"department_name" xml:"DepartmentName"`
	DepartmentCode   string `json:"department_code" xml:"DepartmentCode"`
	DoctorName       string `json:"doctor_name" xml:"DoctorName"`
	DoctorCode       string `json:"doctor_code" xml:"DoctorCode"`
	Cito             string `json:"cito" xml:"Cito"`
	DiagnosisName    string `json:"diagnosisName" xml:"DiagnosisName"`
	DiagnosisCode    string `json:"diagnosis_code" xml:"DiagnosisCode"`
	Comment          string `json:"comment" xml:"Comment"`
	EComment         string `json:"ecomment" xml:"eComment"`
	Comment1         string `json:"comment_1" xml:"Comment1"`
	PregnancyWeek    string `json:"pregnancy_week" xml:"PregnancyWeek"`
	CyclePeriod      string `json:"cycle_period" xml:"CyclePeriod"`
	LastMenstruation string `json:"last_menstruation" xml:"LastMenstruation"`
	DiuresisMl       string `json:"diuresis_ml" xml:"DiuresisMl"`
	WeightKg         string `json:"weight_kg" xml:"WeightKg"`
	HeightCm         string `json:"height_cm" xml:"HeightCm"`
	PayCategoryId    string `json:"pay_category_id" xml:"PayCategoryId"`
}

type AssaysOrders struct {
	ItemCode string `json:"item_code"`
}

type AssaysItem struct {
	Barcode         string       `json:"barcode"`
	BiomaterialCode string       `json:"biomaterial_code"`
	Orders          AssaysOrders `json:"orders"`
}

type Assays struct {
	Item []AssaysItem `json:"item"`
}
type PatientInfoXML struct {
	XMLName         xml.Name `xml:"Patient"`
	MisId           string   `json:"mis_id" xml:"MisId"`
	LastName        string   `json:"last_name" xml:"LastName"`
	FirstName       string   `json:"first_name" xml:"FirstName"`
	MiddleName      string   `json:"middle_name" xml:"MiddleName"`
	Gender          string   `json:"gender" xml:"Gender"`
	PayCategoryId   string   `json:"pay_category_id" xml:"PayCategoryId"`
	BirthDate       string   `json:"birth_date" xml:"BirthDate"`
	Email           string   `json:"email" xml:"Email"`
	Phone           string   `json:"phone" xml:"Phone"`
	DocumentNumber  string   `json:"documentNumber" xml:"DocumentNumber"`
	BodyTemperature string   `json:"bodyTemperature" xml:"BodyTemperature"`
}

type ReferralXML struct {
	XMLName          xml.Name `xml:"Referral"`
	MisId            string   `xml:"MisId"`
	Nr               string   `xml:"Nr"`
	Date             string   `xml:"Date"`
	SamplingDate     string   `xml:"SamplingDate"`
	DeliveryDate     string   `xml:"DeliveryDate"`
	DepartmentName   string   `xml:"DepartmentName"`
	DepartmentCode   string   `xml:"DepartmentCode"`
	DoctorName       string   `xml:"DoctorName"`
	DoctorCode       string   `xml:"DoctorCode"`
	Cito             string   `xml:"Cito"`
	DiagnosisName    string   `xml:"DiagnosisName"`
	DiagnosisCode    string   `xml:"DiagnosisCode"`
	Comment          string   `xml:"Comment"`
	EComment         string   `xml:"eComment"`
	Comment1         string   `xml:"Comment1"`
	PregnancyWeek    string   `xml:"PregnancyWeek"`
	CyclePeriod      string   `xml:"CyclePeriod"`
	LastMenstruation string   `xml:"LastMenstruation"`
	DiuresisMl       string   `xml:"DiuresisMl"`
	WeightKg         string   `xml:"WeightKg"`
	HeightCm         string   `xml:"HeightCm"`
	PayCategoryId    string   `xml:"PayCategoryId"`
}

type OrdersItem struct {
	XMLName xml.Name `xml:"Item"`
	Code    string   `xml:"Code,attr"`
}

type Orders struct {
	XMLName xml.Name   `xml:"Orders"`
	Item    OrdersItem `xml:"Item"`
}

type Item struct {
	XMLName         xml.Name `xml:"Item"`
	Barcode         string   `xml:"Barcode,attr"`
	BiomaterialCode string   `xml:"BiomaterialCode,attr"`
	Orders          Orders   `xml:"Orders"`
}

type AssaysXML struct {
	XMLName xml.Name `xml:"Assays"`
	Items   []Item   `xml:"Item"`
}
type CreateDirectionsRequestBody struct {
	XMLName     xml.Name  `xml:"Message"`
	MessageType string    `json:"message_type" xml:"MessageType"`
	Sender      string    `json:"sender" xml:"Sender"`
	Receiver    string    `json:"receiver" xml:"Receiver"`
	Password    string    `json:"password" xml:"Password"`
	Date        string    `json:"date" xml:"Date"`
	Patient     Patient   `json:"patient" xml:"Patient"`
	Referral    Referral  `json:"referral" xml:"Referral"`
	Assays      AssaysXML `json:"assays" xml:"Assays"`
}

type Query struct {
	XMLName xml.Name `xml:"Query"`
	LisId   int      `xml:"LisId"`
	Nr      string   `xml:"Nr"`
	MisId   string   `xml:"MisId"`
}

type QueryReferralResult struct {
	XMLName     xml.Name `xml:"Message"`
	MessageType string   `json:"message_type" xml:"MessageType"`
	Name        string   `json:"name" xml:"Name"`
	Removed     bool     `json:"removed" xml:"Removed"`
	Sender      string   `json:"sender" xml:"Sender"`
	Receiver    string   `json:"receiver" xml:"Receiver"`
	Password    string   `json:"password" xml:"Password"`
	Query       Query    `json:"query" xml:"Query"`
}
