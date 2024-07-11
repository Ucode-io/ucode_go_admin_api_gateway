package gpt

// import (
// 	"bytes"
// 	"encoding/json"
// 	"fmt"
// 	"io"
// 	"log"
// 	"net/http"
// 	"time"

// 	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
// 	"github.com/google/uuid"
// 	_ "github.com/spf13/cast"
// )

// const (
// 	apiKey = ""

// 	createURL    = "https://api.admin.u-code.io/v1/object/"
// 	createMethod = "POST"

// 	updateURL    = "https://api.admin.u-code.io/v1/object/"
// 	updateMethod = "PUT"

// 	getListURL    = "https://api.admin.u-code.io/v1/object/get-list/"
// 	getListMethod = "POST"

// 	getSingleURL    = "https://api.admin.u-code.io/v1/object/"
// 	getSingleMethod = "GET"

// 	multipleUpdateUrl    = "https://api.admin.u-code.io/v1/object/multiple-update/"
// 	multipleUpdateMethod = "PUT"
// )

// func Handle(req []byte) string {

// 	var reqBody RequestBody

// 	if err := json.Unmarshal(req, &reqBody); err != nil {
// 		return Handler("error", err.Error())
// 	}

// 	tableSlug := "get from prompt"

// 	// Make exemple for create...
// 	var (
// 		createTableNameUrl = createURL + tableSlug
// 		// column name get from prompt
// 		createTableNameReq = Request{
// 			Data: map[string]interface{}{
// 				"guid": uuid.NewString(),
// 				"name": "some name",
// 			},
// 		}
// 		// or you can
// 		// createTableNameReq = Request{
// 		// 	Data: reqBody.Data,
// 		// }
// 	)

// 	_, err := DoRequest(createTableNameUrl, createMethod, createTableNameReq)
// 	if err != nil {
// 		return Handler("error", err.Error())
// 	}

// 	// Make exemple for update...
// 	var (
// 		updateTableNameUrl = updateURL + tableSlug + "/guid from prompt"
// 		// column name get from prompt
// 		updateTableNameReq = Request{
// 			Data: map[string]interface{}{
// 				"guid": "guid from prompt",
// 				"name": "some name",
// 			},
// 		}
// 		// or you can
// 		// createTableNameReq = Request{
// 		// 	Data: reqBody.Data,
// 		// }
// 	)

// 	_, err = DoRequest(updateTableNameUrl, updateMethod, updateTableNameReq)
// 	if err != nil {
// 		return Handler("error", err.Error())
// 	}

// 	// Make exemple for get-list
// 	var (
// 		getListTableNameUrl = getListURL + tableSlug
// 		getListTableNameReq = Request{
// 			Data: map[string]interface{}{}, // default u give like this or u can give limit, offset, or other filters which user gives in pormpt
// 		}

// 		// exemple with filters
// 		// getListTableNameReq = Request{
// 		// 	Data: map[string]interface{}{
// 		// 		"name": "John",
// 		// 		"age": map[string]interface{}{
// 		// 			"$gt": 18,
// 		// 		},
// 		// 	},
// 		// }
// 		getListTableNameResp = GetListClientApiResponse{}
// 	)

// 	body, err := DoRequest(getListTableNameUrl, getListMethod, getListTableNameReq)
// 	if err != nil {
// 		return Handler("error", err.Error())
// 	}
// 	if err := json.Unmarshal(body, &getListTableNameResp); err != nil {
// 		return Handler("error", err.Error())
// 	}

// 	//Make exemple for get single
// 	var (
// 		getSingleTableNameUrl  = getSingleURL + tableSlug
// 		getSingleTableNameResp = ClientApiResponse{}
// 	)

// 	body, err = DoRequest(getSingleTableNameUrl, getSingleMethod, nil)
// 	if err != nil {
// 		return Handler("error", err.Error())
// 	}
// 	if err := json.Unmarshal(body, &getSingleTableNameResp); err != nil {
// 		return Handler("error", err.Error())
// 	}

// 	//Make exemple for multiple update

