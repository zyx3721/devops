<template>
  <div class="system-monitor">
    <!-- 概览卡片 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="QPS" :value="metrics.qps">
            <template #prefix><ThunderboltOutlined style="color: #1890ff" /></template>
            <template #suffix>req/s</template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="平均响应时间" :value="metrics.avgLatency" suffix="ms">
            <template #prefix><ClockCircleOutlined style="color: #52c41a" /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="错误率" :value="metrics.errorRate" suffix="%" :precision="2">
            <template #prefix><WarningOutlined :style="{ color: metrics.errorRate > 1 ? '#ff4d4f' : '#52c41a' }" /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="活跃分布式锁" :value="locks.length">
            <template #prefix><LockOutlined style="color: #722ed1" /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <a-tabs v-model:activeKey="activeTab">
      <!-- 概览 Tab -->
      <a-tab-pane key="overview" tab="概览">
        <a-row :gutter="16">
          <a-col :span="16">
            <a-card title="请求趋势" size="small">
              <div ref="chartRef" style="height: 300px"></div>
            </a-card>
          </a-col>
          <a-col :span="8">
            <a-card title="请求分布" size="small">
              <div ref="pieChartRef" style="height: 300px"></div>
            </a-card>
          </a-col>
        </a-row>
      </a-tab-pane>

      <!-- 链路追踪 Tab -->
      <a-tab-pane key="tracing" tab="链路追踪">
        <a-card :bordered="false">
          <a-form layout="inline" style="margin-bottom: 16px">
            <a-form-item>
              <a-input v-model:value="traceFilter.trace_id" placeholder="Trace ID" style="width: 200px" allowClear />
            </a-form-item>
            <a-form-item>
              <a-select v-model:value="traceFilter.service" placeholder="服务" style="width: 150px" allowClear>
                <a-select-option value="api-gateway">API Gateway</a-select-option>
                <a-select-option value="user-service">User Service</a-select-option>
                <a-select-option value="pipeline-service">Pipeline Service</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item>
              <a-range-picker v-model:value="traceFilter.time_range" show-time />
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="fetchTraces">查询</a-button>
            </a-form-item>
          </a-form>

          <a-table :columns="traceColumns" :data-source="traces" :loading="tracesLoading" row-key="trace_id" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <a-tag :color="record.status === 'success' ? 'green' : 'red'">{{ record.status }}</a-tag>
              </template>
              <template v-if="column.key === 'duration'">
                <span :style="{ color: record.duration > 1000 ? '#ff4d4f' : '#52c41a' }">{{ record.duration }}ms</span>
              </template>
              <template v-if="column.key === 'action'">
                <a-button type="link" size="small" @click="showTraceDetail(record)">详情</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 性能指标 Tab -->
      <a-tab-pane key="metrics" tab="性能指标">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-card title="API 响应时间 Top 10" size="small">
              <a-table :columns="apiLatencyColumns" :data-source="apiLatencyTop" :pagination="false" size="small">
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'latency'">
                    <a-progress :percent="Math.min(record.avg_latency / 10, 100)" :show-info="false" size="small" style="width: 80px" />
                    <span style="margin-left: 8px">{{ record.avg_latency }}ms</span>
                  </template>
                </template>
              </a-table>
            </a-card>
          </a-col>
          <a-col :span="12">
            <a-card title="错误率 Top 10" size="small">
              <a-table :columns="errorRateColumns" :data-source="errorRateTop" :pagination="false" size="small">
                <template #bodyCell="{ column, record }">
                  <template v-if="column.key === 'error_rate'">
                    <a-progress :percent="record.error_rate" :status="record.error_rate > 5 ? 'exception' : 'normal'" size="small" style="width: 80px" />
                    <span style="margin-left: 8px">{{ record.error_rate }}%</span>
                  </template>
                </template>
              </a-table>
            </a-card>
          </a-col>
        </a-row>
      </a-tab-pane>

      <!-- 分布式锁 Tab -->
      <a-tab-pane key="locks" tab="分布式锁">
        <a-card :bordered="false">
          <a-alert message="分布式锁用于保证分布式环境下的资源互斥访问" type="info" show-icon style="margin-bottom: 16px" />
          
          <a-table :columns="lockColumns" :data-source="locks" :loading="locksLoading" row-key="key" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'status'">
                <a-badge status="processing" text="锁定中" />
              </template>
              <template v-if="column.key === 'ttl'">
                <a-countdown :value="record.expires_at" format="mm:ss" />
              </template>
              <template v-if="column.key === 'action'">
                <a-popconfirm title="强制释放锁可能导致数据不一致，确定继续？" @confirm="releaseLock(record.key)">
                  <a-button type="link" size="small" danger>强制释放</a-button>
                </a-popconfirm>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <!-- 链路详情弹窗 -->
    <a-modal v-model:open="traceDetailVisible" title="链路详情" width="800px" :footer="null">
      <template v-if="currentTrace">
        <a-descriptions :column="2" bordered size="small" style="margin-bottom: 16px">
          <a-descriptions-item label="Trace ID">{{ currentTrace.trace_id }}</a-descriptions-item>
          <a-descriptions-item label="总耗时">{{ currentTrace.duration }}ms</a-descriptions-item>
          <a-descriptions-item label="服务">{{ currentTrace.service }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="currentTrace.status === 'success' ? 'green' : 'red'">{{ currentTrace.status }}</a-tag>
          </a-descriptions-item>
        </a-descriptions>

        <a-timeline>
          <a-timeline-item v-for="(span, index) in currentTrace.spans" :key="index" :color="span.status === 'error' ? 'red' : 'green'">
            <p><strong>{{ span.operation }}</strong> <span style="color: #999">{{ span.duration }}ms</span></p>
            <p style="color: #666; font-size: 12px">{{ span.service }} | {{ span.start_time }}</p>
          </a-timeline-item>
        </a-timeline>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, defineExpose } from 'vue'
