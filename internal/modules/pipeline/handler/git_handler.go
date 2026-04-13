package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/service/pipeline"
	"devops/pkg/dto"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
	"devops/pkg/response"
)

func init() {
	ioc.Api.RegisterContainer("GitHandler", &GitApiHandler{})
}

// GitApiHandler IOC容器注册的处理器
type GitApiHandler struct {
	handler *GitHandler
}

func (h *GitApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	gitSvc := pipeline.NewGitService(db)

	h.handler = NewGitHandler(gitSvc)

	root := cfg.Application.GinRootRouter().Group("git")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *GitApiHandler) Register(r gin.IRouter) {
	// Git 仓库管理
	r.GET("/repos", h.handler.ListRepos)
	r.GET("/repos/:id", h.handler.GetRepo)
	r.POST("/repos", h.handler.CreateRepo)
	r.PUT("/repos/:id", h.handler.UpdateRepo)
	r.DELETE("/repos/:id", middleware.RequireAdmin(), h.handler.DeleteRepo)

	// 仓库操作
	r.POST("/repos/test", h.handler.TestConnection)
	r.GET("/repos/:id/branches", h.handler.GetBranches)
	r.GET("/repos/:id/tags", h.handler.GetTags)
	r.POST("/repos/:id/regenerate-secret", middleware.RequireAdmin(), h.handler.RegenerateSecret)

	// Webhook 回调（无需认证）
	// 注意：Webhook 路由需要在无认证的路由组中注册
}

// GitHandler Git 仓库处理器
type GitHandler struct {
	gitSvc *pipeline.GitService
}

// NewGitHandler 创建 Git 仓库处理器
func NewGitHandler(gitSvc *pipeline.GitService) *GitHandler {
	return &GitHandler{
		gitSvc: gitSvc,
	}
}

// ListRepos 获取仓库列表
func (h *GitHandler) ListRepos(c *gin.Context) {
	var req dto.GitRepoListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.gitSvc.List(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetRepo 获取仓库详情
func (h *GitHandler) GetRepo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	result, err := h.gitSvc.Get(c.Request.Context(), uint(id))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// CreateRepo 创建仓库
func (h *GitHandler) CreateRepo(c *gin.Context) {
	var req dto.GitRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.gitSvc.Create(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "创建成功", result)
}

// UpdateRepo 更新仓库
func (h *GitHandler) UpdateRepo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var req dto.GitRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.gitSvc.Update(c.Request.Context(), uint(id), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "更新成功", result)
}

// DeleteRepo 删除仓库
func (h *GitHandler) DeleteRepo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.gitSvc.Delete(c.Request.Context(), uint(id)); err != nil {
		response.FromError(c, err)
		return
	}

	response.SuccessWithMessage(c, "删除成功", nil)
}

// TestConnection 测试仓库连接
func (h *GitHandler) TestConnection(c *gin.Context) {
	var req dto.GitTestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	result, err := h.gitSvc.TestConnection(c.Request.Context(), &req)
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetBranches 获取分支列表
func (h *GitHandler) GetBranches(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	result, err := h.gitSvc.GetBranches(c.Request.Context(), uint(id))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// GetTags 获取 Tag 列表
func (h *GitHandler) GetTags(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	result, err := h.gitSvc.GetTags(c.Request.Context(), uint(id))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, result)
}

// RegenerateSecret 重新生成 Webhook Secret
func (h *GitHandler) RegenerateSecret(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	secret, err := h.gitSvc.RegenerateWebhookSecret(c.Request.Context(), uint(id))
	if err != nil {
		response.FromError(c, err)
		return
	}

	response.Success(c, gin.H{"webhook_secret": secret})
}
