<template>
  <div class="canary-detail">
    <!-- 基本信息 -->
    <a-descriptions title="基本信息" :column="2" bordered size="small">
      <a-descriptions-item label="应用">
        <a-tag color="blue">{{ record.app_name }}</a-tag>
      </a-descriptions-item>
      <a-descriptions-item label="环境">
        <a-tag :color="getEnvColor(record.env_name)">{{ getEnvLabel(record.env_name) }}</a-tag>
      </a-descriptions-item>
      <a-descriptions-item label="状态">
        <a-badge :status="getStatusBadge(status.status)" :text="getStatusText(status.status)" />
      </a-descriptions-item>
      <a-descriptions-item label="开始时间">{{ status.started_at || record.created_at }}</a-descriptions-item>
      <a-descriptions-item label="灰度策略" :span="2">
        <a-space>
          <a-tag color="purple">流量权重: {{ canaryPercent }}%</a-tag>
          <a-tag v-if="status.canary_header" color="cyan">Header: {{ status.canary_header }}</a-tag>
          <a-tag v-if="status.canary_cookie" color="orange">Cookie: {{ status.canary_cookie }}</a-tag>
        </a-space>
      </a-descriptions-item>
      <a-descriptions-item label="灰度镜像" :span="2">
        <a-typography-text code>{{ status.canary_image || record.image_tag }}</a-typography-text>
      </a-descriptions-item>
      <a-descriptions-item label="稳定镜像" :span="2">
        <a-typography-text code>{{ status.stable_image || '-' }}</a-typography-text>
      </a-descriptions-item>
    </a-descriptions>

    <!-- 灰度进度 -->
    <a-card title="灰度进度" style="margin-top: 16px" size="small">
      <a-row :gutter="24">
        <a-col :span="12">
          <div class="progress-section">
            <div class="progress-title">流量分配</div>
            <div ref="trafficChartRef" style="height: 200px"></div>
          </div>
        </a-col>
        <a-col :span="12">
          <div class="progress-section">
            <div class="progress-title">副本状态</div>
            <a-row :gutter="16" style="margin-top: 16px">
              <a-col :span="12">
                <a-statistic title="灰度副本" :value="`${status.canary_ready}/${status.canary_replicas}`">
                  <template #suffix>
                    <CheckCircleOutlined v-if="status.canary_healthy" style="color: #52c41a" />
                    <WarningOutlined v-else style="color: #faad14" />
                  </template>
                </a-statistic>
              </a-col>
              <a-col :span="12">
                <a-statistic title="稳定副本" :value="`${status.stable_ready}/${status.stable_replicas}`" />
              </a-col>
            </a-row>
            <a-progress :percent="canaryPercent" :success="{ percent: stablePercent }" style="margin-top: 24px" />
            <div style="display: flex; justify-content: space-between; margin-top: 8px; font-size: 12px; color: #666">
              <span><span style="color: #1890ff">■</span> 灰度版本 {{ canaryPercent }}%</span>
              <span><span style="color: #52c41a">■</span> 稳定版本 {{ stablePercent }}%</span>
            </div>
          </div>
        </a-col>
      </a-row>
    </a-card>

    <!-- 灰度步骤 -->
    <a-card title="发布步骤" style="margin-top: 16px" size="small">
      <a-steps :current="currentStep" size="small">
        <a-step title="创建灰度" description="创建灰度Deployment" />
        <a-step title="流量切换" description="部分流量切到灰度版本" />
        <a-step title="观察验证" description="监控灰度版本运行状态" />
        <a-step title="全量/回滚" description="确认后全量发布或回滚" />
      </a-steps>
    </a-card>

    <!-- 健康指标 -->
    <a-card title="健康指标" style="margin-top: 16px" size="small">
      <a-row :gutter="16">
        <a-col :span="6">
          <a-statistic title="错误率" :value="status.error_rate || '0%'" :value-style="{ color: getErrorRateColor(status.error_rate) }" />
        </a-col>
        <a-col :span="6">
          <a-statistic title="健康状态" :value="status.canary_healthy ? '健康' : '异常'" :value-style="{ color: status.canary_healthy ? '#52c41a' : '#ff4d4f' }" />
        </a-col>
        <a-col :span="6">
          <a-statistic title="运行时长" :value="runningDuration" />
        </a-col>
        <a-col :span="6">
          <a-statistic title="重启次数" :value="status.restart_count || 0" />
        </a-col>
      </a-row>
    </a-card>

    <!-- 操作按钮 -->
    <div class="action-buttons" v-if="status.status === 'canary_running'">
      <a-space>
        <a-button @click="handleRefresh" :loading="refreshing">
          <ReloadOutlined /> 刷新状态
        </a-button>
        <a-button type="primary" @click="showAdjustModal = true">
          <SettingOutlined /> 调整比例
        </a-button>
        <a-popconfirm title="确定全量发布？灰度版本将替换稳定版本" @confirm="handlePromote">
          <a-button type="primary" style="background: #52c41a; border-color: #52c41a">
            <CheckOutlined /> 全量发布
          </a-button>
        </a-popconfirm>
        <a-popconfirm title="确定回滚？将删除灰度版本，恢复稳定版本" @confirm="handleRollback">
          <a-button danger>
            <RollbackOutlined /> 回滚
          </a-button>
        </a-popconfirm>
      </a-space>
    </div>

    <!-- 调整比例弹窗 -->
    <a-modal v-model:open="showAdjustModal" title="调整灰度比例" @ok="handleAdjust" :confirm-loading="adjusting">
      <a-form :label-col="{ span: 6 }">
        <a-form-item label="当前比例">
          <a-tag color="blue">{{ canaryPercent }}%</a-tag>
        </a-form-item>
        <a-form-item label="新比例">
          <a-slider v-model:value="newPercent" :min="1" :max="100" :marks="{ 10: '10%', 30: '30%', 50: '50%', 100: '100%' }" />
        </a-form-item>
        <a-form-item label="快速选择">
          <a-space>
            <a-button size="small" @click="newPercent = 10">10%</a-button>
            <a-button size="small" @click="newPercent = 30">30%</a-button>
            <a-button size="small" @click="newPercent = 50">50%</a-button>
            <a-button size="small" @click="newPercent = 80">80%</a-button>
            <a-button size="small" @click="newPercent = 100">100%</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { message } from 'ant-design-vue'
