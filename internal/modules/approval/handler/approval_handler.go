package handler

import (
	"devops/internal/service/approval"
	"devops/pkg/excel"
	"devops/pkg/middleware"
	"devops/pkg/response"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type ApprovalHandler struct {
	service *approval.ApprovalService
}

func NewApprovalHandler(service *approval.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{service: service}
}

// GetPendingList 获取待审批列表
// @Summary 获取待审批列表
// @Tags 审批管理
// @Produce json
// @Success 200 {object} response.Response
// @Router /api/v1/approval/pending [get]
func (h *ApprovalHandler) GetPendingList(c *gin.Context) {
	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	records, err := h.service.GetPendingList(c.Request.Context(), uid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取待审批列表失败")
		return
	}

	response.Success(c, records)
}

// Approve 审批通过
// @Summary 审批通过
// @Tags 审批管理
// @Accept json
// @Produce json
// @Param id path int true "部署记录ID"
// @Param body body object true "审批意见"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/{id}/approve [post]
func (h *ApprovalHandler) Approve(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
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

	if err := h.service.Approve(c.Request.Context(), uint(id), uid, uname, req.Comment); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// Reject 审批拒绝
// @Summary 审批拒绝
// @Tags 审批管理
// @Accept json
// @Produce json
// @Param id path int true "部署记录ID"
// @Param body body object true "拒绝原因"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/{id}/reject [post]
func (h *ApprovalHandler) Reject(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
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

	if err := h.service.Reject(c.Request.Context(), uint(id), uid, uname, req.Reason); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// Cancel 取消审批
// @Summary 取消审批
// @Tags 审批管理
// @Param id path int true "部署记录ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/{id}/cancel [post]
func (h *ApprovalHandler) Cancel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
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

	if err := h.service.Cancel(c.Request.Context(), uint(id), uid, uname); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetHistory 获取审批历史
// @Summary 获取审批历史
// @Tags 审批管理
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param app_id query int false "应用ID"
// @Param env query string false "环境"
// @Param status query string false "状态"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/history [get]
func (h *ApprovalHandler) GetHistory(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	env := c.Query("env")
	status := c.Query("status")

	var appID *uint
	if appIDStr := c.Query("app_id"); appIDStr != "" {
		id, err := strconv.ParseUint(appIDStr, 10, 32)
		if err == nil {
			appIDUint := uint(id)
			appID = &appIDUint
		}
	}

	records, total, err := h.service.GetHistory(c.Request.Context(), page, pageSize, appID, env, status)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取审批历史失败")
		return
	}

	response.Success(c, gin.H{
		"list":  records,
		"total": total,
		"page":  page,
		"size":  pageSize,
	})
}

// GetApprovalRecords 获取某个部署记录的审批记录
// @Summary 获取审批记录
// @Tags 审批管理
// @Param id path int true "部署记录ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/{id}/records [get]
func (h *ApprovalHandler) GetApprovalRecords(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	records, err := h.service.GetApprovalRecords(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取审批记录失败")
		return
	}

	response.Success(c, records)
}

// CheckApprovalRequired 检查是否需要审批
// @Summary 检查是否需要审批
// @Tags 审批管理
// @Param app_id query int true "应用ID"
// @Param env query string true "环境"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/check [get]
func (h *ApprovalHandler) CheckApprovalRequired(c *gin.Context) {
	appIDStr := c.Query("app_id")
	env := c.Query("env")

	if appIDStr == "" || env == "" {
		response.Error(c, http.StatusBadRequest, "缺少必要参数")
		return
	}

	appID, err := strconv.ParseUint(appIDStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的应用ID")
		return
	}

	required, approvers, err := h.service.CheckApprovalRequired(c.Request.Context(), uint(appID), env)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "检查审批要求失败")
		return
	}

	response.Success(c, gin.H{
		"required":  required,
		"approvers": approvers,
	})
}

// ExportHistory 导出审批历史
// @Summary 导出审批历史
// @Tags 审批管理
// @Produce application/octet-stream
// @Param format query string true "导出格式 (excel/csv)"
// @Param env query string false "环境"
// @Param status query string false "状态"
// @Param start_time query string false "开始时间"
// @Param end_time query string false "结束时间"
// @Success 200 {file} file
// @Router /api/v1/approval/history/export [get]
func (h *ApprovalHandler) ExportHistory(c *gin.Context) {
	format := c.DefaultQuery("format", "excel")
	env := c.Query("env")
	status := c.Query("status")
	startTime := c.Query("start_time")
	endTime := c.Query("end_time")

	// 获取所有符合条件的记录
	records, err := h.service.GetHistoryForExport(c.Request.Context(), env, status, startTime, endTime)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取审批历史失败")
		return
	}

	if format == "csv" {
		h.exportCSV(c, records)
	} else {
		h.exportExcel(c, records)
	}
}

