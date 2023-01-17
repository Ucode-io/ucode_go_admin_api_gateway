package handlers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/pkg/util"

	"ucode/ucode_go_api_gateway/api/status_http"
	"github.com/gin-gonic/gin"
)

// CreateDirections godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID create_patient
// @Router /v1/alfalab/directions [POST]
// @Summary Create Directions
// @Description Create Directions
// @Tags AlfaLab
// @Accept json
// @Produce json
// @Param table body models.CreateDirections true "CreatePatientRequestBody"
// @Success 201 {object} status_http.Response{data=models.Field} "Field data"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) CreateDirections(c *gin.Context) {
	var (
		directions  models.CreateDirections
		requestBody models.CreateDirectionsRequestBody
		items       []models.Item
	)

	err := c.ShouldBindJSON(&directions)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	for _, direction := range directions.Assays.Item {
		items = append(items, models.Item{
			Barcode:         direction.Barcode,
			BiomaterialCode: direction.BiomaterialCode,
			Orders: models.Orders{
				Item: models.OrdersItem{
					Code: direction.Orders.ItemCode,
				},
			},
		})
	}

	requestBody = models.CreateDirectionsRequestBody{
		MessageType: "query-create-referral",
		Sender:      "50389",
		Receiver:    "SwissLab",
		Password:    "gU4by567rUtBe457BJd",
		Patient: models.Patient{
			MisId:           directions.Patient.MisId,
			FirstName:       directions.Patient.FirstName,
			MiddleName:      directions.Patient.MiddleName,
			LastName:        directions.Patient.LastName,
			Gender:          directions.Patient.Gender,
			PayCategoryId:   directions.Patient.PayCategoryId,
			BirthDate:       directions.Patient.BirthDate,
			Email:           directions.Patient.Email,
			Phone:           directions.Patient.Phone,
			DocumentNumber:  directions.Patient.DocumentNumber,
			BodyTemperature: directions.Patient.BodyTemperature,
		},
		Referral: models.Referral{
			MisId:            directions.Referral.MisId,
			Nr:               directions.Referral.Nr,
			Date:             directions.Referral.Date,
			SamplingDate:     directions.Referral.SamplingDate,
			DeliveryDate:     directions.Referral.DeliveryDate,
			DepartmentName:   directions.Referral.DepartmentName,
			DepartmentCode:   directions.Referral.DepartmentCode,
			DoctorName:       directions.Referral.DoctorName,
			DoctorCode:       directions.Referral.DoctorCode,
			Cito:             directions.Referral.Cito,
			DiagnosisName:    directions.Referral.DiagnosisName,
			DiagnosisCode:    directions.Referral.DiagnosisCode,
			Comment:          directions.Referral.Comment,
			EComment:         directions.Referral.EComment,
			Comment1:         directions.Referral.Comment1,
			PregnancyWeek:    directions.Referral.PregnancyWeek,
			CyclePeriod:      directions.Referral.CyclePeriod,
			LastMenstruation: directions.Referral.LastMenstruation,
			DiuresisMl:       directions.Referral.DiuresisMl,
			WeightKg:         directions.Referral.WeightKg,
			HeightCm:         directions.Referral.HeightCm,
			PayCategoryId:    directions.Referral.PayCategoryId,
		},
		Assays: models.AssaysXML{
			Items: items,
		},
	}

	resp, err := util.DoXMLRequest("http://95.211.223.217:9901", "POST", requestBody)
	if err != nil {
		h.handleResponse(c, status_http.InvalidArgument, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

// GetReferral godoc
// @Security ApiKeyAuth
// @Param Resource-Id header string true "Resource-Id"
// @ID query_referral_result
// @Router /v1/alfalab/referral [GET]
// @Summary query referral result
// @Description Query Referral Result
// @Tags AlfaLab
// @Accept json
// @Produce json
// @Param nr query string false "nr"
// @Success 200 {object} status_http.Response{data=object_builder_service.BarcodeGenerateRes} "Barcode"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *Handler) GetReferral(c *gin.Context) {
	url := "http://95.211.223.217:9901"
	method := "POST"

	payload := strings.NewReader(`<?xml version="1.0" encoding="Windows-1251"?>
									<Message
									MessageType="query-referral-results"
									Date="24.10.2022 08:22:21"
									Sender="50027"
									Receiver="SwissLab"
									Password="gU4by567rUtBe457BJd">
										<Query
											LisId="14908254"
											Nr="310"
											MisId="14908254"
										/>
									</Message>`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", ": application/x-www-form-urlencoded")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}

// Need to delete
