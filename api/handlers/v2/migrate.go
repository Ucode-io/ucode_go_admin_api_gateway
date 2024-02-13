package v2

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

type DataTableWrapper1 struct {
	Data *obs.CreateTableRequest
}

type DataTableWrapper2 struct {
	Data *obs.UpdateTableRequest
}

type DataFieldWrapper1 struct {
	Data *obs.CreateFieldRequest
}

type DataFieldWrapper2 struct {
	Data *obs.Field
}

type DataViewWrapper struct {
	Data *obs.View
}

func (h *HandlerV2) MigrateUp(c *gin.Context) {
	req := []*models.MigrateUp{}

	if err := c.ShouldBindJSON(&req); err != nil {
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
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	userId, _ := c.Get("user_id")

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
		var (
			actionSource          = v.ActionSource
			actionType            = strings.Split(v.ActionType, " ")[0]
			nodeType              = resource.NodeType
			resourceEnvironmentId = resource.ResourceEnvironmentId

			logReq = &models.CreateVersionHistoryRequest{
				Services:     services,
				NodeType:     nodeType,
				ProjectId:    resourceEnvironmentId,
				ActionSource: v.ActionSource,
				ActionType:   v.ActionType,
				UsedEnvironments: map[string]bool{
					cast.ToString(environmentId): true,
				},
				UserInfo: cast.ToString(userId),
			}
		)

		if actionSource == "TABLE" {
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				previous DataTableWrapper2
				request  DataTableWrapper1
				response DataTableWrapper1
			)

			err := json.Unmarshal([]byte(cast.ToString(v.Previous)), &previous)
			if err != nil {
				log.Println(err)
				return
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Request)), &request)
			if err != nil {
				log.Println(err)
				return
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Response)), &response)
			if err != nil {
				log.Println(err)
				return
			}

			logReq.Request = request.Data
			logReq.TableSlug = request.Data.Slug

			switch actionType {
			case "CREATE":
				request.Data.Id = response.Data.Id

				createTable, err := services.GetBuilderServiceByType(nodeType).Table().Create(
					context.Background(),
					request.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
				logReq.Response = createTable
			case "UPDATE":
				updateTable, err := services.GetBuilderServiceByType(nodeType).Table().Update(
					context.Background(),
					previous.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
				logReq.Response = updateTable
			}
		} else if actionSource == "FIELD" {
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				current1 DataFieldWrapper1
				current2 DataFieldWrapper2
			)

			err = json.Unmarshal([]byte(cast.ToString(v.Current)), &current1)
			if err != nil {
				log.Println(err)
				return
			}

			switch actionType {
			case "CREATE":
				createField, err := services.GetBuilderServiceByType(nodeType).Field().Create(
					context.Background(),
					current1.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
				logReq.Response = createField
			case "UPDATE":
				updateField, err := services.GetBuilderServiceByType(nodeType).Field().Update(
					context.Background(),
					current2.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
				logReq.Response = updateField
			case "DELETE":
				_, err := services.GetBuilderServiceByType(nodeType).Field().Delete(
					context.Background(),
					&obs.FieldPrimaryKey{Id: current1.Data.Id},
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
				logReq.Previous = current1.Data
			}
		} else if actionSource == "VIEW" {
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				previous DataViewWrapper
			)

			err = json.Unmarshal([]byte(cast.ToString(v.Previous)), &previous)
			if err != nil {
				log.Println(err)
				return
			}

			switch actionSource {
			case "CREATE":
				createView, err := services.GetBuilderServiceByType(nodeType).View().Create(
					context.Background(),
					&view,
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
			case "UPDATE":
				
			case "DELETE":
				logReq.Previous = previous.Data
				_, err := services.GetBuilderServiceByType(resource.NodeType).View().Delete(
					context.Background(),
					&obs.ViewPrimaryKey{
						Id:        previous.Data.Id,
						ProjectId: resource.ResourceEnvironmentId,
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
			}
		}
	}
}
