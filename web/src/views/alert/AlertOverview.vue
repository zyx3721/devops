<template>
  <div class="alert-overview">
    <!-- 统计卡片 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="4">
        <a-card size="small" hoverable @click="goTo('/alert/history', { ack_status: 'pending' })">
          <a-statistic title="待处理" :value="stats.pending_count" :value-style="{ color: '#cf1322' }">
            <template #prefix><ExclamationCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="4">
        <a-card size="small" hoverable @click="goTo('/alert/history')">
          <a-statistic title="今日告警" :value="stats.today_count" :value-style="{ color: '#fa8c16' }">
            <template #prefix><AlertOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="4">
        <a-card size="small" hoverable @click="goTo('/alert/config')">
          <a-statistic title="启用配置" :value="stats.enabled_count" :value-style="{ color: '#52c41a' }">
            <template #prefix><CheckCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="4">
        <a-card size="small" hoverable @click="goTo('/alert/silence')">
          <a-statistic title="活跃静默" :value="stats.active_silence_count" :value-style="{ color: '#1890ff' }">
            <template #prefix><StopOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="4">
        <a-card size="small" hoverable @click="goTo('/alert/escalation')">
          <a-statistic title="升级规则" :value="stats.enabled_escalation_count" :value-style="{ color: '#722ed1' }">
            <template #prefix><ArrowUpOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="4">
        <a-card size="small">
          <a-statistic title="本周告警" :value="weekCount" :value-style="{ color: '#13c2c2' }">
            <template #prefix><CalendarOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16">
      <!-- 告警趋势 -->
      <a-col :span="16">
        <a-card title="告警趋势（近7天）" :bordered="false" :loading="loadingTrend">
          <div ref="trendChartRef" style="height: 280px"></div>
        </a-card>
      </a-col>
      <!-- 告警分布 -->
      <a-col :span="8">
        <a-card title="告警类型分布" :bordered="false" :loading="loadingStats">
          <div ref="typeChartRef" style="height: 280px"></div>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top: 16px">
      <!-- 级别分布 -->
      <a-col :span="8">
        <a-card title="告警级别分布" :bordered="false" :loading="loadingStats">
          <div ref="levelChartRef" style="height: 240px"></div>
        </a-card>
      </a-col>
      <!-- 最近告警 -->
      <a-col :span="16">
        <a-card title="最近告警" :bordered="false" :loading="loadingRecent">
          <template #extra>
            <a-button type="link" @click="goTo('/alert/history')">查看全部</a-button>
          </template>
          <a-table :columns="recentColumns" :data-source="recentAlerts" row-key="id" size="small" :pagination="false">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'created_at'">{{ formatTime(record.created_at) }}</template>
              <template v-if="column.key === 'type'">
                <a-tag :color="getTypeColor(record.type)" size="small">{{ getTypeLabel(record.type) }}</a-tag>
              </template>
              <template v-if="column.key === 'level'">
                <a-tag :color="getLevelColor(record.level)" size="small">{{ getLevelLabel(record.level) }}</a-tag>
              </template>
              <template v-if="column.key === 'ack_status'">
                <a-badge :status="getAckStatusBadge(record.ack_status)" :text="getAckStatusLabel(record.ack_status)" />
              </template>
              <template v-if="column.key === 'action'">
                <a-button v-if="record.ack_status === 'pending'" type="link" size="small" @click="ackAlert(record.id)">确认</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { AlertOutlined, CheckCircleOutlined, StopOutlined, ArrowUpOutlined, ExclamationCircleOutlined, CalendarOutlined } from '@ant-design/icons-vue'
import * as echarts from 'echarts'
import { alertApi, type AlertStats, type AlertHistory } from '@/services/alert'

const router = useRouter()
const loadingStats = ref(false)
const loadingTrend = ref(false)
const loadingRecent = ref(false)

const stats = ref<AlertStats>({ type_stats: [], level_stats: [], ack_stats: [], today_count: 0, pending_count: 0, enabled_count: 0, active_silence_count: 0, enabled_escalation_count: 0 })
const recentAlerts = ref<AlertHistory[]>([])
const weekCount = ref(0)
const trendData = ref<{ date: string; count: number }[]>([])

const trendChartRef = ref<HTMLElement>()
const typeChartRef = ref<HTMLElement>()
const levelChartRef = ref<HTMLElement>()
let trendChart: echarts.ECharts | null = null
let typeChart: echarts.ECharts | null = null
let levelChart: echarts.ECharts | null = null

const recentColumns = [
  { title: '时间', key: 'created_at', width: 140 },
  { title: '类型', key: 'type', width: 100 },
  { title: '级别', key: 'level', width: 70 },
  { title: '标题', dataIndex: 'title', ellipsis: true },
  { title: '状态', key: 'ack_status', width: 80 },
  { title: '操作', key: 'action', width: 70 }
]

const typeLabels: Record<string, string> = { jenkins_build: 'Jenkins构建', k8s_pod: 'K8s Pod', health_check: '健康检查' }
const typeColors: Record<string, string> = { jenkins_build: '#f5222d', k8s_pod: '#722ed1', health_check: '#1890ff' }
const levelLabels: Record<string, string> = { info: '信息', warning: '警告', error: '错误', critical: '严重' }
const levelColors: Record<string, string> = { info: '#1890ff', warning: '#faad14', error: '#f5222d', critical: '#eb2f96' }
const ackStatusLabels: Record<string, string> = { pending: '待处理', acked: '已确认', resolved: '已解决' }

