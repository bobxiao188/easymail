package handler

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	domain "easymail/internal/domain/management"
	"easymail/internal/domain/shared"
	"easymail/internal/app/management"

	"github.com/gin-gonic/gin"
)

// ==================== Agent CRUD ====================

// ListPostfixAgentsHandler lists all Postfix agents.
func (h *Handler) ListPostfixAgentsHandler(c *gin.Context) {
	keyword := c.Query("keyword")
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("pageSize"), 20)

	agents, total, err := h.postfixService.ListAgents(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	jsonPage(c, agents, total, page, pageSize)
}

// GetPostfixAgentHandler gets a single Postfix agent.
func (h *Handler) GetPostfixAgentHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}
	agent, err := h.postfixService.GetAgent(c.Request.Context(), id)
	if err != nil {
		code := http.StatusInternalServerError
		if err == domain.ErrPostfixAgentNotFound {
			code = http.StatusNotFound
		}
		jsonError(c, code, err.Error())
		return
	}
	// Mask token in response
	safeAgent := *agent
	if safeAgent.Token != "" {
		safeAgent.Token = maskToken(safeAgent.Token)
	}
	jsonData(c, safeAgent)
}

// CreatePostfixAgentHandler creates a new Postfix agent.
func (h *Handler) CreatePostfixAgentHandler(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Host        string `json:"host" binding:"required"`
		Token       string `json:"token" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, formatBindingError(err))
		return
	}
	agent, err := h.postfixService.CreateAgent(c.Request.Context(), req.Name, req.Host, req.Token, req.Description)
	if err != nil {
		code := http.StatusInternalServerError
		if err == domain.ErrPostfixAgentExists || err == domain.ErrPostfixAgentTokenEmpty {
			code = http.StatusBadRequest
		}
		jsonError(c, code, err.Error())
		return
	}
	jsonData(c, agent)
}

// UpdatePostfixAgentHandler updates an existing Postfix agent.
func (h *Handler) UpdatePostfixAgentHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}
	var req struct {
		Name        string `json:"name" binding:"required"`
		Host        string `json:"host" binding:"required"`
		Token       string `json:"token"`
		Description string `json:"description"`
		Enabled     bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, formatBindingError(err))
		return
	}
	if err := h.postfixService.UpdateAgent(c.Request.Context(), id, req.Name, req.Host, req.Token, req.Description, req.Enabled); err != nil {
		code := http.StatusInternalServerError
		if err == domain.ErrPostfixAgentNotFound {
			code = http.StatusNotFound
		}
		jsonError(c, code, err.Error())
		return
	}
	jsonSuccess(c, "agent updated")
}

// DeletePostfixAgentHandler deletes a Postfix agent.
func (h *Handler) DeletePostfixAgentHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}
	if err := h.postfixService.DeleteAgent(c.Request.Context(), id); err != nil {
		code := http.StatusInternalServerError
		if err == domain.ErrPostfixAgentNotFound {
			code = http.StatusNotFound
		}
		jsonError(c, code, err.Error())
		return
	}
	jsonSuccess(c, "agent deleted")
}

// CheckPostfixAgentStatusHandler checks the live status of a Postfix agent.
func (h *Handler) CheckPostfixAgentStatusHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}
	status, err := h.postfixService.CheckAgentStatus(c.Request.Context(), id)
	if err != nil {
		jsonError(c, http.StatusServiceUnavailable, err.Error())
		return
	}
	jsonData(c, status)
}

// ==================== Global Settings ====================

// GetPostfixSettingsHandler returns global Postfix settings.
func (h *Handler) GetPostfixSettingsHandler(c *gin.Context) {
	settings, err := h.postfixService.GetSettings(c.Request.Context())
	if err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	jsonData(c, settings)
}

// UpdatePostfixSettingsHandler updates global Postfix settings.
func (h *Handler) UpdatePostfixSettingsHandler(c *gin.Context) {
	var req struct {
		EasyMailHost string `json:"easymailHost"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, formatBindingError(err))
		return
	}
	settings := &management.PostfixSettings{
		EasyMailHost: req.EasyMailHost,
	}
	if err := h.postfixService.UpdateSettings(c.Request.Context(), settings); err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	jsonSuccess(c, "settings updated")
}

// GetPostfixVariablesHandler returns available variables for parameter values.
func (h *Handler) GetPostfixVariablesHandler(c *gin.Context) {
	variables := h.postfixService.GetVariables(c.Request.Context())
	jsonData(c, variables)
}

// GetLocalIPsHandler returns all local IP addresses.
func (h *Handler) GetLocalIPsHandler(c *gin.Context) {
	ips := h.postfixService.GetLocalIPs(c.Request.Context())
	jsonData(c, ips)
}

// ==================== Config Parameter CRUD ====================

