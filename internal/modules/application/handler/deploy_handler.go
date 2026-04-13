package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/models"
	"devops/internal/repository"
	"devops/internal/service/approval"
	"devops/internal/service/deploy"
	"devops/internal/service/jenkins"
	"devops/internal/service/kubernetes"
	"devops/pkg/ioc"
	"devops/pkg/logger"
	"devops/pkg/middleware"
	"devops/pkg/response"
	"devops/pkg/validator"
)

var deployLog = logger.L().WithField("module", "deploy")

func init() {
	ioc.Api.RegisterContainer("DeployHandler", &DeployApiHandler{})
}

type DeployApiHandler struct {
	handler *DeployHandler
}

func (h *DeployApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	h.handler = NewDeployHandler(db)

	root := cfg.Application.GinRootRouter().Group("deploy")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *DeployApiHandler) Register(r gin.IRouter) {
	// 部署记录（含审批）
	r.POST("/records", h.handler.CreateDeploy)
	r.POST("/records/emergency", h.handler.CreateEmergencyDeploy)
	r.GET("/records", h.handler.ListRecords)
	r.GET("/records/:id", h.handler.GetRecord)
	r.POST("/records/:id/cancel", h.handler.CancelDeploy)
	r.POST("/records/:id/approve", h.handler.ApproveDeploy)
	r.POST("/records/:id/reject", h.handler.RejectDeploy)
	r.POST("/records/:id/execute", h.handler.ExecuteDeploy)

	// 回滚
	r.POST("/rollback", h.handler.CreateRollback)
	r.GET("/rollback/:appId/:env/available", h.handler.GetAvailableRollback)

	// 锁管理
	r.GET("/locks/:appId/:env", h.handler.GetLockStatus)
	r.POST("/locks/:appId/:env/release", h.handler.ReleaseLock)

	// 发布窗口
	r.GET("/window/:appId/:env", h.handler.GetDeployWindowStatus)

	// 统计
	r.GET("/stats", h.handler.GetStats)
}

type DeployHandler struct {
	service *deploy.Service
	db      *gorm.DB
}

func NewDeployHandler(db *gorm.DB) *DeployHandler {
	recordRepo := repository.NewDeployRecordRepository(db)
	lockRepo := repository.NewDeployLockRepository(db)
	approvalRepo := repository.NewApprovalRecordRepository(db)
	appRepo := repository.NewApplicationRepository(db)
	jenkinsClient := jenkins.NewClient()
	k8sManager := kubernetes.NewK8sClientManager(db)

	// 创建审批链相关 Repository 和 Service
	chainRepo := repository.NewApprovalChainRepository(db)
	nodeRepo := repository.NewApprovalNodeRepository(db)
	instanceRepo := repository.NewApprovalInstanceRepository(db)
	nodeInstanceRepo := repository.NewApprovalNodeInstanceRepository(db)
	actionRepo := repository.NewApprovalActionRepository(db)

	chainService := approval.NewChainService(chainRepo, nodeRepo)
	nodeExecutor := approval.NewNodeExecutor(nodeInstanceRepo, actionRepo, instanceRepo)
	approverResolver := approval.NewApproverResolver(db)
	instanceService := approval.NewInstanceService(instanceRepo, nodeInstanceRepo, chainService, nodeExecutor, approverResolver)

	// 创建 deploy service 并注入审批链服务
	deployService := deploy.NewService(recordRepo, lockRepo, approvalRepo, appRepo, jenkinsClient, k8sManager)
	deployService.SetApprovalChainServices(chainService, instanceService)

	return &DeployHandler{
		service: deployService,
		db:      db,
	}
}

// CreateDeployDTO 创建部署参数
type CreateDeployDTO struct {
	ApplicationID uint   `json:"application_id" validate:"required,gt=0" label:"应用ID"`
	EnvName       string `json:"env_name" validate:"required" label:"环境"`
	Version       string `json:"version" label:"版本"`
	Branch        string `json:"branch" label:"分支"`
	CommitID      string `json:"commit_id" label:"提交ID"`
	ImageTag      string `json:"image_tag" label:"镜像标签"`
	DeployMethod  string `json:"deploy_method" label:"部署方式"` // jenkins, k8s
	Description   string `json:"description" label:"说明"`
}

