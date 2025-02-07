package v1

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
	"github.com/spf13/cast"
	"golang.org/x/exp/rand"
)

type CreateProjectCardRequest struct {
	Pan    string `json:"pan"`
	Expire string `json:"expire"`
}

type CreateCardResponse struct {
	Id     int         `json:"id"`
	Result *CardResult `json:"result"`
	Error  *Error      `json:"error"`
}

type CardResult struct {
	Card Card `json:"card"`
}

type Card struct {
	Number    string `json:"number"`
	Expire    string `json:"expire"`
	Token     string `json:"token"`
	Recurrent bool   `json:"recurrent"`
	Verify    bool   `json:"verify"`
	Type      string `json:"type"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

type GetVerifyCodeResponse struct {
	Result *VerifyCodeResult `json:"result"`
	Error  *Error            `json:"error"`
}

type VerifyCodeResult struct {
	Sent  bool   `json:"sent"`
	Phone string `json:"phone"`
	Wait  int    `json:"wait"`
	Token string `json:"token"`
}

type VerifyCardRequest struct {
	Token string `json:"token"`
	Code  string `json:"code"`
}

func (h *HandlerV1) CreateProjectCard(c *gin.Context) {
	var request CreateProjectCardRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	token, id, err := createCard(request.Pan, request.Expire)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	verifyResponse, err := getVerifyCode(token, id)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}
	verifyResponse.Token = token

	h.handleResponse(c, status_http.OK, verifyResponse)
}

func (h *HandlerV1) VerifyCard(c *gin.Context) {
	var request VerifyCardRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	payload := map[string]any{
		"id":     random5DigitNumber(),
		"method": "cards.verify",
		"params": map[string]any{
			"token": request.Token,
			"code":  request.Code,
		},
	}

	response, err := sendRequest[CreateCardResponse](payload)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}
	if response.Error != nil {
		h.handleResponse(c, status_http.InternalServerError, response.Error.Message)
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	resp, err := h.companyServices.Billing().CreateCard(
		c.Request.Context(),
		&company_service.CreateProjectCardRequest{
			Pan:        response.Result.Card.Number,
			Expire:     response.Result.Card.Expire,
			PaymeToken: response.Result.Card.Token,
			ProjectId:  cast.ToString(projectId),
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.InternalServerError, err.Error())
		return
	}

	h.handleResponse(c, status_http.Created, resp)
}

func createCard(pan, expire string) (string, int, error) {
	payload := map[string]any{
		"id":     random5DigitNumber(),
		"method": "cards.create",
		"params": map[string]any{
			"card": map[string]any{
				"number": pan,
				"expire": expire,
			},
			"save": true,
		},
	}

	response, err := sendRequest[CreateCardResponse](payload)
	if err != nil {
		return "", 0, err
	}

	if response.Error != nil {
		return "", 0, errors.New(response.Error.Message)
	}

	if response.Result == nil || response.Result.Card.Token == "" {
		return "", 0, errors.New("missing token in response")
	}

	return response.Result.Card.Token, response.Id, nil
}

func getVerifyCode(token string, id int) (*VerifyCodeResult, error) {
	payload := map[string]any{
		"id":     id,
		"method": "cards.get_verify_code",
		"params": map[string]any{
			"token": token,
		},
	}

	response, err := sendRequest[GetVerifyCodeResponse](payload)
	if err != nil {
		return nil, err
	}

	if response.Error != nil {
		return nil, errors.New(response.Error.Message)
	}

	if response.Result == nil {
		return nil, errors.New("missing response")
	}

	return response.Result, nil
}

func sendRequest[T any](payload map[string]any) (*T, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://checkout.test.paycom.uz/api", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth", "5e730e8e0b852a417aa49ceb")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var responseData T
	if err := json.NewDecoder(resp.Body).Decode(&responseData); err != nil {
		return nil, err
	}

	return &responseData, nil
}

func random5DigitNumber() int {
	rand.Seed(uint64(time.Now().UnixNano()))
	min, max := 10000, 99999
	return rand.Intn(max-min+1) + min
}