// ListPostfixConfigParamsHandler lists Postfix configuration parameters.
func (h *Handler) ListPostfixConfigParamsHandler(c *gin.Context) {
	keyword := c.Query("keyword")
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("pageSize"), 20)

	params, total, err := h.postfixService.ListConfigParams(c.Request.Context(), keyword, page, pageSize)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	jsonPage(c, params, total, page, pageSize)
}

// GetPostfixConfigParamHandler gets a single configuration parameter.
func (h *Handler) GetPostfixConfigParamHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid config id")
		return
	}
	param, err := h.postfixService.GetConfigParam(c.Request.Context(), id)
	if err != nil {
		code := http.StatusInternalServerError
		if err == domain.ErrPostfixConfigNotFound {
			code = http.StatusNotFound
		}
		jsonError(c, code, err.Error())
		return
	}
	jsonData(c, param)
}

// CreatePostfixConfigParamHandler creates a new user-defined config parameter.
func (h *Handler) CreatePostfixConfigParamHandler(c *gin.Context) {
	var req struct {
		ParamName   string `json:"paramName" binding:"required"`
		ParamValue  string `json:"paramValue" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, formatBindingError(err))
		return
	}
	param, err := h.postfixService.CreateConfigParam(c.Request.Context(), req.ParamName, req.ParamValue, req.Description)
	if err != nil {
		code := http.StatusInternalServerError
		if err == domain.ErrPostfixConfigDuplicate {
			code = http.StatusConflict
		}
		jsonError(c, code, err.Error())
		return
	}
	jsonData(c, param)
}

// UpdatePostfixConfigParamHandler updates a user-defined config parameter value.
func (h *Handler) UpdatePostfixConfigParamHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid config id")
		return
	}
	var req struct {
		ParamValue string `json:"paramValue" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, formatBindingError(err))
		return
	}
	if err := h.postfixService.UpdateConfigParam(c.Request.Context(), id, req.ParamValue); err != nil {
		code := http.StatusInternalServerError
		if err == domain.ErrPostfixConfigNotFound {
			code = http.StatusNotFound
		} else if err == domain.ErrPostfixConfigNotEditable {
			code = http.StatusForbidden
		}
		jsonError(c, code, err.Error())
		return
	}
	jsonSuccess(c, "config parameter updated")
}

// DeletePostfixConfigParamHandler deletes a user-defined config parameter.
func (h *Handler) DeletePostfixConfigParamHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid config id")
		return
	}
	if err := h.postfixService.DeleteConfigParam(c.Request.Context(), id); err != nil {
		code := http.StatusInternalServerError
		if err == domain.ErrPostfixConfigNotFound {
			code = http.StatusNotFound
		} else if err == domain.ErrPostfixConfigNotEditable {
			code = http.StatusForbidden
		}
		jsonError(c, code, err.Error())
		return
	}
	jsonSuccess(c, "config parameter deleted")
}

// ==================== Config Generation and Delivery ====================

// PreviewPostfixConfigHandler generates a preview of the rendered configuration.
func (h *Handler) PreviewPostfixConfigHandler(c *gin.Context) {
	preview, err := h.postfixService.GeneratePreview(c.Request.Context())
	if err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	jsonData(c, preview)
}

// PushPostfixConfigHandler pushes configuration to an agent (without applying).
func (h *Handler) PushPostfixConfigHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}
	if err := h.postfixService.PushConfig(c.Request.Context(), id); err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	jsonSuccess(c, "config pushed to agent")
}

// ApplyPostfixConfigHandler applies the pushed configuration on an agent.
func (h *Handler) ApplyPostfixConfigHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}
	if err := h.postfixService.ApplyConfig(c.Request.Context(), id); err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	jsonSuccess(c, "config applied on agent")
}

// RollbackPostfixConfigHandler rolls back configuration on an agent.
func (h *Handler) RollbackPostfixConfigHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}
	if err := h.postfixService.RollbackConfig(c.Request.Context(), id); err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	jsonSuccess(c, "config rolled back on agent")
}

// PushAndApplyPostfixConfigHandler pushes and immediately applies config on an agent.
func (h *Handler) PushAndApplyPostfixConfigHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}
	if err := h.postfixService.PushAndApply(c.Request.Context(), id); err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	jsonSuccess(c, "config pushed and applied on agent")
}

// ==================== Delivery Logs ====================

// ListPostfixDeliveryLogsHandler lists delivery logs for an agent.
func (h *Handler) ListPostfixDeliveryLogsHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}
	limit := parseInt(c.Query("limit"), 50)
	logs, err := h.postfixService.ListDeliveryLogs(c.Request.Context(), id, limit)
	if err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	jsonData(c, logs)
}

// ==================== Status Summary ====================

