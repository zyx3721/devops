package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/models"
	"devops/internal/repository"
	"devops/pkg/ioc"
	"devops/pkg/logger"
	"devops/pkg/middleware"
	"devops/pkg/response"
)

var appLog = logger.L().WithField("module", "application")

func init() {
	ioc.Api.RegisterContainer("ApplicationHandler", &ApplicationApiHandler{})
}

type ApplicationApiHandler struct {
	handler *ApplicationHandler
}

func (h *ApplicationApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewApplicationHandler(db)

	root := cfg.Application.GinRootRouter().Group("app")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	// 添加 /applications 路由别名，兼容前端请求
	appRoot := cfg.Application.GinRootRouter().Group("applications")
	appRoot.Use(middleware.AuthMiddleware())
	h.Register(appRoot)

	return nil
}

func (h *ApplicationApiHandler) Register(r gin.IRouter) {
	// 查看权限 - 所有登录用户可访问
	r.GET("", h.handler.ListApplications)
	r.GET("/:id", h.handler.GetApplication)
	r.GET("/:id/envs", h.handler.ListAppEnvs)
	r.GET("/:id/deploys", h.handler.ListDeployRecords)
	r.GET("/deploys", h.handler.ListAllDeployRecords)
	r.GET("/stats", h.handler.GetStats)
	r.GET("/teams", h.handler.GetTeams)

	// 管理权限 - 需要管理员
	r.POST("", middleware.RequireAdmin(), h.handler.CreateApplication)
	r.PUT("/:id", middleware.RequireAdmin(), h.handler.UpdateApplication)
	r.DELETE("/:id", middleware.RequireAdmin(), h.handler.DeleteApplication)
	r.POST("/:id/envs", middleware.RequireAdmin(), h.handler.CreateAppEnv)
	r.PUT("/:id/envs/:envId", middleware.RequireAdmin(), h.handler.UpdateAppEnv)
	r.DELETE("/:id/envs/:envId", middleware.RequireAdmin(), h.handler.DeleteAppEnv)
}

type ApplicationHandler struct {
	appRepo    *repository.ApplicationRepository
	envRepo    *repository.ApplicationEnvRepository
	deployRepo *repository.DeployRecordRepository
	db         *gorm.DB
}

func NewApplicationHandler(db *gorm.DB) *ApplicationHandler {
	return &ApplicationHandler{
		appRepo:    repository.NewApplicationRepository(db),
		envRepo:    repository.NewApplicationEnvRepository(db),
		deployRepo: repository.NewDeployRecordRepository(db),
		db:         db,
	}
}

// ListApplications godoc
// @Summary 获取应用列表
// @Description 分页获取应用列表，支持按名称、团队、状态、语言筛选
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param name query string false "应用名称"
// @Param team query string false "团队"
// @Param status query string false "状态"
// @Param language query string false "开发语言"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=response.PageData} "成功"
// @Failure 500 {object} response.Response "服务器错误"
// @Security BearerAuth
// @Router /app [get]
func (h *ApplicationHandler) ListApplications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := repository.ApplicationFilter{
		Name:     c.Query("name"),
		Team:     c.Query("team"),
		Status:   c.Query("status"),
		Language: c.Query("language"),
	}

	apps, total, err := h.appRepo.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Page(c, apps, total, page, pageSize)
}

// GetApplication godoc
// @Summary 获取应用详情
// @Description 根据ID获取应用详情及环境配置
// @Tags 应用管理
// @Accept json
// @Produce json
// @Param id path int true "应用ID"
// @Success 200 {object} response.Response{data=object{app=models.Application,envs=[]models.ApplicationEnv}} "成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 404 {object} response.Response "应用不存在"
// @Security BearerAuth
// @Router /app/{id} [get]
func (h *ApplicationHandler) GetApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	app, err := h.appRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		response.NotFound(c, "应用不存在")
		return
	}

	// 获取环境配置
	envs, _ := h.envRepo.GetByAppID(c.Request.Context(), uint(id))

	response.Success(c, gin.H{"app": app, "envs": envs})
}

