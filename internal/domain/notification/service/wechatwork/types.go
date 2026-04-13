package wechatwork

import "encoding/json"

// APIResponse 统一API响应结构
type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

// SendRequest 发送消息请求结构
type SendRequest struct {
	MsgType string          `json:"msg_type"`
	ToUser  string          `json:"to_user"`
	ToParty string          `json:"to_party"`
	ToTag   string          `json:"to_tag"`
	Content json.RawMessage `json:"content"`
}

// AppMessage 应用消息
type AppMessage struct {
	ToUser   string       `json:"touser,omitempty"`
	ToParty  string       `json:"toparty,omitempty"`
	ToTag    string       `json:"totag,omitempty"`
	MsgType  string       `json:"msgtype"`
	AgentID  int64        `json:"agentid"`
	Text     *TextMsg     `json:"text,omitempty"`
	Markdown *MarkdownMsg `json:"markdown,omitempty"`
	TextCard *TextCardMsg `json:"textcard,omitempty"`
	News     *NewsMsg     `json:"news,omitempty"`
}

// TextMsg 文本消息
type TextMsg struct {
	Content string `json:"content"`
}

// MarkdownMsg Markdown消息
type MarkdownMsg struct {
	Content string `json:"content"`
}

// TextCardMsg 文本卡片消息
type TextCardMsg struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	URL         string `json:"url"`
	BtnTxt      string `json:"btntxt,omitempty"`
}

// NewsMsg 图文消息
type NewsMsg struct {
	Articles []NewsArticle `json:"articles"`
}

// NewsArticle 图文消息文章
type NewsArticle struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
	PicURL      string `json:"picurl,omitempty"`
}

// WebhookMessage Webhook消息结构
type WebhookMessage struct {
	MsgType  string           `json:"msgtype"`
	Text     *WebhookText     `json:"text,omitempty"`
	Markdown *WebhookMarkdown `json:"markdown,omitempty"`
	News     *WebhookNews     `json:"news,omitempty"`
}

// WebhookText 文本消息
type WebhookText struct {
	Content             string   `json:"content"`
	MentionedList       []string `json:"mentioned_list,omitempty"`
	MentionedMobileList []string `json:"mentioned_mobile_list,omitempty"`
}

// WebhookMarkdown Markdown消息
type WebhookMarkdown struct {
	Content string `json:"content"`
}

// WebhookNews 图文消息
type WebhookNews struct {
	Articles []WebhookArticle `json:"articles"`
}

// WebhookArticle 图文消息文章
type WebhookArticle struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
	PicURL      string `json:"picurl,omitempty"`
}

// TokenResponse 获取token响应
type TokenResponse struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// UserInfo 用户信息
type UserInfo struct {
	UserID     string `json:"userid"`
	Name       string `json:"name"`
	Mobile     string `json:"mobile"`
	Email      string `json:"email"`
	Department []int  `json:"department"`
	Avatar     string `json:"avatar"`
}
