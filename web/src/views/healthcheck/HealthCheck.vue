<template>
  <div class="health-check">
    <div class="page-header">
      <h1>健康检查</h1>
      <a-space>
        <a-tag :color="getOverallStatusColor(overallStatus.status)" style="font-size: 14px; padding: 4px 12px">
          {{ getOverallStatusText(overallStatus.status) }}
        </a-tag>
        <a-button @click="$router.push('/healthcheck/ssl-cert')">
          <template #icon><SafetyCertificateOutlined /></template>
          SSL 证书检查
        </a-button>
        <a-button type="primary" @click="showConfigModal()">
          <template #icon><PlusOutlined /></template>
          添加检查
        </a-button>
      </a-space>
    </div>

    <!-- 统计卡片 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-card>
          <a-statistic title="检查项" :value="stats.enabled_count" :value-style="{ color: '#1890ff' }">
            <template #prefix><MonitorOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="健康" :value="stats.healthy_count" :value-style="{ color: '#52c41a' }">
            <template #prefix><CheckCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="异常" :value="stats.unhealthy_count" :value-style="{ color: '#cf1322' }">
            <template #prefix><CloseCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="成功率" :value="successRate" suffix="%" :precision="1" :value-style="{ color: successRate >= 90 ? '#52c41a' : '#fa8c16' }">
            <template #prefix><PercentageOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <a-tabs v-model:activeKey="activeTab">
      <!-- 检查配置 -->
      <a-tab-pane key="config" tab="检查配置">
        <a-card :bordered="false">
          <a-table :columns="configColumns" :data-source="configs" :loading="loadingConfigs" row-key="id" :pagination="configPagination" @change="onConfigTableChange">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'name'">
                <div>
                  <span>{{ record.name }}</span>
                  <div v-if="record.target_name" class="sub-text">{{ record.target_name }}</div>
                </div>
              </template>
              <template v-if="column.key === 'type'">
                <a-tag :color="getTypeColor(record.type)">{{ getTypeLabel(record.type) }}</a-tag>
              </template>
              <template v-if="column.key === 'last_status'">
                <a-badge :status="getStatusBadge(record.last_status)" :text="getStatusText(record.last_status)" />
              </template>
              <template v-if="column.key === 'last_check_at'">
                {{ formatTime(record.last_check_at) }}
              </template>
              <template v-if="column.key === 'interval'">
                {{ formatInterval(record.interval) }}
              </template>
              <template v-if="column.key === 'enabled'">
                <a-switch :checked="record.enabled" @change="toggleConfig(record)" size="small" />
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="checkNow(record)" :loading="checkingId === record.id">检查</a-button>
                  <a-button type="link" size="small" @click="showConfigModal(record)">编辑</a-button>
                  <a-popconfirm title="确定删除？" @confirm="deleteConfig(record.id)">
                    <a-button type="link" size="small" danger>删除</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 检查历史 -->
      <a-tab-pane key="history" tab="检查历史">
        <a-card :bordered="false">
          <a-table :columns="historyColumns" :data-source="histories" :loading="loadingHistories" row-key="id" :pagination="historyPagination" @change="onHistoryTableChange">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'created_at'">
                {{ formatTime(record.created_at) }}
              </template>
              <template v-if="column.key === 'type'">
                <a-tag :color="getTypeColor(record.type)">{{ getTypeLabel(record.type) }}</a-tag>
              </template>
              <template v-if="column.key === 'status'">
                <a-badge :status="getStatusBadge(record.status)" :text="getStatusText(record.status)" />
              </template>
              <template v-if="column.key === 'response_time_ms'">
                {{ record.response_time_ms }}ms
              </template>
              <template v-if="column.key === 'alert_sent'">
                <a-tag v-if="record.alert_sent" color="orange">已告警</a-tag>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <!-- 配置编辑弹窗 -->
    <a-modal v-model:open="configModalVisible" :title="editingConfigId ? '编辑检查配置' : '添加检查配置'" @ok="saveConfig" :confirm-loading="savingConfig" width="600px">
      <a-form :model="editingConfig" layout="vertical">
        <a-form-item label="名称" required>
          <a-input v-model:value="editingConfig.name" placeholder="如：Jenkins 主节点" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="检查类型" required>
              <a-select v-model:value="editingConfig.type" @change="onTypeChange">
                <a-select-option value="jenkins">Jenkins</a-select-option>
                <a-select-option value="k8s">Kubernetes</a-select-option>
                <a-select-option value="oa">OA 地址</a-select-option>
                <a-select-option value="custom">自定义 URL</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item v-if="editingConfig.type === 'jenkins'" label="Jenkins 实例">
              <a-select v-model:value="editingConfig.target_id" placeholder="选择实例">
                <a-select-option v-for="inst in jenkinsInstances" :key="inst.id" :value="inst.id">{{ inst.name }}</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item v-else-if="editingConfig.type === 'k8s'" label="K8s 集群">
              <a-select v-model:value="editingConfig.target_id" placeholder="选择集群">
                <a-select-option v-for="cluster in k8sClusters" :key="cluster.id" :value="cluster.id">{{ cluster.name }}</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item v-else-if="editingConfig.type === 'oa'" label="OA 地址">
              <a-select v-model:value="editingConfig.target_id" placeholder="选择地址">
                <a-select-option v-for="addr in oaAddresses" :key="addr.id" :value="addr.id">{{ addr.name }}</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item v-else label="URL">
              <a-input v-model:value="editingConfig.url" placeholder="https://..." />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="检查间隔（秒）">
              <a-input-number v-model:value="editingConfig.interval" :min="60" :max="86400" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="超时时间（秒）">
              <a-input-number v-model:value="editingConfig.timeout" :min="1" :max="60" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="重试次数">
              <a-input-number v-model:value="editingConfig.retry_count" :min="0" :max="10" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-divider orientation="left">告警配置</a-divider>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="告警平台">
              <a-select v-model:value="editingConfig.alert_platform" placeholder="选择平台" allow-clear @change="onAlertPlatformChange">
                <a-select-option value="feishu">飞书</a-select-option>
                <a-select-option value="dingtalk">钉钉</a-select-option>
                <a-select-option value="wechatwork">企业微信</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="告警机器人">
              <a-select v-model:value="editingConfig.alert_bot_id" placeholder="选择机器人" allow-clear :loading="loadingBots">
                <a-select-option v-for="bot in currentAlertBots" :key="bot.id" :value="bot.id">{{ bot.name }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item><a-checkbox v-model:checked="editingConfig.enabled">启用检查</a-checkbox></a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item><a-checkbox v-model:checked="editingConfig.alert_enabled">启用告警</a-checkbox></a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, MonitorOutlined, CheckCircleOutlined, CloseCircleOutlined, PercentageOutlined, SafetyCertificateOutlined } from '@ant-design/icons-vue'
