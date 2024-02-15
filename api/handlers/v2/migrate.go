package v2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"ucode/ucode_go_api_gateway/api/models"
	"ucode/ucode_go_api_gateway/api/status_http"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	obs "ucode/ucode_go_api_gateway/genproto/object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

type DataTableWrapper struct {
	Data *obs.CreateTableRequest
}

type DataUpdateTableWrapper struct {
	Data *obs.UpdateTableRequest
}

type DataCreateFieldWrapper struct {
	Data *obs.CreateFieldRequest
}

type DataFieldWrapper struct {
	Data *obs.Field
}

type DataCreateRelationWrapper struct {
	Data *obs.CreateRelationRequest
}

type DataCreateMenuWrapper struct {
	Data *obs.CreateMenuRequest
}

type DataUpdateMenuWrapper struct {
	Data *obs.Menu
}

type DataCreateViewWrapper struct {
	Data *obs.CreateViewRequest
}

type DataUpdateViewWrapper struct {
	Data *obs.View
}

type DataCreateLayoutWrapper struct {
	Data *obs.CreateLayoutRequest
}

type DataUpdateLayoutWrapper struct {
	Data *obs.LayoutRequest
}

// MigrateUp godoc
// @Security ApiKeyAuth
// @ID migrate_up
// @Router /v2/version/history/migrate/up/{environment_id} [POST]
// @Summary Migrate up
// @Description Migrate up
// @Tags VersionHistory
// @Accept json
// @Produce json
// @Param environment_id path string true "environment_id"
// @Param migrate body models.MigrateUpRequest true "MigrateUpRequest"
// @Success 200 {object} status_http.Response{data=models.MigrateUpResponse} "Upbody"
// @Response 400 {object} status_http.Response{data=string} "Invalid Argument"
// @Failure 500 {object} status_http.Response{data=string} "Server Error"
func (h *HandlerV2) MigrateUp(c *gin.Context) {
	var (
		resp models.MigrateUpResponse
		req  models.MigrateUpRequest
		ids  []string
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	namespace := c.GetString("namespace")
	services, err := h.GetService(namespace)
	if err != nil {
		h.handleResponse(c, status_http.Forbidden, err)
		return
	}

	migrateRequest := req.Data

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	currentEnvironmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(currentEnvironmentId.(string)) {
		err = errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	environmentId := currentEnvironmentId.(string)

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId,
			ServiceType:   pb.ServiceType_BUILDER_SERVICE,
		},
	)
	if err != nil {
		h.handleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	for _, v := range migrateRequest {
		var (
			actionSource  = v.ActionSource
			actionType    = strings.Split(v.ActionType, " ")[0]
			nodeType      = resource.NodeType
			resourceEnvId = resource.ResourceEnvironmentId

			logReq = &models.CreateVersionHistoryRequest{
				Services:     services,
				NodeType:     nodeType,
				ProjectId:    resourceEnvId,
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
				previous      DataTableWrapper
				current       DataTableWrapper
				currentUpdate DataUpdateTableWrapper
			)

			err := json.Unmarshal([]byte(cast.ToString(v.Request)), &current)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Previous)), &previous)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Current)), &currentUpdate)
			if err != nil {
				continue
			}

			current.Data.ProjectId = resourceEnvId
			current.Data.EnvId = cast.ToString(environmentId)
			currentUpdate.Data.ProjectId = resourceEnvId
			currentUpdate.Data.EnvId = cast.ToString(environmentId)
			logReq.Request = current.Data
			logReq.TableSlug = current.Data.Slug

			switch actionType {
			case "CREATE":
				fmt.Println("HERE AGAIN")
				_, err = services.GetBuilderServiceByType(nodeType).Table().Create(
					context.Background(),
					current.Data,
				)
				if err != nil {
					fmt.Println("CREATE TABLE ERROR: ", err)
					logReq.Response = err.Error()
					continue
				}
				logReq.Current = current.Data
				logReq.Response = current.Data
				ids = append(ids, v.Id)
			case "UPDATE":
				logReq.Previous = previous.Data

				_, err = services.GetBuilderServiceByType(nodeType).Table().Update(
					context.Background(),
					currentUpdate.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Current = current.Data
				logReq.Response = current.Data
				ids = append(ids, v.Id)
			}
		} else if actionSource == "FIELD" {
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				current       DataCreateFieldWrapper
				previous      DataFieldWrapper
				currentUpdate DataFieldWrapper
			)

			err := json.Unmarshal([]byte(cast.ToString(v.Current)), &current)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Previous)), &previous)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Current)), &currentUpdate)
			if err != nil {
				continue
			}

			current.Data.ProjectId = resourceEnvId
			current.Data.EnvId = cast.ToString(environmentId)
			currentUpdate.Data.ProjectId = resourceEnvId
			currentUpdate.Data.EnvId = cast.ToString(environmentId)
			previous.Data.ProjectId = resourceEnvId
			previous.Data.EnvId = cast.ToString(environmentId)
			logReq.TableSlug = current.Data.TableId

			switch actionType {
			case "CREATE":
				createField, err := services.GetBuilderServiceByType(nodeType).Field().Create(
					context.Background(),
					current.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Request = current.Data
				logReq.Current = current.Data
				logReq.Response = createField
				ids = append(ids, v.Id)
			case "UPDATE":
				logReq.Previous = previous.Data

				updateField, err := services.GetBuilderServiceByType(nodeType).Field().Update(
					context.Background(),
					currentUpdate.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Request = currentUpdate.Data
				logReq.Current = updateField
				logReq.Response = updateField
				ids = append(ids, v.Id)
			case "DELETE":
				logReq.Previous = previous.Data
				_, err := services.GetBuilderServiceByType(nodeType).Field().Delete(
					context.Background(),
					&obs.FieldPrimaryKey{
						Id:        previous.Data.Id,
						ProjectId: resourceEnvId,
						EnvId:     cast.ToString(environmentId),
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				ids = append(ids, v.Id)
			}
		} else if actionSource == "RELATION" {
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				request  DataCreateRelationWrapper
				response DataCreateRelationWrapper
			)

			err := json.Unmarshal([]byte(cast.ToString(v.Request)), &request)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Response)), &response)
			if err != nil {
				continue
			}

			request.Data.ProjectId = resourceEnvId
			request.Data.EnvId = cast.ToString(environmentId)
			response.Data.ProjectId = resourceEnvId
			response.Data.EnvId = cast.ToString(environmentId)
			logReq.TableSlug = request.Data.RelationTableSlug

			switch actionType {
			case "CREATE":
				createRelation, err := services.GetBuilderServiceByType(nodeType).Relation().Create(
					context.Background(),
					request.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Request = request.Data
				logReq.Current = createRelation
				logReq.Response = createRelation
				ids = append(ids, v.Id)
			case "DELETE":
				logReq.Previous = response.Data
				_, err := services.GetBuilderServiceByType(nodeType).Field().Delete(
					context.Background(),
					&obs.FieldPrimaryKey{
						Id:        response.Data.Id,
						ProjectId: resourceEnvId,
						EnvId:     cast.ToString(environmentId),
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				ids = append(ids, v.Id)
			}
		} else if actionSource == "MENU" {
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				request  DataCreateMenuWrapper
				previous DataCreateMenuWrapper
				current  DataUpdateMenuWrapper
			)

			err := json.Unmarshal([]byte(cast.ToString(v.Request)), &request)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Previous)), &previous)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Request)), &current)
			if err != nil {
				continue
			}

			request.Data.ProjectId = resourceEnvId
			request.Data.EnvId = cast.ToString(environmentId)
			previous.Data.ProjectId = resourceEnvId
			previous.Data.EnvId = cast.ToString(environmentId)
			current.Data.ProjectId = resourceEnvId
			current.Data.EnvId = cast.ToString(environmentId)
			logReq.TableSlug = "Menu"

			switch actionType {
			case "CREATE":
				createMenu, err := services.GetBuilderServiceByType(nodeType).Menu().Create(
					context.Background(),
					request.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Request = request.Data
				logReq.Current = createMenu
				logReq.Response = createMenu
				ids = append(ids, v.Id)
			case "UPDATE":
				logReq.Previous = previous.Data
				updatemenu, err := services.GetBuilderServiceByType(nodeType).Menu().Update(
					context.Background(),
					current.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Request = current.Data
				logReq.Current = updatemenu
				logReq.Response = updatemenu
				ids = append(ids, v.Id)
			case "DELETE":
				logReq.Previous = previous.Data
				_, err = services.GetBuilderServiceByType(nodeType).Menu().Delete(
					context.Background(),
					&obs.MenuPrimaryKey{
						Id:        previous.Data.Id,
						ProjectId: resourceEnvId,
						EnvId:     cast.ToString(environmentId),
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				ids = append(ids, v.Id)
			}
		} else if actionSource == "VIEW" {
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				request  DataCreateViewWrapper
				previous DataCreateViewWrapper
				current  DataUpdateViewWrapper
			)

			err := json.Unmarshal([]byte(cast.ToString(v.Request)), &request)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Previous)), &previous)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Request)), &current)
			if err != nil {
				continue
			}

			request.Data.ProjectId = resourceEnvId
			request.Data.EnvId = cast.ToString(environmentId)
			previous.Data.ProjectId = resourceEnvId
			previous.Data.EnvId = cast.ToString(environmentId)
			current.Data.ProjectId = resourceEnvId
			current.Data.EnvId = cast.ToString(environmentId)
			logReq.TableSlug = "View"

			switch actionType {
			case "CREATE":
				createView, err := services.GetBuilderServiceByType(nodeType).View().Create(
					context.Background(),
					request.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Request = request.Data
				logReq.Current = createView
				logReq.Response = createView
				ids = append(ids, v.Id)
			case "UPDATE":
				logReq.Previous = previous.Data
				updateView, err := services.GetBuilderServiceByType(nodeType).View().Update(
					context.Background(),
					current.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Request = current.Data
				logReq.Current = updateView
				logReq.Response = updateView
				ids = append(ids, v.Id)
			case "DELETE":
				logReq.Previous = previous.Data
				_, err = services.GetBuilderServiceByType(nodeType).View().Delete(
					context.Background(),
					&obs.ViewPrimaryKey{
						Id:        previous.Data.Id,
						ProjectId: resourceEnvId,
						EnvId:     cast.ToString(environmentId),
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				ids = append(ids, v.Id)
			}
		} else if actionSource == "LAYOUT" {
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				request  DataCreateLayoutWrapper
				previous DataUpdateLayoutWrapper
				current  DataUpdateLayoutWrapper
			)

			err := json.Unmarshal([]byte(cast.ToString(v.Request)), &request)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Previous)), &previous)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Request)), &current)
			if err != nil {
				continue
			}

			request.Data.ProjectId = resourceEnvId
			request.Data.EnvId = cast.ToString(environmentId)
			previous.Data.ProjectId = resourceEnvId
			previous.Data.EnvId = cast.ToString(environmentId)
			current.Data.ProjectId = resourceEnvId
			current.Data.EnvId = cast.ToString(environmentId)
			logReq.TableSlug = "View"

			switch actionType {
			case "UPDATE":
				logReq.Previous = previous.Data
				updateLayout, err := services.GetBuilderServiceByType(nodeType).Layout().Update(
					context.Background(),
					current.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Request = current.Data
				logReq.Current = updateLayout
				logReq.Response = updateLayout
				ids = append(ids, v.Id)
			case "DELETE":
				logReq.Previous = previous.Data
				_, err = services.GetBuilderServiceByType(nodeType).Layout().RemoveLayout(
					context.Background(),
					&obs.LayoutPrimaryKey{
						Id:        previous.Data.Id,
						ProjectId: resourceEnvId,
						EnvId:     cast.ToString(environmentId),
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				ids = append(ids, v.Id)
			}
		}
	}

	resp.Ids = ids

	h.handleResponse(c, status_http.OK, resp)
}

