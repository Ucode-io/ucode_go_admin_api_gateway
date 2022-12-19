package helper

import (
	"encoding/json"
	"strconv"
	"strings"

	pb "ucode/ucode_go_api_gateway/genproto/auth_service"
	pbObject "ucode/ucode_go_api_gateway/genproto/object_builder_service"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func MarshalToStruct(data interface{}, resp interface{}) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}

	err = json.Unmarshal(js, resp)
	if err != nil {
		return err
	}

	return nil
}

func ConvertMapToStruct(inputMap map[string]interface{}) (*structpb.Struct, error) {
	marshledInputMap, err := json.Marshal(inputMap)
	outputStruct := &structpb.Struct{}
	if err != nil {
		return outputStruct, err
	}
	err = protojson.Unmarshal(marshledInputMap, outputStruct)

	return outputStruct, err
}

func GetURLWithTableSlug(c *gin.Context) string {
	url := c.FullPath()
	if strings.Contains(url, ":table_slug") {
		tableSlug := c.Param("table_slug")
		url = strings.Replace(url, ":table_slug", tableSlug, -1)
	}

	return url
}

func ReplaceQueryParams(namedQuery string, params map[string]interface{}) (string, []interface{}) {
	var (
		i    int = 1
		args     = make([]interface{}, 0, len(params))
	)

	for k, v := range params {
		if k != "" {
			oldsize := len(namedQuery)
			namedQuery = strings.ReplaceAll(namedQuery, ":"+k, "$"+strconv.Itoa(i))

			if oldsize != len(namedQuery) {
				args = append(args, v)
				i++
			}
		}
	}

	return namedQuery, args
}

func ConvertPbToAnotherPb(data *pbObject.V2LoginResponse) *pb.V2LoginResponse {
	res := &pb.V2LoginResponse{}
	res.UserId = data.UserId
	res.LoginTableSlug = data.LoginTableSlug
	tables := make([]*pb.Table, 0, len(data.ClientType.Tables))
	for _, v := range data.ClientType.Tables {
		table := &pb.Table{}
		table.Data = v.Data
		table.Icon = v.Icon
		table.Label = v.Label
		table.Slug = v.Slug
		table.ViewLabel = v.ViewLabel
		table.ViewSlug = v.ViewSlug
		tables = append(tables, table)
	}

	res.ClientType = &pb.ClientType{
		Id:           data.ClientType.Guid,
		Name:         data.ClientType.Name,
		ConfirmBy:    pb.ConfirmStrategies(data.ClientType.ConfirmBy),
		SelfRegister: data.ClientType.SelfRegister,
		SelfRecover:  data.ClientType.SelfRecover,
		ProjectId:    data.ClientType.ProjectId,
		Tables:       tables,
	}

	res.ClientPlatform = &pb.ClientPlatform{
		Id:        data.ClientPlatform.Guid,
		Name:      data.ClientPlatform.Name,
		ProjectId: data.ClientPlatform.ProjectId,
		Subdomain: data.ClientPlatform.Subdomain,
	}
	permissions := make([]*pb.RecordPermission, 0, len(data.Permissions))
	for _, v := range data.Permissions {
		permission := &pb.RecordPermission{}
		permission.ClientTypeId = v.ClientTypeId
		permission.Id = v.Guid
		permission.Read = v.Read
		permission.Write = v.Write
		permission.Delete = v.Delete
		permission.Update = v.Update
		permission.RoleId = v.RoleId
		permission.TableSlug = v.TableSlug
		permissions = append(permissions, permission)
	}
	res.Permissions = permissions
	res.Role = &pb.Role{
		Id:               data.Role.Guid,
		ClientTypeId:     data.Role.ClientTypeId,
		Name:             data.Role.Name,
		ClientPlatformId: data.Role.ClientPlatformId,
		ProjectId:        data.Role.ProjectId,
	}
	return res
}


func ConverPhoneNumberToMongoPhoneFormat(input string) string {
	//input +998995677777
	input = input[4:]
	// input  = 995677777
	changedEl := input[:2]
	input = "(" + changedEl + ") " + input[2:5] + "-" + input[5:7] + "-" + input[7:]
	// input = (99) 567-77-77
	return input
}