import { healthCheckApi, type HealthCheckConfig, type HealthCheckHistory, type HealthCheckStats, type OverallStatus } from '@/services/healthcheck'
import { jenkinsInstanceApi } from '@/services/jenkins'
import { k8sClusterApi } from '@/services/k8s'
import { oaAddressApi } from '@/services/oa'
import { feishuBotApi, type FeishuBot } from '@/services/feishu'
import { dingtalkBotApi, type DingtalkBot } from '@/services/dingtalk'
import { wechatworkBotApi, type WechatWorkBot } from '@/services/wechatwork'

const activeTab = ref('config')
const loadingConfigs = ref(false)
const loadingHistories = ref(false)
const savingConfig = ref(false)
const configModalVisible = ref(false)
const editingConfigId = ref<number | undefined>(undefined)
const checkingId = ref<number | null>(null)

const configs = ref<HealthCheckConfig[]>([])
const histories = ref<HealthCheckHistory[]>([])
const stats = ref<HealthCheckStats>({ type_stats: [], status_stats: [], enabled_count: 0, healthy_count: 0, unhealthy_count: 0 })
const overallStatus = ref<OverallStatus>({ status: 'unknown', healthy: 0, unhealthy: 0, unknown: 0, total: 0 })
const jenkinsInstances = ref<any[]>([])
const k8sClusters = ref<any[]>([])
const oaAddresses = ref<any[]>([])
const feishuBots = ref<FeishuBot[]>([])
const dingtalkBots = ref<DingtalkBot[]>([])
const wechatworkBots = ref<WechatWorkBot[]>([])
const loadingBots = ref(false)

const editingConfig = reactive<Partial<HealthCheckConfig>>({
  name: '', type: 'jenkins', target_id: undefined, url: '', interval: 300, timeout: 10, retry_count: 3,
  enabled: true, alert_enabled: true, alert_platform: '', alert_bot_id: undefined
})

const currentAlertBots = computed(() => {
  switch (editingConfig.alert_platform) {
    case 'feishu': return feishuBots.value
    case 'dingtalk': return dingtalkBots.value
    case 'wechatwork': return wechatworkBots.value
    default: return []
  }
})