// CreateEmergencyDeployDTO 创建紧急部署参数
type CreateEmergencyDeployDTO struct {
	ApplicationID   uint   `json:"application_id" validate:"required,gt=0" label:"应用ID"`
	EnvName         string `json:"env_name" validate:"required" label:"环境"`
	Version         string `json:"version" label:"版本"`
	Branch          string `json:"branch" label:"分支"`
	CommitID        string `json:"commit_id" label:"提交ID"`
	ImageTag        string `json:"image_tag" label:"镜像标签"`
	DeployMethod    string `json:"deploy_method" label:"部署方式"`
	Description     string `json:"description" label:"说明"`
	EmergencyReason string `json:"emergency_reason" validate:"required" label:"紧急原因"`
}

// CreateDeploy godoc
// @Summary 创建部署记录
// @Description 创建新的部署记录（需要审批的环境会进入待审批状态）
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param request body CreateDeployDTO true "部署参数"
// @Success 200 {object} response.Response{data=models.DeployRecord} "成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 404 {object} response.Response "应用不存在"
// @Security BearerAuth
// @Router /deploy/records [post]
func (h *DeployHandler) CreateDeploy(c *gin.Context) {
	var dto CreateDeployDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if errs := validator.Validate(dto); len(errs) > 0 {
		response.ValidationError(c, errs)
		return
	}

	userID, _ := middleware.GetUserID(c)
	username, _ := middleware.GetUsername(c)

	record := &models.DeployRecord{
		ApplicationID: dto.ApplicationID,
		EnvName:       dto.EnvName,
		Version:       dto.Version,
		Branch:        dto.Branch,
		CommitID:      dto.CommitID,
		ImageTag:      dto.ImageTag,
		DeployType:    deploy.DeployTypeDeploy,
		DeployMethod:  dto.DeployMethod,
		Description:   dto.Description,
		Operator:      username,
		OperatorID:    userID,
	}

	if err := h.service.CreateDeploy(c.Request.Context(), record); err != nil {
		if errors.Is(err, deploy.ErrApplicationNotFound) {
			response.NotFound(c, "应用不存在")
			return
		}
		if errors.Is(err, deploy.ErrOutsideDeployWindow) {
			response.BadRequest(c, "当前不在发布窗口期内，请使用紧急发布")
			return
		}
		if errors.Is(err, deploy.ErrEmergencyRequired) {
			response.BadRequest(c, "窗口期外发布需要使用紧急发布接口")
			return
		}
		deployLog.WithError(err).Error("创建部署记录失败")
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, record)
}

// CreateEmergencyDeploy godoc
// @Summary 创建紧急部署
// @Description 创建紧急部署记录（可绕过发布窗口限制和审批）
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param request body CreateEmergencyDeployDTO true "紧急部署参数"
// @Success 200 {object} response.Response{data=models.DeployRecord} "成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 404 {object} response.Response "应用不存在"
// @Security BearerAuth
// @Router /deploy/records/emergency [post]
func (h *DeployHandler) CreateEmergencyDeploy(c *gin.Context) {
	var dto CreateEmergencyDeployDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if errs := validator.Validate(dto); len(errs) > 0 {
		response.ValidationError(c, errs)
		return
	}

	userID, _ := middleware.GetUserID(c)
	username, _ := middleware.GetUsername(c)

	record := &models.DeployRecord{
		ApplicationID: dto.ApplicationID,
		EnvName:       dto.EnvName,
		Version:       dto.Version,
		Branch:        dto.Branch,
		CommitID:      dto.CommitID,
		ImageTag:      dto.ImageTag,
		DeployType:    deploy.DeployTypeDeploy,
		DeployMethod:  dto.DeployMethod,
		Description:   dto.Description,
		Operator:      username,
		OperatorID:    userID,
	}

	if err := h.service.CreateDeployWithEmergency(c.Request.Context(), record, true, dto.EmergencyReason); err != nil {
		if errors.Is(err, deploy.ErrApplicationNotFound) {
			response.NotFound(c, "应用不存在")
			return
		}
		if errors.Is(err, deploy.ErrOutsideDeployWindow) {
			response.BadRequest(c, "当前不在发布窗口期内且不允许紧急发布")
			return
		}
		deployLog.WithError(err).Error("创建紧急部署记录失败")
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, record)
}

