package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/domain/notification/service/feishu"
	"devops/internal/models"
	"devops/internal/repository"
)

// SearchUser 搜索用户
func (h *FeishuHandler) SearchUser(c *gin.Context) {
	var req struct {
		Query string `json:"query"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if req.Query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "query is required"})
		return
	}

	users, err := h.client.SearchUser(c.Request.Context(), req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data":    users,
	})
}

// GetUser 获取用户信息
func (h *FeishuHandler) GetUser(c *gin.Context) {
	userID := c.Param("id")
	userIDType := c.DefaultQuery("user_id_type", "user_id")

	user, err := h.client.GetUserByID(c.Request.Context(), userID, userIDType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data":    user,
	})
}

// ListChats 获取群列表
func (h *FeishuHandler) ListChats(c *gin.Context) {
	pageToken := c.Query("page_token")
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	chats, nextPageToken, err := h.client.GetChatList(c.Request.Context(), pageToken, pageSize)
	if err != nil {
		// 如果是配置问题，返回空列表而不是错误
		if err.Error() == "feishu app_id or app_secret not configured" ||
			err.Error() == "feishu not configured or token unavailable: feishu app_id or app_secret not configured" {
			h.logger.Debug("Feishu not configured, returning empty chat list")
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "Success",
				"data": gin.H{
					"list":       []map[string]any{},
					"page_token": "",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":       chats,
			"page_token": nextPageToken,
		},
	})
}

// CreateChat 创建群聊
func (h *FeishuHandler) CreateChat(c *gin.Context) {
	var req struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		UserIDs     []string `json:"user_ids"`
		UserIDType  string   `json:"user_id_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "name is required"})
		return
	}

	if req.UserIDType == "" {
		req.UserIDType = "user_id"
	}

	chatID, err := h.client.CreateChat(c.Request.Context(), req.Name, req.Description, req.UserIDs, req.UserIDType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"chat_id": chatID,
		},
	})
}

// AddChatMembers 添加群成员
func (h *FeishuHandler) AddChatMembers(c *gin.Context) {
	chatID := c.Param("id")

	var req struct {
		UserIDs    []string `json:"user_ids"`
		UserIDType string   `json:"user_id_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if len(req.UserIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "user_ids is required"})
		return
	}

	if req.UserIDType == "" {
		req.UserIDType = "user_id"
	}

	err := h.client.AddChatMembers(c.Request.Context(), chatID, req.UserIDs, req.UserIDType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
	})
}

// SetUserToken 设置用户访问令牌
func (h *FeishuHandler) SetUserToken(c *gin.Context) {
	var req struct {
		AppID        string `json:"app_id"`
		UserToken    string `json:"user_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}

	if req.UserToken == "" && req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "user_token or refresh_token is required"})
		return
	}

	if req.AppID != "" {
		cfg, _ := config.LoadConfig()
		tokenRepo := repository.NewFeishuUserTokenRepository(cfg.GetDB())
		token := &models.FeishuUserToken{
			AppID:        req.AppID,
			AccessToken:  req.UserToken,
			RefreshToken: req.RefreshToken,
			ExpiresAt:    time.Now().Add(2 * time.Hour),
		}
		if err := tokenRepo.Save(c.Request.Context(), token); err != nil {
			h.logger.Error("Failed to save user token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存失败: " + err.Error()})
			return
		}
		h.logger.Info("User token saved for app: %s", req.AppID)
	}

	h.client.SetUserToken(req.UserToken, req.RefreshToken)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "User token set successfully",
		"data": gin.H{
			"has_refresh_token": req.RefreshToken != "",
		},
	})
}

