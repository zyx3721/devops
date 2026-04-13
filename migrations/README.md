# 数据库迁移脚本

此目录包含 DevOps 平台数据库初始化与升级所需的 SQL 脚本。

## 目录结构

| 文件 | 用途 | 执行时机 |
|------|------|----------|
| `init_tables.sql` | 全量建表 + 初始数据（113张表） | 全新部署时执行 |
| `upgrades.sql` | 存量数据库升级补丁 | 已有数据库升级时执行 |
| `update_alert_channels.sql` | 告警通知渠道更新 | 按需执行 |

---

## 全新部署

```bash
# 创建数据库
mysql -h 127.0.0.1 -u root -p -e "CREATE DATABASE IF NOT EXISTS devops DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"

# 初始化所有表结构和初始数据
mysql -h 127.0.0.1 -u root -p devops < migrations/init_tables.sql
```

完成后使用以下账号登录：
- 用户名：`admin`
- 密码：`admin123`
- 角色：超级管理员

---

## 存量数据库升级

适用于已有数据库，按顺序执行升级补丁：

```bash
mysql -h 127.0.0.1 -u root -p devops < migrations/upgrades.sql
```

`upgrades.sql` 包含以下补丁（按顺序）：

1. 飞书相关补丁 — 补充 `feishu_apps.webhook` 字段及缺失表
2. 流水线模板补丁 — 补充 `pipeline_templates` 的 `language`/`framework` 字段
3. 负载均衡字段修复 — `traffic_loadbalance_config.hash_key` 从 ENUM 改为 VARCHAR
4. 制品仓库监控补丁 — 为 `artifact_repositories` 添加连接状态监控字段
5. 告警静默字段补丁 — 为 `log_alert_history` 添加 `silenced`/`silence_id` 字段

> 补丁中的 ALTER TABLE 如列已存在会报错，可忽略该错误继续执行。

---

## init_tables.sql 表清单