import {
  CheckCircleOutlined,
  WarningOutlined,
  ReloadOutlined,
  SettingOutlined,
  CheckOutlined,
  RollbackOutlined
} from '@ant-design/icons-vue'
import * as echarts from 'echarts'
import { canaryApi } from '@/services/canary'

const props = defineProps<{
  record: any
}>()

const emit = defineEmits(['refresh', 'close'])

const refreshing = ref(false)
const adjusting = ref(false)
const showAdjustModal = ref(false)
const newPercent = ref(30)
const trafficChartRef = ref<HTMLElement | null>(null)
let trafficChart: echarts.ECharts | null = null
let refreshTimer: number | null = null

const status = ref({
  status: props.record.status || 'canary_running',
  canary_replicas: 0,
  stable_replicas: 0,
  canary_ready: 0,
  stable_ready: 0,
  canary_image: props.record.image_tag,
  stable_image: '',
  started_at: '',
  canary_healthy: true,
  error_rate: '0%',
  restart_count: 0,
  traffic_percent: props.record.canary_percent || 10,
  canary_header: '',
  canary_cookie: ''
})

const canaryPercent = computed(() => {
  return status.value.traffic_percent || props.record.canary_percent || 10
})

const stablePercent = computed(() => 100 - canaryPercent.value)

const currentStep = computed(() => {
  if (status.value.status === 'success') return 4
  if (status.value.status === 'rolled_back') return 3
  if (status.value.canary_ready > 0) return 2
  return 1
})

const runningDuration = computed(() => {
  const startTime = status.value.started_at || props.record.created_at
  if (!startTime) return '-'
  const start = new Date(startTime).getTime()
  const now = Date.now()
  const diff = Math.floor((now - start) / 1000 / 60)
  if (diff < 60) return `${diff}分钟`
  return `${Math.floor(diff / 60)}小时${diff % 60}分钟`
})

