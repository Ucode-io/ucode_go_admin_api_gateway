package tests

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/manveru/faker"
	"github.com/spf13/cast"
	sdk "github.com/ucode-io/ucode_sdk"
)

const (
	// Base Creds
	BaseUrlAuthStaging = "https://auth-api.ucode.run"
	BaseUrlStaging     = "https://admin-api.ucode.run"
	requestTimeout     = time.Second * 5
	Email              = "abdulbositkabilov@gmail.com"

	// Mongo Creds Prod
	appId       = "P-bgh4cmZxaWTXWscpH6sUa9gGlsuvKyZO"
	BaseUrl     = "https://api.admin.u-code.io"
	BaseUrlAuth = "https://api.auth.u-code.io"

	ProjectId            = "f05fdd8d-f949-4999-9593-5686ac272993"
	ResourceId           = "b74a3b18-6531-45fc-8e05-0b9709af8faa"
	EnvironmentId        = "e8b82a93-b87f-4103-abc4-b5a017f540a4"
	ClientTypeId         = "ce3e630f-5399-4557-94f5-f142b411ed6b"
	RoleId               = "d1523cf2-c684-4c14-b021-413011ffb375"
	EmployeeClientTypeId = "8a69fc80-6316-4f84-8914-3e7ebae03dc7"
	EmployeeRoleId       = "b64ac7b7-9ec9-42e0-b720-1267ca1e42f7"

	// Mongo Staging Creds
	appIdStaging              = "P-5LYaWn6pNhTUHfD9fEgXFgonn3sDBaLz"
	ProjectIdMongo            = "462baeca-37b0-4355-addc-b8ae5d26995d"
	ResourceIdMongo           = "05df5e41-1066-474e-8435-3781e0841603"
	EnvironmentIdMongo        = "ad41c493-8697-4f23-979a-341722465748"
	ClientTypeIdMongo         = "038569f5-4dfc-44e6-bee1-6f390aa8195d"
	RoleIdMongo               = "75228002-988e-4418-b10e-656c01c4cc68"
	EmployeeClientTypeIdMongo = "91b36ca9-7576-403c-9f57-db1cc0c3e1ce"
	EmployeeRoleIdMongo       = "58e45143-f11e-4d8d-84d5-90b6435ab564"

	// Postgres Staging Creds
	appIdPg         = "P-1cDW9ISg1ko1rSvffoSg1m0gJRHl7vgh"
	ProjectIdPg     = "8e83e7d6-954e-4c13-bb85-2119c245dcea"
	ResourceIdPg    = "835206e8-f971-41f0-838b-54ae6c53ca97"
	EnvironmentIdPg = "7ab0af4a-6ae2-417f-a8f0-e45315ab0b60"
	ClientTypeIdPg  = "fd941777-68e2-4a4f-acd1-a664cceaa4ea"
	RoleIdPg        = "dda0320f-999f-4018-b172-70d4b2bd6792"

	EmployeeClientTypeIdPg = "61f967dd-cd9f-496e-99b4-cd32177baba2"
	EmployeeRoleIdPg       = "ed239299-a987-4fa0-b0ef-9b7d69081d93"
)

var (
	UcodeApi           = sdk.New(&sdk.Config{BaseURL: BaseUrl, RequestTimeout: requestTimeout, AppId: appId})
	UcodeApiForPg      = sdk.New(&sdk.Config{BaseURL: BaseUrlStaging, RequestTimeout: requestTimeout, AppId: appIdPg})
	UcodeApiForStaging = sdk.New(&sdk.Config{
		BaseURL:        BaseUrlStaging,
		RequestTimeout: requestTimeout,
		AppId:          appIdPg,
		ProjectId:      ProjectIdPg,
		BaseAuthUrl:    BaseUrlAuthStaging,
	})
	fakeData *faker.Faker
)