// GetPostfixConfigStatusHandler returns the overall config status summary.
func (h *Handler) GetPostfixConfigStatusHandler(c *gin.Context) {
	summary, err := h.postfixService.GetConfigStatusSummary(c.Request.Context())
	if err != nil {
		jsonError(c, http.StatusInternalServerError, err.Error())
		return
	}
	jsonData(c, summary)
}

// ==================== Queue Management ====================

// ListQueueHandler lists messages in the mail queue.
func (h *Handler) ListQueueHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}

	// Parse filter parameters
	filter := &domain.QueueFilter{
		Status:    c.Query("status"),
		Sender:    c.Query("sender"),
		Recipient: c.Query("recipient"),
		QueueID:   c.Query("queueId"),
		Page:      parseInt(c.Query("page"), 1),
		PageSize:  parseInt(c.Query("pageSize"), 50),
	}

	response, err := h.postfixService.ListQueue(c.Request.Context(), id, filter)
	if err != nil {
		jsonError(c, http.StatusServiceUnavailable, err.Error())
		return
	}
	jsonData(c, response)
}

// GetQueueStatsHandler returns queue statistics.
func (h *Handler) GetQueueStatsHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}

	stats, err := h.postfixService.GetQueueStats(c.Request.Context(), id)
	if err != nil {
		jsonError(c, http.StatusServiceUnavailable, err.Error())
		return
	}
	jsonData(c, stats)
}

// DeleteQueueMessagesHandler deletes specified messages from the queue.
func (h *Handler) DeleteQueueMessagesHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}

	var req struct {
		MessageIDs []string `json:"messageIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "messageIds is required")
		return
	}

	if err := h.postfixService.DeleteQueueMessages(c.Request.Context(), id, req.MessageIDs); err != nil {
		jsonError(c, http.StatusServiceUnavailable, err.Error())
		return
	}
	jsonSuccess(c, "messages deleted")
}

// ResendQueueMessagesHandler resends specified messages in the queue.
func (h *Handler) ResendQueueMessagesHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}

	var req struct {
		MessageIDs []string `json:"messageIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonError(c, http.StatusBadRequest, "messageIds is required")
		return
	}

	if err := h.postfixService.ResendQueueMessages(c.Request.Context(), id, req.MessageIDs); err != nil {
		jsonError(c, http.StatusServiceUnavailable, err.Error())
		return
	}
	jsonSuccess(c, "messages resent")
}

// FlushQueueHandler flushes the entire queue.
func (h *Handler) FlushQueueHandler(c *gin.Context) {
	id, err := shared.ParseGlobalID(c.Param("id"))
	if err != nil {
		jsonError(c, http.StatusBadRequest, "invalid agent id")
		return
	}

	if err := h.postfixService.FlushQueue(c.Request.Context(), id); err != nil {
		jsonError(c, http.StatusServiceUnavailable, err.Error())
		return
	}
	jsonSuccess(c, "queue flushed")
}

// ==================== Helpers ====================

func parseInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var v int
	if _, err := fmt.Sscanf(s, "%d", &v); err != nil || v < 1 {
		return defaultVal
	}
	return v
}

func maskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}

func jsonData(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "ok", "data": data})
}

func jsonPage(c *gin.Context, data interface{}, total int64, page, pageSize int) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "ok",
		"data":    data,
		"meta": gin.H{
			"page":       page,
			"pageSize":   pageSize,
			"total":      total,
			"totalPages": totalPages,
			"hasPrev":    page > 1,
			"hasNext":    page < totalPages,
		},
	})
}

func jsonSuccess(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": message})
}

func jsonError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"code": -1, "message": message})
}

// bindingFieldRegex matches gin validator error fields like "Key: 'Host' Error:..."
var bindingFieldRegex = regexp.MustCompile(`Key:\s*'(\w+)'`)

// formatBindingError aggregates gin validator errors into a user-friendly message.
// For example: "Key: 'Host' Error:Field validation for 'Host' failed on the 'required' tag"
// becomes: "Host 不能为空"
func formatBindingError(err error) string {
	msg := err.Error()
	fields := bindingFieldRegex.FindAllStringSubmatch(msg, -1)
	if len(fields) == 0 {
		return msg
	}

	var parts []string
	for _, m := range fields {
		field := m[1]
		// Map field names to Chinese labels
		switch field {
		case "Name":
			parts = append(parts, "名称不能为空")
		case "Host":
			parts = append(parts, "主机地址不能为空")
		case "Token":
			parts = append(parts, "令牌不能为空")
		case "ParamName":
			parts = append(parts, "参数名不能为空")
		case "ParamValue":
			parts = append(parts, "参数值不能为空")
		default:
			parts = append(parts, fmt.Sprintf("%s 不能为空", field))
		}
	}
	return strings.Join(parts, "，")
}