// GetUserTokenStatus 获取用户令牌状态
func (h *FeishuHandler) GetUserTokenStatus(c *gin.Context) {
	appID := c.Query("app_id")

	if appID != "" {
		cfg, _ := config.LoadConfig()
		tokenRepo := repository.NewFeishuUserTokenRepository(cfg.GetDB())
		token, err := tokenRepo.GetByAppID(c.Request.Context(), appID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "Success",
				"data": gin.H{
					"has_token":     false,
					"is_valid":      false,
					"refresh_token": "",
					"expires_at":    "",
					"status":        "未配置",
				},
			})
			return
		}

		refreshToken := token.RefreshToken
		if len(refreshToken) > 20 {
			refreshToken = refreshToken[:10] + "..." + refreshToken[len(refreshToken)-10:]
		}

		isValid := token.ExpiresAt.After(time.Now())
		status := "正常"
		if !isValid {
			status = "已过期，请重新授权"
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "Success",
			"data": gin.H{
				"has_token":     token.AccessToken != "" || token.RefreshToken != "",
				"is_valid":      isValid,
				"refresh_token": refreshToken,
				"expires_at":    token.ExpiresAt.Format("2006-01-02 15:04:05"),
				"status":        status,
			},
		})
		return
	}

	hasToken := h.client.HasUserToken()
	refreshToken := ""
	if hasToken {
		refreshToken = h.client.GetRefreshToken()
		if len(refreshToken) > 20 {
			refreshToken = refreshToken[:10] + "..." + refreshToken[len(refreshToken)-10:]
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"has_token":     hasToken,
			"refresh_token": refreshToken,
		},
	})
}

// OAuthAuthorize 跳转到飞书授权页面
func (h *FeishuHandler) OAuthAuthorize(c *gin.Context) {
	var app models.FeishuApp
	if h.appRepo != nil {
		apps, _, _ := h.appRepo.List(c.Request.Context(), 1, 1)
		if len(apps) > 0 {
			app = apps[0]
		}
	}

	if app.AppID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "No Feishu app configured"})
		return
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	redirectURI := fmt.Sprintf("%s://%s/app/api/v1/feishu/oauth/callback", scheme, c.Request.Host)

	authURL := fmt.Sprintf(
		"https://open.feishu.cn/open-apis/authen/v1/authorize?app_id=%s&redirect_uri=%s&state=feishu_oauth",
		app.AppID,
		redirectURI,
	)

	c.Redirect(http.StatusFound, authURL)
}

// OAuthCallback 飞书授权回调
func (h *FeishuHandler) OAuthCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Missing authorization code"})
		return
	}

	token, err := h.client.GetTenantAccessToken(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to get tenant token: " + err.Error()})
		return
	}

	tokenURL := "https://open.feishu.cn/open-apis/authen/v1/oidc/access_token"
	payload := map[string]string{
		"grant_type": "authorization_code",
		"code":       code,
	}

	payloadData, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", tokenURL, bytes.NewBuffer(payloadData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to get user token: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	if code, ok := result["code"].(float64); ok && code != 0 {
		msg, _ := result["msg"].(string)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "OAuth failed: " + msg})
		return
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Invalid response format"})
		return
	}

	accessToken, _ := data["access_token"].(string)
	refreshToken, _ := data["refresh_token"].(string)

	if refreshToken == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "No refresh token in response"})
		return
	}

	h.client.SetUserToken(accessToken, refreshToken)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, `
<!DOCTYPE html>
<html>
<head><title>授权成功</title></head>
<body style="text-align:center;padding-top:100px;font-family:sans-serif;">
<h1>✅ 飞书授权成功</h1>
<p>User Token 已保存，现在可以使用用户搜索功能了。</p>
<p>此页面可以关闭。</p>
<script>setTimeout(function(){window.close();}, 3000);</script>
</body>
</html>
`)
}

// GetCallbackStatus 获取回调状态
func (h *FeishuHandler) GetCallbackStatus(c *gin.Context) {
	mgr := feishu.GetCallbackManager()
	if mgr == nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "Success",
			"data": gin.H{
				"running_apps": []string{},
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"running_apps": mgr.GetRunningApps(),
		},
	})
}

// RefreshCallbacks 刷新回调
func (h *FeishuHandler) RefreshCallbacks(c *gin.Context) {
	mgr := feishu.GetCallbackManager()
	if mgr == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Callback manager not initialized"})
		return
	}

	if err := mgr.RefreshCallbacks(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Callbacks refreshed",
		"data": gin.H{
			"running_apps": mgr.GetRunningApps(),
		},
	})
}

// ListMessageLogs 获取消息发送日志列表
func (h *FeishuHandler) ListMessageLogs(c *gin.Context) {
	if h.logRepo == nil {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	msgType := c.Query("msg_type")
	source := c.Query("source")

	list, total, err := h.logRepo.List(c.Request.Context(), page, pageSize, msgType, source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Success",
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

// GetMessageLog 获取单条消息日志
func (h *FeishuHandler) GetMessageLog(c *gin.Context) {
	if h.logRepo == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid ID"})
		return
	}

	log, err := h.logRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "Success", "data": log})
}
