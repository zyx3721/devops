# SSL 证书检查前端功能说明

## 概述

SSL 证书检查功能提供了完整的 HTTPS 域名证书监控和告警能力，帮助运维团队及时发现和处理即将过期的证书。

## 功能特性

### 1. 证书监控管理

#### 批量导入域名
- 支持批量导入多个域名
- 每行一个域名，支持带端口（默认 443）
- 自动去重和验证
- 统一配置检查间隔、超时时间、重试次数
- 统一配置告警阈值（严重/警告/提醒）
- 统一配置告警平台和机器人

**使用示例：**
```
example.com
api.example.com:8443
www.example.com
```

#### 单个域名管理
- 添加单个域名监控
- 编辑域名配置
- 删除域名监控
- 启用/禁用监控
- 立即检查证书

### 2. 证书查询和筛选

#### 多维度筛选
- **告警级别筛选**：已过期、严重、警告、提醒、正常
- **关键字搜索**：支持域名模糊搜索
- **排序方式**：
  - 剩余天数升序（即将过期优先）
  - 剩余天数降序
  - 创建时间降序

#### 证书列表展示
- 域名和端口
- 证书剩余天数（带颜色标识）
- 告警级别（带状态徽章）
- 证书过期时间
- 证书颁发者
- 最后检查时间
- 启用状态
- 操作按钮（检查/编辑/删除）

### 3. 统计概览

四个关键指标卡片：
- **监控域名**：总监控域名数量
- **即将过期**：处于警告状态的证书数量
- **已过期**：已经过期的证书数量
- **正常**：状态正常的证书数量

### 4. 告警配置

#### 告警阈值
- **严重告警**：默认 7 天，证书即将在 7 天内过期
- **警告告警**：默认 30 天，证书即将在 30 天内过期
- **提醒告警**：默认 60 天，证书即将在 60 天内过期

#### 告警平台
支持三种告警平台：
- 飞书
- 钉钉
- 企业微信

每个平台可以选择对应的机器人进行告警推送。

### 5. 证书报告导出

- 支持导出 JSON 格式的证书报告
- 包含所有证书的详细信息
- 支持按筛选条件导出
- 文件名自动包含日期

## 页面路由

- **健康检查主页**：`/healthcheck`
- **SSL 证书检查**：`/healthcheck/ssl-cert`

## API 接口

### 证书列表查询
```typescript
GET /healthcheck/ssl-domains
参数：
  - page: 页码
  - page_size: 每页数量
  - alert_level: 告警级别
  - keyword: 关键字
  - sort_by: 排序方式
```

### 即将过期证书
```typescript
GET /healthcheck/ssl-domains/expiring
参数：
  - days: 天数阈值
```

### 批量导入域名
```typescript
POST /healthcheck/ssl-domains/import
请求体：
{
  "domains": ["example.com", "api.example.com:8443"],
  "interval": 86400,
  "timeout": 10,
  "retry_count": 3,
  "critical_days": 7,
  "warning_days": 30,
  "notice_days": 60,
  "alert_platform": "feishu",
  "alert_bot_id": 1
}
```

### 批量配置告警阈值
```typescript
PUT /healthcheck/ssl-domains/alert-config
请求体：
{
  "ids": [1, 2, 3],
  "critical_days": 7,
  "warning_days": 30,
  "notice_days": 60,
  "alert_platform": "feishu",
  "alert_bot_id": 1
}
```

### 导出报告
```typescript
GET /healthcheck/ssl-domains/export
参数：
  - alert_level: 告警级别
  - keyword: 关键字
```

### 单个域名操作
```typescript
POST /healthcheck/configs              # 创建
PUT /healthcheck/configs/:id           # 更新
DELETE /healthcheck/configs/:id        # 删除
POST /healthcheck/configs/:id/toggle   # 切换启用状态
POST /healthcheck/configs/:id/check    # 立即检查
```

## 数据模型

### SSLCertConfig
```typescript
interface SSLCertConfig {
  id?: number
  name: string                    // 名称
  url: string                     // 域名（可带端口）
  type: 'ssl_cert'                // 类型固定为 ssl_cert
  interval: number                // 检查间隔（秒）
  timeout: number                 // 超时时间（秒）
  retry_count: number             // 重试次数
  enabled: boolean                // 是否启用
  alert_enabled: boolean          // 是否启用告警
  alert_platform?: string         // 告警平台
  alert_bot_id?: number           // 告警机器人 ID
  
  // 证书信息
  cert_expiry_date?: string       // 证书过期时间
  cert_days_remaining?: number    // 剩余天数
  cert_issuer?: string            // 证书颁发者
  cert_subject?: string           // 证书主题
  cert_serial_number?: string     // 证书序列号
  
  // 告警阈值
  critical_days?: number          // 严重告警阈值（天）
  warning_days?: number           // 警告告警阈值（天）
  notice_days?: number            // 提醒告警阈值（天）
  
  // 告警状态
  last_alert_level?: string       // 最后告警级别
  last_alert_at?: string          // 最后告警时间
  last_check_at?: string          // 最后检查时间
}
```

## 颜色和状态映射

### 告警级别颜色
- **已过期（expired）**：红色 (red)
- **严重（critical）**：红色 (red)
- **警告（warning）**：橙色 (orange)
- **提醒（notice）**：蓝色 (blue)
- **正常（normal）**：绿色 (green)

### 状态徽章
- **已过期**：error
- **严重**：error
- **警告**：warning
- **提醒**：processing
- **正常**：success

## 使用流程

### 1. 批量导入域名
1. 点击"批量导入"按钮
2. 在文本框中输入域名列表（每行一个）
3. 配置默认参数（检查间隔、超时、重试次数）
4. 配置告警阈值（严重/警告/提醒天数）
5. 选择告警平台和机器人
6. 点击"确定"完成导入

### 2. 查看证书状态
1. 在列表中查看所有监控的域名
2. 使用筛选条件快速定位问题证书
3. 查看证书详细信息（剩余天数、过期时间、颁发者等）

### 3. 处理即将过期的证书
1. 使用"告警级别"筛选查看即将过期的证书
2. 或使用"排序"功能按剩余天数升序排列
3. 点击"检查"按钮立即检查证书状态
4. 根据告警信息及时更新证书

### 4. 导出证书报告
1. 设置筛选条件（可选）
2. 点击"导出报告"按钮
3. 下载 JSON 格式的报告文件
4. 用于存档或进一步分析

## 注意事项

1. **检查间隔**：建议设置为 86400 秒（24 小时），避免频繁检查
2. **超时时间**：建议设置为 10 秒，确保网络延迟不影响检查
3. **重试次数**：建议设置为 3 次，提高检查可靠性
4. **告警阈值**：根据实际情况调整，建议：
   - 严重：7 天（需要立即处理）
   - 警告：30 天（需要尽快处理）
   - 提醒：60 天（提前准备）
5. **告警冷却期**：系统会自动控制告警频率，避免重复告警

## 技术栈

- **框架**：Vue 3 + TypeScript
- **UI 组件**：Ant Design Vue
- **状态管理**：Composition API
- **HTTP 客户端**：Axios
- **路由**：Vue Router

## 相关文件

- **页面组件**：`devops/web/src/views/healthcheck/SSLCertCheck.vue`
- **服务层**：`devops/web/src/services/healthcheck.ts`
- **路由配置**：`devops/web/src/router/index.ts`
- **后端 API**：`devops/internal/modules/monitoring/handler/healthcheck_handler.go`
