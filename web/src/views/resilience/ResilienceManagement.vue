<template>
  <div class="resilience-management">
    <!-- 概览卡片 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="限流规则" :value="overview.rateLimit.totalConfigs">
            <template #suffix>
              <span style="font-size: 14px; color: #52c41a">/ {{ overview.rateLimit.enabledConfigs }} 启用</span>
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="请求拒绝率" :value="overview.rateLimit.rejectionRate" suffix="%" :precision="2">
            <template #prefix>
              <StopOutlined style="color: #ff4d4f" />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="熔断器状态">
            <template #formatter>
              <a-space>
                <a-badge status="success" :text="`${overview.circuitBreaker.closedBreakers} 正常`" />
                <a-badge status="warning" :text="`${overview.circuitBreaker.halfOpenBreakers} 半开`" />
                <a-badge status="error" :text="`${overview.circuitBreaker.openBreakers} 熔断`" />
              </a-space>
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="平均成功率" :value="overview.circuitBreaker.avgSuccessRate" suffix="%" :precision="2">
            <template #prefix>
              <CheckCircleOutlined style="color: #52c41a" />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <a-tabs v-model:activeKey="activeTab">
      <!-- 限流配置 Tab -->
      <a-tab-pane key="ratelimit" tab="限流配置">
        <a-card :bordered="false">
          <template #extra>
            <a-button type="primary" @click="showRateLimitModal()">
              <PlusOutlined /> 新增规则
            </a-button>
          </template>

          <a-table :columns="rateLimitColumns" :data-source="rateLimitConfigs" :loading="loading" row-key="id">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'limit_type'">
                <a-tag :color="getLimitTypeColor(record.limit_type)">{{ getLimitTypeText(record.limit_type) }}</a-tag>
              </template>
              <template v-if="column.key === 'limit'">
                <span>{{ record.requests_per_min }} 次/分钟</span>
                <a-tooltip title="突发容量">
                  <span style="color: #999; margin-left: 8px">(突发: {{ record.burst_size }})</span>
                </a-tooltip>
              </template>
              <template v-if="column.key === 'enabled'">
                <a-switch v-model:checked="record.enabled" size="small" @change="toggleRateLimitEnabled(record)" />
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="showRateLimitModal(record)">编辑</a-button>
                  <a-popconfirm title="确定删除？" @confirm="deleteRateLimitConfig(record.id)">
                    <a-button type="link" size="small" danger>删除</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 熔断器 Tab -->
      <a-tab-pane key="circuit" tab="熔断器">
        <a-card :bordered="false">
          <a-table :columns="circuitColumns" :data-source="circuitBreakers" :loading="circuitLoading" row-key="name">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'state'">
                <a-badge :status="getStateStatus(record.state)" :text="getStateText(record.state)" />
              </template>
              <template v-if="column.key === 'success_rate'">
                <a-progress :percent="record.success_rate" :status="getProgressStatus(record.success_rate)" size="small" style="width: 120px" />
              </template>
              <template v-if="column.key === 'stats'">
                <a-space>
                  <a-tooltip title="总请求">
                    <span><ApiOutlined /> {{ record.requests }}</span>
                  </a-tooltip>
                  <a-tooltip title="成功">
                    <span style="color: #52c41a"><CheckOutlined /> {{ record.successes }}</span>
                  </a-tooltip>
                  <a-tooltip title="失败">
                    <span style="color: #ff4d4f"><CloseOutlined /> {{ record.failures }}</span>
                  </a-tooltip>
                </a-space>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="showCircuitDetail(record)">详情</a-button>
                  <a-popconfirm title="确定重置熔断器？" @confirm="resetCircuitBreaker(record.name)">
                    <a-button type="link" size="small" :disabled="record.state === 'closed'">重置</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 实时监控 Tab -->
      <a-tab-pane key="monitor" tab="实时监控">
        <a-card :bordered="false">
          <a-row :gutter="16">
            <a-col :span="12">
              <a-card title="限流统计" size="small">
                <a-table :columns="rateLimitStatsColumns" :data-source="rateLimitStats" :pagination="false" size="small" row-key="key">
                  <template #bodyCell="{ column, record }">
                    <template v-if="column.key === 'usage'">
                      <a-progress :percent="Math.round((record.current_count / record.limit) * 100)" size="small" style="width: 100px" />
                    </template>
                    <template v-if="column.key === 'action'">
                      <a-popconfirm title="确定重置？" @confirm="resetRateLimit(record.key)">
                        <a-button type="link" size="small">重置</a-button>
                      </a-popconfirm>
                    </template>
                  </template>
                </a-table>
              </a-card>
            </a-col>
            <a-col :span="12">
              <a-card title="最近事件" size="small">
                <a-timeline>
                  <a-timeline-item v-for="(event, index) in recentEvents" :key="index" :color="getEventColor(event.type)">
                    <p style="margin-bottom: 4px">
                      <a-tag :color="getEventColor(event.type)" size="small">{{ getEventTypeText(event.type) }}</a-tag>
                      <span style="color: #999; font-size: 12px">{{ formatTime(event.time) }}</span>
                    </p>
                    <p style="margin: 0">{{ event.target }}: {{ event.description }}</p>
                  </a-timeline-item>
                </a-timeline>
              </a-card>
            </a-col>
          </a-row>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <!-- 限流配置弹窗 -->
    <a-modal v-model:open="rateLimitModalVisible" :title="editingRateLimit ? '编辑限流规则' : '新增限流规则'" @ok="saveRateLimitConfig" :confirm-loading="saving">
      <a-form :model="rateLimitForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="规则名称" required>
          <a-input v-model:value="rateLimitForm.name" placeholder="如：api_read" />
        </a-form-item>
        <a-form-item label="匹配路径" required>
          <a-input v-model:value="rateLimitForm.endpoint" placeholder="如：/api/v1/*" />
        </a-form-item>
        <a-form-item label="限流类型" required>
          <a-select v-model:value="rateLimitForm.limit_type">
            <a-select-option value="global">全局限流</a-select-option>
            <a-select-option value="ip">按 IP 限流</a-select-option>
            <a-select-option value="user">按用户限流</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="请求限制" required>
          <a-input-number v-model:value="rateLimitForm.requests_per_min" :min="1" style="width: 150px" />
          <span style="margin-left: 8px">次/分钟</span>
        </a-form-item>
        <a-form-item label="突发容量">
          <a-input-number v-model:value="rateLimitForm.burst_size" :min="1" style="width: 150px" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="rateLimitForm.description" :rows="2" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 熔断器详情弹窗 -->
    <a-modal v-model:open="circuitDetailVisible" :title="`熔断器详情: ${currentCircuit?.name}`" :footer="null" width="600px">
      <template v-if="currentCircuit">
        <a-descriptions :column="2" bordered size="small">
          <a-descriptions-item label="名称">{{ currentCircuit.name }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-badge :status="getStateStatus(currentCircuit.state)" :text="getStateText(currentCircuit.state)" />
          </a-descriptions-item>
          <a-descriptions-item label="总请求">{{ currentCircuit.requests }}</a-descriptions-item>
          <a-descriptions-item label="成功率">{{ currentCircuit.success_rate?.toFixed(2) }}%</a-descriptions-item>
          <a-descriptions-item label="成功次数">{{ currentCircuit.successes }}</a-descriptions-item>
          <a-descriptions-item label="失败次数">{{ currentCircuit.failures }}</a-descriptions-item>
          <a-descriptions-item label="连续失败">{{ currentCircuit.consecutive_fails }}</a-descriptions-item>
          <a-descriptions-item label="最后失败">{{ currentCircuit.last_failure || '-' }}</a-descriptions-item>
        </a-descriptions>

        <a-divider>配置参数</a-divider>
        <a-descriptions :column="2" size="small">
          <a-descriptions-item label="失败阈值">5 次</a-descriptions-item>
          <a-descriptions-item label="成功阈值">3 次</a-descriptions-item>
          <a-descriptions-item label="超时时间">30 秒</a-descriptions-item>
          <a-descriptions-item label="半开最大请求">5 次</a-descriptions-item>
        </a-descriptions>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, StopOutlined, CheckCircleOutlined, ApiOutlined, CheckOutlined, CloseOutlined } from '@ant-design/icons-vue'
