package handler

import (
	"devops/internal/config"
	"devops/internal/modules/monitoring/repository"
	"devops/internal/service/kubernetes"
	"devops/internal/service/logs"
	"devops/pkg/ioc"
	"devops/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func init() {
	ioc.Api.RegisterContainer("LogsHandler", &LogsApiHandler{})
}

type LogsApiHandler struct {
	logHandler       *LogHandler
	exportHandler    *ExportHandler
	highlightHandler *HighlightHandler
	alertHandler     *AlertHandler
	templateHandler  *TemplateHandler
	bookmarkHandler  *BookmarkHandler
	statsHandler     *StatsHandler
	compareHandler   *CompareHandler
}

func (h *LogsApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := cfg.GetDB()
	clientMgr := kubernetes.NewK8sClientManager(db)

	// 创建 Repository
	silenceRepo := repository.NewAlertSilenceRepository(db)

	// 创建服务
	adapter := logs.NewK8sLogAdapter(clientMgr)
	queryService := logs.NewQueryService(adapter)
	streamService := logs.NewStreamService(adapter)
	parserService := logs.NewParserService()
	contextService := logs.NewContextService(adapter)
	savedQueryService := logs.NewSavedQueryService(db)
	alertService := logs.NewAlertService(db, adapter, nil, silenceRepo)
	bookmarkService := logs.NewBookmarkService(db)
	statsService := logs.NewStatsService(adapter)
	compareService := logs.NewCompareService(adapter)

	// 创建处理器
	h.logHandler = NewLogHandler(queryService, streamService, parserService, contextService, savedQueryService)
	h.exportHandler = NewExportHandler(queryService)
	h.highlightHandler = NewHighlightHandler(db)
	h.alertHandler = NewAlertHandler(alertService)
	h.templateHandler = NewTemplateHandler(db, parserService)
	h.bookmarkHandler = NewBookmarkHandler(bookmarkService)
	h.statsHandler = NewStatsHandler(statsService)
	h.compareHandler = NewCompareHandler(compareService)

	root := cfg.Application.GinRootRouter().Group("logs")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *LogsApiHandler) Register(r gin.IRouter) {
	// WebSocket 日志流
	r.GET("/stream", h.logHandler.StreamLogs)
	r.GET("/stream/multi", h.logHandler.StreamMultiPodLogs)

	// 日志查询
	r.GET("/query", h.logHandler.QueryLogs)

	// 日志上下文
	r.GET("/context", h.logHandler.GetLogContext)

	// 获取容器列表
	r.GET("/containers/:cluster_id/:namespace/:pod_name", h.logHandler.GetContainers)

	// 日志解析
	r.POST("/parse", h.logHandler.ParseLog)

	// 日志导出
	r.POST("/export", h.exportHandler.ExportLogs)
	r.GET("/export/:task_id", h.exportHandler.GetExportStatus)
	r.GET("/export/:task_id/download", h.exportHandler.DownloadExport)
	r.POST("/export/:task_id/cancel", h.exportHandler.CancelExport)

	// 染色规则
	r.GET("/highlight-rules", h.highlightHandler.ListHighlightRules)
	r.GET("/highlight-rules/presets", h.highlightHandler.GetPresetRules)
	r.POST("/highlight-rules", h.highlightHandler.CreateHighlightRule)
	r.PUT("/highlight-rules/:id", h.highlightHandler.UpdateHighlightRule)
	r.DELETE("/highlight-rules/:id", h.highlightHandler.DeleteHighlightRule)
	r.POST("/highlight-rules/:id/toggle", h.highlightHandler.ToggleHighlightRule)

	// 快捷查询
	r.GET("/saved-queries", h.logHandler.ListSavedQueries)
	r.POST("/saved-queries", h.logHandler.CreateSavedQuery)
	r.PUT("/saved-queries/:id", h.logHandler.UpdateSavedQuery)
	r.DELETE("/saved-queries/:id", h.logHandler.DeleteSavedQuery)
	r.POST("/saved-queries/:id/use", h.logHandler.UseSavedQuery)

	// 告警规则
	r.GET("/alert-rules", h.alertHandler.ListAlertRules)
	r.POST("/alert-rules", h.alertHandler.CreateAlertRule)
	r.PUT("/alert-rules/:id", h.alertHandler.UpdateAlertRule)
	r.DELETE("/alert-rules/:id", h.alertHandler.DeleteAlertRule)
	r.POST("/alert-rules/:id/toggle", h.alertHandler.ToggleAlertRule)
	r.GET("/alert-history", h.alertHandler.ListAlertHistory)

	// 解析模板
	r.GET("/templates", h.templateHandler.ListTemplates)
	r.GET("/templates/presets", h.templateHandler.GetPresetTemplates)
	r.POST("/templates", h.templateHandler.CreateTemplate)
	r.PUT("/templates/:id", h.templateHandler.UpdateTemplate)
	r.DELETE("/templates/:id", h.templateHandler.DeleteTemplate)
	r.POST("/templates/test", h.templateHandler.TestTemplate)
	// 兼容前端路径
	r.GET("/parse-templates", h.templateHandler.ListTemplates)
	r.GET("/parse-templates/presets", h.templateHandler.GetPresetTemplates)
	r.POST("/parse-templates", h.templateHandler.CreateTemplate)
	r.PUT("/parse-templates/:id", h.templateHandler.UpdateTemplate)
	r.DELETE("/parse-templates/:id", h.templateHandler.DeleteTemplate)
	r.POST("/parse-templates/test", h.templateHandler.TestTemplate)

	// 书签
	r.GET("/bookmarks", h.bookmarkHandler.ListBookmarks)
	r.POST("/bookmarks", h.bookmarkHandler.CreateBookmark)
	r.PUT("/bookmarks/:id", h.bookmarkHandler.UpdateBookmark)
	r.DELETE("/bookmarks/:id", h.bookmarkHandler.DeleteBookmark)
	r.POST("/bookmarks/:id/share", h.bookmarkHandler.ShareBookmark)
	r.GET("/bookmarks/shared/:token", h.bookmarkHandler.GetSharedBookmark)

	// 日志统计
	r.GET("/stats", h.statsHandler.GetStats)

	// 日志对比
	r.POST("/compare", h.compareHandler.CompareLogs)
}
