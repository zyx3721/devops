// Package feishu 飞书客户端封装
//
// 本包提供飞书开放平台 API 的封装，支持以下功能：
//   - 令牌管理（tenant_access_token / user_access_token）
//   - 消息发送（文本、卡片）
//   - 用户搜索和查询
//   - 群聊管理
//
// 文件结构:
//   - client.go  - 客户端核心结构和初始化
//   - token.go   - 令牌获取和刷新
//   - message.go - 消息发送
//   - user.go    - 用户搜索和查询
//   - chat.go    - 群聊管理
package feishu

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"
	"time"

	"devops/internal/config"
	"devops/internal/domain/notification/repository"
	"devops/pkg/logger"
)

// Client 飞书客户端封装
// 提供飞书 API 的统一访问接口
type Client struct {
	appID           string
	appSecret       string
	logger          *logger.Logger
	httpClient      *http.Client
	tenantToken     string
	tokenExpireAt   time.Time
	userToken       string
	userTokenExpire time.Time
	refreshToken    string
	mu              sync.RWMutex
	tokenRepo       *repository.FeishuUserTokenRepository
}

// NewClient 创建新的飞书客户端
// 使用配置文件中的 AppID 和 AppSecret
func NewClient(cfg *config.Config) *Client {
	log := logger.NewLogger(cfg.LogLevel)

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        cfg.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
			IdleConnTimeout:     cfg.IdleConnTimeout,
		},
	}

	log.Info("Feishu client initialized")

	return &Client{
		appID:      cfg.FeishuAppID,
		appSecret:  cfg.FeishuAppSecret,
		logger:     log,
		httpClient: httpClient,
	}
}

// NewClientWithApp 使用指定的 AppID 和 AppSecret 创建飞书客户端
// 用于动态创建不同应用的客户端
func NewClientWithApp(appID, appSecret string) *Client {
	log := logger.NewLogger("INFO")

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	log.Info("Feishu client initialized with app: %s", appID)

	return &Client{
		appID:      appID,
		appSecret:  appSecret,
		logger:     log,
		httpClient: httpClient,
	}
}

// SetTokenRepository 设置 token 仓储
// 用于持久化用户令牌
func (c *Client) SetTokenRepository(repo *repository.FeishuUserTokenRepository) {
	c.tokenRepo = repo
}

// GetLogger 获取日志记录器
func (c *Client) GetLogger() *logger.Logger {
	return c.logger
}

// GetAppID 获取应用ID
func (c *Client) GetAppID() string {
	return c.appID
}

// IsConfigured 检查客户端是否已正确配置
// 返回 true 表示 app_id 和 app_secret 都已设置
func (c *Client) IsConfigured() bool {
	return c.appID != "" && c.appSecret != ""
}

// generateUUID 生成 UUID
// 用于生成唯一标识符
func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
