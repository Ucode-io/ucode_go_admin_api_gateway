package v1

import (
	"ucode/ucode_go_api_gateway/api/status_http"
	"ucode/ucode_go_api_gateway/config"
	pb "ucode/ucode_go_api_gateway/genproto/company_service"
	pbo "ucode/ucode_go_api_gateway/genproto/new_object_builder_service"
	"ucode/ucode_go_api_gateway/pkg/util"
	"ucode/ucode_go_api_gateway/services"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
)

// ==================== Helper ====================

func (h *HandlerV1) getAiChatServices(c *gin.Context) (services.ServiceManagerI, string, error) {
	projectId, ok := c.Get("project_id")
	if !ok || !util.IsValidUUID(projectId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, config.ErrProjectIdValid)
		return nil, "", config.ErrProjectIdValid
	}

	environmentId, ok := c.Get("environment_id")
	if !ok || !util.IsValidUUID(environmentId.(string)) {
		h.HandleResponse(c, status_http.InvalidArgument, config.ErrEnvironmentIdValid)
		return nil, "", config.ErrEnvironmentIdValid
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
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return nil, "", err
	}

	if resource.ResourceType != pb.ResourceType_POSTGRESQL {
		h.HandleResponse(c, status_http.InvalidArgument, "resource type not supported")
		return nil, "", config.ErrProjectIdValid
	}

	service, err := h.GetProjectSrvc(c.Request.Context(), projectId.(string), resource.NodeType)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return nil, "", err
	}

	return service, resource.ResourceEnvironmentId, nil
}

// ==================== Chat Endpoints ====================

func (h *HandlerV1) CreateAiChat(c *gin.Context) {
	var (
		request pbo.CreateChatRequest
		project *pbo.McpProject
	)

	if err := c.ShouldBindJSON(&request); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	if len(request.GetProjectId()) == 0 {
		if request.GetTitle() == "" && request.GetDescription() == "" {
			h.HandleResponse(c, status_http.InvalidArgument, "project_id or project_name is required")
			return
		}

		if request.GetTitle() == "" {
			request.Title = request.GetDescription()
		}

		project, err = service.GoObjectBuilderService().McpProject().CreateMcpProject(
			c.Request.Context(),
			&pbo.CreateMcpProjectReqeust{
				ResourceEnvId: resourceEnvId,
				Title:         request.GetTitle(),
				Description:   request.GetDescription(),
			},
		)
		if err != nil {
			h.HandleResponse(c, status_http.GRPCError, err.Error())
			return
		}

		request.ProjectId = project.Id
	}

	request.ResourceEnvId = resourceEnvId

	response, err := service.GoObjectBuilderService().AiChat().CreateChat(
		c.Request.Context(), &request,
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.Created, response)
}

func (h *HandlerV1) GetAllChats(c *gin.Context) {
	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	var (
		title          = c.Query("title")
		model          = c.Query("model")
		projectId      = c.Query("project_id")
		orderBy        = c.Query("order_by")
		orderDirection = c.Query("order_direction")
		limit          = cast.ToInt32(c.DefaultQuery("limit", "20"))
		offset         = cast.ToInt32(c.DefaultQuery("offset", "0"))
	)

	response, err := service.GoObjectBuilderService().AiChat().GetAllChats(
		c.Request.Context(),
		&pbo.GetAllChatsRequest{
			ResourceEnvId:  resourceEnvId,
			Title:          title,
			Model:          model,
			ProjectId:      projectId,
			OrderBy:        orderBy,
			OrderDirection: orderDirection,
			Limit:          limit,
			Offset:         offset,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) GetProjectChat(c *gin.Context) {
	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	var (
		withMessages = c.DefaultQuery("with_messages", "true") == "true"
		projectId    = c.Param("project-id")
	)

	chat, err := service.GoObjectBuilderService().AiChat().GetChatByProjectId(
		c.Request.Context(),
		&pbo.ChatByProjectIdRequest{
			ResourceEnvId: resourceEnvId,
			ProjectId:     projectId,
			WithMessages:  withMessages,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return

	}

	h.HandleResponse(c, status_http.OK, chat)
}

func (h *HandlerV1) UpdateAiChat(c *gin.Context) {
	var (
		request pbo.UpdateChatRequest
		chatId  = c.Param("chat-id")
	)

	if err := c.ShouldBindJSON(&request); err != nil {
		h.HandleResponse(c, status_http.BadRequest, err.Error())
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	request.ResourceEnvId = resourceEnvId
	request.Id = chatId

	response, err := service.GoObjectBuilderService().AiChat().UpdateChat(
		c.Request.Context(), &request,
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) DeleteAiChat(c *gin.Context) {
	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	var chatId = c.Param("chat-id")

	response, err := service.GoObjectBuilderService().AiChat().DeleteChat(
		c.Request.Context(),
		&pbo.ChatPrimaryKey{
			ResourceEnvId: resourceEnvId,
			Id:            chatId,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

// ==================== Message Endpoints ====================

func (h *HandlerV1) GetAiChatMessages(c *gin.Context) {
	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	var (
		chatId = c.Param("chat-id")
		limit  = cast.ToInt32(c.DefaultQuery("limit", "50"))
		offset = cast.ToInt32(c.DefaultQuery("offset", "0"))
	)

	response, err := service.GoObjectBuilderService().AiChat().GetMessages(
		c.Request.Context(),
		&pbo.GetMessagesRequest{
			ResourceEnvId: resourceEnvId,
			ChatId:        chatId,
			Limit:         limit,
			Offset:        offset,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}

func (h *HandlerV1) DeleteAiChatMessage(c *gin.Context) {
	var messageId = c.Param("message_id")
	if !util.IsValidUUID(messageId) {
		h.HandleResponse(c, status_http.InvalidArgument, "invalid message_id")
		return
	}

	service, resourceEnvId, err := h.getAiChatServices(c)
	if err != nil {
		return
	}

	response, err := service.GoObjectBuilderService().AiChat().DeleteMessage(
		c.Request.Context(),
		&pbo.MessagePrimaryKey{
			ResourceEnvId: resourceEnvId,
			Id:            messageId,
		},
	)
	if err != nil {
		h.HandleResponse(c, status_http.GRPCError, err.Error())
		return
	}

	h.HandleResponse(c, status_http.OK, response)
}
