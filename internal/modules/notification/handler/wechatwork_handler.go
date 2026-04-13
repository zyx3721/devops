package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"devops/internal/config"
	"devops/internal/domain/notification/service/wechatwork"
	"devops/internal/models"
	"devops/internal/repository"
	"devops/pkg/ioc"
	"devops/pkg/logger"
)

func init() {
	ioc.Api.RegisterContainer("WechatWorkHandler", &WechatWorkApiHandler{})
}

type WechatWorkApiHandler struct {
	handler *WechatWorkHandler
}

func (h *WechatWorkApiHandler) Init() error {
	cfg, _ := config.LoadConfig()
	h.handler = NewWechatWorkHandler(cfg.GetDB())

	root := cfg.Application.GinRootRouter().Group("wechatwork")
	h.Register(root)
	return nil
}

func (h *WechatWorkApiHandler) Register(r gin.IRouter) {
	r.POST("/send-message", h.handler.SendMessage)
	r.POST("/send-webhook", h.handler.SendWebhook)
	r.POST("/user/search", h.handler.SearchUser)
	r.GET("/logs", h.handler.ListMessageLogs)

	// 应用管理
	app := r.Group("/app")
	{
		app.GET("", h.handler.ListApps)
		app.GET("/:id", h.handler.GetApp)
		app.POST("", h.handler.CreateApp)
		app.PUT("/:id", h.handler.UpdateApp)
		app.DELETE("/:id", h.handler.DeleteApp)
		app.POST("/:id/default", h.handler.SetDefaultApp)
		app.GET("/:id/bindings", h.handler.GetAppBindings)
	}

	// 机器人管理
	bot := r.Group("/bot")
	{
		bot.GET("", h.handler.ListBots)
		bot.GET("/:id", h.handler.GetBot)
		bot.POST("", h.handler.CreateBot)
		bot.PUT("/:id", h.handler.UpdateBot)
		bot.DELETE("/:id", h.handler.DeleteBot)
	}
}

type WechatWorkHandler struct {
	logger  *logger.Logger
	appRepo *repository.WechatWorkAppRepository
	botRepo *repository.WechatWorkBotRepository
	logRepo *repository.WechatWorkMessageLogRepository
	db      *gorm.DB
}

func NewWechatWorkHandler(db *gorm.DB) *WechatWorkHandler {
	return &WechatWorkHandler{
		logger:  logger.NewLogger("INFO"),
		appRepo: repository.NewWechatWorkAppRepository(db),
		botRepo: repository.NewWechatWorkBotRepository(db),
		logRepo: repository.NewWechatWorkMessageLogRepository(db),
		db:      db,
	}
}

// SendMessage 发送应用消息
func (h *WechatWorkHandler) SendMessage(c *gin.Context) {
	var req struct {
		AppID   uint   `json:"app_id"`
		ToUser  string `json:"to_user"`
		ToParty string `json:"to_party"`
		ToTag   string `json:"to_tag"`
		MsgType string `json:"msg_type"`
		Content string `json:"content"`
		Title   string `json:"title"`
		URL     string `json:"url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var app *models.WechatWorkApp
	var err error
	if req.AppID > 0 {
		app, err = h.appRepo.GetByID(c.Request.Context(), req.AppID)
	} else {
		app, err = h.appRepo.GetDefault(c.Request.Context())
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "应用不存在"})
		return
	}

	client := wechatwork.NewClient(app.CorpID, app.AgentID, app.Secret)

	msg := &wechatwork.AppMessage{
		ToUser:  req.ToUser,
		ToParty: req.ToParty,
		ToTag:   req.ToTag,
		MsgType: req.MsgType,
	}

	switch req.MsgType {
	case "text":
		msg.Text = &wechatwork.TextMsg{Content: req.Content}
	case "markdown":
		msg.Markdown = &wechatwork.MarkdownMsg{Content: req.Content}
	case "textcard":
		msg.TextCard = &wechatwork.TextCardMsg{Title: req.Title, Description: req.Content, URL: req.URL}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "不支持的消息类型"})
		return
	}

	err = client.SendMessage(c.Request.Context(), msg)

	logEntry := &models.WechatWorkMessageLog{
		MsgType: req.MsgType,
		ToUser:  req.ToUser,
		ToParty: req.ToParty,
		ToTag:   req.ToTag,
		Content: req.Content,
		Title:   req.Title,
		Source:  "manual",
		Status:  "success",
		AppID:   app.ID,
	}
	if err != nil {
		logEntry.Status = "failed"
		logEntry.ErrorMsg = err.Error()
	}
	h.logRepo.Create(c.Request.Context(), logEntry)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "发送成功"})
}

// SendWebhook 发送Webhook消息
func (h *WechatWorkHandler) SendWebhook(c *gin.Context) {
	var req struct {
		BotID               uint            `json:"bot_id"`
		MsgType             string          `json:"msg_type"`
		Content             json.RawMessage `json:"content"`
		MentionedList       []string        `json:"mentioned_list"`
		MentionedMobileList []string        `json:"mentioned_mobile_list"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	bot, err := h.botRepo.GetByID(c.Request.Context(), req.BotID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "机器人不存在"})
		return
	}

	client := wechatwork.NewClient("", 0, "")

	msg := &wechatwork.WebhookMessage{MsgType: req.MsgType}

	var content map[string]string
	json.Unmarshal(req.Content, &content)

	switch req.MsgType {
	case "text":
		msg.Text = &wechatwork.WebhookText{
			Content:             content["content"],
			MentionedList:       req.MentionedList,
			MentionedMobileList: req.MentionedMobileList,
		}
	case "markdown":
		msg.Markdown = &wechatwork.WebhookMarkdown{Content: content["content"]}
	}

	err = client.SendWebhookMessage(c.Request.Context(), bot.WebhookURL, msg)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "发送成功"})
}