import request from '@/utils/request'

const activeTab = ref('ratelimit')
const loading = ref(false)
const circuitLoading = ref(false)
const saving = ref(false)
const rateLimitModalVisible = ref(false)
const circuitDetailVisible = ref(false)
const editingRateLimit = ref<any>(null)
const currentCircuit = ref<any>(null)

const rateLimitConfigs = ref<any[]>([])
const circuitBreakers = ref<any[]>([])
const rateLimitStats = ref<any[]>([])
const recentEvents = ref<any[]>([])

const overview = reactive({
  rateLimit: { totalConfigs: 0, enabledConfigs: 0, totalRequests: 0, rejectedCount: 0, rejectionRate: 0 },
  circuitBreaker: { totalBreakers: 0, openBreakers: 0, halfOpenBreakers: 0, closedBreakers: 0, avgSuccessRate: 0 }
})

const rateLimitForm = reactive({
  name: '',
  endpoint: '',
  limit_type: 'global',
  requests_per_min: 100,
  burst_size: 20,
  description: ''
})

const rateLimitColumns = [
  { title: '规则名称', dataIndex: 'name', key: 'name' },
  { title: '匹配路径', dataIndex: 'endpoint', key: 'endpoint' },
  { title: '限流类型', key: 'limit_type', width: 100 },
  { title: '限制', key: 'limit', width: 180 },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '启用', key: 'enabled', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const circuitColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '状态', key: 'state', width: 100 },
  { title: '成功率', key: 'success_rate', width: 150 },
  { title: '统计', key: 'stats', width: 200 },
  { title: '连续失败', dataIndex: 'consecutive_fails', key: 'consecutive_fails', width: 100 },
  { title: '操作', key: 'action', width: 120 }
]