const configPagination = reactive({ current: 1, pageSize: 10, total: 0 })
const historyPagination = reactive({ current: 1, pageSize: 20, total: 0 })

const successRate = computed(() => {
  const total = stats.value.healthy_count + stats.value.unhealthy_count
  return total > 0 ? (stats.value.healthy_count / total) * 100 : 100
})

const configColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '类型', dataIndex: 'type', key: 'type', width: 100 },
  { title: '状态', dataIndex: 'last_status', key: 'last_status', width: 90 },
  { title: '最后检查', dataIndex: 'last_check_at', key: 'last_check_at', width: 170 },
  { title: '间隔', dataIndex: 'interval', key: 'interval', width: 80 },
  { title: '启用', dataIndex: 'enabled', key: 'enabled', width: 70 },
  { title: '操作', key: 'action', width: 150 }
]

const historyColumns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '配置', dataIndex: 'config_name', key: 'config_name' },
  { title: '类型', dataIndex: 'type', key: 'type', width: 100 },
  { title: '目标', dataIndex: 'target_name', key: 'target_name' },
  { title: '状态', dataIndex: 'status', key: 'status', width: 90 },
  { title: '响应时间', dataIndex: 'response_time_ms', key: 'response_time_ms', width: 100 },
  { title: '告警', key: 'alert_sent', width: 80 }
]

const typeLabels: Record<string, string> = { jenkins: 'Jenkins', k8s: 'K8s', oa: 'OA', custom: '自定义' }
const typeColors: Record<string, string> = { jenkins: 'red', k8s: 'purple', oa: 'orange', custom: 'blue' }

