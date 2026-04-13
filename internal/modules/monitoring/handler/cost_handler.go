package handler

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/service/cost"
	"devops/internal/service/kubernetes"
	"devops/pkg/dto"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
	"devops/pkg/response"
)

func init() {
	ioc.Api.RegisterContainer("CostHandler", &CostApiHandler{})
}

// CostApiHandler IOC容器注册的处理器
type CostApiHandler struct {
	handler *CostHandler
}

func (h *CostApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	svc := cost.NewCostService(db)
	h.handler = NewCostHandler(svc)

	// 启动成本调度器
	clientMgr := kubernetes.NewK8sClientManager(db)
	scheduler := cost.NewCostScheduler(db, clientMgr)
	scheduler.Start()

	root := cfg.Application.GinRootRouter().Group("cost")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *CostApiHandler) Register(r gin.IRouter) {
	// 成本概览
	r.GET("/overview", h.handler.GetOverview)
	r.GET("/trend", h.handler.GetTrend)
	r.GET("/distribution", h.handler.GetDistribution)
	r.GET("/usage", h.handler.GetResourceUsage)

	// 高级分析
	r.GET("/forecast", h.handler.GetForecast)
	r.GET("/waste", h.handler.GetWasteDetection)
	r.GET("/health-score", h.handler.GetHealthScore)
	r.GET("/comparison", h.handler.GetComparison)

	// 多维度分析
	r.GET("/app", h.handler.GetAppCost)
	r.GET("/team", h.handler.GetTeamCost)
	r.GET("/node", h.handler.GetNodeCost)
	r.GET("/pvc", h.handler.GetPVCCost)
	r.GET("/env", h.handler.GetEnvCost)
	r.GET("/allocation", h.handler.GetCostAllocation)

	// 优化建议
	r.GET("/suggestions", h.handler.GetSuggestions)
	r.POST("/suggestions/:id/apply", middleware.RequireAdmin(), h.handler.ApplySuggestion)
	r.POST("/suggestions/:id/ignore", middleware.RequireAdmin(), h.handler.IgnoreSuggestion)

	// 告警管理
	r.GET("/alerts", h.handler.GetAlerts)
	r.POST("/alerts/:id/acknowledge", h.handler.AcknowledgeAlert)

	// 预算管理
	r.GET("/budgets", h.handler.GetBudgets)
	r.POST("/budgets", middleware.RequireAdmin(), h.handler.SaveBudget)

	// 报表导出
	r.GET("/export", h.handler.ExportReport)

	// 成本配置 - 需要管理员权限
	r.GET("/config", h.handler.GetConfig)
	r.POST("/config", middleware.RequireAdmin(), h.handler.SaveConfig)
}

// CostHandler 成本管理处理器
type CostHandler struct {
	svc *cost.CostService
}

// NewCostHandler 创建成本管理处理器
func NewCostHandler(svc *cost.CostService) *CostHandler {
	return &CostHandler{svc: svc}
}