// ListRecords godoc
// @Summary 获取部署记录列表
// @Description 分页获取部署记录列表
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param application_id query int false "应用ID"
// @Param app_name query string false "应用名称"
// @Param env_name query string false "环境"
// @Param status query string false "状态"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} response.Response{data=response.PageData} "成功"
// @Security BearerAuth
// @Router /deploy/records [get]
func (h *DeployHandler) ListRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	appID, _ := strconv.ParseUint(c.Query("application_id"), 10, 64)

	filter := repository.DeployRecordFilter{
		ApplicationID: uint(appID),
		AppName:       c.Query("app_name"),
		EnvName:       c.Query("env_name"),
		Status:        c.Query("status"),
	}

	records, total, err := h.service.ListRecords(c.Request.Context(), filter, page, pageSize)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Page(c, records, total, page, pageSize)
}

// GetRecord godoc
// @Summary 获取部署记录详情
// @Description 根据ID获取部署记录详情及审批记录
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param id path int true "记录ID"
// @Success 200 {object} response.Response{data=object{record=models.DeployRecord,approvals=[]models.ApprovalRecord}} "成功"
// @Failure 404 {object} response.Response "记录不存在"
// @Security BearerAuth
// @Router /deploy/records/{id} [get]
func (h *DeployHandler) GetRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	record, err := h.service.GetRecord(c.Request.Context(), uint(id))
	if err != nil {
		if errors.Is(err, deploy.ErrRecordNotFound) {
			response.NotFound(c, "部署记录不存在")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	// 获取审批记录
	approvals, _ := h.service.GetApprovalRecords(c.Request.Context(), uint(id))

	response.Success(c, gin.H{
		"record":    record,
		"approvals": approvals,
	})
}

// CancelDeploy godoc
// @Summary 取消部署
// @Description 取消待审批的部署记录
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param id path int true "记录ID"
// @Success 200 {object} response.Response "成功"
// @Failure 400 {object} response.Response "状态不允许取消"
// @Failure 404 {object} response.Response "记录不存在"
// @Security BearerAuth
// @Router /deploy/records/{id}/cancel [post]
func (h *DeployHandler) CancelDeploy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	if err := h.service.CancelDeploy(c.Request.Context(), uint(id)); err != nil {
		if errors.Is(err, deploy.ErrRecordNotFound) {
			response.NotFound(c, "部署记录不存在")
			return
		}
		if errors.Is(err, deploy.ErrInvalidStatus) {
			response.BadRequest(c, "当前状态不允许取消")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c)
}

// ApprovalDTO 审批参数
type ApprovalDTO struct {
	Comment string `json:"comment" label:"审批意见"`
}

// ApproveDeploy godoc
// @Summary 审批通过
// @Description 审批通过部署记录
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param id path int true "记录ID"
// @Param request body ApprovalDTO false "审批意见"
// @Success 200 {object} response.Response "成功"
// @Failure 400 {object} response.Response "状态不允许审批"
// @Failure 403 {object} response.Response "不能审批自己的请求"
// @Failure 404 {object} response.Response "记录不存在"
// @Security BearerAuth
// @Router /deploy/records/{id}/approve [post]
func (h *DeployHandler) ApproveDeploy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	var dto ApprovalDTO
	c.ShouldBindJSON(&dto)

	userID, _ := middleware.GetUserID(c)
	username, _ := middleware.GetUsername(c)

	if err := h.service.ApproveDeploy(c.Request.Context(), uint(id), userID, username, dto.Comment); err != nil {
		if errors.Is(err, deploy.ErrRecordNotFound) {
			response.NotFound(c, "部署记录不存在")
			return
		}
		if errors.Is(err, deploy.ErrInvalidStatus) {
			response.BadRequest(c, "当前状态不允许审批")
			return
		}
		if errors.Is(err, deploy.ErrSelfApprovalForbidden) {
			response.Forbidden(c, "不能审批自己的请求")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c)
}

// RejectDTO 拒绝参数
type RejectDTO struct {
	Reason string `json:"reason" validate:"required" label:"拒绝原因"`
}

// RejectDeploy godoc
// @Summary 审批拒绝
// @Description 拒绝部署记录
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param id path int true "记录ID"
// @Param request body RejectDTO true "拒绝原因"
// @Success 200 {object} response.Response "成功"
// @Failure 400 {object} response.Response "状态不允许审批"
// @Failure 404 {object} response.Response "记录不存在"
// @Security BearerAuth
// @Router /deploy/records/{id}/reject [post]
func (h *DeployHandler) RejectDeploy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	var dto RejectDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if errs := validator.Validate(dto); len(errs) > 0 {
		response.ValidationError(c, errs)
		return
	}

	userID, _ := middleware.GetUserID(c)
	username, _ := middleware.GetUsername(c)

	if err := h.service.RejectDeploy(c.Request.Context(), uint(id), userID, username, dto.Reason); err != nil {
		if errors.Is(err, deploy.ErrRecordNotFound) {
			response.NotFound(c, "部署记录不存在")
			return
		}
		if errors.Is(err, deploy.ErrInvalidStatus) {
			response.BadRequest(c, "当前状态不允许审批")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c)
}

// ExecuteDeploy godoc
// @Summary 执行部署
// @Description 执行部署（已审批或无需审批的记录）
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param id path int true "记录ID"
// @Success 200 {object} response.Response "成功"
// @Failure 400 {object} response.Response "状态不允许执行"
// @Failure 404 {object} response.Response "记录不存在"
// @Failure 409 {object} response.Response "应用环境已被锁定"
// @Security BearerAuth
// @Router /deploy/records/{id}/execute [post]
func (h *DeployHandler) ExecuteDeploy(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "ID格式错误")
		return
	}

	userID, _ := middleware.GetUserID(c)
	username, _ := middleware.GetUsername(c)

	if err := h.service.ExecuteDeploy(c.Request.Context(), uint(id), userID, username); err != nil {
		if errors.Is(err, deploy.ErrRecordNotFound) {
			response.NotFound(c, "部署记录不存在")
			return
		}
		if errors.Is(err, deploy.ErrInvalidStatus) {
			response.BadRequest(c, "当前状态不允许执行")
			return
		}
		if errors.Is(err, deploy.ErrLockExists) {
			response.Conflict(c, "应用环境已被锁定，请稍后重试")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c)
}

