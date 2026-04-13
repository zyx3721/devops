<template>
  <div class="cost-overview">
    <a-row :gutter="16">
      <a-col :span="6">
        <a-select v-model:value="selectedCluster" style="width: 100%" placeholder="全部集群" allowClear @change="fetchData">
          <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</a-select-option>
        </a-select>
      </a-col>
      <a-col :span="4">
        <a-select v-model:value="days" style="width: 100%" @change="fetchData">
          <a-select-option :value="7">最近7天</a-select-option>
          <a-select-option :value="30">最近30天</a-select-option>
          <a-select-option :value="90">最近90天</a-select-option>
        </a-select>
      </a-col>
      <a-col :span="14" style="text-align: right">
        <a-button type="primary" @click="exportReport" :loading="exporting">
          <template #icon><DownloadOutlined /></template>
          导出报表
        </a-button>
      </a-col>
    </a-row>

    <!-- 健康评分 -->
    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="6">
        <a-card>
          <div style="text-align: center">
            <a-progress type="circle" :percent="healthScore.overall_score" :width="120" :stroke-color="getGradeColor(healthScore.grade)">
              <template #format><span style="font-size: 32px; font-weight: bold">{{ healthScore.grade || '-' }}</span></template>
            </a-progress>
            <div style="margin-top: 12px; font-size: 16px">健康评分 {{ healthScore.overall_score || 0 }} 分</div>
          </div>
        </a-card>
      </a-col>
      <a-col :span="18">
        <a-card title="评分详情">
          <a-row :gutter="16">
            <a-col :span="4" v-for="dim in healthScore.dimensions" :key="dim.name">
              <a-statistic :title="dim.name" :value="dim.score" :suffix="`/${dim.max_score}`" />
              <a-progress :percent="dim.score / dim.max_score * 100" :show-info="false" size="small" :status="dim.status === 'good' ? 'success' : dim.status === 'warning' ? 'active' : 'exception'" />
            </a-col>
          </a-row>
          <div v-if="healthScore.recommendations?.length" style="margin-top: 16px">
            <span style="color: #999">改进建议：</span>
            <a-tag v-for="(r, i) in healthScore.recommendations" :key="i" color="orange" style="margin: 4px">{{ r }}</a-tag>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 成本统计 -->
    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="6">
        <a-card>
          <a-statistic title="总成本" :value="overview.total_cost" :precision="2" prefix="¥" :value-style="{ color: '#1890ff', fontSize: '28px' }">
            <template #suffix>
              <span :style="{ color: overview.cost_change_rate >= 0 ? '#ff4d4f' : '#52c41a', fontSize: '14px' }">
                {{ overview.cost_change_rate >= 0 ? '↑' : '↓' }}{{ Math.abs(overview.cost_change_rate || 0).toFixed(1) }}%
              </span>
            </template>
          </a-statistic>
          <div style="color: #999; margin-top: 8px">同比: {{ overview.yoy_cost_change_rate?.toFixed(1) || 0 }}%</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="资源浪费" :value="overview.wasted_cost" :precision="2" prefix="¥" :value-style="{ color: '#ff4d4f', fontSize: '28px' }" />
          <div style="color: #999; margin-top: 8px">占比 {{ overview.wasted_percentage?.toFixed(1) || 0 }}% | 闲置 {{ overview.idle_resources || 0 }} 个</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="预算使用" :value="overview.budget_percent" :precision="1" suffix="%" :value-style="{ color: getBudgetColor(overview.budget_percent), fontSize: '28px' }" />
          <div style="color: #999; margin-top: 8px">¥{{ (overview.budget_used || 0).toFixed(0) }} / ¥{{ (overview.budget_total || 0).toFixed(0) }}</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="潜在节省" :value="overview.potential_savings" :precision="2" prefix="¥" :value-style="{ color: '#52c41a', fontSize: '28px' }" />
          <div style="color: #999; margin-top: 8px">{{ overview.suggestion_count || 0 }} 条优化建议</div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 成本构成 -->
    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="12">
        <a-card title="成本构成">
          <a-row :gutter="16">
            <a-col :span="8"><a-statistic title="CPU成本" :value="overview.cpu_cost" :precision="2" prefix="¥" /></a-col>
            <a-col :span="8"><a-statistic title="内存成本" :value="overview.memory_cost" :precision="2" prefix="¥" /></a-col>
            <a-col :span="8"><a-statistic title="存储成本" :value="overview.storage_cost" :precision="2" prefix="¥" /></a-col>
          </a-row>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card title="资源利用率">
          <a-row :gutter="16">
            <a-col :span="12">
              <div style="margin-bottom: 8px">CPU 平均利用率</div>
              <a-progress :percent="overview.avg_cpu_usage" :status="getUsageStatus(overview.avg_cpu_usage)" />
            </a-col>
            <a-col :span="12">
              <div style="margin-bottom: 8px">内存 平均利用率</div>
              <a-progress :percent="overview.avg_memory_usage" :status="getUsageStatus(overview.avg_memory_usage)" />
            </a-col>
          </a-row>
        </a-card>
      </a-col>
    </a-row>

    <!-- 预测 -->
    <a-card title="成本预测" style="margin-top: 16px">
      <a-row :gutter="16">
        <a-col :span="6"><a-statistic title="本月已产生" :value="forecast.current_month_cost" :precision="2" prefix="¥" /></a-col>
        <a-col :span="6"><a-statistic title="预测月底" :value="forecast.predicted_month_cost" :precision="2" prefix="¥" :value-style="{ color: '#1890ff' }" /></a-col>
        <a-col :span="6"><a-statistic title="预测下月" :value="forecast.next_month_cost" :precision="2" prefix="¥" /></a-col>
        <a-col :span="6"><a-statistic title="预测置信度" :value="forecast.confidence" :precision="0" suffix="%" :value-style="{ color: '#52c41a' }" /></a-col>
      </a-row>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { DownloadOutlined } from '@ant-design/icons-vue'
