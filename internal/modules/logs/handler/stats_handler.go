package handler

import (
	"net/http"

	"devops/internal/service/logs"
	"devops/pkg/dto"
	"devops/pkg/response"

	"github.com/gin-gonic/gin"
)

// StatsHandler 日志统计处理器
type StatsHandler struct {
	statsService *logs.StatsService
}

// NewStatsHandler 创建统计处理器
func NewStatsHandler(statsService *logs.StatsService) *StatsHandler {
	return &StatsHandler{
		statsService: statsService,
	}
}

// GetStats 获取日志统计
// @Summary 获取日志统计
// @Tags 日志中心
// @Param cluster_id query int true "集群ID"
// @Param namespace query string true "命名空间"
// @Param pod_name query string false "Pod名称"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Param interval query string false "时间间隔(hour/day)"
// @Success 200 {object} response.Response{data=dto.LogStatsResponse}
// @Router /api/v1/logs/stats [get]
func (h *StatsHandler) GetStats(c *gin.Context) {
	var req dto.LogStatsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	stats, err := h.statsService.GetStats(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, stats)
}