const rateLimitStatsColumns = [
  { title: 'Key', dataIndex: 'key', key: 'key', ellipsis: true },
  { title: '使用率', key: 'usage', width: 130 },
  { title: '剩余', dataIndex: 'remaining', key: 'remaining', width: 60 },
  { title: '操作', key: 'action', width: 60 }
]

const getLimitTypeColor = (type: string) => ({ global: 'blue', ip: 'orange', user: 'green' }[type] || 'default')
const getLimitTypeText = (type: string) => ({ global: '全局', ip: '按IP', user: '按用户' }[type] || type)
const getStateStatus = (state: string) => ({ closed: 'success', 'half-open': 'warning', open: 'error' }[state] || 'default') as any
const getStateText = (state: string) => ({ closed: '正常', 'half-open': '半开', open: '熔断' }[state] || state)
const getProgressStatus = (rate: number) => rate >= 95 ? 'success' : rate >= 80 ? 'normal' : 'exception'
const getEventColor = (type: string) => ({ rate_limit: 'orange', circuit_open: 'red', circuit_close: 'green' }[type] || 'blue')
const getEventTypeText = (type: string) => ({ rate_limit: '限流', circuit_open: '熔断', circuit_close: '恢复' }[type] || type)
const formatTime = (time: string) => new Date(time).toLocaleString()

const fetchOverview = async () => {
  try {
    const res = await request.get('/resilience/overview')
    const data = res.data || res
    if (data) {
      Object.assign(overview.rateLimit, data.rate_limit || {})
      Object.assign(overview.circuitBreaker, data.circuit_breaker || {})
      recentEvents.value = data.recent_events || []
    }
  } catch (e) { console.error('获取概览失败', e) }
}

const fetchRateLimitConfigs = async () => {
  loading.value = true
  try {
    const res = await request.get('/resilience/ratelimit/configs')
    const data = res.data || res
    rateLimitConfigs.value = data?.items || []
  } catch (e) { console.error('获取限流配置失败', e) }
  finally { loading.value = false }
}

const fetchCircuitBreakers = async () => {
  circuitLoading.value = true
  try {
    const res = await request.get('/resilience/circuit/breakers')
    circuitBreakers.value = res.data || res || []
  } catch (e) { console.error('获取熔断器失败', e) }
  finally { circuitLoading.value = false }
}

const fetchRateLimitStats = async () => {
  try {
    const res = await request.get('/resilience/ratelimit/stats')
    rateLimitStats.value = res.data || res || []
  } catch (e) { console.error('获取限流统计失败', e) }
}

const showRateLimitModal = (record?: any) => {
  editingRateLimit.value = record || null
  if (record) {
    Object.assign(rateLimitForm, record)
  } else {
    Object.assign(rateLimitForm, { name: '', endpoint: '', limit_type: 'global', requests_per_min: 100, burst_size: 20, description: '' })
  }
  rateLimitModalVisible.value = true
}

const saveRateLimitConfig = async () => {
  if (!rateLimitForm.name || !rateLimitForm.endpoint) {
    message.warning('请填写必填项')
    return
  }
  saving.value = true
  try {
    if (editingRateLimit.value) {
      await request.put(`/resilience/ratelimit/configs/${editingRateLimit.value.id}`, rateLimitForm)
    } else {
      await request.post('/resilience/ratelimit/configs', rateLimitForm)
    }
    message.success('保存成功')
    rateLimitModalVisible.value = false
    fetchRateLimitConfigs()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

const deleteRateLimitConfig = async (id: number) => {
  try {
    await request.delete(`/resilience/ratelimit/configs/${id}`)
    message.success('删除成功')
    fetchRateLimitConfigs()
  } catch (e) { message.error('删除失败') }
}

const toggleRateLimitEnabled = async (record: any) => {
  try {
    await request.put(`/resilience/ratelimit/configs/${record.id}`, { ...record })
    message.success(record.enabled ? '已启用' : '已禁用')
  } catch (e) { message.error('操作失败') }
}

const resetRateLimit = async (key: string) => {
  try {
    await request.post('/resilience/ratelimit/reset', { key })
    message.success('重置成功')
    fetchRateLimitStats()
  } catch (e) { message.error('重置失败') }
}

const showCircuitDetail = (record: any) => {
  currentCircuit.value = record
  circuitDetailVisible.value = true
}

const resetCircuitBreaker = async (name: string) => {
  try {
    await request.post(`/resilience/circuit/breakers/${name}/reset`)
    message.success('熔断器已重置')
    fetchCircuitBreakers()
  } catch (e) { message.error('重置失败') }
}

onMounted(() => {
  fetchOverview()
  fetchRateLimitConfigs()
  fetchCircuitBreakers()
  fetchRateLimitStats()
})
</script>

<style scoped>
.resilience-management :deep(.ant-tabs-nav) {
  margin-bottom: 16px;
}
</style>