func (h *HandlerV2) MigrateDown(c *gin.Context) {
	var (
		ids  []string
		req  models.MigrateUpRequest
		resp models.MigrateUpResponse
	)

	if err := c.ShouldBindJSON(&req); err != nil {
		h.handleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	migrateRequest := req.Data

	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.handleResponse(c, status_http.InvalidArgument, "project id is an invalid uuid")
		return
	}

	environmentId := c.Param("environment_id")
	if !ok || !util.IsValidUUID(environmentId) {
		err := errors.New("error getting environment id | not valid")
		h.handleResponse(c, status_http.BadRequest, err)
		return
	}

	userId, _ := c.Get("user_id")

	resource, err := h.companyServices.ServiceResource().GetSingle(
		c.Request.Context(),
		&pb.GetSingleServiceResourceReq{
			ProjectId:     projectId.(string),
			EnvironmentId: environmentId,
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

	for _, v := range migrateRequest {
		var (
			actionSource  = v.ActionSource
			actionType    = strings.Split(v.ActionType, " ")[0]
			nodeType      = resource.NodeType
			resourceEnvId = resource.ResourceEnvironmentId

			logReq = &models.CreateVersionHistoryRequest{
				Services:     services,
				NodeType:     nodeType,
				ProjectId:    resourceEnvId,
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
				previous DataUpdateTableWrapper
				current  DataTableWrapper
			)

			err := json.Unmarshal([]byte(cast.ToString(v.Current)), &current)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Previous)), &previous)
			if err != nil {
				continue
			}

			current.Data.ProjectId = resourceEnvId
			current.Data.EnvId = cast.ToString(environmentId)
			logReq.Request = current.Data
			logReq.TableSlug = current.Data.Slug

			switch actionType {
			case "CREATE":
				_, err := services.GetBuilderServiceByType(nodeType).Table().Delete(
					context.Background(),
					&obs.TablePrimaryKey{
						Id:        current.Data.Id,
						ProjectId: resourceEnvId,
						EnvId:     cast.ToString(environmentId),
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Previous = current.Data
				ids = append(ids, v.Id)
			case "UPDATE":
				_, err = services.GetBuilderServiceByType(nodeType).Table().Update(
					context.Background(),
					previous.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				ids = append(ids, v.Id)
			}
		} else if actionSource == "FIELD" {
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				current        DataCreateFieldWrapper
				previous       DataFieldWrapper
				createPrevious DataCreateFieldWrapper
				currentUpdate  DataFieldWrapper
			)

			err := json.Unmarshal([]byte(cast.ToString(v.Current)), &current)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Previous)), &previous)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Current)), &currentUpdate)
			if err != nil {
				continue
			}

			current.Data.ProjectId = resourceEnvId
			current.Data.EnvId = cast.ToString(environmentId)
			currentUpdate.Data.ProjectId = resourceEnvId
			currentUpdate.Data.EnvId = cast.ToString(environmentId)
			previous.Data.ProjectId = resourceEnvId
			previous.Data.EnvId = cast.ToString(environmentId)
			logReq.TableSlug = current.Data.TableId

			switch actionType {
			case "CREATE":
				createField, err := services.GetBuilderServiceByType(nodeType).Field().Delete(
					context.Background(),
					&obs.FieldPrimaryKey{
						ProjectId: current.Data.ProjectId,
						EnvId:     current.Data.EnvId,
						Id:        current.Data.Id,
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Request = current.Data
				logReq.Current = current.Data
				logReq.Response = createField
				ids = append(ids, v.Id)
			case "UPDATE":
				logReq.Previous = previous.Data

				updateField, err := services.GetBuilderServiceByType(nodeType).Field().Update(
					context.Background(),
					previous.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Request = currentUpdate.Data
				logReq.Current = updateField
				logReq.Response = updateField
				ids = append(ids, v.Id)
			case "DELETE":
				logReq.Previous = previous.Data
				_, err := services.GetBuilderServiceByType(nodeType).Field().Create(
					context.Background(),
					createPrevious.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				ids = append(ids, v.Id)
			}
		} else if actionSource == "RELATION" {
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				request  DataCreateRelationWrapper
				response DataCreateRelationWrapper
				previous DataCreateRelationWrapper
			)

			err := json.Unmarshal([]byte(cast.ToString(v.Request)), &request)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Response)), &response)
			if err != nil {
				continue
			}

			err = json.Unmarshal([]byte(cast.ToString(v.Previous)), &previous)
			if err != nil {
				continue
			}

			request.Data.ProjectId = resourceEnvId
			request.Data.EnvId = cast.ToString(environmentId)
			response.Data.ProjectId = resourceEnvId
			response.Data.EnvId = cast.ToString(environmentId)
			previous.Data.ProjectId = resourceEnvId
			previous.Data.EnvId = cast.ToString(environmentId)
			logReq.TableSlug = request.Data.RelationTableSlug

			switch actionType {
			case "CREATE":
				logReq.Previous = response.Data
				_, err := services.GetBuilderServiceByType(nodeType).Field().Delete(
					context.Background(),
					&obs.FieldPrimaryKey{
						Id:        response.Data.Id,
						ProjectId: resourceEnvId,
						EnvId:     cast.ToString(environmentId),
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				ids = append(ids, v.Id)
			case "DELETE":
				createRelation, err := services.GetBuilderServiceByType(nodeType).Relation().Create(
					context.Background(),
					previous.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Request = request.Data
				logReq.Current = createRelation
				logReq.Response = createRelation
				ids = append(ids, v.Id)
			}
		} else if actionSource == "MENU" {
			defer func() {
				go h.versionHistory(c, logReq)
			}()

			var (
				current       DataUpdateMenuWrapper
				previous      DataUpdateMenuWrapper
				createRequest DataCreateMenuWrapper
				request       DataUpdateMenuWrapper
				response      DataUpdateMenuWrapper
			)

			if cast.ToString(v.Current) != "" {
				err = json.Unmarshal([]byte(cast.ToString(v.Current)), &current)
				if err != nil {
					fmt.Println("Unamrshal error 1")
					continue
				}
				current.Data.ProjectId = resourceEnvId
				current.Data.EnvId = environmentId
			}

			if cast.ToString(v.Previous) != "" {
				err = json.Unmarshal([]byte(cast.ToString(v.Previous)), &previous)
				if err != nil {
					fmt.Println("Unamrshal error 2")
					continue
				}
				previous.Data.ProjectId = resourceEnvId
				previous.Data.EnvId = environmentId
			}

			if cast.ToString(v.Previous) != "" {
				err = json.Unmarshal([]byte(cast.ToString(v.Previous)), &createRequest)
				if err != nil {
					fmt.Println("Unamrshal error 3")
					continue
				}
				createRequest.Data.ProjectId = resourceEnvId
				createRequest.Data.EnvId = environmentId
			}

			if cast.ToString(v.Request) != "" {
				err = json.Unmarshal([]byte(cast.ToString(v.Request)), &request)
				if err != nil {
					fmt.Println("Unamrshal error 4")
					continue
				}
				request.Data.ProjectId = resourceEnvId
				request.Data.EnvId = environmentId
			}

			if cast.ToString(v.Response) != "" {
				err = json.Unmarshal([]byte(cast.ToString(v.Response)), &response)
				if err != nil {
					fmt.Println("Unamrshal error 5")
					continue
				}
				response.Data.ProjectId = resourceEnvId
				response.Data.EnvId = environmentId
			}

			switch actionType {
			case "CREATE":
				logReq.ActionType = "DELETE MENU"
				logReq.Previous = current.Data
				_, err = services.GetBuilderServiceByType(nodeType).Menu().Delete(
					context.Background(),
					&obs.MenuPrimaryKey{
						Id:        current.Data.Id,
						ProjectId: resourceEnvId,
						EnvId:     cast.ToString(environmentId),
					},
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				ids = append(ids, v.Id)
			case "UPDATE":
				logReq.Request = previous.Data
				logReq.Previous = current.Data
				updateMenu, err := services.GetBuilderServiceByType(nodeType).Menu().Update(
					context.Background(),
					previous.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					return
				}
				logReq.Current = updateMenu
				logReq.Response = updateMenu
				ids = append(ids, v.Id)
			case "DELETE":
				logReq.ActionType = "CREATE MENU"
				createMenu, err := services.GetBuilderServiceByType(nodeType).Menu().Create(
					context.Background(),
					createRequest.Data,
				)
				if err != nil {
					logReq.Response = err.Error()
					continue
				}
				logReq.Request = createRequest.Data
				logReq.Response = createMenu
				logReq.Current = createMenu
				ids = append(ids, v.Id)
			}
		}
	}

	resp.Ids = ids

	h.handleResponse(c, status_http.OK, resp)
}