func (h *ApprovalHandler) exportCSV(c *gin.Context, records []map[string]interface{}) {
	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=approval-history.csv")

	// 写入 BOM 以支持 Excel 打开中文
	c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})

	// 写入表头
	c.Writer.WriteString("应用名称,环境,版本,状态,申请人,审批人,申请时间,审批时间,发布说明\n")

	// 写入数据
	for _, r := range records {
		line := formatCSVLine(r)
		c.Writer.WriteString(line + "\n")
	}
}

func (h *ApprovalHandler) exportExcel(c *gin.Context, records []map[string]interface{}) {
	// 使用 excelize 生成真正的 Excel 文件
	exporter := excel.NewExporter()

	// 设置表头
	headers := []string{"应用名称", "环境", "版本", "状态", "申请人", "审批人", "申请时间", "审批时间", "发布说明"}
	exporter.SetHeaders(headers)

	// 设置列宽
	exporter.SetColumnWidth("A", 20) // 应用名称
	exporter.SetColumnWidth("B", 12) // 环境
	exporter.SetColumnWidth("C", 15) // 版本
	exporter.SetColumnWidth("D", 12) // 状态
	exporter.SetColumnWidth("E", 15) // 申请人
	exporter.SetColumnWidth("F", 15) // 审批人
	exporter.SetColumnWidth("G", 20) // 申请时间
	exporter.SetColumnWidth("H", 20) // 审批时间
	exporter.SetColumnWidth("I", 30) // 发布说明

	// 写入数据
	for _, r := range records {
		row := []interface{}{
			getStringValue(r, "app_name"),
			getStringValue(r, "env_name"),
			getStringValue(r, "version"),
			getStringValue(r, "status"),
			getStringValue(r, "operator"),
			getStringValue(r, "approver_name"),
			getStringValue(r, "created_at"),
			getStringValue(r, "approved_at"),
			getStringValue(r, "description"),
		}
		exporter.AddRow(row)
	}

	// 设置响应头
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=approval-history.xlsx")

	// 输出到响应
	if err := exporter.SaveToWriter(c.Writer); err != nil {
		c.String(http.StatusInternalServerError, "生成 Excel 文件失败")
		return
	}

	exporter.Close()
}

// getStringValue 安全地从 map 中获取字符串值
func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok && v != nil {
		if str, ok := v.(string); ok {
			return str
		}
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func formatCSVLine(r map[string]interface{}) string {
	getString := func(key string) string {
		if v, ok := r[key]; ok && v != nil {
			return escapeCSV(v.(string))
		}
		return ""
	}

	return getString("app_name") + "," +
		getString("env_name") + "," +
		getString("version") + "," +
		getString("status") + "," +
		getString("operator") + "," +
		getString("approver_name") + "," +
		getString("created_at") + "," +
		getString("approved_at") + "," +
		getString("description")
}

func escapeCSV(s string) string {
	if s == "" {
		return ""
	}
	// 如果包含逗号、引号或换行，需要用引号包裹
	needQuote := false
	for _, c := range s {
		if c == ',' || c == '"' || c == '\n' || c == '\r' {
			needQuote = true
			break
		}
	}
	if needQuote {
		// 将引号替换为两个引号
		escaped := ""
		for _, c := range s {
			if c == '"' {
				escaped += "\"\""
			} else {
				escaped += string(c)
			}
		}
		return "\"" + escaped + "\""
	}
	return s
}

// GetDeployRequest 获取发布请求详情
// @Summary 获取发布请求详情
// @Tags 审批管理
// @Param id path int true "发布请求ID"
// @Success 200 {object} response.Response
// @Router /api/v1/approval/request/{id} [get]
func (h *ApprovalHandler) GetDeployRequest(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "无效的ID")
		return
	}

	uid, ok := middleware.GetUserID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "用户未登录")
		return
	}

	request, records, approvers, canApprove, err := h.service.GetDeployRequestDetail(c.Request.Context(), uint(id), uid)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "获取发布请求失败")
		return
	}

	response.Success(c, gin.H{
		"request":     request,
		"records":     records,
		"approvers":   approvers,
		"can_approve": canApprove,
	})
}
