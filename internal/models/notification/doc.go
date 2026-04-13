// Package notification 定义消息通知相关的数据模型
//
// 本包包含与第三方消息平台集成相关的所有数据模型，包括：
//   - 飞书（Feishu）：机器人、应用、消息日志、用户令牌
//   - 钉钉（DingTalk）：机器人、应用、消息日志
//   - 企业微信（WeChatWork）：机器人、应用、消息日志
//
// 使用示例:
//
//	import "devops/internal/models/notification"
//
//	// 创建飞书机器人
//	bot := &notification.FeishuBot{
//	    Name:       "DevOps通知",
//	    WebhookURL: "https://open.feishu.cn/...",
//	}
package notification
