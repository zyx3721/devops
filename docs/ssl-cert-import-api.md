# SSL证书批量导入API文档

## 接口概述

批量导入SSL域名接口允许管理员一次性添加多个域名进行SSL证书监控。

## 接口详情

### 请求

- **URL**: `POST /healthcheck/ssl-domains/import`
- **权限**: 需要管理员权限
- **Content-Type**: `application/json`

### 请求参数

```json
{
  "domains": ["example.com", "api.example.com:8443", "www.example.com"],
  "interval": 3600,
  "timeout": 10,
  "critical_days": 7,
  "warning_days": 30,
  "notice_days": 60,
  "alert_enabled": true,
  "alert_platform": "feishu",
  "alert_bot_id": 1
}
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| domains | array[string] | 是 | 域名列表，支持以下格式：<br>- `example.com` (使用默认端口443)<br>- `example.com:8443` (指定端口)<br>- `https://example.com` (自动去除协议前缀) |
| interval | int | 是 | 检查间隔（秒），最小60秒 |
| timeout | int | 是 | 超时时间（秒），范围1-300 |
| critical_days | int | 是 | 严重告警阈值（天），必须 < warning_days |
| warning_days | int | 是 | 警告告警阈值（天），必须 < notice_days |
| notice_days | int | 是 | 提醒告警阈值（天） |
| alert_enabled | bool | 否 | 是否启用告警，默认false |
| alert_platform | string | 否 | 告警平台（feishu/dingtalk/wechat） |
| alert_bot_id | int | 否 | 告警机器人ID |

### 响应

#### 成功响应

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "success_count": 2,
    "failed_count": 1,
    "failed_domains": [
      {
        "domain": "invalid..com",
        "error": "Invalid domain format"
      }
    ]
  }
}
```

#### 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 状态码，0表示成功 |
| message | string | 响应消息 |
| data.success_count | int | 成功导入的域名数量 |
| data.failed_count | int | 失败的域名数量 |
| data.failed_domains | array | 失败的域名列表及错误原因 |

#### 错误响应

**400 Bad Request** - 参数验证失败

```json
{
  "code": 400,
  "message": "Invalid alert thresholds: critical_days < warning_days < notice_days required"
}
```

**401 Unauthorized** - 未认证

```json
{
  "code": 401,
  "message": "Authentication required"
}
```

**403 Forbidden** - 权限不足

```json
{
  "code": 403,
  "message": "Admin permission required"
}
```

**500 Internal Server Error** - 服务器错误

```json
{
  "code": 500,
  "message": "Failed to create configurations: database error"
}
```

## 域名验证规则

### 有效域名格式

- 标准域名：`example.com`
- 带端口：`example.com:8443`
- 带协议：`https://example.com` (协议会被自动去除)
- 子域名：`api.example.com`
- IP地址：`192.168.1.1` 或 `192.168.1.1:8443`

### 无效域名格式

- 空字符串
- 包含连续的点：`invalid..com`
- 以点或连字符开头/结尾：`.example.com`, `example.com.`, `-example.com`, `example.com-`
- 包含特殊字符：`example@com`, `example#com`

## 去重逻辑

系统会自动检测并跳过重复的域名：

1. **批次内去重**：同一请求中的重复域名只会处理一次
2. **数据库去重**：已存在于数据库中的域名会被跳过

域名标准化规则：
- 转换为小写
- 去除协议前缀（http://, https://）
- 去除路径部分
- 添加默认端口443（如果未指定）

例如，以下域名被视为相同：
- `example.com`
- `EXAMPLE.COM`
- `https://example.com`
- `example.com:443`

## 使用示例

### cURL 示例

```bash
curl -X POST http://localhost:8080/healthcheck/ssl-domains/import \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "domains": [
      "example.com",
      "api.example.com:8443",
      "www.example.com"
    ],
    "interval": 3600,
    "timeout": 10,
    "critical_days": 7,
    "warning_days": 30,
    "notice_days": 60,
    "alert_enabled": true,
    "alert_platform": "feishu",
    "alert_bot_id": 1
  }'
```

### JavaScript 示例

