package handler

import (
	"net/http"
	"strconv"

	"devops/internal/service/logs"
	"devops/pkg/dto"
	"devops/pkg/response"

	"github.com/gin-gonic/gin"
)

// AlertHandler 告警处理器
type AlertHandler struct {
	alertService *logs.AlertService
}

// NewAlertHandler 创建告警处理器
func NewAlertHandler(alertService *logs.AlertService) *AlertHandler {
	return &AlertHandler{
		alertService: alertService,
	}
}

// ListAlertRules 获取告警规则列表
// @Summary 获取告警规则列表
// @Tags 日志告警
// @Param cluster_id query int false "集群ID"
// @Param namespace query string false "命名空间"
// @Success 200 {object} response.Response{data=[]dto.LogAlertRuleResponse}
// @Router /api/v1/logs/alert-rules [get]
func (h *AlertHandler) ListAlertRules(c *gin.Context) {
	clusterID, _ := strconv.ParseInt(c.Query("cluster_id"), 10, 64)
	namespace := c.Query("namespace")

	rules, err := h.alertService.ListRules(clusterID, namespace)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, rules)
}

// CreateAlertRule 创建告警规则
// @Summary 创建告警规则
// @Tags 日志告警
// @Accept json
// @Produce json
// @Param body body dto.LogAlertRuleRequest true "告警规则"
// @Success 200 {object} response.Response{data=dto.LogAlertRuleResponse}
// @Router /api/v1/logs/alert-rules [post]
func (h *AlertHandler) CreateAlertRule(c *gin.Context) {
	var req dto.LogAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	userID := c.GetInt64("user_id")
	rule, err := h.alertService.CreateRule(userID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, rule)
}

// UpdateAlertRule 更新告警规则
// @Summary 更新告警规则
// @Tags 日志告警
// @Accept json
// @Produce json
// @Param id path int true "规则ID"
// @Param body body dto.LogAlertRuleRequest true "告警规则"
// @Success 200 {object} response.Response{data=dto.LogAlertRuleResponse}
// @Router /api/v1/logs/alert-rules/{id} [put]
func (h *AlertHandler) UpdateAlertRule(c *gin.Context) {
	ruleID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	var req dto.LogAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	rule, err := h.alertService.UpdateRule(ruleID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, rule)
}

// DeleteAlertRule 删除告警规则
// @Summary 删除告警规则
// @Tags 日志告警
// @Param id path int true "规则ID"
// @Success 200 {object} response.Response
// @Router /api/v1/logs/alert-rules/{id} [delete]
func (h *AlertHandler) DeleteAlertRule(c *gin.Context) {
	ruleID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	if err := h.alertService.DeleteRule(ruleID); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// ToggleAlertRule 切换告警规则状态
// @Summary 切换告警规则状态
// @Tags 日志告警
// @Param id path int true "规则ID"
// @Success 200 {object} response.Response{data=map[string]bool}
// @Router /api/v1/logs/alert-rules/{id}/toggle [post]
func (h *AlertHandler) ToggleAlertRule(c *gin.Context) {
	ruleID, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	enabled, err := h.alertService.ToggleRule(ruleID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{"enabled": enabled})
}

// ListAlertHistory 获取告警历史
// @Summary 获取告警历史
// @Tags 日志告警
// @Param rule_id query int false "规则ID"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} response.Response{data=[]dto.LogAlertHistoryResponse}
// @Router /api/v1/logs/alert-history [get]
func (h *AlertHandler) ListAlertHistory(c *gin.Context) {
	ruleID, _ := strconv.ParseInt(c.Query("rule_id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	histories, total, err := h.alertService.ListHistory(ruleID, page, pageSize)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, gin.H{
		"items": histories,
		"total": total,
		"page":  page,
	})
}