import { message } from 'ant-design-vue'
import { ThunderboltOutlined, ClockCircleOutlined, WarningOutlined, LockOutlined } from '@ant-design/icons-vue'
import type { ECharts } from 'echarts'
import * as echarts from 'echarts'

// 显式声明组件使用的图标
void ThunderboltOutlined
void ClockCircleOutlined
void WarningOutlined
void LockOutlined

const activeTab = ref('overview')
const tracesLoading = ref(false)
const locksLoading = ref(false)
const traceDetailVisible = ref(false)
const chartRef = ref<HTMLElement | null>(null)
const pieChartRef = ref<HTMLElement | null>(null)
let chart: ECharts | null = null
let pieChart: ECharts | null = null

const metrics = reactive({ qps: 1234, avgLatency: 45, errorRate: 0.12 })
const traces = ref<any[]>([])
const locks = ref<any[]>([])
const currentTrace = ref<any>(null)
const traceFilter = reactive({ trace_id: '', service: '', time_range: null as any })

const apiLatencyTop = ref([
  { endpoint: '/api/v1/pipeline/run', avg_latency: 856, count: 1200 },
  { endpoint: '/api/v1/k8s/pods', avg_latency: 423, count: 3400 },
  { endpoint: '/api/v1/deploy/check', avg_latency: 312, count: 890 },
  { endpoint: '/api/v1/logs/search', avg_latency: 245, count: 2100 },
  { endpoint: '/api/v1/users', avg_latency: 89, count: 5600 },
])

const errorRateTop = ref([
  { endpoint: '/api/v1/webhook/github', error_rate: 8.5, errors: 42 },
  { endpoint: '/api/v1/notify/feishu', error_rate: 3.2, errors: 18 },
  { endpoint: '/api/v1/jenkins/build', error_rate: 2.1, errors: 12 },
  { endpoint: '/api/v1/k8s/exec', error_rate: 1.5, errors: 8 },
  { endpoint: '/api/v1/deploy/rollback', error_rate: 0.8, errors: 3 },
])

const traceColumns = [
  { title: 'Trace ID', dataIndex: 'trace_id', key: 'trace_id', width: 200 },
  { title: '服务', dataIndex: 'service', key: 'service' },
  { title: '操作', dataIndex: 'operation', key: 'operation' },
  { title: '状态', key: 'status', width: 80 },
  { title: '耗时', key: 'duration', width: 100 },
  { title: '时间', dataIndex: 'start_time', key: 'start_time', width: 180 },
  { title: '操作', key: 'action', width: 80 }
]

const apiLatencyColumns = [
  { title: 'API', dataIndex: 'endpoint', key: 'endpoint', ellipsis: true },
  { title: '平均耗时', key: 'latency', width: 180 },
  { title: '请求数', dataIndex: 'count', key: 'count', width: 80 }
]

const errorRateColumns = [
  { title: 'API', dataIndex: 'endpoint', key: 'endpoint', ellipsis: true },
  { title: '错误率', key: 'error_rate', width: 180 },
  { title: '错误数', dataIndex: 'errors', key: 'errors', width: 80 }
]

