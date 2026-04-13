package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/models"
	"devops/internal/modules/auth/repository"
	"devops/pkg/ioc"
	"devops/pkg/logger"
	"devops/pkg/middleware"
)

var rbacLog = logger.L().WithField("module", "rbac")

func init() {
	ioc.Api.RegisterContainer("RBACHandler", &RBACApiHandler{})
}

type RBACApiHandler struct {
	handler *RBACHandler
}

func (h *RBACApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewRBACHandler(db)

	// 初始化默认权限和角色
	h.handler.InitDefaultPermissions()

	root := cfg.Application.GinRootRouter().Group("rbac")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *RBACApiHandler) Register(r gin.IRouter) {
	// 角色管理 - 查看
	r.GET("/roles", h.handler.ListRoles)
	r.GET("/roles/:id", h.handler.GetRole)
	r.GET("/roles/:id/permissions", h.handler.GetRolePermissions)
	// 角色管理 - 需要超级管理员
	r.POST("/roles", middleware.RequireSuperAdmin(), h.handler.CreateRole)
	r.PUT("/roles/:id", middleware.RequireSuperAdmin(), h.handler.UpdateRole)
	r.DELETE("/roles/:id", middleware.RequireSuperAdmin(), h.handler.DeleteRole)
	r.PUT("/roles/:id/permissions", middleware.RequireSuperAdmin(), h.handler.SetRolePermissions)

	// 权限管理 - 查看
	r.GET("/permissions", h.handler.ListPermissions)
	r.GET("/permissions/grouped", h.handler.GetPermissionsGrouped)

	// 用户角色 - 需要管理员
	r.GET("/users/:id/roles", h.handler.GetUserRoles)
	r.PUT("/users/:id/roles", middleware.RequireAdmin(), h.handler.SetUserRoles)
	r.GET("/users/:id/permissions", h.handler.GetUserPermissions)

	// 当前用户权限 - 所有登录用户可访问
	r.GET("/my-permissions", h.handler.GetMyPermissions)
	r.GET("/check", h.handler.CheckPermission)
}

type RBACHandler struct {
	roleRepo     *repository.RoleRepository
	permRepo     *repository.PermissionRepository
	rolePermRepo *repository.RolePermissionRepository
	userRoleRepo *repository.UserRoleRepository
	db           *gorm.DB
}

func NewRBACHandler(db *gorm.DB) *RBACHandler {
	return &RBACHandler{
		roleRepo:     repository.NewRoleRepository(db),
		permRepo:     repository.NewPermissionRepository(db),
		rolePermRepo: repository.NewRolePermissionRepository(db),
		userRoleRepo: repository.NewUserRoleRepository(db),
		db:           db,
	}
}

