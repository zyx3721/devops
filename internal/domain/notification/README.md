# Notification Domain

消息通知领域模块，采用 DDD（领域驱动设计）分层结构。

## 目录结构

```
internal/domain/notification/
├── model/           # 领域模型
│   ├── feishu.go    # 飞书模型
│   ├── dingtalk.go  # 钉钉模型
│   ├── wechatwork.go # 企业微信模型
│   └── doc.go       # 包文档
├── repository/      # 数据仓储层
│   ├── feishu_repo.go
│   ├── dingtalk_repo.go
│   └── wechatwork_repo.go
├── service/         # 业务服务层
│   ├── feishu/      # 飞书服务
│   │   ├── client.go       # 客户端核心
│   │   ├── token.go        # 令牌管理
│   │   ├── message.go      # 消息发送
│   │   ├── user.go         # 用户管理
│   │   ├── chat.go         # 群聊管理
│   │   ├── callback.go     # 回调管理
│   │   ├── callback_handler.go # 卡片回调处理
│   │   ├── card_builder.go # 卡片构建
│   │   ├── store.go        # 请求存储
│   │   ├── sender.go       # 消息发送器
│   │   └── types.go        # 类型定义
│   ├── dingtalk/    # 钉钉服务
│   │   ├── client.go
│   │   └── types.go
│   └── wechatwork/  # 企业微信服务
│       ├── client.go
│       └── types.go
└── README.md
```

## 迁移状态

- [x] Model 层 - 已完成
- [x] Repository 层 - 已完成
- [x] Service 层 - 已完成（完整迁移）
- [ ] Handler 层 - 保留在 internal/modules/notification/handler

## 使用方式

```go
import (
    "devops/internal/domain/notification/model"
    "devops/internal/domain/notification/repository"
    "devops/internal/domain/notification/service/feishu"
    "devops/internal/domain/notification/service/dingtalk"
    "devops/internal/domain/notification/service/wechatwork"
)

// 使用 model
app := &model.FeishuApp{
    Name:    "DevOps",
    AppID:   "cli_xxx",
}

// 使用 repository
repo := repository.NewFeishuAppRepository(db)
repo.Create(ctx, app)

// 使用 service
feishuClient := feishu.NewClient(cfg)
dingtalkClient := dingtalk.NewClient(appKey, appSecret, agentID)
wechatClient := wechatwork.NewClient(corpID, agentID, secret)

// 发送消息
feishuClient.SendMessage(ctx, receiveID, "open_id", "text", content)
```

## 飞书服务功能

- 令牌管理（tenant_access_token / user_access_token）
- 消息发送（文本、卡片）
- 用户搜索和查询
- 群聊管理
- WebSocket 回调
- 卡片交互处理
- Jenkins 构建触发

## 钉钉服务功能

- 访问令牌管理
- 工作通知消息
- Webhook 消息
- 用户搜索

## 企业微信服务功能

- 访问令牌管理
- 应用消息
- Webhook 消息
- 用户搜索
