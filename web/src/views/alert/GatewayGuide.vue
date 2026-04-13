<template>
  <div class="gateway-guide">
    <a-card :bordered="false" title="告警网关接入指南">
      <template #extra>
        <a-tag color="blue">API Version v1</a-tag>
      </template>
      
      <a-alert
        message="什么是告警网关？"
        description="告警网关是统一的告警接收入口。它负责接收来自 Prometheus、Grafana 或自定义脚本的告警请求，经过清洗、静默检查和路由匹配后，通过飞书、钉钉等渠道发送通知。"
        type="info"
        show-icon
        style="margin-bottom: 24px"
      />

      <a-tabs v-model:activeKey="activeTab">
        <a-tab-pane key="generic" tab="通用接入 (Generic)">
          <div class="guide-content">
            <h3>接口地址</h3>
            <div class="url-box">
              <code>{{ baseUrl }}/app/api/v1/gateway/event/generic</code>
              <a-button type="link" size="small" @click="copyText(`${baseUrl}/app/api/v1/gateway/event/generic`)">复制</a-button>
            </div>

            <h3>请求格式 (JSON)</h3>
            <pre class="code-block">{{ genericExample }}</pre>

            <h3>参数说明</h3>
            <a-table :columns="paramColumns" :data-source="paramData" :pagination="false" size="small" bordered />
          </div>
        </a-tab-pane>

        <a-tab-pane key="prometheus" tab="Prometheus">
          <div class="guide-content">
            <h3>Alertmanager 配置 (alertmanager.yml)</h3>
            <p>在 <code>receivers</code> 部分添加 webhook 配置：</p>
            <pre class="code-block">{{ prometheusExample }}</pre>
            <p>接口地址: <code>{{ baseUrl }}/app/api/v1/gateway/event/prometheus</code></p>
          </div>
        </a-tab-pane>

        <a-tab-pane key="grafana" tab="Grafana">
          <div class="guide-content">
            <h3>Webhook 配置</h3>
            <ul>
              <li><strong>Type</strong>: Webhook</li>
              <li><strong>URL</strong>: <code>{{ baseUrl }}/app/api/v1/gateway/event/grafana</code></li>
              <li><strong>HTTP Method</strong>: POST</li>
            </ul>
            <p>Grafana 会自动发送其标准格式的告警 Payload，网关会自动解析。</p>
          </div>
        </a-tab-pane>
        
        <a-tab-pane key="test" tab="在线测试">
          <div class="test-panel">
            <a-form layout="vertical">
              <a-form-item label="测试接口">
                <a-select v-model:value="testSource">
                  <a-select-option value="generic">通用 (Generic)</a-select-option>
                  <a-select-option value="prometheus">Prometheus</a-select-option>
                </a-select>
              </a-form-item>
              <a-form-item label="Payload">
                <a-textarea v-model:value="testPayload" :rows="10" style="font-family: monospace" />
              </a-form-item>
              <a-form-item>
                <a-button type="primary" :loading="sending" @click="sendTest">发送测试请求</a-button>
              </a-form-item>
            </a-form>
            
            <div v-if="testResult" class="result-box">
              <h4>响应结果:</h4>
              <pre>{{ testResult }}</pre>
            </div>
          </div>
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { message } from 'ant-design-vue'
import request from '@/services/api'

const activeTab = ref('generic')
const baseUrl = window.location.origin
const sending = ref(false)
const testResult = ref('')
const testSource = ref('generic')

const genericExample = `{
  "title": "服务OOM告警",
  "content": "检测到服务内存使用率超过90%，请及时处理。",
  "level": "critical",
  "labels": {
    "alertname": "OOMAlert",
    "service": "payment-service",
    "env": "prod"
  }
}`

const prometheusExample = `receivers:
- name: 'web.hook'
  webhook_configs:
  - url: '${window.location.origin}/app/api/v1/gateway/event/prometheus'
    send_resolved: true`

const testPayload = ref(genericExample)

const paramColumns = [
  { title: '字段', dataIndex: 'field', width: 150 },
  { title: '类型', dataIndex: 'type', width: 100 },
  { title: '必填', dataIndex: 'required', width: 80 },
  { title: '说明', dataIndex: 'desc' }
]

const paramData = [
  { field: 'title', type: 'string', required: '是', desc: '告警标题' },
  { field: 'content', type: 'string', required: '是', desc: '告警详细内容' },
  { field: 'level', type: 'string', required: '否', desc: '级别: info, warning, error, critical (默认 warning)' },
  { field: 'labels', type: 'object', required: '否', desc: '标签集合，用于路由匹配和静默规则。建议包含 alertname' },
  { field: 'fingerprint', type: 'string', required: '否', desc: '告警指纹，用于去重。如果不传，网关会自动生成' }
]

const copyText = (text: string) => {
  navigator.clipboard.writeText(text).then(() => {
    message.success('复制成功')
  })
}

const sendTest = async () => {
  try {
    const json = JSON.parse(testPayload.value)
    sending.value = true
    testResult.value = ''
    
    // 直接调用网关接口
    // 注意：在开发环境中，前端代理可能已经配置了 /app/api -> 后端
    // 如果没有，这里可能需要调整 URL
    const res = await request.post(`/gateway/event/${testSource.value}`, json)
    
    testResult.value = JSON.stringify(res, null, 2)
    if (res.code === 0 || res.message) {
      message.success('发送成功')
    } else {
      message.warn('发送可能失败，请检查响应')
    }
  } catch (e: any) {
    message.error('发送失败: ' + e.message)
    testResult.value = e.message
  } finally {
    sending.value = false
  }
}
</script>

<style scoped>
.gateway-guide {
  max-width: 1200px;
  margin: 0 auto;
}
.guide-content {
  padding: 16px 0;
}
.url-box {
  background: #f5f5f5;
  padding: 12px;
  border-radius: 4px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.url-box code {
  font-family: monospace;
  color: #eb2f96;
}
.code-block {
  background: #282c34;
  color: #abb2bf;
  padding: 16px;
  border-radius: 4px;
  font-family: monospace;
  white-space: pre-wrap;
  margin-bottom: 16px;
}
h3 {
  margin-bottom: 12px;
  margin-top: 24px;
  font-weight: 600;
}
h3:first-child {
  margin-top: 0;
}
.result-box {
  margin-top: 24px;
  background: #f0f2f5;
  padding: 16px;
  border-radius: 4px;
}
</style>
