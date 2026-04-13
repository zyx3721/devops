# 🚨 监控告警模块 (Monitoring Module)

## 功能概述
负责系统监控、健康检查和告警通知。

## 文件结构
```
monitoring/
├── handler/                    # HTTP处理器
│   ├── alert_handler.go       # 告警管理接口
│   └── healthcheck_handler.go # 健康检查接口
└── repository/                 # 数据访问层
    ├── alert_repo.go          # 告警数据操作
    └── healthcheck_repo.go    # 健康检查数据操作
```

## 主要功能
- **告警管理**: 告警规则配置、告警历史查询
- **健康检查**: 服务健康监控、状态检查
- **通知集成**: 告警消息推送到各个平台

## API接口
- `GET /alerts` - 获取告警列表
- `POST /alerts` - 创建告警规则
- `PUT /alerts/:id` - 更新告警规则
- `DELETE /alerts/:id` - 删除告警规则
- `GET /healthcheck` - 获取健康检查配置
- `POST /healthcheck` - 创建健康检查

## 相关Service
- `internal/service/healthcheck/` - 健康检查业务逻辑