func TestMain(m *testing.M) {
	fakeData, _ = faker.New("en")

	body, err := UcodeApiForStaging.DoRequest(BaseUrl+"/v1/table", http.MethodPost, TableReq,
		map[string]string{
			"Resource-Id":    ResourceId,
			"Environment-Id": EnvironmentId,
			"X-API-KEY":      UcodeApi.Config().AppId,
			"Authorization":  "API-KEY",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	bodyAsMap := cast.ToStringMap(string(body))
	tableId := cast.ToString(cast.ToStringMap(bodyAsMap["data"])["id"])

	_, err = UcodeApiForStaging.DoRequest(BaseUrl+"/v1/table/"+tableId, http.MethodDelete, map[string]interface{}{},
		map[string]string{
			"Resource-Id":    ResourceId,
			"Environment-Id": EnvironmentId,
			"X-API-KEY":      UcodeApi.Config().AppId,
			"Authorization":  "API-KEY",
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

var TableReq = map[string]interface{}{
	"show_in_menu": true,
	"label":        "Integration table",
	"slug":         "integration_table",
	"attributes": map[string]interface{}{
		"label":    "",
		"label_en": "pikachu",
	},
}

var ExcelReq = map[string]interface{}{
	"data": map[string]interface{}{
		"field_ids": []string{
			"fd33c3ed-9326-4f76-9c43-706689369738",
			"e4eec7bd-0748-4543-b7b3-ad7559173e83",
			"8cea34f5-432b-4530-96fa-954b5d015ef2",
			"51a6ffe5-99b4-417e-8ff5-57961a82f427",
			"22645a76-fd31-4a1d-8f49-bb2310ef5ec5",
			"e6262d69-5b55-4d3b-994c-80260cd039a0",
			"4fa0f7b3-b12e-4728-b2c0-ce56b625b02f",
			"76979418-a57f-4d57-aa95-b07a3513fca3",
			"b6dcf877-fb2c-470a-8b4d-7730d174da63",
			"814bcbb1-c086-4267-8641-4dc1425791e8",
			"3b23ec04-9c3f-4f04-9345-57c198ac8b11",
			"341f6ab9-fd2f-4279-b2d8-161867c3c354",
			"deafef96-8371-483e-a43e-a65f180c563f",
			"7fb671e5-7e9e-475d-a184-0a7b97442064",
			"984f5ea9-df64-48bc-8a57-9d067e4a1b0d",
			"2f11a4ce-d421-402d-8ed0-6be5d8fa6be9",
			"a378b474-3726-4bc5-8388-1b1cd5ac0d29",
			"076101e1-c7af-452a-a812-4770941a223b",
			"5e119b85-2e4a-4bf8-ab28-45d40163ea7f",
			"2f6620af-aac3-4d2b-98c7-55826a920da9",
			"266e6249-cbf4-4eb1-8d8f-e7a6c55092d8",
		},
		"language": "en",
		"search":   "",
		"view_fields": []string{
			"single_line_field",
			"multi_line_field",
			"number_field",
			"formula_in_frontend_field",
			"internation_phone_field",
			"email_field",
		},
	},
}

var ExcelReqPg = map[string]interface{}{
	"data": map[string]interface{}{
		"field_ids": []string{
			"540c6dee-09bb-4a8c-a541-0189590eadfb",
			"3dee395d-fa70-4f22-a8cf-06b66e017357",
			"f4bcc98e-2ad0-46d7-a214-a45137b8dd85",
			"cde90b05-7d67-4825-9546-c77be4582367",
			"bf5d2845-0aa2-4056-918b-24ca33a162e7",
			"931570f3-2c87-43be-b2f2-5f7c38543724",
			"7997808e-2029-4205-9bf5-28d47f829625",
			"70005076-2cf5-483f-8b73-a0265d30e060",
			"cd293c06-32c5-4baa-8fd3-91bfc1635f40",
			"7f191bb2-4709-4eaf-9428-c9ead48cca50",
			"efadfaf1-6e39-4b8a-9811-6419b894b7fc",
			"a442b74d-c751-40d5-811f-aa6d34f4a9ce",
			"b1d4ad1e-2046-453e-b9c5-bfe825900ff8",
			"539d1622-97fe-42ea-9d83-8c43b2e33022",
			"4f01c613-7c19-4b6c-9c8d-961003fef1f4",
			"648d4fcf-c0bd-4144-b122-3082605bb736",
			"c6773299-02e2-4c8c-9af4-87c327f8d7a8",
			"74e67292-ed63-4d78-9c9f-341c5d39b205",
			"8a445db6-8837-43b6-9db8-0bc251733fb7",
			"dfd37ee8-ad7d-457e-a11a-2c7d735c7ba3",
			"c29bf7a6-9235-4632-a8cf-02e8c52bc800",
			"c2c2e208-cebc-43bc-ac8a-7626939c695f",
		},
		"language": "en",
		"search":   "",
		"view_fields": []string{
			"single_line_field",
			"multi_line_field",
			"number_field",
			"increment_id_field",
			"formula_in_frontend_field",
			"internation_phone_field",
			"email_field",
		},
	},
}

func Login() (string, error) {
	var (
		body = map[string]any{
			"username":        "bosit_test_002",
			"password":        "bosit_test_002",
			"company_id":      "d324e78a-e57b-4951-b83b-910e02521012",
			"project_id":      "462baeca-37b0-4355-addc-b8ae5d26995d",
			"environment_id":  "ad41c493-8697-4f23-979a-341722465748",
			"client_type":     "91b36ca9-7576-403c-9f57-db1cc0c3e1ce",
			"tables":          []map[string]string{{"object_id": "aa6243da-4966-4cca-994e-85893a1df97c", "table_slug": "company"}},
			"environment_ids": []string{"ad41c493-8697-4f23-979a-341722465748"},
		}
	)

	loginResp, _, err := UcodeApiForStaging.Auth().Login(body).Headers(map[string]string{}).Exec()
	if err != nil {
		return "", err
	}

	return loginResp.Data.Token.AccessToken, nil
}

func LoginPg() (string, error) {
	var (
		url  = "https://auth-api.ucode.run/v2/login?project-id=8e83e7d6-954e-4c13-bb85-2119c245dcea"
		body = map[string]interface{}{
			"username":        "bosit_test_001",
			"password":        "bosit_test_001",
			"company_id":      "1ada9292-c76a-453d-a323-559538baa0ee",
			"project_id":      "8e83e7d6-954e-4c13-bb85-2119c245dcea",
			"environment_id":  "7ab0af4a-6ae2-417f-a8f0-e45315ab0b60",
			"client_type":     "61f967dd-cd9f-496e-99b4-cd32177baba2",
			"tables":          []map[string]string{{"object_id": "17c91c72-5666-4f47-95a0-356c1398493b", "table_slug": "company"}},
			"environment_ids": []string{"7ab0af4a-6ae2-417f-a8f0-e45315ab0b60"},
		}
	)

	bodyBytes, err := UcodeApiForPg.DoRequest(url, "POST", body, map[string]string{})
	if err != nil {
		return "", err
	}

	var response LoginResponse

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return "", err
	}

	return response.Data.Token.AccessToken, nil
}

type LoginResponse struct {
	Data struct {
		Token Token `json:"token"`
	} `json:"data"`
}

type Token struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type GetListApiResponse struct {
	Status string `json:"status"`
	Data   struct {
		Data struct {
			Response []map[string]interface{} `json:"response"`
		} `json:"data"`
	} `json:"data"`
}

type ExcelResponse struct {
	Data struct {
		Data struct {
			Link string `json:"link"`
		} `json:"data"`
	} `json:"data"`
}

type RegisterResponse struct {
	Status      string `json:"status"`
	Description string `json:"description"`
	Data        struct {
		UserID string `json:"user_id"`
	} `json:"data"`
}

type SmsResponse struct {
	Status      string `json:"status"`
	Description string `json:"description"`
	Data        struct {
		SmsID       string `json:"sms_id"`
		GoogleAcces bool   `json:"google_acces"`
		UserFound   bool   `json:"user_found"`
	} `json:"data"`
}
