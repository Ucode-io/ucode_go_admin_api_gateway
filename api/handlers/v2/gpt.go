package v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	"ucode/ucode_go_api_gateway/pkg/gpt"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// SendToGpt godoc
// @Security ApiKeyAuth
// @ID send_to_gpt
// @Router /v2/send-to-gpt [POST]
// @Summary Send To Gpt
// @Description Send To Gpt
// @Tags GPT
// @Accept json
// @Produce json
// @Param object body models.SendToGptRequest true "SendToGptRequestBody"
// @Success 201 {object} status_http.Response{data=string} "Success"
// @Response 400 {object} status_http.Response{data=string} "Bad Request"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) SendToGpt(c *gin.Context) {
	var (
		reqBody models.SendToGptRequest
		logReq  []models.CreateVersionHistoryRequest
	)

	err := c.ShouldBindJSON(&reqBody)
	if err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId.(string),
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	respMessages := []models.Message{}

	userId, _ := c.Get("user_id")

	respMessages = append(respMessages, models.Message{
		Role:    "user",
		Content: reqBody.Promt,
	})

	toolCalls, err := gpt.SendReqToGPT(respMessages)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		projectId.(string),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	for _, toolCall := range toolCalls {
		var (
			functionCall = toolCall.Function
			functionName = functionCall.Name
			arguments    map[string]interface{}
		)

		err = json.Unmarshal([]byte(functionCall.Arguments), &arguments)
		if err != nil {
			fmt.Println("Error parsing function arguments:", err)
			continue
		}

		switch functionName {
		case "create_menu":
			_, err = gpt.CreateMenu(&models.CreateMenuAI{
				Label:    cast.ToString(arguments["name"]),
				UserId:   userId.(string),
				Resource: resource,
				Service:  services,
			})
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		case "delete_menu", "delete_table":
			logReq, err = gpt.DeleteMenu(&models.DeleteMenuAI{
				Label:    cast.ToString(arguments["name"]),
				UserId:   cast.ToString(userId),
				Resource: resource,
				Service:  services,
			})
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		case "update_menu":
			logReq, err = gpt.UpdateMenu(&models.UpdateMenuAI{
				OldLabel: cast.ToString(arguments["old_name"]),
				NewLabel: cast.ToString(arguments["new_name"]),
				UserId:   cast.ToString(userId),
				Resource: resource,
				Service:  services,
			})
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		case "create_table":

			_, ok := arguments["menu"]
			if !ok {
				arguments["menu"] = "c57eedc3-a954-4262-a0af-376c65b5a284"
			}

			logReq, err = gpt.CreateTable(&models.CreateTableAI{
				Label:         cast.ToString(arguments["name"]),
				TableSlug:     cast.ToString(arguments["table_slug"]),
				Menu:          cast.ToString(arguments["menu"]),
				EnvironmentId: resource.EnvironmentId,
				UserId:        userId.(string),
				Resource:      resource,
				Service:       services,
			})
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		case "update_table":

			logReq, err = gpt.UpdateTable(&models.UpdateTableAI{
				OldLabel: cast.ToString(arguments["old_name"]),
				NewLabel: cast.ToString(arguments["new_name"]),
				Resource: resource,
				Service:  services,
			})
		case "create_field":

			if cast.ToString(arguments["table"]) != "" {
				logReq, err = gpt.CreateField(&models.CreateFieldAI{
					Label:    cast.ToString(arguments["label"]),
					Slug:     cast.ToString(arguments["slug"]),
					Type:     cast.ToString(arguments["type"]),
					Table:    cast.ToString(arguments["table"]),
					UserId:   userId.(string),
					Resource: resource,
					Service:  services,
				})
				if err != nil {
					h.handleResponse(c, status_http.GRPCError, err.Error())
					return
				}
			} else {
				h.handleResponse(c, status_http.BadRequest, "Table not found please give table's label")
			}

		case "update_field":

			logReq, err = gpt.UpdateField(&models.UpdateFieldAI{
				NewLabel: cast.ToString(arguments["new_label"]),
				OldLabel: cast.ToString(arguments["old_label"]),
				NewType:  cast.ToString(arguments["new_type"]),
				Table:    cast.ToString(arguments["table"]),
				UserId:   userId.(string),
				Resource: resource,
				Service:  services,
			})
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		case "delete_field":

			logReq, err = gpt.DeleteField(&models.DeleteFieldAI{
				Label:    cast.ToString(arguments["label"]),
				Table:    cast.ToString(arguments["table"]),
				UserId:   userId.(string),
				Resource: resource,
				Service:  services,
			})
			if err != nil {
				h.handleResponse(c, status_http.GRPCError, err.Error())
				return
			}
		case "create_relation":

			fmt.Println("CREATE RELATION >>>>>>")

			fmt.Println(arguments["table_from"])
			fmt.Println(arguments["table_to"])
			fmt.Println(arguments["relation_type"])
			fmt.Println(arguments["view_field"])
			fmt.Println(arguments["view_type"])

		default:

			h.handleResponse(c, status_http.BadRequest, "Unknown function: "+functionName)
			return
		}

		for _, log := range logReq {
			switch resource.ResourceType {
			case pb.ResourceType_MONGODB:
				go h.versionHistory(c, &log)
			case pb.ResourceType_POSTGRESQL:
				go h.versionHistoryGo(c, &log)
			}
		}
	}

	h.handleResponse(c, status_http.OK, "Success")
}
