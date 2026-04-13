package feishu

import (
	"context"
	"fmt"
	"sync"

	"devops/internal/config"
	"devops/internal/models"
	"devops/pkg/logger"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkevent "github.com/larksuite/oapi-sdk-go/v3/event"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
	"gorm.io/gorm"
)

var cardActionHandler func(context.Context, *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error)

// SetCardActionHandler 设置卡片交互回调处理器
func SetCardActionHandler(h func(context.Context, *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error)) {
	cardActionHandler = h
}

// CallbackManager 管理多个飞书应用的回调
type CallbackManager struct {
	db      *gorm.DB
	log     *logger.Logger
	clients map[string]*larkws.Client // key: app_id
	mu      sync.RWMutex
}

var callbackManager *CallbackManager

// InitCallbackManager 初始化回调管理器
func InitCallbackManager(db *gorm.DB, cfg *config.Config) *CallbackManager {
	callbackManager = &CallbackManager{
		db:      db,
		log:     logger.NewLogger(cfg.LogLevel),
		clients: make(map[string]*larkws.Client),
	}
	return callbackManager
}

// GetCallbackManager 获取回调管理器
func GetCallbackManager() *CallbackManager {
	return callbackManager
}

// StartAllCallbacks 启动所有飞书应用的回调
func (m *CallbackManager) StartAllCallbacks() {
	if m.db == nil {
		m.log.Error("Database not initialized")
		return
	}

	var apps []models.FeishuApp
	if err := m.db.Where("status = ?", "active").Find(&apps).Error; err != nil {
		m.log.Error("Failed to get Feishu apps: %v", err)
		return
	}

	m.log.Info("Starting callbacks for %d Feishu apps", len(apps))

	for _, app := range apps {
		m.log.Info("Starting callback for app: %s (AppID: %s)", app.Name, app.AppID)
		m.StartCallback(app.AppID, app.AppSecret, app.Name)
	}
}

// StartCallback 为单个应用启动回调
func (m *CallbackManager) StartCallback(appID, appSecret, appName string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[appID]; exists {
		m.log.Info("Callback for app %s (%s) already running", appName, appID)
		return
	}

	eventHandler := dispatcher.NewEventDispatcher("", "").
		OnP2CardActionTrigger(func(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
			if cardActionHandler != nil {
				return cardActionHandler(ctx, event)
			}
			m.log.Info("Card action trigger received: %s", larkcore.Prettify(event))
			return nil, nil
		}).
		OnP2CardURLPreviewGet(func(ctx context.Context, event *callback.URLPreviewGetEvent) (*callback.URLPreviewGetResponse, error) {
			m.log.Info("URL preview request received for URL: %s", larkcore.Prettify(event))
			return nil, nil
		}).
		OnCustomizedEvent("im.chat.access_event.bot_p2p_chat_entered_v1", func(ctx context.Context, event *larkevent.EventReq) error {
			m.log.Info("User entered P2P chat with bot: %s", string(event.Body))
			return nil
		}).
		OnCustomizedEvent("im.message.receive_v1", func(ctx context.Context, event *larkevent.EventReq) error {
			m.log.Info("Message received: %s", string(event.Body))
			return nil
		})

	cli := larkws.NewClient(appID, appSecret,
		larkws.WithEventHandler(eventHandler),
		larkws.WithLogLevel(larkcore.LogLevelInfo),
	)

	m.clients[appID] = cli

	go func() {
		m.log.Info("Starting Feishu WebSocket callback for app: %s (%s)", appName, appID)
		err := cli.Start(context.Background())
		if err != nil {
			m.log.Error("Failed to start Feishu WebSocket client for %s (%s): %v", appName, appID, err)
			m.mu.Lock()
			delete(m.clients, appID)
			m.mu.Unlock()
		} else {
			m.log.Info("Feishu WebSocket callback started successfully for app: %s (%s)", appName, appID)
		}
	}()
}

// StopCallback 停止单个应用的回调
func (m *CallbackManager) StopCallback(appID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if cli, exists := m.clients[appID]; exists {
		_ = cli
		delete(m.clients, appID)
		m.log.Info("Stopped callback for app: %s", appID)
	}
}

// GetRunningApps 获取正在运行回调的应用列表
func (m *CallbackManager) GetRunningApps() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	apps := make([]string, 0, len(m.clients))
	for appID := range m.clients {
		apps = append(apps, appID)
	}
	return apps
}

// RefreshCallbacks 刷新回调（重新加载数据库中的应用）
func (m *CallbackManager) RefreshCallbacks() error {
	if m.db == nil {
		return fmt.Errorf("database not initialized")
	}

	var apps []models.FeishuApp
	if err := m.db.Where("status = ?", "active").Find(&apps).Error; err != nil {
		return err
	}

	activeAppIDs := make(map[string]bool)
	for _, app := range apps {
		activeAppIDs[app.AppID] = true
	}

	m.mu.RLock()
	toStop := []string{}
	for appID := range m.clients {
		if !activeAppIDs[appID] {
			toStop = append(toStop, appID)
		}
	}
	m.mu.RUnlock()

	for _, appID := range toStop {
		m.StopCallback(appID)
	}

	for _, app := range apps {
		m.StartCallback(app.AppID, app.AppSecret, app.Name)
	}

	return nil
}

// RegisterCallback 注册飞书回调（兼容旧接口，使用默认配置）
func RegisterCallback(cfg *config.Config) {
	log := logger.NewLogger(cfg.LogLevel)

	eventHandler := dispatcher.NewEventDispatcher("", "").
		OnP2CardActionTrigger(func(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
			if cardActionHandler != nil {
				return cardActionHandler(ctx, event)
			}
			log.Info("Card action trigger received: %s", larkcore.Prettify(event))
			return nil, nil
		}).
		OnP2CardURLPreviewGet(func(ctx context.Context, event *callback.URLPreviewGetEvent) (*callback.URLPreviewGetResponse, error) {
			log.Info("URL preview request received for URL: %s", larkcore.Prettify(event))
			return nil, nil
		}).
		OnCustomizedEvent("im.chat.access_event.bot_p2p_chat_entered_v1", func(ctx context.Context, event *larkevent.EventReq) error {
			log.Info("User entered P2P chat with bot: %s", string(event.Body))
			return nil
		}).
		OnCustomizedEvent("im.message.receive_v1", func(ctx context.Context, event *larkevent.EventReq) error {
			log.Info("Message received: %s", string(event.Body))
			return nil
		})

	cli := larkws.NewClient(cfg.FeishuAppID, cfg.FeishuAppSecret,
		larkws.WithEventHandler(eventHandler),
		larkws.WithLogLevel(larkcore.LogLevelDebug),
	)

	err := cli.Start(context.Background())
	if err != nil {
		log.Error("Failed to start Feishu WebSocket client: %v", err)
	}
}