const lockColumns = [
  { title: '锁 Key', dataIndex: 'key', key: 'key' },
  { title: '持有者', dataIndex: 'holder', key: 'holder' },
  { title: '状态', key: 'status', width: 100 },
  { title: '剩余时间', key: 'ttl', width: 120 },
  { title: '获取时间', dataIndex: 'acquired_at', key: 'acquired_at', width: 180 },
  { title: '操作', key: 'action', width: 100 }
]

const initCharts = () => {
  if (chartRef.value) {
    chart = echarts.init(chartRef.value)
    chart.setOption({
      tooltip: { trigger: 'axis' },
      legend: { data: ['QPS', '错误数'] },
      xAxis: { type: 'category', data: ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '24:00'] },
      yAxis: [{ type: 'value', name: 'QPS' }, { type: 'value', name: '错误数' }],
      series: [
        { name: 'QPS', type: 'line', smooth: true, data: [820, 932, 901, 1234, 1290, 1330, 1120], areaStyle: { opacity: 0.3 } },
        { name: '错误数', type: 'bar', yAxisIndex: 1, data: [5, 8, 12, 15, 10, 8, 6] }
      ]
    })
  }

  if (pieChartRef.value) {
    pieChart = echarts.init(pieChartRef.value)
    pieChart.setOption({
      tooltip: { trigger: 'item' },
      legend: { orient: 'vertical', left: 'left' },
      series: [{
        type: 'pie', radius: '60%',
        data: [
          { value: 4500, name: 'API 请求' },
          { value: 2800, name: 'K8s 操作' },
          { value: 1200, name: '流水线' },
          { value: 800, name: '部署' },
          { value: 500, name: '其他' }
        ]
      }]
    })
  }
}

const fetchTraces = () => {
  tracesLoading.value = true
  setTimeout(() => {
    traces.value = [
      { trace_id: 'abc123def456', service: 'api-gateway', operation: 'POST /api/v1/pipeline/run', status: 'success', duration: 856, start_time: '2026-01-12 10:30:15' },
      { trace_id: 'xyz789ghi012', service: 'user-service', operation: 'GET /api/v1/users/1', status: 'success', duration: 45, start_time: '2026-01-12 10:30:10' },
      { trace_id: 'mno345pqr678', service: 'pipeline-service', operation: 'POST /api/v1/deploy', status: 'error', duration: 1234, start_time: '2026-01-12 10:29:55' },
    ]
    tracesLoading.value = false
  }, 500)
}

const fetchLocks = () => {
  locksLoading.value = true
  setTimeout(() => {
    locks.value = [
      { key: 'deploy:app-1:prod', holder: 'pipeline-worker-1', acquired_at: '2026-01-12 10:28:00', expires_at: Date.now() + 120000 },
      { key: 'pipeline:run:123', holder: 'pipeline-worker-2', acquired_at: '2026-01-12 10:29:30', expires_at: Date.now() + 60000 },
      { key: 'k8s:scale:deployment-x', holder: 'k8s-controller', acquired_at: '2026-01-12 10:30:00', expires_at: Date.now() + 30000 },
    ]
    locksLoading.value = false
  }, 500)
}

const showTraceDetail = (trace: any) => {
  currentTrace.value = {
    ...trace,
    spans: [
      { operation: 'HTTP Request', service: 'api-gateway', duration: 5, status: 'success', start_time: '10:30:15.000' },
      { operation: 'Auth Check', service: 'auth-service', duration: 12, status: 'success', start_time: '10:30:15.005' },
      { operation: 'DB Query', service: 'pipeline-service', duration: 45, status: 'success', start_time: '10:30:15.017' },
      { operation: 'K8s API Call', service: 'k8s-service', duration: 780, status: 'success', start_time: '10:30:15.062' },
      { operation: 'Response', service: 'api-gateway', duration: 14, status: 'success', start_time: '10:30:15.842' },
    ]
  }
  traceDetailVisible.value = true
}

const releaseLock = (key: string) => {
  message.success(`锁 ${key} 已释放`)
  locks.value = locks.value.filter(l => l.key !== key)
}

onMounted(() => {
  initCharts()
  fetchTraces()
  fetchLocks()
})

onUnmounted(() => {
  chart?.dispose()
  pieChart?.dispose()
})

// 暴露给模板使用
defineExpose({
  activeTab, tracesLoading, locksLoading, traceDetailVisible,
  metrics, traces, locks, currentTrace, traceFilter,
  apiLatencyTop, errorRateTop, traceColumns, apiLatencyColumns,
  errorRateColumns, lockColumns, fetchTraces, showTraceDetail, releaseLock
})
</script>

<style scoped>
.system-monitor :deep(.ant-tabs-nav) { margin-bottom: 16px; }
</style>