```javascript
const response = await fetch('http://localhost:8080/healthcheck/ssl-domains/import', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer YOUR_TOKEN'
  },
  body: JSON.stringify({
    domains: [
      'example.com',
      'api.example.com:8443',
      'www.example.com'
    ],
    interval: 3600,
    timeout: 10,
    critical_days: 7,
    warning_days: 30,
    notice_days: 60,
    alert_enabled: true,
    alert_platform: 'feishu',
    alert_bot_id: 1
  })
});

const result = await response.json();
console.log(`成功导入 ${result.data.success_count} 个域名`);
if (result.data.failed_count > 0) {
  console.log('失败的域名:', result.data.failed_domains);
}
```

### Python 示例

```python
import requests

url = 'http://localhost:8080/healthcheck/ssl-domains/import'
headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer YOUR_TOKEN'
}
data = {
    'domains': [
        'example.com',
        'api.example.com:8443',
        'www.example.com'
    ],
    'interval': 3600,
    'timeout': 10,
    'critical_days': 7,
    'warning_days': 30,
    'notice_days': 60,
    'alert_enabled': True,
    'alert_platform': 'feishu',
    'alert_bot_id': 1
}

response = requests.post(url, json=data, headers=headers)
result = response.json()

print(f"成功导入 {result['data']['success_count']} 个域名")
if result['data']['failed_count'] > 0:
    print('失败的域名:', result['data']['failed_domains'])
```

## 注意事项

1. **告警阈值顺序**：必须满足 `critical_days < warning_days < notice_days`
2. **检查间隔**：建议设置为3600秒（1小时）或更长，避免频繁检查
3. **超时时间**：根据网络环境设置合理的超时时间，建议10-30秒
4. **批量大小**：建议每次导入不超过100个域名，避免请求超时
5. **权限要求**：此接口需要管理员权限，普通用户无法访问

## 后续操作

导入成功后，系统会：

1. 自动创建SSL证书检查配置
2. 按照设置的间隔定期检查证书
3. 根据证书剩余天数发送不同级别的告警
4. 记录每次检查的历史记录

您可以通过以下接口查看和管理导入的域名：

- `GET /healthcheck/configs?type=ssl_cert` - 查询所有SSL证书配置
- `GET /healthcheck/configs/:id` - 查询单个配置详情
- `PUT /healthcheck/configs/:id` - 更新配置
- `DELETE /healthcheck/configs/:id` - 删除配置
- `POST /healthcheck/configs/:id/check` - 立即执行检查


---

# SSL证书批量配置告警阈值API文档

## 接口概述

批量更新告警阈值接口允许管理员一次性修改多个SSL证书配置的告警阈值设置。

## 接口详情

### 请求

- **URL**: `PUT /healthcheck/ssl-domains/alert-config`
- **权限**: 需要管理员权限
- **Content-Type**: `application/json`

### 请求参数

```json
{
  "config_ids": [1, 2, 3],
  "critical_days": 7,
  "warning_days": 30,
  "notice_days": 60
}
```

#### 参数说明

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| config_ids | array[int] | 是 | 配置ID列表，至少包含1个ID |
| critical_days | int | 是 | 严重告警阈值（天），必须 < warning_days |
| warning_days | int | 是 | 警告告警阈值（天），必须 < notice_days |
| notice_days | int | 是 | 提醒告警阈值（天） |

### 响应

#### 成功响应

```json
{
  "code": 0,
  "message": "Success",
  "data": {
    "updated_count": 3
  }
}
```

#### 响应字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| code | int | 状态码，0表示成功 |
| message | string | 响应消息 |
| data.updated_count | int | 成功更新的配置数量 |

#### 错误响应

**400 Bad Request** - 参数验证失败

```json
{
  "code": 400,
  "message": "Invalid alert thresholds: critical_days < warning_days < notice_days required"
}
```

```json
{
  "code": 400,
  "message": "Key: 'BatchAlertConfigRequest.ConfigIDs' Error:Field validation for 'ConfigIDs' failed on the 'min' tag"
}
```

**401 Unauthorized** - 未认证

```json
{
  "code": 401,
  "message": "Authentication required"
}
```

**403 Forbidden** - 权限不足

```json
{
  "code": 403,
  "message": "Admin permission required"
}
```

## 告警阈值说明

告警阈值定义了证书剩余天数与告警级别的对应关系：