// InitDefaultPermissions 初始化默认权限和角色
func (h *RBACHandler) InitDefaultPermissions() {
	ctx := context.Background()
	// 定义默认权限
	defaultPerms := []models.Permission{
		{Name: "jenkins:read", DisplayName: "Jenkins 查看", Resource: "jenkins", Action: "read"},
		{Name: "jenkins:write", DisplayName: "Jenkins 操作", Resource: "jenkins", Action: "write"},
		{Name: "jenkins:admin", DisplayName: "Jenkins 管理", Resource: "jenkins", Action: "admin"},
		{Name: "k8s:read", DisplayName: "K8s 查看", Resource: "k8s", Action: "read"},
		{Name: "k8s:write", DisplayName: "K8s 操作", Resource: "k8s", Action: "write"},
		{Name: "k8s:admin", DisplayName: "K8s 管理", Resource: "k8s", Action: "admin"},
		{Name: "feishu:read", DisplayName: "飞书 查看", Resource: "feishu", Action: "read"},
		{Name: "feishu:write", DisplayName: "飞书 操作", Resource: "feishu", Action: "write"},
		{Name: "feishu:admin", DisplayName: "飞书 管理", Resource: "feishu", Action: "admin"},
		{Name: "dingtalk:read", DisplayName: "钉钉 查看", Resource: "dingtalk", Action: "read"},
		{Name: "dingtalk:write", DisplayName: "钉钉 操作", Resource: "dingtalk", Action: "write"},
		{Name: "dingtalk:admin", DisplayName: "钉钉 管理", Resource: "dingtalk", Action: "admin"},
		{Name: "wechatwork:read", DisplayName: "企业微信 查看", Resource: "wechatwork", Action: "read"},
		{Name: "wechatwork:write", DisplayName: "企业微信 操作", Resource: "wechatwork", Action: "write"},
		{Name: "wechatwork:admin", DisplayName: "企业微信 管理", Resource: "wechatwork", Action: "admin"},
		{Name: "oa:read", DisplayName: "OA 查看", Resource: "oa", Action: "read"},
		{Name: "oa:write", DisplayName: "OA 操作", Resource: "oa", Action: "write"},
		{Name: "oa:admin", DisplayName: "OA 管理", Resource: "oa", Action: "admin"},
		{Name: "app:read", DisplayName: "应用 查看", Resource: "app", Action: "read"},
		{Name: "app:write", DisplayName: "应用 操作", Resource: "app", Action: "write"},
		{Name: "app:admin", DisplayName: "应用 管理", Resource: "app", Action: "admin"},
		{Name: "user:read", DisplayName: "用户 查看", Resource: "user", Action: "read"},
		{Name: "user:write", DisplayName: "用户 操作", Resource: "user", Action: "write"},
		{Name: "user:admin", DisplayName: "用户 管理", Resource: "user", Action: "admin"},
		{Name: "audit:read", DisplayName: "审计 查看", Resource: "audit", Action: "read"},
		{Name: "system:admin", DisplayName: "系统 管理", Resource: "system", Action: "admin"},
	}

	for _, perm := range defaultPerms {
		h.db.Where("name = ?", perm.Name).FirstOrCreate(&perm)
	}

	// 定义默认角色
	defaultRoles := []models.Role{
		{Name: "admin", DisplayName: "管理员", Description: "拥有所有权限", IsSystem: true, Status: "active"},
		{Name: "operator", DisplayName: "运维人员", Description: "可以操作 Jenkins、K8s 等资源", IsSystem: true, Status: "active"},
		{Name: "developer", DisplayName: "开发人员", Description: "可以查看资源和触发构建", IsSystem: true, Status: "active"},
		{Name: "viewer", DisplayName: "只读用户", Description: "只能查看资源", IsSystem: true, Status: "active"},
	}

	for _, role := range defaultRoles {
		var existing models.Role
		if h.db.Where("name = ?", role.Name).First(&existing).Error != nil {
			h.db.Create(&role)
		}
	}

	// 为管理员角色分配所有权限
	var adminRole models.Role
	if h.db.Where("name = ?", "admin").First(&adminRole).Error == nil {
		var allPerms []models.Permission
		h.db.Find(&allPerms)
		var permIDs []uint
		for _, p := range allPerms {
			permIDs = append(permIDs, p.ID)
		}
		h.rolePermRepo.SetRolePermissions(ctx, adminRole.ID, permIDs)
	}

	// 为运维人员分配权限
	var operatorRole models.Role
	if h.db.Where("name = ?", "operator").First(&operatorRole).Error == nil {
		var perms []models.Permission
		h.db.Where("action IN ?", []string{"read", "write"}).Find(&perms)
		var permIDs []uint
		for _, p := range perms {
			permIDs = append(permIDs, p.ID)
		}
		h.rolePermRepo.SetRolePermissions(ctx, operatorRole.ID, permIDs)
	}

	// 为开发人员分配权限
	var devRole models.Role
	if h.db.Where("name = ?", "developer").First(&devRole).Error == nil {
		var perms []models.Permission
		h.db.Where("action = ? OR (resource IN ? AND action = ?)", "read", []string{"jenkins", "app"}, "write").Find(&perms)
		var permIDs []uint
		for _, p := range perms {
			permIDs = append(permIDs, p.ID)
		}
		h.rolePermRepo.SetRolePermissions(ctx, devRole.ID, permIDs)
	}

	// 为只读用户分配权限
	var viewerRole models.Role
	if h.db.Where("name = ?", "viewer").First(&viewerRole).Error == nil {
		var perms []models.Permission
		h.db.Where("action = ?", "read").Find(&perms)
		var permIDs []uint
		for _, p := range perms {
			permIDs = append(permIDs, p.ID)
		}
		h.rolePermRepo.SetRolePermissions(ctx, viewerRole.ID, permIDs)
	}
}

func (h *RBACHandler) ListRoles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	roles, total, err := h.roleRepo.List(c.Request.Context(), page, pageSize)
	if err != nil {
		rbacLog.WithError(err).Error("查询角色列表失败")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询角色列表失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  roles,
			"total": total,
		},
	})
}

