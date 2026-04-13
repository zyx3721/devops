package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"devops/internal/models"
	"devops/internal/service/approval"
	"devops/pkg/middleware"
	"devops/pkg/response"
)

// ApprovalChainHandler 审批链处理器
type ApprovalChainHandler struct {
	chainService    *approval.ChainService
	instanceService *approval.InstanceService
	nodeExecutor    *approval.NodeExecutor
}

// NewApprovalChainHandler 创建审批链处理器
func NewApprovalChainHandler(
	chainService *approval.ChainService,
	instanceService *approval.InstanceService,
	nodeExecutor *approval.NodeExecutor,
) *ApprovalChainHandler {
	return &ApprovalChainHandler{
		chainService:    chainService,
		instanceService: instanceService,
		nodeExecutor:    nodeExecutor,
	}
}

// ============================================================================
// 审批链管理
// ============================================================================

// ListChains 获取审批链列表
// @Summary 获取审批链列表
// @Tags 审批链管理
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param app_id query int false "应用ID"
// @Param env query string false "环境"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/chains [get]
func (h *ApprovalChainHandler) ListChains(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	var appID *uint
	if appIDStr := c.Query("app_id"); appIDStr != "" {
		id, err := strconv.ParseUint(appIDStr, 10, 32)
		if err == nil {
			appIDUint := uint(id)
			appID = &appIDUint
		}
	}
	env := c.Query("env")

	chains, total, err := h.chainService.List(c.Request.Context(), page, pageSize, appID, env)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取审批链列表失败")
		return
	}

	response.Success(c, gin.H{
		"list":  chains,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// CreateChain 创建审批链
// @Summary 创建审批链
// @Tags 审批链管理
// @Accept json
// @Produce json
// @Param body body models.ApprovalChain true "审批链信息"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/chains [post]
func (h *ApprovalChainHandler) CreateChain(c *gin.Context) {
	var chain models.ApprovalChain
	if err := c.ShouldBindJSON(&chain); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if chain.Name == "" {
		response.Error(c, http.StatusBadRequest, "审批链名称不能为空")
		return
	}

	if err := h.chainService.Create(c.Request.Context(), &chain); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, chain)
}

// GetChain 获取审批链详情
// @Summary 获取审批链详情
// @Tags 审批链管理
// @Produce json
// @Param id path int true "审批链ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/chains/{id} [get]
func (h *ApprovalChainHandler) GetChain(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	chain, err := h.chainService.GetWithNodes(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, chain)
}

// UpdateChain 更新审批链
// @Summary 更新审批链
// @Tags 审批链管理
// @Accept json
// @Produce json
// @Param id path int true "审批链ID"
// @Param body body models.ApprovalChain true "审批链信息"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/chains/{id} [put]
func (h *ApprovalChainHandler) UpdateChain(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	var chain models.ApprovalChain
	if err := c.ShouldBindJSON(&chain); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	chain.ID = uint(id)
	if err := h.chainService.Update(c.Request.Context(), &chain); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, chain)
}

// DeleteChain 删除审批链
// @Summary 删除审批链
// @Tags 审批链管理
// @Param id path int true "审批链ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/chains/{id} [delete]
func (h *ApprovalChainHandler) DeleteChain(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	if err := h.chainService.Delete(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// ============================================================================
// 审批节点管理
// ============================================================================

// AddNode 添加审批节点
// @Summary 添加审批节点
// @Tags 审批链管理
// @Accept json
// @Produce json
// @Param id path int true "审批链ID"
// @Param body body models.ApprovalNode true "节点信息"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/chains/{id}/nodes [post]
func (h *ApprovalChainHandler) AddNode(c *gin.Context) {
	chainID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的审批链ID")
		return
	}

	var node models.ApprovalNode
	if err := c.ShouldBindJSON(&node); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	node.ChainID = uint(chainID)
	if node.Name == "" {
		response.Error(c, http.StatusBadRequest, "节点名称不能为空")
		return
	}
	if node.Approvers == "" {
		response.Error(c, http.StatusBadRequest, "审批人不能为空")
		return
	}

	if err := h.chainService.AddNode(c.Request.Context(), &node); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, node)
}

// UpdateNode 更新审批节点
// @Summary 更新审批节点
// @Tags 审批链管理
// @Accept json
// @Produce json
// @Param id path int true "审批链ID"
// @Param nodeId path int true "节点ID"
// @Param body body models.ApprovalNode true "节点信息"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/chains/{id}/nodes/{nodeId} [put]
func (h *ApprovalChainHandler) UpdateNode(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("nodeId"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的节点ID")
		return
	}

	var node models.ApprovalNode
	if err := c.ShouldBindJSON(&node); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	node.ID = uint(nodeID)
	if err := h.chainService.UpdateNode(c.Request.Context(), &node); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, node)
}

