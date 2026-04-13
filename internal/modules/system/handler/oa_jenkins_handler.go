package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/domain/notification/service/feishu"
	"devops/internal/models"
	"devops/internal/repository"
	"devops/internal/service/notification"
	"devops/internal/service/oa"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
)

func init() {
	ioc.Api.RegisterContainer("OAJenkinsHandler", &OAJenkinsApiHandler{})
}

type OAJenkinsApiHandler struct {
	handler *OAJenkinsHandler
}

func (h *OAJenkinsApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	db := repository.GetDB(context.Background())
	h.handler = NewOAJenkinsHandler(db)

	root := cfg.Application.GinRootRouter().Group("jk")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *OAJenkinsApiHandler) Register(r gin.IRouter) {
	r.POST("/test-flow", middleware.RequireAdmin(), h.handler.TestFlow)
}

type OAJenkinsHandler struct {
	oaNotifyRepo    *repository.OANotifyConfigRepository
	feishuAppRepo   *repository.FeishuAppRepository
	templateService *notification.TemplateService
}

func NewOAJenkinsHandler(db repository.GormDB) *OAJenkinsHandler {
	// GormDB is interface, we need *gorm.DB
	// repository.GetDB returns *gorm.DB, but let's check what NewOAJenkinsHandler received.
	// In Init(), I passed *gorm.DB.
	// But wait, repository.GormDB interface in base.go:
	// type GormDB interface { WithContext(ctx context.Context) *gorm.DB }
	// *gorm.DB satisfies this.

	// However, the repos need *gorm.DB.
	gormDB := db.WithContext(context.Background())

	return &OAJenkinsHandler{
		oaNotifyRepo:    repository.NewOANotifyConfigRepository(gormDB),
		feishuAppRepo:   repository.NewFeishuAppRepository(gormDB),
		templateService: notification.NewTemplateService(repository.NewMessageTemplateRepository(gormDB)),
	}
}

type TestFlowRequest struct {
	ReceiveID     string `json:"receive_id"`
	ReceiveIDType string `json:"receive_id_type"`
}

func (h *OAJenkinsHandler) TestFlow(c *gin.Context) {
	var req TestFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "error": err.Error()})
		return
	}

	if req.ReceiveID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "error": "receive_id is required"})
		return
	}

	go func() {
		ctx := context.Background()
		h.simulateOAFlow(ctx, req.ReceiveID, req.ReceiveIDType)
	}()

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "Test flow started"})
}

// simulateOAFlow 模拟 OA 推送 -> 生成卡片 -> 发送卡片 的流程
func (h *OAJenkinsHandler) simulateOAFlow(ctx context.Context, receiveID, receiveIDType string) {
	oaData, err := oa.GetLatestJson()
	if err != nil {
		h.sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("❌ 获取 OA 数据失败: %v", err))
		return
	}

	dummy := &oa.JenkinsJob{}
	jobs, err := dummy.HandleLatestJson(oaData)
	if err != nil {
		h.sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("❌ 解析 OA 数据失败: %v", err))
		return
	}

	if len(jobs) == 0 {
		h.sendFeishuMessage(ctx, receiveID, receiveIDType, "⚠️ OA 数据中没有找到 Job")
		return
	}

	var services []feishu.Service
	for _, job := range jobs {
		actions := []string{"gray", "rollback", "restart"}

		services = append(services, feishu.Service{
			Name:     job.JobName + "-prod",
			ObjectID: job.JobName,
			Actions:  actions,
			Branches: []string{job.JobBranch},
		})
	}

	requestID := fmt.Sprintf("req_test_%d", time.Now().UnixNano())
	cardReq := feishu.GrayCardRequest{
		Title:         "应用发布申请 (测试)",
		Services:      services,
		ReceiveID:     receiveID,
		ReceiveIDType: receiveIDType,
	}

	feishu.GlobalStore.Save(requestID, cardReq)

	// 获取客户端
	client, err := h.getFeishuClient(ctx)
	if err != nil {
		h.sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("❌ 获取飞书客户端失败: %v", err))
		return
	}

	// 使用模板渲染
	data := map[string]interface{}{
		"Title":     "应用发布申请 (测试)",
		"Services":  services,
		"RequestID": requestID,
	}
	cardContent, err := h.templateService.Render(ctx, "JENKINS_FLOW_CARD", data)
	if err != nil {
		h.sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("❌ 渲染卡片失败: %v", err))
		return
	}

	err = client.SendMessage(ctx, receiveID, receiveIDType, "interactive", cardContent)
	if err != nil {
		h.sendFeishuMessage(ctx, receiveID, receiveIDType, fmt.Sprintf("❌ 发送卡片失败: %v", err))
		return
	}

	h.sendFeishuMessage(ctx, receiveID, receiveIDType, "✅ 卡片已发送，请点击卡片按钮测试 Jenkins 触发")
}

func (h *OAJenkinsHandler) getFeishuClient(ctx context.Context) (*feishu.Client, error) {
	notifyConfig, err := h.oaNotifyRepo.GetDefault(ctx)
	if err != nil {
		return nil, err
	}
	// 如果没有默认配置，尝试直接获取默认飞书应用
	if notifyConfig == nil {
		app, err := h.feishuAppRepo.GetDefault(ctx)
		if err != nil {
			return nil, err
		}
		if app == nil {
			return nil, fmt.Errorf("no default feishu app found")
		}
		return feishu.NewClientWithApp(app.AppID, app.AppSecret), nil
	}

	var feishuApp *models.FeishuApp
	if notifyConfig.AppID > 0 {
		feishuApp, err = h.feishuAppRepo.GetByID(ctx, notifyConfig.AppID)
	} else {
		feishuApp, err = h.feishuAppRepo.GetDefault(ctx)
	}

	if err != nil {
		return nil, err
	}
	if feishuApp == nil {
		return nil, fmt.Errorf("no feishu app found")
	}

	return feishu.NewClientWithApp(feishuApp.AppID, feishuApp.AppSecret), nil
}

func (h *OAJenkinsHandler) sendFeishuMessage(ctx context.Context, receiveID, receiveIDType, content string) {
	client, err := h.getFeishuClient(ctx)
	if err != nil {
		fmt.Printf("Failed to get feishu client: %v. Content: %s\n", err, content)
		return
	}

	msgContent := map[string]interface{}{
		"text": content,
	}
	msgBytes, _ := json.Marshal(msgContent)

	client.SendMessage(ctx, receiveID, receiveIDType, "text", string(msgBytes))
}
