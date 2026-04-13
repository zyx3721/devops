# 📦 业务模块总览 (Modules Overview)

## 🎯 模块化设计理念

将原本分散在 `handler/` 和 `repository/` 目录下的 **50个文件** 按业务功能重新组织为 **7个模块**，每个模块职责清晰，便于开发和维护。

## 📁 模块结构

```
modules/
├── auth/           🔐 认证授权模块 (3 handlers, 2 repos)
├── application/    📱 应用管理模块 (3 handlers, 2 repos)
├── approval/       ✅ 审批流程模块 (5 handlers, 7 repos)
├── infrastructure/ 🔨 基础设施模块 (7 handlers, 2 repos)
├── notification/   💬 消息通知模块 (5 handlers, 3 repos)
├── monitoring/     🚨 监控告警模块 (2 handlers, 2 repos)
└── system/         ⚙️ 系统管理模块 (4 handlers, 2 repos)
```

## 🔍 模块详情

| 模块 | 功能描述 | Handler数 | Repository数 | 主要功能 |
|------|----------|-----------|--------------|----------|
| **🔐 auth** | 认证授权 | 3 | 2 | 用户登录、权限管理、RBAC |
| **📱 application** | 应用管理 | 3 | 2 | 应用配置、部署管理、发布锁 |
| **✅ approval** | 审批流程 | 5 | 7 | 多级审批、审批链、发布窗口 |
| **🔨 infrastructure** | 基础设施 | 7 | 2 | Jenkins集成、K8s管理 |
| **💬 notification** | 消息通知 | 5 | 3 | 飞书、钉钉、企微集成 |
| **🚨 monitoring** | 监控告警 | 2 | 2 | 健康检查、告警管理 |
| **⚙️ system** | 系统管理 | 4 | 2 | 仪表盘、审计日志、OA集成 |

## 🚀 使用指南

### 开发新功能
1. 确定功能属于哪个模块
2. 在对应模块的 `handler/` 目录添加HTTP接口
3. 在对应模块的 `repository/` 目录添加数据操作
4. 在 `internal/service/` 目录添加业务逻辑

### 查找现有功能
- **用户相关**: 去 `auth/` 模块
- **应用部署**: 去 `application/` 模块  
- **审批流程**: 去 `approval/` 模块
- **Jenkins/K8s**: 去 `infrastructure/` 模块
- **消息通知**: 去 `notification/` 模块
- **监控告警**: 去 `monitoring/` 模块
- **系统管理**: 去 `system/` 模块

### 模块间依赖
- 各模块的 `handler` 和 `repository` 相对独立
- 共享的业务逻辑在 `internal/service/` 目录
- 公共组件在 `pkg/` 目录
- 数据模型在 `internal/models/` 目录

## 📋 重构收益

### 开发体验 ✨
- **快速定位**: 不用在29个handler文件中找，直接去对应模块
- **模块化开发**: 团队成员可以专注特定模块
- **新人友好**: 按模块学习，降低认知负担

### 代码质量 🔧
- **职责清晰**: 每个模块功能边界明确
- **易于维护**: 相关文件聚合，修改时不用跨目录
- **减少冲突**: 不同模块开发很少修改同一目录

### 团队协作 👥
- **清晰分工**: 每个人负责特定模块
- **并行开发**: 模块间依赖少，可以并行开发
- **代码审查**: 按模块审查，更容易发现问题

## 🔄 迁移说明

### 文件移动
- 原 `internal/handler/*.go` → `internal/modules/*/handler/*.go`
- 原 `internal/repository/*.go` → `internal/modules/*/repository/*.go`
- `internal/repository/base.go` 保持原位置

### Import路径
- 文件移动后，相对于项目根目录的import路径保持不变
- 无需修改现有的import语句

### Service目录
- `internal/service/` 目录结构保持不变
- 各模块通过相对路径引用对应的service

## 📖 参考文档

每个模块都有详细的README文档：
- [🔐 auth/README.md](auth/README.md) - 认证授权模块
- [📱 application/README.md](application/README.md) - 应用管理模块
- [✅ approval/README.md](approval/README.md) - 审批流程模块
- [🔨 infrastructure/README.md](infrastructure/README.md) - 基础设施模块
- [💬 notification/README.md](notification/README.md) - 消息通知模块
- [🚨 monitoring/README.md](monitoring/README.md) - 监控告警模块
- [⚙️ system/README.md](system/README.md) - 系统管理模块