// SearchUser 搜索用户
func (h *WechatWorkHandler) SearchUser(c *gin.Context) {
	var req struct {
		Query string `json:"query"`
		AppID uint   `json:"app_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	var app *models.WechatWorkApp
	var err error
	if req.AppID > 0 {
		app, err = h.appRepo.GetByID(c.Request.Context(), req.AppID)
	} else {
		app, err = h.appRepo.GetDefault(c.Request.Context())
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "应用不存在"})
		return
	}

	client := wechatwork.NewClient(app.CorpID, app.AgentID, app.Secret)
	users, err := client.SearchUser(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": users})
}

// ListMessageLogs 获取消息日志
func (h *WechatWorkHandler) ListMessageLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	msgType := c.Query("msg_type")
	source := c.Query("source")

	list, total, err := h.logRepo.List(c.Request.Context(), page, pageSize, msgType, source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": gin.H{"list": list, "total": total}})
}

// 应用管理
func (h *WechatWorkHandler) ListApps(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))

	list, total, err := h.appRepo.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": gin.H{"list": list, "total": total}})
}

func (h *WechatWorkHandler) GetApp(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	app, err := h.appRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": app})
}

func (h *WechatWorkHandler) CreateApp(c *gin.Context) {
	var app models.WechatWorkApp
	if err := c.ShouldBindJSON(&app); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	app.ID = 0
	if app.Status == "" {
		app.Status = "active"
	}
	if err := h.appRepo.Create(c.Request.Context(), &app); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": app})
}

func (h *WechatWorkHandler) UpdateApp(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var app models.WechatWorkApp
	if err := c.ShouldBindJSON(&app); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	app.ID = uint(id)
	if err := h.appRepo.Update(c.Request.Context(), &app); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": app})
}

func (h *WechatWorkHandler) DeleteApp(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.appRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

func (h *WechatWorkHandler) SetDefaultApp(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.appRepo.SetDefault(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}

// GetAppBindings 获取企业微信应用绑定的 Jenkins 实例和 K8s 集群
func (h *WechatWorkHandler) GetAppBindings(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	// 获取绑定的 Jenkins 实例
	var jenkinsBindings []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	h.db.Table("jenkins_instances").
		Select("jenkins_instances.id, jenkins_instances.name, jenkins_instances.url").
		Joins("JOIN jenkins_wechat_work_apps ON jenkins_wechat_work_apps.jenkins_instance_id = jenkins_instances.id").
		Where("jenkins_wechat_work_apps.wechat_work_app_id = ?", id).
		Scan(&jenkinsBindings)

	// 获取绑定的 K8s 集群
	var k8sBindings []struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	h.db.Table("k8s_clusters").
		Select("k8s_clusters.id, k8s_clusters.name").
		Joins("JOIN k8s_cluster_wechat_work_apps ON k8s_cluster_wechat_work_apps.k8s_cluster_id = k8s_clusters.id").
		Where("k8s_cluster_wechat_work_apps.wechat_work_app_id = ?", id).
		Scan(&k8sBindings)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"jenkins_instances": jenkinsBindings,
			"k8s_clusters":      k8sBindings,
		},
	})
}

// 机器人管理
func (h *WechatWorkHandler) ListBots(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "100"))

	list, total, err := h.botRepo.List(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": gin.H{"list": list, "total": total}})
}

func (h *WechatWorkHandler) GetBot(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	bot, err := h.botRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": bot})
}

func (h *WechatWorkHandler) CreateBot(c *gin.Context) {
	var bot models.WechatWorkBot
	if err := c.ShouldBindJSON(&bot); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	bot.ID = 0
	if bot.Status == "" {
		bot.Status = "active"
	}
	if err := h.botRepo.Create(c.Request.Context(), &bot); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": bot})
}

func (h *WechatWorkHandler) UpdateBot(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var bot models.WechatWorkBot
	if err := c.ShouldBindJSON(&bot); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	bot.ID = uint(id)
	if err := h.botRepo.Update(c.Request.Context(), &bot); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": bot})
}

func (h *WechatWorkHandler) DeleteBot(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := h.botRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success"})
}