const fetchStatus = async () => {
  try {
    const res = await canaryApi.getStatus(props.record.id)
    if (res?.code === 0 && res.data) {
      status.value = { ...status.value, ...res.data }
      updateChart()
    }
  } catch (error) {
    // ignore
  }
}

const handleRefresh = async () => {
  refreshing.value = true
  await fetchStatus()
  refreshing.value = false
  message.success('状态已刷新')
}

const handleAdjust = async () => {
  adjusting.value = true
  try {
    const res = await canaryApi.adjust(props.record.id, newPercent.value)
    if (res?.code === 0) {
      message.success('灰度比例已调整')
      showAdjustModal.value = false
      await fetchStatus()
      emit('refresh')
    } else {
      message.error(res?.message || '调整失败')
    }
  } catch (error) {
    message.error('调整失败')
  } finally {
    adjusting.value = false
  }
}

const handlePromote = async () => {
  try {
    const res = await canaryApi.promote(props.record.id)
    if (res?.code === 0) {
      message.success('全量发布成功')
      emit('refresh')
      emit('close')
    } else {
      message.error(res?.message || '全量发布失败')
    }
  } catch (error) {
    message.error('全量发布失败')
  }
}

const handleRollback = async () => {
  try {
    const res = await canaryApi.rollback(props.record.id)
    if (res?.code === 0) {
      message.success('回滚成功')
      emit('refresh')
      emit('close')
    } else {
      message.error(res?.message || '回滚失败')
    }
  } catch (error) {
    message.error('回滚失败')
  }
}

const initChart = () => {
  if (!trafficChartRef.value) return
  trafficChart = echarts.init(trafficChartRef.value)
  updateChart()
}

const updateChart = () => {
  if (!trafficChart) return
  trafficChart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: {c}%' },
    series: [{
      type: 'pie',
      radius: ['50%', '70%'],
      avoidLabelOverlap: false,
      label: { show: true, position: 'center', formatter: `{a|灰度}\n{b|${canaryPercent.value}%}`, rich: { a: { fontSize: 12, color: '#666' }, b: { fontSize: 24, fontWeight: 'bold', color: '#1890ff' } } },
      data: [
        { value: canaryPercent.value, name: '灰度版本', itemStyle: { color: '#1890ff' } },
        { value: stablePercent.value, name: '稳定版本', itemStyle: { color: '#52c41a' } }
      ]
    }]
  })
}

const getEnvColor = (env: string) => {
  const map: Record<string, string> = { dev: 'green', test: 'blue', staging: 'orange', prod: 'red' }
  return map[env] || 'default'
}

const getEnvLabel = (env: string) => {
  const map: Record<string, string> = { dev: '开发', test: '测试', staging: '预发', prod: '生产' }
  return map[env] || env
}

const getStatusBadge = (s: string) => {
  const map: Record<string, 'processing' | 'success' | 'warning' | 'error' | 'default'> = {
    canary_running: 'processing',
    success: 'success',
    rolled_back: 'warning',
    failed: 'error'
  }
  return map[s] || 'default'
}

const getStatusText = (s: string) => {
  const map: Record<string, string> = {
    canary_running: '进行中',
    success: '已完成',
    rolled_back: '已回滚',
    failed: '失败'
  }
  return map[s] || s
}

const getErrorRateColor = (rate: string) => {
  if (!rate) return '#52c41a'
  const num = parseFloat(rate)
  if (num > 5) return '#ff4d4f'
  if (num > 1) return '#faad14'
  return '#52c41a'
}

watch(() => props.record, () => {
  fetchStatus()
}, { immediate: true })

onMounted(() => {
  initChart()
  // 自动刷新
  if (props.record.status === 'canary_running') {
    refreshTimer = window.setInterval(fetchStatus, 10000)
  }
})

onUnmounted(() => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
  }
  if (trafficChart) {
    trafficChart.dispose()
  }
})
</script>

<style scoped>
.canary-detail {
  padding: 8px 0;
}

.progress-section {
  padding: 8px;
}

.progress-title {
  font-weight: 500;
  margin-bottom: 8px;
}

.action-buttons {
  margin-top: 24px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
  text-align: center;
}
</style>