// 	var (
// 		multipleUpdateTableNameUrl = multipleUpdateUrl + tableSlug
// 		multipleUpdateTableNameReq = MultipleUpdateRequest{
// 			Data: Data{},
// 		}
// 		objects = []map[string]interface{}{}
// 	)

// 	for _, obj := range getListTableNameResp.Data.Data.Response { // or other data for loop
// 		objects = append(objects, obj) // ther some info about multiple update in  map[string]interface{} if you don't give guid system create this data if uyou give then he updates
// 	}

// 	multipleUpdateTableNameReq.Data.Objects = objects

// 	_, err = DoRequest(multipleUpdateTableNameUrl, multipleUpdateMethod, multipleUpdateTableNameReq)
// 	if err != nil {
// 		return Handler("error", err.Error())
// 	}

// 	return Handler("OK", "Successfully updated!")
// }

// func DoRequest(url string, method string, body interface{}) ([]byte, error) {
// 	data, err := json.Marshal(&body)
// 	if err != nil {
// 		return nil, err
// 	}
// 	client := &http.Client{
// 		Timeout: time.Duration(10 * time.Second),
// 	}

// 	request, err := http.NewRequest(method, url, bytes.NewBuffer(data))
// 	if err != nil {
// 		return nil, err
// 	}

// 	request.Header.Add("authorization", "API-KEY")
// 	request.Header.Add("X-API-KEY", apiKey)

// 	resp, err := client.Do(request)
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer resp.Body.Close()

// 	respByte, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return respByte, nil
// }

// func Handler(status, message string) string {
// 	var (
// 		response Response
// 		Message  = make(map[string]interface{})
// 	)

// 	sendMessage("", status, message)
// 	response.Status = status
// 	data := Request{
// 		Data: map[string]interface{}{
// 			"data": message,
// 		},
// 	}
// 	response.Data = data.Data
// 	Message["message"] = message
// 	respByte, _ := json.Marshal(response)
// 	return string(respByte)
// }

// func sendMessage(functionName, errorStatus string, message interface{}) {
// 	bot, err := tgbotapi.NewBotAPI("")
// 	if err != nil {
// 		log.Panic(err)
// 	}

// 	chatID := int64(0)
// 	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("message from %s function: %s\n%s", functionName, errorStatus, message))
// 	_, err = bot.Send(msg)
// 	if err != nil {
// 		log.Panic(err)
// 	}
// }

// type Response struct {
// 	Status string                 `json:"status"`
// 	Data   map[string]interface{} `json:"data"`
// }

// // This struct for request of ofs unmarshal
// type RequestBody struct {
// 	ObjectIDs []string               `json:"object_ids"`
// 	Data      map[string]interface{} `json:"data"`
// }

// // Response of single data get
// type ClientApiResponse struct {
// 	Data ClientApiData `json:"data"`
// }

// type ClientApiData struct {
// 	Data ClientApiResp `json:"data"`
// }

// type ClientApiResp struct {
// 	Response map[string]interface{} `json:"response"`
// }

// // All request struct for get-list, create, update
// type Request struct {
// 	Data map[string]interface{} `json:"data"`
// }

// // This struct for create multiple update data
// type MultipleUpdateRequest struct {
// 	Data Data `json:"data"`
// }

// type Data struct {
// 	Objects []map[string]interface{} `json:"objects"`
// }

// // This for response get-list data
// type GetListClientApiResponse struct {
// 	Data GetListClientApiData `json:"data"`
// }

// type GetListClientApiData struct {
// 	Data GetListClientApiResp `json:"data"`
// }

// type GetListClientApiResp struct {
// 	Response []map[string]interface{} `json:"response"`
// }

// type ResponseModel struct {
// 	Status string                 `json:"status"`
// 	Data   map[string]interface{} `json:"data"`
// }

// type CreateResponseBody struct {
// 	Data CreateResponseModel `json:"data"`
// }

// type CreateResponseModel struct {
// 	Data CreateResponse `json:"data"`
// }

// type CreateResponse struct {
// 	Data map[string]interface{} `json:"data"`
// }