| 编号 | 表名 | 说明 |
|------|------|------|
| 1 | users | 用户表 |
| 2 | audit_logs | 审计日志 |
| 3 | jenkins_instances | Jenkins 实例 |
| 4 | jenkins_builds | Jenkins 构建记录 |
| 5 | k8s_clusters | K8s 集群 |
| 6 | cron_hpa | 定时 HPA |
| 7 | feishu_apps | 飞书应用 |
| 8 | feishu_bots | 飞书机器人 |
| 9 | feishu_message_logs | 飞书消息记录 |
| 10 | feishu_user_tokens | 飞书用户令牌 |
| 11 | feishu_requests | 飞书请求记录 |
| 12 | dingtalk_bots | 钉钉机器人 |
| 13 | dingtalk_apps | 钉钉应用 |
| 14 | dingtalk_message_logs | 钉钉消息记录 |
| 15 | wechat_work_apps | 企业微信应用 |
| 16 | wechat_work_bots | 企业微信机器人 |
| 17 | wechat_work_message_logs | 企业微信消息记录 |
| 18 | message_templates | 消息模板 |
| 19 | oa_configs | OA 配置 |
| 20 | oa_approval_records | OA 审批记录 |
| 21 | system_configs | 系统配置 |
| 22 | sys_message_templates | 系统消息模板 |
| 23 | alert_configs | 告警配置 |
| 24 | alert_histories | 告警历史 |
| 25 | log_alert_history | 日志告警历史 |
| 26 | applications | 应用 |
| 27 | application_envs | 应用环境 |
| 28 | deploy_records | 部署记录 |
| 29 | deploy_locks | 部署锁 |
| 30 | approval_records | 审批记录 |
| 31 | health_check_configs | 健康检查配置 |
| 32 | health_check_histories | 健康检查历史 |
| 33 | roles | 角色 |
| 34 | permissions | 权限 |
| 35 | role_permissions | 角色权限关联 |
| 36 | user_roles | 用户角色关联 |
| 37 | approval_rules | 审批规则 |
| 38 | deploy_windows | 发布窗口 |
| 39 | approval_chains | 审批链 |
| 40 | approval_nodes | 审批节点 |
| 41 | approval_instances | 审批实例 |
| 42 | approval_node_instances | 审批节点实例 |
| 43 | approval_actions | 审批操作记录 |
| 44 | pipeline_templates | 流水线模板 |
| 45 | pipeline_template_ratings | 模板评分 |
| 46 | pipeline_stage_templates | 阶段模板 |
| 47 | pipeline_step_templates | 步骤模板 |
| 48 | build_caches | 构建缓存 |
| 49 | build_resource_quotas | 构建资源配额 |
| 50 | build_resource_usages | 构建资源使用记录 |
| 51 | parallel_build_configs | 并行构建配置 |
| 52 | artifact_repositories | 制品仓库 |
| 53 | artifact_registry_connection_history | 制品仓库连接历史 |
| 54 | artifacts | 制品 |
| 55 | artifact_versions | 制品版本 |
| 56 | artifact_scan_results | 制品扫描结果 |
| 57 | artifact_promotions | 制品晋级记录 |
| 58 | traffic_ratelimit_rules | 限流规则 |
| 59 | traffic_circuitbreaker_rules | 熔断规则 |
| 60 | traffic_routing_rules | 流量路由规则 |
| 61 | traffic_loadbalance_config | 负载均衡配置 |
| 62 | traffic_timeout_config | 超时重试配置 |
| 63 | traffic_mirror_rules | 流量镜像规则 |
| 64 | traffic_fault_rules | 故障注入规则 |
| 65 | traffic_operation_logs | 流量治理操作日志 |
| 66 | traffic_statistics | 流量统计 |
| 67 | traffic_rule_versions | 规则版本 |
| 68 | canary_releases | 金丝雀发布配置 |
| 69 | blue_green_deployments | 蓝绿部署配置 |
| 70 | ai_conversations | AI 会话 |
| 71 | ai_messages | AI 消息 |
| 72 | ai_knowledge | AI 知识库 |
| 73 | ai_operation_logs | AI 操作审计日志 |
| 74 | ai_llm_configs | AI LLM 配置 |
| 75 | ai_message_feedbacks | AI 消息反馈 |
| 76 | oa_data | OA 数据 |
| 77 | oa_addresses | OA 地址配置 |
| 78 | oa_notify_configs | OA 通知配置 |
| 79 | alert_silence_rules | 告警静默规则 |
| 80 | alert_notification_channels | 告警通知渠道 |
| 81 | alert_channel_mappings | 告警渠道映射 |
| 82 | log_alert_rules | 日志告警规则 |
| 83 | log_highlight_rules | 日志高亮规则 |
| 84 | log_parse_templates | 日志解析模板 |
| 85 | log_datasources | 日志数据源 |
| 86 | log_bookmarks | 日志书签 |
| 87 | log_saved_queries | 日志保存查询 |
| 88 | k8s_cluster_feishu_apps | K8s集群飞书应用关联 |
| 89 | k8s_cluster_dingtalk_apps | K8s集群钉钉应用关联 |
| 90 | k8s_cluster_wechat_work_apps | K8s集群企业微信应用关联 |
| 91 | jenkins_feishu_apps | Jenkins飞书应用关联 |
| 92 | jenkins_dingtalk_apps | Jenkins钉钉应用关联 |
| 93 | jenkins_wechat_work_apps | Jenkins企业微信应用关联 |
| 94 | resource_costs | 资源成本记录 |
| 95 | cost_summaries | 成本汇总 |
| 96 | cost_suggestions | 成本优化建议 |
| 97 | cost_configs | 成本计费配置 |
| 98 | cost_budgets | 成本预算 |
| 99 | cost_alerts | 成本告警记录 |
| 100 | resource_activities | 资源活动记录 |
| 101 | traffic_rule_templates | 流量规则模板 |
| 102 | app_ratelimit_rules | 应用限流规则关联 |
| 103 | app_mirror_rules | 应用镜像规则关联 |
| 104 | app_fault_rules | 应用故障注入规则关联 |
| 105 | image_registries | 镜像仓库 |
| 106 | image_scans | 镜像扫描结果 |
| 107 | compliance_rules | 合规检查规则 |
| 108 | config_checks | 配置合规检查记录 |
| 109 | security_audit_logs | 安全审计日志 |
| 110 | security_reports | 安全报告 |
| 111 | encryption_keys | 加密密钥管理 |
| 112 | tasks | 异步任务 |
| 113 | pipelines | 流水线 |

---

## 注意事项

- MySQL 版本推荐 8.0+
- 字符集必须使用 `utf8mb4`
- 生产环境执行前务必备份数据库
- `upgrades.sql` 中的 DELIMITER 语法需通过 mysql 客户端执行，不支持部分 GUI 工具