// RollbackDTO 回滚参数
type RollbackDTO struct {
	ApplicationID uint   `json:"application_id" validate:"required,gt=0" label:"应用ID"`
	EnvName       string `json:"env_name" validate:"required" label:"环境"`
}

// CreateRollback godoc
// @Summary 创建回滚
// @Description 创建回滚到上一个成功版本
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param request body RollbackDTO true "回滚参数"
// @Success 200 {object} response.Response{data=models.DeployRecord} "成功"
// @Failure 400 {object} response.Response "没有可回滚的版本"
// @Security BearerAuth
// @Router /deploy/rollback [post]
func (h *DeployHandler) CreateRollback(c *gin.Context) {
	var dto RollbackDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if errs := validator.Validate(dto); len(errs) > 0 {
		response.ValidationError(c, errs)
		return
	}

	userID, _ := middleware.GetUserID(c)
	username, _ := middleware.GetUsername(c)

	record, err := h.service.CreateRollback(c.Request.Context(), dto.ApplicationID, dto.EnvName, userID, username)
	if err != nil {
		if errors.Is(err, deploy.ErrNoRollbackVersion) {
			response.BadRequest(c, "没有可回滚的版本")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, record)
}

// GetAvailableRollback godoc
// @Summary 获取可回滚版本
// @Description 获取指定应用环境的可回滚版本
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param appId path int true "应用ID"
// @Param env path string true "环境"
// @Success 200 {object} response.Response{data=models.DeployRecord} "成功"
// @Failure 400 {object} response.Response "没有可回滚的版本"
// @Security BearerAuth
// @Router /deploy/rollback/{appId}/{env}/available [get]
func (h *DeployHandler) GetAvailableRollback(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("appId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "应用ID格式错误")
		return
	}

	envName := c.Param("env")

	record, err := h.service.GetAvailableRollback(c.Request.Context(), uint(appID), envName)
	if err != nil {
		if errors.Is(err, deploy.ErrNoRollbackVersion) {
			response.BadRequest(c, "没有可回滚的版本")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, record)
}

// GetLockStatus godoc
// @Summary 获取锁定状态
// @Description 获取指定应用环境的锁定状态
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param appId path int true "应用ID"
// @Param env path string true "环境"
// @Success 200 {object} response.Response{data=models.DeployLock} "成功"
// @Security BearerAuth
// @Router /deploy/locks/{appId}/{env} [get]
func (h *DeployHandler) GetLockStatus(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("appId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "应用ID格式错误")
		return
	}

	envName := c.Param("env")

	lock, err := h.service.GetLockStatus(c.Request.Context(), uint(appID), envName)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"locked": lock != nil,
		"lock":   lock,
	})
}

