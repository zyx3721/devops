package handler

import (
	"net/http"

	"devops/internal/service/logs"
	"devops/pkg/dto"
	"devops/pkg/response"

	"github.com/gin-gonic/gin"
)

// CompareHandler 日志对比处理器
type CompareHandler struct {
	compareService *logs.CompareService
}

// NewCompareHandler 创建对比处理器
func NewCompareHandler(compareService *logs.CompareService) *CompareHandler {
	return &CompareHandler{
		compareService: compareService,
	}
}

// CompareLogs 对比日志
// @Summary 对比日志
// @Tags 日志中心
// @Accept json
// @Produce json
// @Param body body dto.LogCompareRequest true "对比请求"
// @Success 200 {object} response.Response{data=dto.LogCompareResponse}
// @Router /api/v1/logs/compare [post]
func (h *CompareHandler) CompareLogs(c *gin.Context) {
	var req dto.LogCompareRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.compareService.Compare(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, result)
}