import { costApi } from '@/services/cost'
import { k8sClusterApi } from '@/services/k8s'
import dayjs from 'dayjs'

const selectedCluster = ref<number>()
const days = ref(30)
const clusters = ref<any[]>([])
const overview = ref<any>({})
const healthScore = ref<any>({ dimensions: [] })
const forecast = ref<any>({})
const exporting = ref(false)

const fetchData = async () => {
  const [o, h, f]: any[] = await Promise.all([
    costApi.getOverview(selectedCluster.value, days.value),
    costApi.getHealthScore(selectedCluster.value),
    costApi.getForecast(selectedCluster.value, 30)
  ])
  if (o?.code === 0) overview.value = o.data || {}
  if (h?.code === 0) healthScore.value = h.data || { dimensions: [] }
  if (f?.code === 0) forecast.value = f.data || {}
}

const fetchClusters = async () => {
  const res: any = await k8sClusterApi.list()
  if (res?.code === 0) clusters.value = res.data?.items || []
}

const exportReport = async () => {
  exporting.value = true
  try {
    const endTime = dayjs().format('YYYY-MM-DD')
    const startTime = dayjs().subtract(days.value, 'day').format('YYYY-MM-DD')
    const res = await costApi.exportReport({
      cluster_id: selectedCluster.value,
      start_time: startTime,
      end_time: endTime
    })
    const blob = new Blob([res as any], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `cost_report_${startTime}_${endTime}.xlsx`
    link.click()
    window.URL.revokeObjectURL(url)
    message.success('导出成功')
  } catch (e) {
    message.error('导出失败')
  } finally {
    exporting.value = false
  }
}

const getGradeColor = (g: string) => ({ A: '#52c41a', B: '#73d13d', C: '#faad14', D: '#ff7a45', F: '#ff4d4f' }[g] || '#999')
const getBudgetColor = (p: number) => p > 100 ? '#ff4d4f' : p > 80 ? '#faad14' : '#52c41a'
const getUsageStatus = (u: number) => u < 30 ? 'exception' : u < 60 ? 'active' : 'success'

onMounted(async () => { await fetchClusters(); fetchData() })
</script>