func (h *RBACHandler) GetRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "ID格式错误"})
		return
	}

	role, err := h.roleRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "角色不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": role})
}

func (h *RBACHandler) CreateRole(c *gin.Context) {
	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		rbacLog.WithError(err).Warn("创建角色参数错误")
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误: " + err.Error()})
		return
	}

	role.ID = 0
	role.IsSystem = false

	if err := h.roleRepo.Create(c.Request.Context(), &role); err != nil {
		rbacLog.WithError(err).Error("创建角色失败: %s", role.Name)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败: " + err.Error()})
		return
	}

	rbacLog.Info("创建角色成功: %s (ID: %d)", role.Name, role.ID)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功", "data": role})
}

func (h *RBACHandler) UpdateRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "ID格式错误"})
		return
	}

	existing, err := h.roleRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "角色不存在"})
		return
	}

	var role models.Role
	if err := c.ShouldBindJSON(&role); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	role.ID = uint(id)
	role.IsSystem = existing.IsSystem // 保持系统角色标记

	if err := h.roleRepo.Update(c.Request.Context(), &role); err != nil {
		rbacLog.WithError(err).Error("更新角色失败: %s", role.Name)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新角色失败"})
		return
	}

	rbacLog.Info("更新角色成功: %s (ID: %d)", role.Name, id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": role})
}

func (h *RBACHandler) DeleteRole(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "ID格式错误"})
		return
	}

	role, err := h.roleRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "角色不存在"})
		return
	}

	if role.IsSystem {
		rbacLog.Warn("尝试删除系统内置角色: %s", role.Name)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "系统内置角色不可删除"})
		return
	}

	if err := h.roleRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		rbacLog.WithError(err).Error("删除角色失败: %s", role.Name)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除角色失败"})
		return
	}

	rbacLog.Info("删除角色成功: %s (ID: %d)", role.Name, id)
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func (h *RBACHandler) GetRolePermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "ID格式错误"})
		return
	}

	perms, err := h.rolePermRepo.GetRolePermissions(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "获取角色权限失败"})
		return
	}

	permIDs, _ := h.rolePermRepo.GetRolePermissionIDs(c.Request.Context(), uint(id))

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"permissions":    perms,
			"permission_ids": permIDs,
		},
	})
}

func (h *RBACHandler) SetRolePermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	var req struct {
		PermissionIDs []uint `json:"permission_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.rolePermRepo.SetRolePermissions(c.Request.Context(), uint(id), req.PermissionIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

func (h *RBACHandler) ListPermissions(c *gin.Context) {
	perms, err := h.permRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": perms})
}

func (h *RBACHandler) GetPermissionsGrouped(c *gin.Context) {
	perms, err := h.permRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 按资源分组
	grouped := make(map[string][]models.Permission)
	for _, p := range perms {
		grouped[p.Resource] = append(grouped[p.Resource], p)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": grouped})
}

func (h *RBACHandler) GetUserRoles(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	roles, err := h.userRoleRepo.GetUserRoles(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	roleIDs, _ := h.userRoleRepo.GetUserRoleIDs(c.Request.Context(), uint(id))

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"roles":    roles,
			"role_ids": roleIDs,
		},
	})
}

func (h *RBACHandler) SetUserRoles(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	var req struct {
		RoleIDs []uint `json:"role_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if err := h.userRoleRepo.SetUserRoles(c.Request.Context(), uint(id), req.RoleIDs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

func (h *RBACHandler) GetUserPermissions(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	perms, err := h.userRoleRepo.GetUserPermissions(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": perms})
}

func (h *RBACHandler) GetMyPermissions(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Unauthorized"})
		return
	}

	perms, err := h.userRoleRepo.GetUserPermissions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	// 转换为权限名称列表
	permNames := make([]string, len(perms))
	for i, p := range perms {
		permNames[i] = p.Name
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"permissions":      perms,
			"permission_names": permNames,
		},
	})
}

func (h *RBACHandler) CheckPermission(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "Unauthorized"})
		return
	}

	resource := c.Query("resource")
	action := c.Query("action")

	if resource == "" || action == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "resource and action are required"})
		return
	}

	hasPermission, err := h.userRoleRepo.HasPermission(c.Request.Context(), userID, resource, action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"has_permission": hasPermission,
		},
	})
}
