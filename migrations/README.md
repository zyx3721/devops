# 🗄️ Database Migrations & Scripts

此目录包含 DevOps 平台数据库初始化、功能更新和修复所需的所有 SQL 脚本。

## 📂 目录结构与分类

### 1. 核心初始化 (Initialization)
> **必须首先执行**，用于建立系统的基础架构。

- **`init_tables.sql`**: 核心业务表结构，已包含基础 RBAC 表结构与初始化数据。
- **`create_rbac_tables.sql`**: 独立 RBAC 建表脚本，适合拆分执行或局部修复时使用。
- **`init_rbac.sql`**: RBAC 初始数据重放脚本，也会补齐旧库缺失的 `permissions.description` 字段和 `resource` 索引。
- **`fix_permissions_description.sql`**: 兼容旧版库结构的 RBAC 修复脚本，补齐 `permissions.description` 字段和 `resource` 索引。

### 2. 功能模块 (Feature Modules)
> 根据部署需求选择性执行，建议全部安装以获得完整体验。

#### 🚦 流量与治理 (Traffic Governance)
- **`traffic_management_v2.sql`**: 流量治理核心表（限流、熔断、路由规则）。
- **`traffic_monitoring_canary.sql`**: 金丝雀发布监控相关表。

#### 📦 流水线与制品 (CI/CD & Artifacts)
- **`pipeline_templates.sql`**: 流水线模板定义表。
- **`build_artifact.sql`**: 构建制品管理表。
- **`artifact_registry_monitoring.sql`**: 制品仓库监控表。

#### 🤖 AI 助手 (AI Copilot)
- **`ai_copilot.sql`**: AI 对话、知识库及向量存储相关表。

#### 🚨 监控与告警 (Monitoring & Alerts)
- **`setup_alert_config.sql`**: 告警配置初始化及默认模板。
- **`update_alert_channels.sql`**: 告警通知渠道更新。
- **`add_log_alert_silence_fields.sql`**: 日志告警静默规则字段。
- **`2026-01-31_ssl_cert_check.sql`**: SSL 证书过期监控表。

### 3. 补丁与增量更新 (Patches & Updates)
> 用于修复特定问题或追加新字段，按需执行。

- **`fix_pipeline_templates_*.sql`**: 修复流水线模板表缺失字段的问题。
- **`fix_applications_table.sql`**: 修复应用表字段。
- **`fix_loadbalance_hash_key.sql`**: 修复负载均衡哈希键配置。
- **`add_user_unique_constraints.sql`**: 补充用户表的唯一性约束。
- **`audit_model_unification.sql`**: 审计日志模型统一化更新。
- **`ADD_PROJECT_TO_FEISHU_BOTS.sql`**: 飞书机器人集成增加项目字段。
- **`ADD_SSL_CERT_FIELDS_SAFE.sql`**: 安全添加 SSL 证书字段。

---

## 🚀 使用指南 (Usage Guide)

### 全新安装 (Fresh Installation)
请严格按照以下顺序执行 SQL 脚本：

```bash
# 1. 基础架构
mysql -u root -p devops < init_tables.sql

# 仅在旧库修复或需要重放 RBAC 数据时执行
mysql -u root -p devops < fix_permissions_description.sql
mysql -u root -p devops < init_rbac.sql

# 2. 核心功能
mysql -u root -p devops < traffic_management_v2.sql
mysql -u root -p devops < pipeline_templates.sql
mysql -u root -p devops < build_artifact.sql
mysql -u root -p devops < ai_copilot.sql

# 3. 监控配置 (新增)
mysql -u root -p devops < setup_alert_config.sql
```

### 故障排查与修复 (Troubleshooting)
如果你在运行中遇到 `Unknown column` 错误，请尝试运行对应的修复脚本：

```bash
# 例如：修复流水线模板字段缺失
mysql -u root -p devops < migrations/fix_pipeline_templates_columns.sql

# 修复 RBAC 权限表缺少 description 字段或缺少 resource 索引
mysql -u root -p devops < migrations/fix_permissions_description.sql
```

---

## ⚠️ 注意事项 (Requirements & Notes)

1.  **MySQL 版本**: 推荐 5.7 或 8.0+。
2.  **字符集**: 必须使用 `utf8mb4`，以支持 Emoji 和多语言（特别是 AI 对话和日志功能）。
3.  **数据备份**: 在生产环境执行任何 `fix_` 或 `update_` 开头的脚本前，请务必备份数据库。
4.  **重复执行**: 大部分脚本包含 `IF NOT EXISTS` 检查，但重复执行可能会报 `Duplicate column` 错误，通常可以忽略。

## 🔐 默认管理员账号

初始化完成后，可使用以下账号登录系统：
- **用户名**: `admin`
- **密码**: `admin123`
- **角色**: 超级管理员 (Super Admin)