// GetOverview 获取成本概览
// @Summary 获取成本概览
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param days query int false "天数，默认30"
// @Success 200 {object} dto.CostOverviewResponse
// @Router /cost/overview [get]
func (h *CostHandler) GetOverview(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	days, _ := strconv.Atoi(c.Query("days"))
	if days <= 0 {
		days = 30
	}

	result, err := h.svc.GetOverview(c.Request.Context(), uint(clusterID), days)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetTrend 获取成本趋势
// @Summary 获取成本趋势
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param days query int false "天数，默认30"
// @Success 200 {object} dto.CostTrendResponse
// @Router /cost/trend [get]
func (h *CostHandler) GetTrend(c *gin.Context) {
	var req dto.CostTrendRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetTrend(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetDistribution 获取成本分布
// @Summary 获取成本分布
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param dimension query string false "维度：namespace/app/team/resource_type"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Param top_n query int false "返回数量，默认10"
// @Success 200 {object} dto.CostDistributionResponse
// @Router /cost/distribution [get]
func (h *CostHandler) GetDistribution(c *gin.Context) {
	var req dto.CostDistributionRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetDistribution(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetResourceUsage 获取资源利用率
// @Summary 获取资源利用率
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param namespace query string false "命名空间"
// @Param top_n query int false "返回数量，默认20"
// @Success 200 {object} dto.ResourceUsageResponse
// @Router /cost/usage [get]
func (h *CostHandler) GetResourceUsage(c *gin.Context) {
	var req dto.ResourceUsageRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetResourceUsage(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetSuggestions 获取成本优化建议
// @Summary 获取成本优化建议
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param status query string false "状态：pending/applied/ignored"
// @Success 200 {object} dto.CostSuggestionListResponse
// @Router /cost/suggestions [get]
func (h *CostHandler) GetSuggestions(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	status := c.Query("status")

	result, err := h.svc.GetSuggestions(c.Request.Context(), uint(clusterID), status)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// ApplySuggestion 应用优化建议
// @Summary 应用优化建议
// @Tags 成本管理
// @Param id path int true "建议ID"
// @Success 200 {object} response.Response
// @Router /cost/suggestions/{id}/apply [post]
func (h *CostHandler) ApplySuggestion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的建议ID")
		return
	}

	userID := c.GetUint("userID")
	if err := h.svc.ApplySuggestion(c.Request.Context(), uint(id), userID); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "应用成功", nil)
}

// IgnoreSuggestion 忽略优化建议
// @Summary 忽略优化建议
// @Tags 成本管理
// @Param id path int true "建议ID"
// @Success 200 {object} response.Response
// @Router /cost/suggestions/{id}/ignore [post]
func (h *CostHandler) IgnoreSuggestion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的建议ID")
		return
	}

	userID := c.GetUint("userID")
	if err := h.svc.IgnoreSuggestion(c.Request.Context(), uint(id), userID); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "已忽略", nil)
}

// GetConfig 获取成本配置
// @Summary 获取成本配置
// @Tags 成本管理
// @Param cluster_id query int true "集群ID"
// @Success 200 {object} dto.CostConfigResponse
// @Router /cost/config [get]
func (h *CostHandler) GetConfig(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	if err != nil || clusterID == 0 {
		response.BadRequest(c, "集群ID不能为空")
		return
	}

	result, err := h.svc.GetConfig(c.Request.Context(), uint(clusterID))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// SaveConfig 保存成本配置
// @Summary 保存成本配置
// @Tags 成本管理
// @Param cluster_id query int true "集群ID"
// @Param body body dto.CostConfigRequest true "配置信息"
// @Success 200 {object} response.Response
// @Router /cost/config [post]
func (h *CostHandler) SaveConfig(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	if err != nil || clusterID == 0 {
		response.BadRequest(c, "集群ID不能为空")
		return
	}

	var req dto.CostConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	if err := h.svc.SaveConfig(c.Request.Context(), uint(clusterID), &req); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "保存成功", nil)
}

// GetForecast 获取成本预测
// @Summary 获取成本预测
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param days query int false "预测天数，默认30"
// @Success 200 {object} dto.CostForecastResponse
// @Router /cost/forecast [get]
func (h *CostHandler) GetForecast(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	days, _ := strconv.Atoi(c.Query("days"))
	if days <= 0 {
		days = 30
	}

	result, err := h.svc.GetForecast(c.Request.Context(), uint(clusterID), days)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetWasteDetection 获取资源浪费检测
// @Summary 获取资源浪费检测
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param days query int false "检测周期，默认7天"
// @Success 200 {object} dto.WasteDetectionResponse
// @Router /cost/waste [get]
func (h *CostHandler) GetWasteDetection(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	days, _ := strconv.Atoi(c.Query("days"))
	if days <= 0 {
		days = 7
	}

	result, err := h.svc.GetWasteDetection(c.Request.Context(), uint(clusterID), days)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetHealthScore 获取成本健康评分
// @Summary 获取成本健康评分
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Success 200 {object} dto.CostHealthScoreResponse
// @Router /cost/health-score [get]
func (h *CostHandler) GetHealthScore(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)

	result, err := h.svc.GetCostHealthScore(c.Request.Context(), uint(clusterID))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetBudgets 获取预算列表
// @Summary 获取预算列表
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Success 200 {object} dto.CostBudgetListResponse
// @Router /cost/budgets [get]
func (h *CostHandler) GetBudgets(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)

	result, err := h.svc.GetBudgetList(c.Request.Context(), uint(clusterID))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// SaveBudget 保存预算
// @Summary 保存预算
// @Tags 成本管理
// @Param body body dto.CostBudgetRequest true "预算信息"
// @Success 200 {object} response.Response
// @Router /cost/budgets [post]
func (h *CostHandler) SaveBudget(c *gin.Context) {
	var req dto.CostBudgetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数格式错误")
		return
	}

	if req.ClusterID == 0 {
		response.BadRequest(c, "集群ID不能为空")
		return
	}

	if err := h.svc.SaveBudget(c.Request.Context(), &req); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "保存成功", nil)
}

// GetComparison 获取成本对比分析
// @Summary 获取成本对比分析
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param period1_start query string true "周期1开始时间"
// @Param period1_end query string true "周期1结束时间"
// @Param period2_start query string true "周期2开始时间"
// @Param period2_end query string true "周期2结束时间"
// @Success 200 {object} dto.CostComparisonResponse
// @Router /cost/comparison [get]
func (h *CostHandler) GetComparison(c *gin.Context) {
	var req dto.CostComparisonRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetComparison(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetAlerts 获取成本告警列表
// @Summary 获取成本告警列表
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param status query string false "状态：active/acknowledged/resolved"
// @Success 200 {array} dto.CostAlertItem
// @Router /cost/alerts [get]
func (h *CostHandler) GetAlerts(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	status := c.Query("status")

	result, err := h.svc.GetAlerts(c.Request.Context(), uint(clusterID), status)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// AcknowledgeAlert 确认告警
// @Summary 确认告警
// @Tags 成本管理
// @Param id path int true "告警ID"
// @Success 200 {object} response.Response
// @Router /cost/alerts/{id}/acknowledge [post]
func (h *CostHandler) AcknowledgeAlert(c *gin.Context) {
	alertID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "告警ID无效")
		return
	}

	// 从上下文获取用户ID
	userID := uint(0)
	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uint)
	}

	if err := h.svc.AcknowledgeAlert(c.Request.Context(), uint(alertID), userID); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "确认成功", nil)
}

