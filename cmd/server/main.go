package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "devops/docs" // swagger docs
	"devops/internal/config"
	"devops/internal/infrastructure/cache"
	"devops/internal/infrastructure/database"
	"devops/internal/repository"
	"devops/internal/service/notification"

	// 引入所有模块的handler包以触发init()函数
	_ "devops/internal/modules/ai/handler"
	_ "devops/internal/modules/application/handler"
	_ "devops/internal/modules/approval/handler"
	_ "devops/internal/modules/artifact/handler"
	_ "devops/internal/modules/auth/handler"
	_ "devops/internal/modules/infrastructure/handler"
	_ "devops/internal/modules/logs/handler"
	_ "devops/internal/modules/monitoring/handler"
	_ "devops/internal/modules/notification/handler"
	_ "devops/internal/modules/pipeline/handler"
	_ "devops/internal/modules/resilience"
	_ "devops/internal/modules/security/handler"
	_ "devops/internal/modules/system/handler"
	"devops/pkg/ioc"
	"devops/pkg/logger"
	"devops/pkg/middleware"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title DevOps 统一运维平台 API
// @version 1.0
// @description DevOps 平台 API 文档，提供 Jenkins、K8s、消息通知、告警等功能的接口
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@devops.local

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /app/api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 输入 Bearer {token}

func main() {
	// 1. 加载基础配置
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// 2. 初始化日志
	log := logger.NewLogger(cfg.LogLevel)
	log.Info("Starting devops service...")

	// 2.1 初始化 Auth 中间件 (解决循环依赖，在 Config 加载后)
	if err := middleware.InitAuth(cfg.JWTSecret); err != nil {
		log.Fatal("Failed to init auth middleware: %v", err)
	}

	// 3. 初始化基础设施 (DB & Redis)
	db, err := database.InitMySQL(cfg)
	if err != nil {
		log.Fatal("Failed to initialize database: %v", err)
	}
	repository.SetDB(db)
	cfg.SetDB(db) // 向后兼容
	database.AutoMigrate(db)

	// 初始化默认消息模板
	if err := notification.InitDefaultTemplates(db); err != nil {
		log.Warn("Failed to init default templates: %v", err)
	}

	rdb, err := cache.InitRedis(cfg)
	if err != nil {
		log.Warn("Redis initialization failed (continuing with memory cache): %v", err)
	} else {
		repository.SetRedis(rdb)
		cfg.SetRedis(rdb) // 向后兼容
	}

	// 4. 初始化 IOC 容器 (业务逻辑注册)
	if err := ioc.Api.Init(); err != nil {
		log.Fatal("Failed to init ioc: %v", err)
	}

	// 5. 注册全局指标和健康检查
	cfg.Application.GinServer().GET("/metrics", gin.WrapH(promhttp.Handler()))
	cfg.Application.GinServer().GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 6. Swagger 文档
	cfg.Application.GinServer().GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("/swagger/doc.json"),
		ginSwagger.DefaultModelsExpandDepth(-1),
	))

	// 7. 启动 HTTP Server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      cfg.Application.GinServer(),
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info("Server starting on port %s...", cfg.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server: %v", err)
		}
	}()

	// 8. 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: %v", err)
	}
	log.Info("Server exited")
}