func (h *ApplicationHandler) CreateApplication(c *gin.Context) {
	var app models.Application
	if err := c.ShouldBindJSON(&app); err != nil {
		appLog.WithError(err).Warn("创建应用参数错误")
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	app.ID = 0
	if userID, ok := middleware.GetUserID(c); ok {
		app.CreatedBy = &userID
	}

	if err := h.appRepo.Create(c.Request.Context(), &app); err != nil {
		appLog.WithError(err).Error("创建应用失败: %s", app.Name)
		response.InternalError(c, "创建失败: "+err.Error())
		return
	}

	response.Success(c, app)
}

func (h *ApplicationHandler) UpdateApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	var app models.Application
	if err := c.ShouldBindJSON(&app); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	app.ID = uint(id)
	if err := h.appRepo.Update(c.Request.Context(), &app); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, app)
}

func (h *ApplicationHandler) DeleteApplication(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	if err := h.appRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c)
}

func (h *ApplicationHandler) ListAppEnvs(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	envs, err := h.envRepo.GetByAppID(c.Request.Context(), uint(id))
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, envs)
}

func (h *ApplicationHandler) CreateAppEnv(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	var env models.ApplicationEnv
	if err := c.ShouldBindJSON(&env); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	env.ID = 0
	env.ApplicationID = uint(id)

	if err := h.envRepo.Create(c.Request.Context(), &env); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, env)
}

func (h *ApplicationHandler) UpdateAppEnv(c *gin.Context) {
	envId, err := strconv.ParseUint(c.Param("envId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "环境ID格式错误")
		return
	}

	var env models.ApplicationEnv
	if err := c.ShouldBindJSON(&env); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	env.ID = uint(envId)
	if err := h.envRepo.Update(c.Request.Context(), &env); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, env)
}

func (h *ApplicationHandler) DeleteAppEnv(c *gin.Context) {
	envId, err := strconv.ParseUint(c.Param("envId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "环境ID格式错误")
		return
	}

	if err := h.envRepo.Delete(c.Request.Context(), uint(envId)); err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c)
}

func (h *ApplicationHandler) ListDeployRecords(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := repository.DeployRecordFilter{
		ApplicationID: uint(id),
		EnvName:       c.Query("env"),
		Status:        c.Query("status"),
	}

	records, total, err := h.deployRepo.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Page(c, records, total, page, pageSize)
}

func (h *ApplicationHandler) ListAllDeployRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := repository.DeployRecordFilter{
		AppName: c.Query("app_name"),
		EnvName: c.Query("env"),
		Status:  c.Query("status"),
	}

	records, total, err := h.deployRepo.List(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Page(c, records, total, page, pageSize)
}

func (h *ApplicationHandler) GetStats(c *gin.Context) {
	type StatItem struct {
		Name  string `json:"name"`
		Count int64  `json:"count"`
	}

	// 应用总数
	var appCount int64
	h.db.Model(&models.Application{}).Count(&appCount)

	// 按团队统计
	var teamStats []StatItem
	h.db.Raw(`SELECT team as name, COUNT(*) as count FROM applications WHERE team != '' GROUP BY team ORDER BY count DESC`).Scan(&teamStats)

	// 按语言统计
	var langStats []StatItem
	h.db.Raw(`SELECT language as name, COUNT(*) as count FROM applications WHERE language != '' GROUP BY language ORDER BY count DESC`).Scan(&langStats)

	// 今日部署数
	var todayDeploys int64
	h.db.Raw(`SELECT COUNT(*) FROM deploy_records WHERE DATE(created_at) = CURDATE()`).Scan(&todayDeploys)

	// 本周部署数
	var weekDeploys int64
	h.db.Raw(`SELECT COUNT(*) FROM deploy_records WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)`).Scan(&weekDeploys)

	// 部署成功率
	var successCount, totalCount int64
	h.db.Raw(`SELECT COUNT(*) FROM deploy_records WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)`).Scan(&totalCount)
	h.db.Raw(`SELECT COUNT(*) FROM deploy_records WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY) AND status = 'success'`).Scan(&successCount)

	successRate := float64(0)
	if totalCount > 0 {
		successRate = float64(successCount) / float64(totalCount) * 100
	}

	response.Success(c, gin.H{
		"app_count":     appCount,
		"team_stats":    teamStats,
		"lang_stats":    langStats,
		"today_deploys": todayDeploys,
		"week_deploys":  weekDeploys,
		"success_rate":  successRate,
	})
}

func (h *ApplicationHandler) GetTeams(c *gin.Context) {
	teams, err := h.appRepo.GetAllTeams(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, teams)
}