const getTypeLabel = (type: string) => typeLabels[type] || type
const getTypeColor = (type: string) => typeColors[type] || 'default'
const getLevelLabel = (level: string) => levelLabels[level] || level
const getLevelColor = (level: string) => levelColors[level] || 'default'
const getAckStatusLabel = (status: string) => ackStatusLabels[status] || status
const getAckStatusBadge = (status: string) => status === 'resolved' ? 'success' : status === 'acked' ? 'processing' : 'warning'
const formatTime = (time: string) => time ? time.replace('T', ' ').substring(0, 16) : '-'

const goTo = (path: string, query?: Record<string, string>) => {
  router.push({ path, query })
}

const fetchStats = async () => {
  loadingStats.value = true
  try {
    const res = await alertApi.getStats()
    if (res.code === 0 && res.data) {
      stats.value = res.data
      await nextTick()
      renderTypeChart()
      renderLevelChart()
    }
  } finally { loadingStats.value = false }
}

const fetchTrend = async () => {
  loadingTrend.value = true
  try {
    // 从历史记录统计近7天数据
    const res = await alertApi.getTrend({ days: 7 })
    if (res.code === 0 && res.data) {
      trendData.value = res.data.items || []
      weekCount.value = res.data.total || 0
    } else {
      // 如果接口不存在，从历史记录计算
      const histRes = await alertApi.listHistories({ page: 1, page_size: 1000 })
      if (histRes.code === 0 && histRes.data) {
        const now = new Date()
        const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000)
        const items = histRes.data.list || []
        const weekItems = items.filter(item => new Date(item.created_at) >= weekAgo)
        weekCount.value = weekItems.length
        
        // 按日期分组
        const dateMap = new Map<string, number>()
        for (let i = 6; i >= 0; i--) {
          const d = new Date(now.getTime() - i * 24 * 60 * 60 * 1000)
          dateMap.set(d.toISOString().substring(5, 10), 0)
        }
        weekItems.forEach(item => {
          const date = item.created_at.substring(5, 10)
          if (dateMap.has(date)) dateMap.set(date, (dateMap.get(date) || 0) + 1)
        })
        trendData.value = Array.from(dateMap.entries()).map(([date, count]) => ({ date, count }))
      }
    }
    await nextTick()
    renderTrendChart()
  } finally { loadingTrend.value = false }
}

const fetchRecent = async () => {
  loadingRecent.value = true
  try {
    const res = await alertApi.listHistories({ page: 1, page_size: 8 })
    if (res.code === 0 && res.data) recentAlerts.value = res.data.list || []
  } finally { loadingRecent.value = false }
}

const ackAlert = async (id: number) => {
  try {
    const res = await alertApi.ackHistory(id)
    if (res.code === 0) { message.success('已确认'); fetchRecent(); fetchStats() }
  } catch (e: any) { message.error(e.message || '操作失败') }
}

const renderTrendChart = () => {
  if (!trendChartRef.value) return
  if (!trendChart) trendChart = echarts.init(trendChartRef.value)
  trendChart.setOption({
    tooltip: { trigger: 'axis' },
    grid: { left: 40, right: 20, top: 20, bottom: 30 },
    xAxis: { type: 'category', data: trendData.value.map(d => d.date) },
    yAxis: { type: 'value', minInterval: 1 },
    series: [{
      type: 'line',
      data: trendData.value.map(d => d.count),
      smooth: true,
      areaStyle: { opacity: 0.3 },
      itemStyle: { color: '#f5222d' }
    }]
  })
}

const renderTypeChart = () => {
  if (!typeChartRef.value || !stats.value.type_stats?.length) return
  if (!typeChart) typeChart = echarts.init(typeChartRef.value)
  typeChart.setOption({
    tooltip: { trigger: 'item' },
    series: [{
      type: 'pie',
      radius: ['45%', '70%'],
      label: { show: true, formatter: '{b}: {c}' },
      data: stats.value.type_stats.map(s => ({
        name: typeLabels[s.name] || s.name,
        value: s.count,
        itemStyle: { color: typeColors[s.name] || '#999' }
      }))
    }]
  })
}

const renderLevelChart = () => {
  if (!levelChartRef.value || !stats.value.level_stats?.length) return
  if (!levelChart) levelChart = echarts.init(levelChartRef.value)
  levelChart.setOption({
    tooltip: { trigger: 'item' },
    series: [{
      type: 'pie',
      radius: ['45%', '70%'],
      label: { show: true, formatter: '{b}: {c}' },
      data: stats.value.level_stats.map(s => ({
        name: levelLabels[s.name] || s.name,
        value: s.count,
        itemStyle: { color: levelColors[s.name] || '#999' }
      }))
    }]
  })
}

onMounted(() => {
  fetchStats()
  fetchTrend()
  fetchRecent()
})
</script>

<style scoped>
.alert-overview :deep(.ant-card-hoverable) {
  cursor: pointer;
}
</style>