const getTypeLabel = (type: string) => typeLabels[type] || type
const getTypeColor = (type: string) => typeColors[type] || 'default'
const getStatusBadge = (status: string) => ({ healthy: 'success', unhealthy: 'error', unknown: 'default' }[status] || 'default')
const getStatusText = (status: string) => ({ healthy: '健康', unhealthy: '异常', unknown: '未知' }[status] || status)
const getOverallStatusColor = (status: string) => ({ healthy: 'green', unhealthy: 'red', unknown: 'default' }[status] || 'default')
const getOverallStatusText = (status: string) => ({ healthy: '系统健康', unhealthy: '存在异常', unknown: '状态未知' }[status] || status)
const formatTime = (time: string | undefined) => time ? time.replace('T', ' ').substring(0, 19) : '-'
const formatInterval = (seconds: number) => {
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟`
  return `${Math.floor(seconds / 3600)}小时`
}

const fetchConfigs = async () => {
  loadingConfigs.value = true
  try {
    const response = await healthCheckApi.listConfigs({ page: configPagination.current, page_size: configPagination.pageSize })
    if (response.code === 0 && response.data) {
      configs.value = response.data.list || []
      configPagination.total = response.data.total
    }
  } catch (error) { console.error('获取配置失败', error) }
  finally { loadingConfigs.value = false }
}

const fetchHistories = async () => {
  loadingHistories.value = true
  try {
    const response = await healthCheckApi.listHistories({ page: historyPagination.current, page_size: historyPagination.pageSize })
    if (response.code === 0 && response.data) {
      histories.value = response.data.list || []
      historyPagination.total = response.data.total
    }
  } catch (error) { console.error('获取历史失败', error) }
  finally { loadingHistories.value = false }
}

const fetchStats = async () => {
  try {
    const [statsRes, statusRes] = await Promise.all([healthCheckApi.getStats(), healthCheckApi.getOverallStatus()])
    if (statsRes.code === 0 && statsRes.data) { stats.value = statsRes.data }
    if (statusRes.code === 0 && statusRes.data) { overallStatus.value = statusRes.data }
  } catch (error) { console.error('获取统计失败', error) }
}

const fetchResources = async () => {
  try {
    const [jenkinsRes, k8sRes, oaRes] = await Promise.all([
      jenkinsInstanceApi.list(),
      k8sClusterApi.list(),
      oaAddressApi.list()
    ])
    if (jenkinsRes.code === 0 && jenkinsRes.data) { jenkinsInstances.value = jenkinsRes.data.items || [] }
    if (k8sRes.code === 0 && k8sRes.data) { k8sClusters.value = k8sRes.data.items || [] }
    if (oaRes.code === 0 && oaRes.data) { oaAddresses.value = oaRes.data.list || [] }
  } catch (error) { console.error('获取资源列表失败', error) }
}

const fetchBots = async () => {
  loadingBots.value = true
  try {
    const [f, d, w] = await Promise.all([feishuBotApi.list(), dingtalkBotApi.list(), wechatworkBotApi.list()])
    if (f.code === 0 && f.data) feishuBots.value = f.data.list || []
    if (d.code === 0 && d.data) dingtalkBots.value = d.data.list || []
    if (w.code === 0 && w.data) wechatworkBots.value = w.data.list || []
  } catch (e) { console.error('获取机器人列表失败', e) }
  finally { loadingBots.value = false }
}

const onConfigTableChange = (pag: any) => { configPagination.current = pag.current; fetchConfigs() }
const onHistoryTableChange = (pag: any) => { historyPagination.current = pag.current; fetchHistories() }
const onTypeChange = () => { editingConfig.target_id = undefined; editingConfig.url = '' }
const onAlertPlatformChange = () => { editingConfig.alert_bot_id = undefined }

const showConfigModal = (config?: HealthCheckConfig) => {
  if (config) {
    editingConfigId.value = config.id
    Object.assign(editingConfig, config)
  } else {
    editingConfigId.value = undefined
    Object.assign(editingConfig, { name: '', type: 'jenkins', target_id: undefined, url: '', interval: 300, timeout: 10, retry_count: 3, enabled: true, alert_enabled: true, alert_platform: '', alert_bot_id: undefined })
  }
  configModalVisible.value = true
}

const saveConfig = async () => {
  if (!editingConfig.name) { message.error('请填写名称'); return }
  savingConfig.value = true
  try {
    const response = editingConfigId.value
      ? await healthCheckApi.updateConfig(editingConfigId.value, editingConfig)
      : await healthCheckApi.createConfig(editingConfig)
    if (response.code === 0) {
      message.success(editingConfigId.value ? '更新成功' : '添加成功')
      configModalVisible.value = false
      fetchConfigs()
      fetchStats()
    } else { message.error(response.message || '保存失败') }
  } catch (error: any) { message.error(error.message || '保存失败') }
  finally { savingConfig.value = false }
}

const deleteConfig = async (id: number) => {
  try {
    const response = await healthCheckApi.deleteConfig(id)
    if (response.code === 0) { message.success('删除成功'); fetchConfigs(); fetchStats() }
    else { message.error(response.message || '删除失败') }
  } catch (error: any) { message.error(error.message || '删除失败') }
}

const toggleConfig = async (config: HealthCheckConfig) => {
  if (!config.id) return
  try {
    const response = await healthCheckApi.toggleConfig(config.id)
    if (response.code === 0) {
      message.success(response.data?.enabled ? '已启用' : '已禁用')
      fetchConfigs()
      fetchStats()
    } else { message.error(response.message || '操作失败') }
  } catch (error: any) { message.error(error.message || '操作失败') }
}

const checkNow = async (config: HealthCheckConfig) => {
  if (!config.id) return
  checkingId.value = config.id
  try {
    const response = await healthCheckApi.checkNow(config.id)
    if (response.code === 0 && response.data) {
      if (response.data.status === 'healthy') {
        message.success(`检查通过，响应时间: ${response.data.response_time_ms}ms`)
      } else {
        message.error(`检查失败: ${response.data.error_msg || '未知错误'}`)
      }
      fetchConfigs()
      fetchHistories()
      fetchStats()
    } else { message.error(response.message || '检查失败') }
  } catch (error: any) { message.error(error.message || '检查失败') }
  finally { checkingId.value = null }
}

onMounted(() => {
  fetchConfigs()
  fetchHistories()
  fetchStats()
  fetchResources()
  fetchBots()
})
</script>

<style scoped>
.health-check { padding: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.page-header h1 { font-size: 20px; font-weight: 500; margin: 0; }
.sub-text { color: #999; font-size: 12px; }

/* 优化卡片间距 */
:deep(.ant-card) {
  margin-bottom: 16px;
}

/* 优化统计卡片 */
:deep(.ant-statistic) {
  text-align: center;
}

/* 限制表格高度，避免页面过长 */
:deep(.ant-table-wrapper) {
  max-height: 600px;
}

:deep(.ant-table-body) {
  max-height: 500px;
  overflow-y: auto;
}

/* 优化滚动条 */
:deep(.ant-table-body::-webkit-scrollbar) {
  width: 8px;
  height: 8px;
}

:deep(.ant-table-body::-webkit-scrollbar-track) {
  background: #f0f0f0;
  border-radius: 4px;
}

:deep(.ant-table-body::-webkit-scrollbar-thumb) {
  background: #bfbfbf;
  border-radius: 4px;
}

:deep(.ant-table-body::-webkit-scrollbar-thumb:hover) {
  background: #999;
}
</style>
