package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/service/kubernetes"
	"devops/internal/service/traffic"
)

// TrafficTestHandler 流量治理测试处理器
type TrafficTestHandler struct {
	db           *gorm.DB
	ruleTester   *traffic.RuleTester
	istioService *kubernetes.IstioService
}

// NewTrafficTestHandler 创建流量治理测试处理器
func NewTrafficTestHandler(db *gorm.DB, istioService *kubernetes.IstioService) *TrafficTestHandler {
	return &TrafficTestHandler{
		db:           db,
		ruleTester:   traffic.NewRuleTester(db, istioService),
		istioService: istioService,
	}
}

// RegisterRoutes 注册路由
func (h *TrafficTestHandler) RegisterRoutes(r *gin.RouterGroup) {
	g := r.Group("/applications/:id/traffic/test")
	{
		g.POST("/ratelimit", h.TestRateLimitRule)
		g.POST("/circuitbreaker", h.TestCircuitBreakerRule)
		g.POST("/routing", h.TestRoutingRule)
		g.POST("/load", h.SimulateLoad)
		g.GET("/results", h.GetTestResults)
	}
}

// TestRateLimitRule 测试限流规则
// @Summary 测试限流规则
// @Tags 流量治理测试
// @Param id path int true "应用ID"
// @Param body body traffic.TestRequest true "测试请求"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/test/ratelimit [post]
func (h *TrafficTestHandler) TestRateLimitRule(c *gin.Context) {
	appID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req traffic.TestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}
	req.AppID = uint(appID)
	req.RuleType = "ratelimit"

	result, err := h.ruleTester.TestRateLimitRule(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "测试失败: " + err.Error()})
		return
	}

	// 保存测试结果
	h.ruleTester.SaveTestResult(c.Request.Context(), uint(appID), result)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// TestCircuitBreakerRule 测试熔断规则
// @Summary 测试熔断规则
// @Tags 流量治理测试
// @Param id path int true "应用ID"
// @Param body body traffic.TestRequest true "测试请求"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/test/circuitbreaker [post]
func (h *TrafficTestHandler) TestCircuitBreakerRule(c *gin.Context) {
	appID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req traffic.TestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}
	req.AppID = uint(appID)
	req.RuleType = "circuitbreaker"

	result, err := h.ruleTester.TestCircuitBreakerRule(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "测试失败: " + err.Error()})
		return
	}

	// 保存测试结果
	h.ruleTester.SaveTestResult(c.Request.Context(), uint(appID), result)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// TestRoutingRule 测试路由规则
// @Summary 测试路由规则
// @Tags 流量治理测试
// @Param id path int true "应用ID"
// @Param body body traffic.TestRequest true "测试请求"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/test/routing [post]
func (h *TrafficTestHandler) TestRoutingRule(c *gin.Context) {
	appID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req traffic.TestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}
	req.AppID = uint(appID)
	req.RuleType = "routing"

	result, err := h.ruleTester.TestRoutingRule(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "测试失败: " + err.Error()})
		return
	}

	// 保存测试结果
	h.ruleTester.SaveTestResult(c.Request.Context(), uint(appID), result)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// SimulateLoad 模拟负载测试
// @Summary 模拟负载测试
// @Tags 流量治理测试
// @Param id path int true "应用ID"
// @Param body body traffic.TestRequest true "测试请求"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/test/load [post]
func (h *TrafficTestHandler) SimulateLoad(c *gin.Context) {
	appID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req traffic.TestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}
	req.AppID = uint(appID)
	req.RuleType = "load_test"

	result, err := h.ruleTester.SimulateLoad(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "测试失败: " + err.Error()})
		return
	}

	// 保存测试结果
	h.ruleTester.SaveTestResult(c.Request.Context(), uint(appID), result)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": result})
}

// GetTestResults 获取测试结果历史
// @Summary 获取测试结果历史
// @Tags 流量治理测试
// @Param id path int true "应用ID"
// @Param rule_type query string false "规则类型"
// @Success 200 {object} gin.H
// @Router /applications/{id}/traffic/test/results [get]
func (h *TrafficTestHandler) GetTestResults(c *gin.Context) {
	appID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	ruleType := c.Query("rule_type")

	query := h.db.Table("traffic_operation_logs").
		Where("app_id = ? AND operation = ?", appID, "test")

	if ruleType != "" {
		query = query.Where("rule_type = ?", ruleType)
	}

	var results []map[string]any
	query.Order("created_at DESC").Limit(50).Find(&results)

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"items": results}})
}
