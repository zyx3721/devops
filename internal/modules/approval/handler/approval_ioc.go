package handler

import (
	"devops/internal/config"
	"devops/internal/domain/notification/service/feishu"
	appHandler "devops/internal/modules/application/handler"
	"devops/internal/repository"
	"devops/internal/service/approval"
	"devops/internal/service/deploy"
	"devops/pkg/ioc"
	"devops/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func init() {
	ioc.Api.RegisterContainer("ApprovalHandler", &ApprovalIOC{})
}

type ApprovalIOC struct {
	ruleHandler         *ApprovalRuleHandler
	windowHandler       *DeployWindowHandler
	approvalHandler     *ApprovalHandler
	lockHandler         *appHandler.DeployLockHandler
	chainHandler        *ApprovalChainHandler
	timeoutChecker      *approval.TimeoutChecker
	lockCleaner         *deploy.LockCleaner
	chainTimeoutHandler *approval.TimeoutHandler
	callbackHandler     *approval.CallbackHandler
}

func (h *ApprovalIOC) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()

	// 创建 Repository
	ruleRepo := repository.NewApprovalRuleRepository(db)
	windowRepo := repository.NewDeployWindowRepository(db)
	chainRepo := repository.NewApprovalChainRepository(db)
	nodeRepo := repository.NewApprovalNodeRepository(db)
	instanceRepo := repository.NewApprovalInstanceRepository(db)
	nodeInstanceRepo := repository.NewApprovalNodeInstanceRepository(db)
	actionRepo := repository.NewApprovalActionRepository(db)

	// 创建 Service
	ruleService := approval.NewRuleService(ruleRepo)
	windowService := approval.NewWindowService(windowRepo)

	// 创建通知服务
	notificationService := approval.NewNotificationService(db)

	approvalService := approval.NewApprovalService(db, ruleService, notificationService)
	lockService := deploy.NewLockService(db)

	// 创建审批链相关 Service
	chainService := approval.NewChainService(chainRepo, nodeRepo)
	nodeExecutor := approval.NewNodeExecutor(nodeInstanceRepo, actionRepo, instanceRepo)
	approverResolver := approval.NewApproverResolver(db)
	instanceService := approval.NewInstanceService(instanceRepo, nodeInstanceRepo, chainService, nodeExecutor, approverResolver)

	// 创建回调处理器并注册到飞书服务
	h.callbackHandler = approval.NewCallbackHandler(nodeExecutor)
	feishu.SetApprovalCallbackHandler(h.callbackHandler)

	// 创建 Handler
	h.ruleHandler = NewApprovalRuleHandler(ruleService)
	h.windowHandler = NewDeployWindowHandler(windowService)
	h.approvalHandler = NewApprovalHandler(approvalService)
	h.lockHandler = appHandler.NewDeployLockHandler(lockService)
	h.chainHandler = NewApprovalChainHandler(chainService, instanceService, nodeExecutor)

	// 创建后台任务
	h.timeoutChecker = approval.NewTimeoutChecker(db, ruleService)
	h.lockCleaner = deploy.NewLockCleaner(lockService)
	h.chainTimeoutHandler = approval.NewTimeoutHandler(nodeInstanceRepo, instanceRepo, nodeExecutor)

	// 注册路由
	root := cfg.Application.GinRootRouter().Group("approval")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	// 注册发布锁路由
	lockRoot := cfg.Application.GinRootRouter().Group("deploy/locks")
	lockRoot.Use(middleware.AuthMiddleware())
	h.RegisterLockRoutes(lockRoot)

	// 启动后台任务
	go h.timeoutChecker.Start()
	go h.lockCleaner.Start()
	h.chainTimeoutHandler.Start()

	return nil
}

func (h *ApprovalIOC) Register(r gin.IRouter) {
	// 审批规则 - 管理员权限
	rules := r.Group("/rules")
	{
		rules.GET("", h.ruleHandler.List)
		rules.GET("/:id", h.ruleHandler.GetByID)
		rules.POST("", middleware.RequireAdmin(), h.ruleHandler.Create)
		rules.PUT("/:id", middleware.RequireAdmin(), h.ruleHandler.Update)
		rules.DELETE("/:id", middleware.RequireAdmin(), h.ruleHandler.Delete)
	}

	// 发布窗口 - 管理员权限
	windows := r.Group("/windows")
	{
		windows.GET("", h.windowHandler.List)
		windows.GET("/:id", h.windowHandler.GetByID)
		windows.GET("/check", h.windowHandler.CheckWindow)
		windows.POST("", middleware.RequireAdmin(), h.windowHandler.Create)
		windows.PUT("/:id", middleware.RequireAdmin(), h.windowHandler.Update)
		windows.DELETE("/:id", middleware.RequireAdmin(), h.windowHandler.Delete)
	}

	// 审批链管理 - 管理员权限
	chains := r.Group("/chains")
	{
		chains.GET("", h.chainHandler.ListChains)
		chains.GET("/:id", h.chainHandler.GetChain)
		chains.POST("", middleware.RequireAdmin(), h.chainHandler.CreateChain)
		chains.PUT("/:id", middleware.RequireAdmin(), h.chainHandler.UpdateChain)
		chains.DELETE("/:id", middleware.RequireAdmin(), h.chainHandler.DeleteChain)
		chains.POST("/:id/nodes", middleware.RequireAdmin(), h.chainHandler.AddNode)
		chains.PUT("/:id/nodes/:nodeId", middleware.RequireAdmin(), h.chainHandler.UpdateNode)
		chains.DELETE("/:id/nodes/:nodeId", middleware.RequireAdmin(), h.chainHandler.DeleteNode)
		chains.PUT("/:id/nodes/reorder", middleware.RequireAdmin(), h.chainHandler.ReorderNodes)
		chains.POST("/:id/test", middleware.RequireAdmin(), h.chainHandler.TestChain)
	}

	// 审批实例 - 查看所有人可访问，取消需要权限
	instances := r.Group("/instances")
	{
		instances.GET("", h.chainHandler.ListInstances)
		instances.GET("/:id", h.chainHandler.GetInstance)
		instances.POST("/:id/cancel", h.chainHandler.CancelInstance)
	}

	// 审批节点操作 - 审批人可操作
	nodes := r.Group("/nodes")
	{
		nodes.POST("/:nodeInstanceId/approve", h.chainHandler.ApproveNode)
		nodes.POST("/:nodeInstanceId/reject", h.chainHandler.RejectNode)
		nodes.POST("/:nodeInstanceId/transfer", h.chainHandler.TransferNode)
	}

	// 审批链待审批列表
	r.GET("/chain/pending", h.chainHandler.GetPendingApprovals)

	// 审批统计
	r.GET("/stats", h.chainHandler.GetStats)

	// 旧版审批操作（保持兼容）
	r.GET("/pending", h.approvalHandler.GetPendingList)
	r.GET("/history", h.approvalHandler.GetHistory)
	r.POST("/:id/approve", h.approvalHandler.Approve)
	r.POST("/:id/reject", h.approvalHandler.Reject)
	r.POST("/:id/cancel", h.approvalHandler.Cancel)
	r.GET("/:id/records", h.approvalHandler.GetApprovalRecords)
}

func (h *ApprovalIOC) RegisterLockRoutes(r gin.IRouter) {
	r.GET("", h.lockHandler.List)
	r.GET("/check", h.lockHandler.CheckLock)
	r.POST("/release", middleware.RequireAdmin(), h.lockHandler.ForceRelease)
}
