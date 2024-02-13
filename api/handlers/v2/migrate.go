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

type DataViewWrapper1 struct {
	Data *obs.CreateViewRequest
}

type DataViewWrapper2 struct {
	Data *obs.View
}

type DataMenuCreateWrapper struct {
	Data *obs.CreateMenuRequest
}

type DataMenuUpdateWrapper struct {
	Data *obs.Menu
}

type DataRelationCreateWrapper struct {
	Data *obs.CreateRelationRequest
}

type DataRelationUpdateWrapper struct {
	Data *obs.UpdateRelationRequest
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

		switch actionSource {
		case "TABLE":
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
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Request)), &request)
			if err != nil {
				log.Println(err)
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Response)), &response)
			if err != nil {
				log.Println(err)
				continue
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
				}
				logReq.Current = updateTable
				logReq.Response = updateTable
			}
		case "FIELD":
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				request  DataFieldWrapper1
				response DataFieldWrapper2
			)

			err = json.Unmarshal([]byte(cast.ToString(v.Request)), &request)
			if err != nil {
				log.Println(err)
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Response)), &response)
			if err != nil {
				log.Println(err)
				continue
			}

			logReq.TableSlug = request.Data.TableId

			switch actionType {
			case "CREATE":
				createField, err := services.GetBuilderServiceByType(nodeType).Field().Create(
					context.Background(),
					request.Data,
				)
				if err != nil {
					logReq.Request = request.Data
					logReq.Response = err.Error()
					log.Println(err)
				}
				logReq.Request = request.Data
				logReq.Response = createField
				logReq.Current = createField
			case "UPDATE":
				oldField, err := services.GetBuilderServiceByType(resource.NodeType).Field().GetByID(
					context.Background(),
					&obs.FieldPrimaryKey{
						Id:        response.Data.Id,
						ProjectId: resource.ResourceEnvironmentId,
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
				}

				logReq.Request = response.Data
				logReq.Previous = oldField

				updateField, err := services.GetBuilderServiceByType(nodeType).Field().Update(
					context.Background(),
					response.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
				}
				logReq.Response = updateField
				logReq.Current = updateField
			case "DELETE":
				oldField, err := services.GetBuilderServiceByType(resource.NodeType).Field().GetByID(
					context.Background(),
					&obs.FieldPrimaryKey{
						Id:        response.Data.Id,
						ProjectId: resource.ResourceEnvironmentId,
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
				}

				logReq.Previous = oldField

				_, err = services.GetBuilderServiceByType(nodeType).Field().Delete(
					context.Background(),
					&obs.FieldPrimaryKey{Id: response.Data.Id},
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
				}
			}
		case "VIEW":
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				request DataViewWrapper1
				current DataViewWrapper2
			)

			err = json.Unmarshal([]byte(cast.ToString(v.Current)), &current)
			if err != nil {
				log.Println(err)
				continue
			}

			logReq.TableSlug = request.Data.TableSlug

			switch actionSource {
			case "CREATE":
				createView, err := services.GetBuilderServiceByType(nodeType).View().Create(
					context.Background(),
					request.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
				}
				logReq.Response = createView
			case "UPDATE":
				resp, err := services.GetBuilderServiceByType(resource.NodeType).View().Update(
					context.Background(),
					current.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
				}
				logReq.Current = resp
			case "DELETE":
				_, err = services.GetBuilderServiceByType(resource.NodeType).View().Delete(
					context.Background(),
					&obs.ViewPrimaryKey{
						Id:        current.Data.Id,
						ProjectId: resource.ResourceEnvironmentId,
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
				}
			}
		case "MENU":
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				previous DataMenuUpdateWrapper
				request  DataMenuCreateWrapper
				response DataMenuCreateWrapper
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
			// logReq.TableSlug = request.Data.Slug

			switch actionType {
			case "CREATE":
				request.Data.Id = response.Data.Id

				createMenu, err := services.GetBuilderServiceByType(nodeType).Menu().Create(
					context.Background(),
					request.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
				logReq.Response = createMenu
			case "UPDATE":
				updatemenu, err := services.GetBuilderServiceByType(nodeType).Menu().Update(
					context.Background(),
					previous.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
				logReq.Response = updatemenu
			case "DELETE":
				deleteMenu, err := services.GetBuilderServiceByType(nodeType).Menu().Delete(
					context.Background(),
					&obs.MenuPrimaryKey{
						Id:        previous.Data.Id,
						ProjectId: resource.ResourceEnvironmentId,
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
				logReq.Response = deleteMenu
			}
		case "RELATION":
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				previous DataRelationUpdateWrapper
				request  DataRelationCreateWrapper
				response DataMenuCreateWrapper
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
			// logReq.TableSlug = request.Data.Slug

			switch actionType {
			case "CREATE":
				request.Data.Id = response.Data.Id

				createRelation, err := services.GetBuilderServiceByType(nodeType).Relation().Create(
					context.Background(),
					request.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
				logReq.Response = createRelation
			case "UPDATE":
				updateRelation, err := services.GetBuilderServiceByType(nodeType).Relation().Update(
					context.Background(),
					previous.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
				logReq.Response = updateRelation
			case "DELETE":
				deleteRelation, err := services.GetBuilderServiceByType(nodeType).Relation().Delete(
					context.Background(),
					&obs.RelationPrimaryKey{
						Id:        previous.Data.Id,
						ProjectId: resource.ResourceEnvironmentId,
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					log.Println(err)
					return
				}
				logReq.Response = deleteRelation
			}
		}
	}
}
