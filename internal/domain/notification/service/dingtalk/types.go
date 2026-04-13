package dingtalk

import "encoding/json"

// APIResponse 统一API响应结构
type APIResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
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
	UserID string `json:"userid"`
	Name   string `json:"name"`
	Mobile string `json:"mobile"`
	Email  string `json:"email"`
	Avatar string `json:"avatar"`
}

// SendRequest 发送消息请求结构
type SendRequest struct {
	MsgType string          `json:"msg_type"`
	Content json.RawMessage `json:"content"`
	AtAll   bool            `json:"at_all"`
	AtUsers []string        `json:"at_users"`
}

// TextMessage 文本消息
type TextMessage struct {
	Content string `json:"content"`
}

// MarkdownMessage Markdown消息
type MarkdownMessage struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// LinkMessage 链接消息
type LinkMessage struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	MessageURL string `json:"messageUrl"`
	PicURL     string `json:"picUrl"`
}

// ActionCardMessage ActionCard消息
type ActionCardMessage struct {
	Title          string `json:"title"`
	Text           string `json:"text"`
	SingleTitle    string `json:"singleTitle,omitempty"`
	SingleURL      string `json:"singleURL,omitempty"`
	BtnOrientation string `json:"btnOrientation,omitempty"`
	Btns           []Btn  `json:"btns,omitempty"`
}

// Btn 按钮
type Btn struct {
	Title     string `json:"title"`
	ActionURL string `json:"actionURL"`
}

// WebhookMessage Webhook消息结构
type WebhookMessage struct {
	MsgType    string             `json:"msgtype"`
	Text       *WebhookText       `json:"text,omitempty"`
	Markdown   *WebhookMarkdown   `json:"markdown,omitempty"`
	Link       *WebhookLink       `json:"link,omitempty"`
	ActionCard *WebhookActionCard `json:"actionCard,omitempty"`
	At         *WebhookAt         `json:"at,omitempty"`
}

// WebhookText 文本消息
type WebhookText struct {
	Content string `json:"content"`
}

// WebhookMarkdown Markdown消息
type WebhookMarkdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

// WebhookLink 链接消息
type WebhookLink struct {
	Title      string `json:"title"`
	Text       string `json:"text"`
	MessageURL string `json:"messageUrl"`
	PicURL     string `json:"picUrl"`
}

// WebhookActionCard ActionCard消息
type WebhookActionCard struct {
	Title          string `json:"title"`
	Text           string `json:"text"`
	SingleTitle    string `json:"singleTitle,omitempty"`
	SingleURL      string `json:"singleURL,omitempty"`
	BtnOrientation string `json:"btnOrientation,omitempty"`
	Btns           []Btn  `json:"btns,omitempty"`
}

// WebhookAt @配置
type WebhookAt struct {
	AtMobiles []string `json:"atMobiles,omitempty"`
	AtUserIds []string `json:"atUserIds,omitempty"`
	IsAtAll   bool     `json:"isAtAll"`
}
