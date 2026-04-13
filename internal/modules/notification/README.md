# 💬 消息通知模块 (Notification Module)

## 功能概述
负责飞书、钉钉、企业微信等消息通知集成。

## 文件结构
```
notification/
├── handler/                        # HTTP处理器
│   ├── feishu_handler.go          # 飞书消息接口
│   ├── feishu_app_handler.go      # 飞书应用管理接口
│   ├── feishu_user_handler.go     # 飞书用户接口
│   ├── dingtalk_handler.go        # 钉钉消息接口
│   └── wechatwork_handler.go      # 企业微信消息接口
└── repository/                     # 数据访问层
    ├── feishu_repo.go             # 飞书数据操作
    ├── dingtalk_repo.go           # 钉钉数据操作
    └── wechatwork_repo.go         # 企业微信数据操作
```

## 主要功能
- **飞书集成**: 消息发送、卡片交互、用户管理
- **钉钉集成**: 消息推送、机器人管理
- **企业微信集成**: 消息通知、应用管理
- **统一通知**: 多平台消息分发

## API接口
- `POST /feishu/send` - 发送飞书消息
- `POST /feishu/card` - 发送飞书卡片
- `GET /feishu/users` - 获取飞书用户
- `POST /dingtalk/send` - 发送钉钉消息
- `POST /wechatwork/send` - 发送企业微信消息

## 相关Service
- `internal/service/feishu/` - 飞书业务逻辑
- `internal/service/dingtalk/` - 钉钉业务逻辑
- `internal/service/wechatwork/` - 企业微信业务逻辑