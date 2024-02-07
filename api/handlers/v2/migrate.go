package v2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

type FieldWrapper struct {
	Data *obs.CreateFieldRequest
}

func (h *HandlerV2) MigrateUp(c *gin.Context) {
	req := []*models.MigrateUp{}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}
	fmt.Println("Req->", req)

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		err := errors.New("error getting environment id | not valid")
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

	services, err := h.GetProjectSrvc(
		c.Request.Context(),
		resource.GetProjectId(),
		resource.NodeType,
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	for _, v := range req {
		fmt.Printf("Heeeey - > %+v", v)
		if v.ActionSource == "FIELD" {
			fmt.Println("Hello World from field")
			var wrapper FieldWrapper
			err := json.Unmarshal([]byte(cast.ToString(v.Previous)), &wrapper)
			if err != nil {
				log.Println(err)
				return
			}

			fmt.Println("I am coming  here")
			fmt.Println("Hey->", v.ActionType)

			if v.ActionType == "DELETE" {
				fmt.Println("hello world from delete")
				_, err := services.GetBuilderServiceByType(resource.NodeType).Field().Delete(
					context.Background(),
					&obs.FieldPrimaryKey{
						Id:        wrapper.Data.Id,
						ProjectId: resource.ResourceEnvironmentId,
					},
				)
				if err != nil {
					h.handleResponse(c, status_http.GRPCError, err.Error())
					return
				}
			}
		}
	}
}