| 剩余天数范围 | 告警级别 | 说明 |
|-------------|---------|------|
| < 0 | expired | 证书已过期 |
| 0 ~ critical_days | critical | 严重告警，需要立即处理 |
| critical_days ~ warning_days | warning | 警告告警，需要尽快处理 |
| warning_days ~ notice_days | notice | 提醒告警，需要关注 |
| >= notice_days | normal | 正常状态，无需告警 |

### 推荐配置

| 场景 | critical_days | warning_days | notice_days |
|------|--------------|--------------|-------------|
| 生产环境 | 7 | 30 | 60 |
| 测试环境 | 3 | 14 | 30 |
| 开发环境 | 1 | 7 | 14 |

## 批量更新行为

1. **部分成功**：如果某些配置ID不存在或更新失败，系统会跳过这些配置，继续更新其他配置
2. **返回统计**：响应中的 `updated_count` 表示实际成功更新的配置数量
3. **日志记录**：失败的更新会记录到日志中，便于排查问题

## 使用示例

### cURL 示例

```bash
curl -X PUT http://localhost:8080/healthcheck/ssl-domains/alert-config \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "config_ids": [1, 2, 3],
    "critical_days": 7,
    "warning_days": 30,
    "notice_days": 60
  }'
```

### JavaScript 示例

```javascript
const response = await fetch('http://localhost:8080/healthcheck/ssl-domains/alert-config', {
  method: 'PUT',
  headers: {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer YOUR_TOKEN'
  },
  body: JSON.stringify({
    config_ids: [1, 2, 3],
    critical_days: 7,
    warning_days: 30,
    notice_days: 60
  })
});

const result = await response.json();
console.log(`成功更新 ${result.data.updated_count} 个配置`);
```

### Python 示例

```python
import requests

url = 'http://localhost:8080/healthcheck/ssl-domains/alert-config'
headers = {
    'Content-Type': 'application/json',
    'Authorization': 'Bearer YOUR_TOKEN'
}
data = {
    'config_ids': [1, 2, 3],
    'critical_days': 7,
    'warning_days': 30,
    'notice_days': 60
}

response = requests.put(url, json=data, headers=headers)
result = response.json()

print(f"成功更新 {result['data']['updated_count']} 个配置")
```

## 使用场景

### 场景1：统一调整所有证书的告警阈值

当需要调整整个系统的告警策略时，可以先查询所有SSL证书配置的ID，然后批量更新：

```bash
# 1. 查询所有SSL证书配置
curl -X GET "http://localhost:8080/healthcheck/configs?type=ssl_cert" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 2. 提取所有ID并批量更新
curl -X PUT http://localhost:8080/healthcheck/ssl-domains/alert-config \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "config_ids": [1, 2, 3, 4, 5],
    "critical_days": 10,
    "warning_days": 45,
    "notice_days": 90
  }'
```

### 场景2：按环境分组调整

为不同环境的证书设置不同的告警阈值：

```javascript
// 生产环境证书 - 更严格的阈值
await updateAlertConfig([1, 2, 3], 7, 30, 60);

// 测试环境证书 - 较宽松的阈值
await updateAlertConfig([4, 5, 6], 3, 14, 30);

async function updateAlertConfig(configIds, critical, warning, notice) {
  const response = await fetch('http://localhost:8080/healthcheck/ssl-domains/alert-config', {
    method: 'PUT',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': 'Bearer YOUR_TOKEN'
    },
    body: JSON.stringify({
      config_ids: configIds,
      critical_days: critical,
      warning_days: warning,
      notice_days: notice
    })
  });
  return await response.json();
}
```

## 注意事项

1. **告警阈值顺序**：必须满足 `critical_days < warning_days < notice_days`，否则请求会被拒绝
2. **配置ID验证**：不存在的配置ID会被自动跳过，不会导致整个请求失败
3. **权限要求**：此接口需要管理员权限，普通用户无法访问
4. **批量大小**：建议每次更新不超过100个配置，避免请求超时
5. **立即生效**：更新后的阈值会在下次证书检查时立即生效

## 相关接口

- `POST /healthcheck/ssl-domains/import` - 批量导入SSL域名
- `GET /healthcheck/configs?type=ssl_cert` - 查询所有SSL证书配置
- `GET /healthcheck/configs/:id` - 查询单个配置详情
- `PUT /healthcheck/configs/:id` - 更新单个配置（包括告警阈值）