// DeleteNode 删除审批节点
// @Summary 删除审批节点
// @Tags 审批链管理
// @Param id path int true "审批链ID"
// @Param nodeId path int true "节点ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/chains/{id}/nodes/{nodeId} [delete]
func (h *ApprovalChainHandler) DeleteNode(c *gin.Context) {
	nodeID, err := strconv.ParseUint(c.Param("nodeId"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的节点ID")
		return
	}

	if err := h.chainService.DeleteNode(c.Request.Context(), uint(nodeID)); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// ReorderNodes 调整节点顺序
// @Summary 调整节点顺序
// @Tags 审批链管理
// @Accept json
// @Produce json
// @Param id path int true "审批链ID"
// @Param body body object true "节点ID顺序"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/chains/{id}/nodes/reorder [put]
func (h *ApprovalChainHandler) ReorderNodes(c *gin.Context) {
	chainID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的审批链ID")
		return
	}

	var req struct {
		NodeIDs []uint `json:"node_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.chainService.ReorderNodes(c.Request.Context(), uint(chainID), req.NodeIDs); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// ============================================================================
// 审批实例管理
// ============================================================================

// ListInstances 获取审批实例列表
// @Summary 获取审批实例列表
// @Tags 审批实例
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param status query string false "状态"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/instances [get]
func (h *ApprovalChainHandler) ListInstances(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	status := c.Query("status")

	instances, total, err := h.instanceService.List(c.Request.Context(), page, pageSize, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取审批实例列表失败")
		return
	}

	response.Success(c, gin.H{
		"list":  instances,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// GetInstance 获取审批实例详情
// @Summary 获取审批实例详情
// @Tags 审批实例
// @Produce json
// @Param id path int true "实例ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/instances/{id} [get]
func (h *ApprovalChainHandler) GetInstance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	instance, err := h.instanceService.GetWithNodeInstances(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, instance)
}

// CancelInstance 取消审批实例
// @Summary 取消审批实例
// @Tags 审批实例
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Param body body object true "取消原因"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/instances/{id}/cancel [post]
func (h *ApprovalChainHandler) CancelInstance(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)

	if err := h.instanceService.Cancel(c.Request.Context(), uint(id), req.Reason); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// ============================================================================
// 审批操作
// ============================================================================

// GetPendingApprovals 获取待审批列表
// @Summary 获取待审批列表
// @Tags 审批操作
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/approval/chain/pending [get]
func (h *ApprovalChainHandler) GetPendingApprovals(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	list, err := h.instanceService.GetPendingList(c.Request.Context(), uid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取待审批列表失败")
		return
	}

	response.Success(c, list)
}

// ApproveNode 审批通过节点
// @Summary 审批通过节点
// @Tags 审批操作
// @Accept json
// @Produce json
// @Param nodeInstanceId path int true "节点实例ID"
// @Param body body object true "审批意见"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/nodes/{nodeInstanceId}/approve [post]
func (h *ApprovalChainHandler) ApproveNode(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("nodeInstanceId"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的节点实例ID")
		return
	}

	var req struct {
		Comment string `json:"comment"`
	}
	c.ShouldBindJSON(&req)

	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	uname, ok := middleware.GetUsername(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	if err := h.nodeExecutor.Approve(c.Request.Context(), uint(nodeInstanceID), uid, uname, req.Comment); err != nil {
		// 业务错误返回 400
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, nil)
}

// RejectNode 审批拒绝节点
// @Summary 审批拒绝节点
// @Tags 审批操作
// @Accept json
// @Produce json
// @Param nodeInstanceId path int true "节点实例ID"
// @Param body body object true "拒绝原因"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/nodes/{nodeInstanceId}/reject [post]
func (h *ApprovalChainHandler) RejectNode(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("nodeInstanceId"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的节点实例ID")
		return
	}

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "请填写拒绝原因")
		return
	}

	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	uname, ok := middleware.GetUsername(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	if err := h.nodeExecutor.Reject(c.Request.Context(), uint(nodeInstanceID), uid, uname, req.Reason); err != nil {
		// 业务错误返回 400
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, nil)
}

// TransferNode 转交审批
// @Summary 转交审批
// @Tags 审批操作
// @Accept json
// @Produce json
// @Param nodeInstanceId path int true "节点实例ID"
// @Param body body object true "转交信息"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/nodes/{nodeInstanceId}/transfer [post]
func (h *ApprovalChainHandler) TransferNode(c *gin.Context) {
	nodeInstanceID, err := strconv.ParseUint(c.Param("nodeInstanceId"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的节点实例ID")
		return
	}

	var req struct {
		ToUserID   uint   `json:"to_user_id" binding:"required"`
		ToUserName string `json:"to_user_name" binding:"required"`
		Reason     string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	uname, ok := middleware.GetUsername(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	if err := h.nodeExecutor.Transfer(c.Request.Context(), uint(nodeInstanceID), uid, uname, req.ToUserID, req.ToUserName, req.Reason); err != nil {
		// 业务错误返回 400
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetStats 获取审批统计
// @Summary 获取审批统计
// @Tags 审批统计
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/approval/stats [get]
func (h *ApprovalChainHandler) GetStats(c *gin.Context) {
	stats, err := h.instanceService.GetStats(c.Request.Context())
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取统计数据失败")
		return
	}

	response.Success(c, stats)
}

// TestChain 测试审批链（手动创建测试实例）
// @Summary 测试审批链
// @Tags 审批链管理
// @Accept json
// @Produce json
// @Param id path int true "审批链ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/chains/{id}/test [post]
func (h *ApprovalChainHandler) TestChain(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	// 获取审批链及节点
	chain, err := h.chainService.GetWithNodes(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	if len(chain.Nodes) == 0 {
		response.Error(c, http.StatusBadRequest, "审批链没有节点，请先添加审批节点")
		return
	}

	// 创建测试审批实例，使用时间戳生成唯一的测试 record_id（使用大数避免与真实记录冲突）
	testRecordID := uint(time.Now().UnixNano() % 1000000000)
	instance, err := h.instanceService.Create(c.Request.Context(), testRecordID, chain)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "创建测试实例失败: "+err.Error())
		return
	}

	response.Success(c, instance)
}
