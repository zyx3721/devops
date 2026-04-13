<template>
  <div class="ssl-cert-check">
    <div class="page-header">
      <h1>SSL 证书检查</h1>
      <a-space>
        <a-button @click="showImportModal">
          <template #icon><ImportOutlined /></template>
          批量导入
        </a-button>
        <a-button type="primary" @click="showConfigModal()">
          <template #icon><PlusOutlined /></template>
          添加域名
        </a-button>
      </a-space>
    </div>

    <!-- 统计卡片 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-card>
          <a-statistic title="监控域名" :value="stats.total" :value-style="{ color: '#1890ff' }">
            <template #prefix><GlobalOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="即将过期" :value="stats.expiring" :value-style="{ color: '#fa8c16' }">
            <template #prefix><ClockCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="已过期" :value="stats.expired" :value-style="{ color: '#cf1322' }">
            <template #prefix><ExclamationCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="正常" :value="stats.normal" :value-style="{ color: '#52c41a' }">
            <template #prefix><CheckCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 筛选和搜索 -->
    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-form layout="inline">
        <a-form-item label="告警级别">
          <a-select v-model:value="filters.alert_level" placeholder="全部" style="width: 120px" allow-clear @change="fetchCerts">
            <a-select-option value="expired">已过期</a-select-option>
            <a-select-option value="critical">严重</a-select-option>
            <a-select-option value="warning">警告</a-select-option>
            <a-select-option value="notice">提醒</a-select-option>
            <a-select-option value="normal">正常</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="关键字">
          <a-input v-model:value="filters.keyword" placeholder="搜索域名" style="width: 200px" @pressEnter="fetchCerts" />
        </a-form-item>
        <a-form-item label="排序">
          <a-select v-model:value="filters.sort_by" style="width: 150px" @change="fetchCerts">
            <a-select-option value="days_asc">剩余天数↑</a-select-option>
            <a-select-option value="days_desc">剩余天数↓</a-select-option>
            <a-select-option value="created_desc">创建时间↓</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" @click="fetchCerts">查询</a-button>
            <a-button @click="resetFilters">重置</a-button>
            <a-button @click="exportReport" :loading="exporting">
              <template #icon><ExportOutlined /></template>
              导出报告
            </a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 证书列表 -->
    <a-card :bordered="false">
      <a-table 
        :columns="columns" 
        :data-source="certs" 
        :loading="loading" 
        row-key="id" 
        :pagination="pagination" 
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'url'">
            <div>
              <a :href="`https://${record.url}`" target="_blank">{{ record.url }}</a>
              <div v-if="record.cert_subject" class="sub-text">{{ record.cert_subject }}</div>
            </div>
          </template>
          
          <template v-if="column.key === 'cert_days_remaining'">
            <a-tag :color="getDaysColor(record.cert_days_remaining, record.last_alert_level)">
              {{ record.cert_days_remaining != null ? `${record.cert_days_remaining} 天` : '-' }}
            </a-tag>
          </template>
          
          <template v-if="column.key === 'last_alert_level'">
            <a-badge :status="getAlertBadge(record.last_alert_level)" :text="getAlertText(record.last_alert_level)" />
          </template>
          
          <template v-if="column.key === 'cert_expiry_date'">
            {{ formatDate(record.cert_expiry_date) }}
          </template>
          
          <template v-if="column.key === 'cert_issuer'">
            <span class="sub-text">{{ record.cert_issuer || '-' }}</span>
          </template>
          
          <template v-if="column.key === 'last_check_at'">
            {{ formatTime(record.last_check_at) }}
          </template>
          
          <template v-if="column.key === 'enabled'">
            <a-switch :checked="record.enabled" @change="toggleCert(record)" size="small" />
          </template>
          
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="checkNow(record)" :loading="checkingId === record.id">
                检查
              </a-button>
              <a-button type="link" size="small" @click="showConfigModal(record)">
                编辑
              </a-button>
              <a-popconfirm title="确定删除？" @confirm="deleteCert(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 批量导入弹窗 -->
    <a-modal 
      v-model:open="importModalVisible" 
      title="批量导入域名" 
      @ok="importDomains" 
      :confirm-loading="importing"
      width="700px"
    >
      <a-form layout="vertical">
        <a-form-item label="域名列表">
          <a-textarea 
            v-model:value="importText" 
            :rows="10" 
            placeholder="每行一个域名，支持带端口，例如：&#10;example.com&#10;api.example.com:8443&#10;www.example.com"
          />
          <div class="hint-text">提示：每行一个域名，支持带端口（默认443）</div>
        </a-form-item>
        
        <a-divider orientation="left">默认配置</a-divider>
        
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="检查间隔（秒）">
              <a-input-number v-model:value="importConfig.interval" :min="3600" :max="86400" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="超时时间（秒）">
              <a-input-number v-model:value="importConfig.timeout" :min="5" :max="60" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="重试次数">
              <a-input-number v-model:value="importConfig.retry_count" :min="0" :max="5" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="严重告警（天）">
              <a-input-number v-model:value="importConfig.critical_days" :min="1" :max="30" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="警告告警（天）">
              <a-input-number v-model:value="importConfig.warning_days" :min="1" :max="60" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="提醒告警（天）">
              <a-input-number v-model:value="importConfig.notice_days" :min="1" :max="90" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="告警平台">
              <a-select v-model:value="importConfig.alert_platform" placeholder="选择平台" allow-clear>
                <a-select-option value="feishu">飞书</a-select-option>
                <a-select-option value="dingtalk">钉钉</a-select-option>
                <a-select-option value="wechatwork">企业微信</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="告警机器人">
              <a-select v-model:value="importConfig.alert_bot_id" placeholder="选择机器人" allow-clear>
                <a-select-option v-for="bot in currentAlertBots" :key="bot.id" :value="bot.id">
                  {{ bot.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>

    <!-- 配置编辑弹窗 -->
    <a-modal 
      v-model:open="configModalVisible" 
      :title="editingId ? '编辑证书配置' : '添加域名'" 
      @ok="saveConfig" 
      :confirm-loading="saving"
      width="600px"
    >
      <a-form :model="editingConfig" layout="vertical">
        <a-form-item label="域名" required>
          <a-input v-model:value="editingConfig.url" placeholder="example.com 或 example.com:8443" />
        </a-form-item>
        
        <a-form-item label="名称">
          <a-input v-model:value="editingConfig.name" placeholder="可选，默认使用域名" />
        </a-form-item>
        
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="检查间隔（秒）">
              <a-input-number v-model:value="editingConfig.interval" :min="3600" :max="86400" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="超时时间（秒）">
              <a-input-number v-model:value="editingConfig.timeout" :min="5" :max="60" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="重试次数">
              <a-input-number v-model:value="editingConfig.retry_count" :min="0" :max="5" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-divider orientation="left">告警阈值</a-divider>
        
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="严重（天）">
              <a-input-number v-model:value="editingConfig.critical_days" :min="1" :max="30" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="警告（天）">
              <a-input-number v-model:value="editingConfig.warning_days" :min="1" :max="60" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="提醒（天）">
              <a-input-number v-model:value="editingConfig.notice_days" :min="1" :max="90" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-divider orientation="left">告警配置</a-divider>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="告警平台">
              <a-select v-model:value="editingConfig.alert_platform" placeholder="选择平台" allow-clear>
                <a-select-option value="feishu">飞书</a-select-option>
                <a-select-option value="dingtalk">钉钉</a-select-option>
                <a-select-option value="wechatwork">企业微信</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="告警机器人">
              <a-select v-model:value="editingConfig.alert_bot_id" placeholder="选择机器人" allow-clear>
                <a-select-option v-for="bot in currentAlertBots" :key="bot.id" :value="bot.id">
                  {{ bot.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item>
              <a-checkbox v-model:checked="editingConfig.enabled">启用检查</a-checkbox>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item>
              <a-checkbox v-model:checked="editingConfig.alert_enabled">启用告警</a-checkbox>
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { 
  PlusOutlined, ImportOutlined, ExportOutlined, GlobalOutlined, 
  ClockCircleOutlined, ExclamationCircleOutlined, CheckCircleOutlined 
} from '@ant-design/icons-vue'
import { sslCertApi, type SSLCertConfig } from '@/services/healthcheck'
import { feishuBotApi, type FeishuBot } from '@/services/feishu'
import { dingtalkBotApi, type DingtalkBot } from '@/services/dingtalk'
import { wechatworkBotApi, type WechatWorkBot } from '@/services/wechatwork'

const loading = ref(false)
const importing = ref(false)
const exporting = ref(false)
const saving = ref(false)
const importModalVisible = ref(false)
const configModalVisible = ref(false)
const editingId = ref<number | undefined>(undefined)
const checkingId = ref<number | null>(null)

const certs = ref<SSLCertConfig[]>([])
const stats = ref({ total: 0, expiring: 0, expired: 0, normal: 0 })
const feishuBots = ref<FeishuBot[]>([])
const dingtalkBots = ref<DingtalkBot[]>([])
const wechatworkBots = ref<WechatWorkBot[]>([])

const importText = ref('')
const importConfig = reactive({
  interval: 86400,
  timeout: 10,
  retry_count: 3,
  critical_days: 7,
  warning_days: 30,
  notice_days: 60,
  alert_platform: '',
  alert_bot_id: undefined as number | undefined
})

const editingConfig = reactive<Partial<SSLCertConfig>>({
  name: '',
  url: '',
  interval: 86400,
  timeout: 10,
  retry_count: 3,
  critical_days: 7,
  warning_days: 30,
  notice_days: 60,
  enabled: true,
  alert_enabled: true,
  alert_platform: '',
  alert_bot_id: undefined
})

const filters = reactive({
  alert_level: undefined as string | undefined,
  keyword: '',
  sort_by: 'days_asc'
})

const pagination = reactive({ current: 1, pageSize: 20, total: 0 })

const currentAlertBots = computed(() => {
  const platform = importModalVisible.value ? importConfig.alert_platform : editingConfig.alert_platform
  switch (platform) {
    case 'feishu': return feishuBots.value
    case 'dingtalk': return dingtalkBots.value
    case 'wechatwork': return wechatworkBots.value
    default: return []
  }
})

const columns = [
  { title: '域名', dataIndex: 'url', key: 'url', width: 250 },
  { title: '剩余天数', dataIndex: 'cert_days_remaining', key: 'cert_days_remaining', width: 100 },
  { title: '告警级别', dataIndex: 'last_alert_level', key: 'last_alert_level', width: 100 },
  { title: '过期时间', dataIndex: 'cert_expiry_date', key: 'cert_expiry_date', width: 120 },
  { title: '颁发者', dataIndex: 'cert_issuer', key: 'cert_issuer', width: 150 },
  { title: '最后检查', dataIndex: 'last_check_at', key: 'last_check_at', width: 170 },
  { title: '启用', dataIndex: 'enabled', key: 'enabled', width: 70 },
  { title: '操作', key: 'action', width: 150, fixed: 'right' }
]

const getDaysColor = (days: number | null, level: string) => {
  if (days === null) return 'default'
  if (level === 'expired') return 'red'
  if (level === 'critical') return 'red'
  if (level === 'warning') return 'orange'
  if (level === 'notice') return 'blue'
  return 'green'
}

const getAlertBadge = (level: string) => {
  const map: Record<string, string> = {
    expired: 'error',
    critical: 'error',
    warning: 'warning',
    notice: 'processing',
    normal: 'success'
  }
  return map[level] || 'default'
}

const getAlertText = (level: string) => {
  const map: Record<string, string> = {
    expired: '已过期',
    critical: '严重',
    warning: '警告',
    notice: '提醒',
    normal: '正常'
  }
  return map[level] || level
}

const formatDate = (date: string | undefined) => {
  return date ? date.substring(0, 10) : '-'
}

const formatTime = (time: string | undefined) => {
  return time ? time.replace('T', ' ').substring(0, 19) : '-'
}

const fetchCerts = async () => {
  loading.value = true
  try {
    const response = await sslCertApi.list({
      page: pagination.current,
      page_size: pagination.pageSize,
      alert_level: filters.alert_level,
      keyword: filters.keyword,
      sort_by: filters.sort_by
    })
    if (response.code === 0 && response.data) {
      certs.value = response.data.list || []
      pagination.total = response.data.total
      updateStats()
    }
  } catch (error) {
    console.error('获取证书列表失败', error)
  } finally {
    loading.value = false
  }
}

const updateStats = () => {
  stats.value = {
    total: certs.value.length,
    expired: certs.value.filter(c => c.last_alert_level === 'expired').length,
    expiring: certs.value.filter(c => ['critical', 'warning', 'notice'].includes(c.last_alert_level || '')).length,
    normal: certs.value.filter(c => c.last_alert_level === 'normal').length
  }
}

const fetchBots = async () => {
  try {
    const [f, d, w] = await Promise.all([
      feishuBotApi.list(),
      dingtalkBotApi.list(),
      wechatworkBotApi.list()
    ])
    if (f.code === 0 && f.data) feishuBots.value = f.data.list || []
    if (d.code === 0 && d.data) dingtalkBots.value = d.data.list || []
    if (w.code === 0 && w.data) wechatworkBots.value = w.data.list || []
  } catch (e) {
    console.error('获取机器人列表失败', e)
  }
}

const onTableChange = (pag: any) => {
  pagination.current = pag.current
  fetchCerts()
}

const resetFilters = () => {
  filters.alert_level = undefined
  filters.keyword = ''
  filters.sort_by = 'days_asc'
  pagination.current = 1
  fetchCerts()
}

const showImportModal = () => {
  importText.value = ''
  importModalVisible.value = true
}

const importDomains = async () => {
  if (!importText.value.trim()) {
    message.error('请输入域名列表')
    return
  }
  
  const domains = importText.value
    .split('\n')
    .map(d => d.trim())
    .filter(d => d.length > 0)
  
  if (domains.length === 0) {
    message.error('请输入有效的域名')
    return
  }
  
  importing.value = true
  try {
    const response = await sslCertApi.importDomains({
      domains,
      ...importConfig
    })
    if (response.code === 0 && response.data) {
      message.success(`成功导入 ${response.data.success_count} 个域名，失败 ${response.data.failed_count} 个`)
      importModalVisible.value = false
      fetchCerts()
    } else {
      message.error(response.message || '导入失败')
    }
  } catch (error: any) {
    message.error(error.message || '导入失败')
  } finally {
    importing.value = false
  }
}

const showConfigModal = (cert?: SSLCertConfig) => {
  if (cert) {
    editingId.value = cert.id
    Object.assign(editingConfig, cert)
  } else {
    editingId.value = undefined
    Object.assign(editingConfig, {
      name: '',
      url: '',
      interval: 86400,
      timeout: 10,
      retry_count: 3,
      critical_days: 7,
      warning_days: 30,
      notice_days: 60,
      enabled: true,
      alert_enabled: true,
      alert_platform: '',
      alert_bot_id: undefined
    })
  }
  configModalVisible.value = true
}

const saveConfig = async () => {
  if (!editingConfig.url) {
    message.error('请填写域名')
    return
  }
  
  saving.value = true
  try {
    const data = {
      ...editingConfig,
      type: 'ssl_cert',
      name: editingConfig.name || editingConfig.url
    }
    
    const response = editingId.value
      ? await sslCertApi.update(editingId.value, data)
      : await sslCertApi.create(data)
    
    if (response.code === 0) {
      message.success(editingId.value ? '更新成功' : '添加成功')
      configModalVisible.value = false
      fetchCerts()
    } else {
      message.error(response.message || '保存失败')
    }
  } catch (error: any) {
    message.error(error.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const deleteCert = async (id: number | undefined) => {
  if (!id) return
  try {
    const response = await sslCertApi.delete(id)
    if (response.code === 0) {
      message.success('删除成功')
      fetchCerts()
    } else {
      message.error(response.message || '删除失败')
    }
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

const toggleCert = async (cert: SSLCertConfig) => {
  if (!cert.id) return
  try {
    const response = await sslCertApi.toggle(cert.id)
    if (response.code === 0) {
      message.success(response.data?.enabled ? '已启用' : '已禁用')
      fetchCerts()
    } else {
      message.error(response.message || '操作失败')
    }
  } catch (error: any) {
    message.error(error.message || '操作失败')
  }
}

const checkNow = async (cert: SSLCertConfig) => {
  if (!cert.id) return
  checkingId.value = cert.id
  try {
    const response = await sslCertApi.checkNow(cert.id)
    if (response.code === 0 && response.data) {
      if (response.data.status === 'healthy') {
        message.success(`检查通过，证书剩余 ${response.data.cert_days_remaining} 天`)
      } else {
        message.error(`检查失败: ${response.data.error_msg || '未知错误'}`)
      }
      fetchCerts()
    } else {
      message.error(response.message || '检查失败')
    }
  } catch (error: any) {
    message.error(error.message || '检查失败')
  } finally {
    checkingId.value = null
  }
}

const exportReport = async () => {
  exporting.value = true
  try {
    const response = await sslCertApi.exportReport(filters)
    if (response.code === 0 && response.data) {
      const blob = new Blob([JSON.stringify(response.data, null, 2)], { type: 'application/json' })
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `ssl-cert-report-${new Date().toISOString().substring(0, 10)}.json`
      a.click()
      window.URL.revokeObjectURL(url)
      message.success('导出成功')
    } else {
      message.error(response.message || '导出失败')
    }
  } catch (error: any) {
    message.error(error.message || '导出失败')
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  fetchCerts()
  fetchBots()
})
</script>

<style scoped>
.ssl-cert-check { padding: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h1 { font-size: 20px; font-weight: 500; margin: 0; }
.sub-text { color: #999; font-size: 12px; margin-top: 4px; }
.hint-text { color: #999; font-size: 12px; margin-top: 8px; }
</style>