// ReleaseLockDTO 释放锁参数
type ReleaseLockDTO struct {
	Reason string `json:"reason" label:"释放原因"`
}

// ReleaseLock godoc
// @Summary 手动释放锁
// @Description 手动释放指定应用环境的锁
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param appId path int true "应用ID"
// @Param env path string true "环境"
// @Param request body ReleaseLockDTO false "释放原因"
// @Success 200 {object} response.Response "成功"
// @Failure 404 {object} response.Response "锁不存在"
// @Security BearerAuth
// @Router /deploy/locks/{appId}/{env}/release [post]
func (h *DeployHandler) ReleaseLock(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("appId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "应用ID格式错误")
		return
	}

	envName := c.Param("env")

	var dto ReleaseLockDTO
	c.ShouldBindJSON(&dto)

	userID, _ := middleware.GetUserID(c)

	reason := dto.Reason
	if reason == "" {
		reason = "手动释放"
	}

	if err := h.service.ReleaseLock(c.Request.Context(), uint(appID), envName, userID, reason); err != nil {
		if errors.Is(err, deploy.ErrLockNotFound) {
			response.NotFound(c, "锁不存在")
			return
		}
		response.InternalError(c, err.Error())
		return
	}

	response.OK(c)
}

// GetDeployWindowStatus godoc
// @Summary 获取发布窗口状态
// @Description 获取指定应用环境的发布窗口状态
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param appId path int true "应用ID"
// @Param env path string true "环境"
// @Success 200 {object} response.Response{data=deploy.DeployWindowStatus} "成功"
// @Security BearerAuth
// @Router /deploy/window/{appId}/{env} [get]
func (h *DeployHandler) GetDeployWindowStatus(c *gin.Context) {
	appID, err := strconv.ParseUint(c.Param("appId"), 10, 64)
	if err != nil {
		response.BadRequest(c, "应用ID格式错误")
		return
	}

	envName := c.Param("env")

	status, err := h.service.GetDeployWindowStatus(c.Request.Context(), uint(appID), envName)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, status)
}

// GetStats godoc
// @Summary 获取部署统计
// @Description 获取部署统计数据
// @Tags 发布管理
// @Accept json
// @Produce json
// @Param application_id query int false "应用ID"
// @Param env_name query string false "环境"
// @Param start_time query string false "开始时间 (2006-01-02)"
// @Param end_time query string false "结束时间 (2006-01-02)"
// @Success 200 {object} response.Response{data=repository.DeployStats} "成功"
// @Security BearerAuth
// @Router /deploy/stats [get]
func (h *DeployHandler) GetStats(c *gin.Context) {
	appID, _ := strconv.ParseUint(c.Query("application_id"), 10, 64)

	filter := repository.DeployStatsFilter{
		ApplicationID: uint(appID),
		EnvName:       c.Query("env_name"),
	}

	if startTime := c.Query("start_time"); startTime != "" {
		if t, err := time.Parse("2006-01-02", startTime); err == nil {
			filter.StartTime = t
		}
	}
	if endTime := c.Query("end_time"); endTime != "" {
		if t, err := time.Parse("2006-01-02", endTime); err == nil {
			filter.EndTime = t.Add(24 * time.Hour)
		}
	}

	stats, err := h.service.GetStats(c.Request.Context(), filter)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, stats)
}
