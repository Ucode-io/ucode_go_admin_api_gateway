package tests

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	sdk "github.com/golanguzb70/ucode-sdk"
	"github.com/manveru/faker"
	"github.com/spf13/cast"
)

const (
	appId       = "P-bgh4cmZxaWTXWscpH6sUa9gGlsuvKyZO"
	appIdPg     = "P-1cDW9ISg1ko1rSvffoSg1m0gJRHl7vgh"
	BaseUrl     = "https://api.admin.u-code.io"
	BaseUrlAuth = "https://api.auth.u-code.io"

	BaseUrlAuthStaging = "https://auth-api.ucode.run"
	BaseUrlStaging     = "https://admin-api.ucode.run"

	requestTimeout = time.Second * 5

	ProjectId     = "f05fdd8d-f949-4999-9593-5686ac272993"
	ResourceId    = "b74a3b18-6531-45fc-8e05-0b9709af8faa"
	EnvironmentId = "e8b82a93-b87f-4103-abc4-b5a017f540a4"
	ClientTypeId  = "ce3e630f-5399-4557-94f5-f142b411ed6b"
	RoleId        = "d1523cf2-c684-4c14-b021-413011ffb375"
	Email         = "abdulbositkabilov@gmail.com"

	EmployeeClientTypeId = "8a69fc80-6316-4f84-8914-3e7ebae03dc7"
	EmployeeRoleId       = "b64ac7b7-9ec9-42e0-b720-1267ca1e42f7"
)

var (
	UcodeApi      = sdk.New(&sdk.Config{BaseURL: BaseUrl, RequestTimeout: requestTimeout, AppId: appId})
	UcodeApiForPg = sdk.New(&sdk.Config{BaseURL: BaseUrlStaging, RequestTimeout: requestTimeout, AppId: appIdPg})
	fakeData      *faker.Faker
)

func TestMain(m *testing.M) {
	fakeData, _ = faker.New("en")

	body, err := UcodeApi.DoRequest(BaseUrl+"/v1/table", http.MethodPost, TableReq,
		map[string]string{
			"Resource-Id":    ResourceId,
			"Environment-Id": EnvironmentId,
			"X-API-KEY":      UcodeApi.Config().AppId,
			"Authorization":  "API-KEY",
		},
	)
	if err == nil {
		log.Fatal(err)
	}

	bodyAsMap := cast.ToStringMap(string(body))
	tableId := cast.ToString(cast.ToStringMap(bodyAsMap["data"])["id"])

	_, err = UcodeApi.DoRequest(BaseUrl+"/v1/table/"+tableId, http.MethodDelete, map[string]interface{}{},
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
			"a716bc3e-611d-413e-b849-2e97e19bf603",
			"2e6cec61-9031-4e98-b8d2-f733c6c19698",
			"6ebc4072-0d5d-4471-b99c-2fb5a34b30cf",
			"c1a4e368-0090-4f69-b20b-76b9e35b37d1",
			"67005876-7525-4551-96f1-915cd2001690",
			"43dcdaac-c702-47c2-85cb-37a383db9bd7",
			"23bb85e8-e2cf-49e1-bc61-2bfe19ead38e",
			"4ca321b1-bc2f-4632-a055-e6a3aec9063c",
			"45612137-f89e-4541-931d-94bc43331f52",
			"fed689ba-477a-4961-bb93-30c89a6105e9",
			"15d6334d-567c-44c7-bd82-fe38a57b8b8b",
			"a7e61d62-10d1-4562-ade1-a27268861941",
			"0770eaa1-c5f0-4e5f-bd78-84e9f151984b",
			"68baa102-9cb2-4b0d-8a9c-5ef139bd183e",
			"0e84701c-d50d-4e27-bd61-a8104282f740",
			"287920d5-cbe3-4d7f-be50-0e30e85c2ac0",
			"a8201d2b-1b98-410e-94e9-d7dd8bd377a5",
			"d1152882-be4f-47a0-b69f-35829cca6b27",
			"9b155bb9-bd92-4408-84ad-b6e298a0df0f",
			"ea152b67-fd58-4251-af17-111db27c634b",
			"f91e5834-26bb-4d58-8eca-6f5f8c332255",
			"3168c528-9954-4e62-8fd9-2821489177ed",
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
		url  = "https://api.auth.u-code.io/v2/login?project-id=f05fdd8d-f949-4999-9593-5686ac272993"
		body = map[string]interface{}{
			"username":        "bosit001",
			"password":        "bosit001",
			"company_id":      "af1f4f6e-1c33-4e9e-99e7-12c0d6ff4bd1",
			"project_id":      "f05fdd8d-f949-4999-9593-5686ac272993",
			"environment_id":  "e8b82a93-b87f-4103-abc4-b5a017f540a4",
			"client_type":     "8a69fc80-6316-4f84-8914-3e7ebae03dc7",
			"tables":          []map[string]string{{"object_id": "ae0183d4-860a-409c-b54c-2818d6b690f7", "table_slug": "Bosit"}},
			"environment_ids": []string{"e8b82a93-b87f-4103-abc4-b5a017f540a4"},
		}
	)

	bodyBytes, err := UcodeApi.DoRequest(url, "POST", body, map[string]string{})
	if err != nil {
		return "", err
	}

	var response LoginResponse

	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return "", err
	}

	return response.Data.Token.AccessToken, nil
}

func LoginPg() (string, error) {
	var (
		url  = "https://auth-api.ucode.run/v2/login?project-id=8e83e7d6-954e-4c13-bb85-2119c245dcea"
		body = map[string]interface{}{
			"username":        "test_for_test",
			"password":        "test_for_test",
			"company_id":      "1ada9292-c76a-453d-a323-559538baa0ee",
			"project_id":      "8e83e7d6-954e-4c13-bb85-2119c245dcea",
			"environment_id":  "7ab0af4a-6ae2-417f-a8f0-e45315ab0b60",
			"client_type":     "61f967dd-cd9f-496e-99b4-cd32177baba2",
			"tables":          []map[string]string{{"object_id": "17c91c72-5666-4f47-95a0-356c1398493b", "table_slug": "company"}},
			"environment_ids": []string{"7ab0af4a-6ae2-417f-a8f0-e45315ab0b60"},
		}
	)

	bodyBytes, err := UcodeApi.DoRequest(url, "POST", body, map[string]string{})
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