// ExportReport 导出成本报表
// @Summary 导出成本报表
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param start_time query string true "开始时间"
// @Param end_time query string true "结束时间"
// @Param report_type query string false "报表类型：overview/comparison"
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Router /cost/export [get]
func (h *CostHandler) ExportReport(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Query("cluster_id"), 10, 64)
	startTimeStr := c.Query("start_time")
	endTimeStr := c.Query("end_time")

	if startTimeStr == "" || endTimeStr == "" {
		response.BadRequest(c, "开始时间和结束时间不能为空")
		return
	}

	startTime, err := time.Parse("2006-01-02", startTimeStr)
	if err != nil {
		response.BadRequest(c, "开始时间格式错误")
		return
	}
	endTime, err := time.Parse("2006-01-02", endTimeStr)
	if err != nil {
		response.BadRequest(c, "结束时间格式错误")
		return
	}

	exporter := cost.NewCostExporter(h.svc.GetDB())
	buf, err := exporter.ExportOverviewReport(c.Request.Context(), uint(clusterID), startTime, endTime)
	if err != nil {
		response.FromError(c, err)
		return
	}

	filename := fmt.Sprintf("cost_report_%s_%s.xlsx", startTimeStr, endTimeStr)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buf.Bytes())
}

// GetAppCost 获取应用维度成本
// @Summary 获取应用维度成本
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Param top_n query int false "返回数量"
// @Success 200 {object} dto.AppCostResponse
// @Router /cost/app [get]
func (h *CostHandler) GetAppCost(c *gin.Context) {
	var req dto.AppCostRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetAppCost(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetTeamCost 获取团队维度成本
// @Summary 获取团队维度成本
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} dto.TeamCostResponse
// @Router /cost/team [get]
func (h *CostHandler) GetTeamCost(c *gin.Context) {
	var req dto.TeamCostRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetTeamCost(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetNodeCost 获取节点成本
// @Summary 获取节点成本
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Success 200 {object} dto.NodeCostResponse
// @Router /cost/node [get]
func (h *CostHandler) GetNodeCost(c *gin.Context) {
	var req dto.NodeCostRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetNodeCost(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetPVCCost 获取PVC存储成本
// @Summary 获取PVC存储成本
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param namespace query string false "命名空间"
// @Success 200 {object} dto.PVCCostResponse
// @Router /cost/pvc [get]
func (h *CostHandler) GetPVCCost(c *gin.Context) {
	var req dto.PVCCostRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetPVCCost(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetEnvCost 获取环境维度成本
// @Summary 获取环境维度成本
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {object} dto.EnvCostResponse
// @Router /cost/env [get]
func (h *CostHandler) GetEnvCost(c *gin.Context) {
	var req dto.EnvCostRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.svc.GetEnvCost(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetCostAllocation 获取成本分摊报表
// @Summary 获取成本分摊报表
// @Tags 成本管理
// @Param cluster_id query int false "集群ID"
// @Param start_time query string true "开始时间"
// @Param end_time query string true "结束时间"
// @Param group_by query string false "分组维度：team/namespace/app"
// @Param include_shared query bool false "是否分摊公共成本"
// @Success 200 {object} dto.CostAllocationReportResponse
// @Router /cost/allocation [get]
func (h *CostHandler) GetCostAllocation(c *gin.Context) {
	var req dto.CostAllocationReportRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if req.StartTime == "" || req.EndTime == "" {
		response.BadRequest(c, "开始时间和结束时间不能为空")
		return
	}

	result, err := h.svc.GetCostAllocationReport(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}
