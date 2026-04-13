<template>
  <div class="cost-comparison">
    <a-card title="成本对比分析" style="margin-bottom: 16px">
      <a-form layout="inline">
        <a-form-item label="集群">
          <a-select v-model:value="form.cluster_id" style="width: 150px" placeholder="全部集群" allowClear>
            <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="周期1">
          <a-range-picker v-model:value="period1" format="YYYY-MM-DD" />
        </a-form-item>
        <a-form-item label="周期2">
          <a-range-picker v-model:value="period2" format="YYYY-MM-DD" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="compare" :loading="loading">对比分析</a-button>
        </a-form-item>
        <a-form-item>
          <a-button @click="setQuickPeriod('week')">周环比</a-button>
          <a-button @click="setQuickPeriod('month')" style="margin-left: 8px">月环比</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <template v-if="result">
      <!-- 总体对比 -->
      <a-row :gutter="16" style="margin-bottom: 16px">
        <a-col :span="6">
          <a-card>
            <a-statistic title="周期1总成本" :value="result.period1?.data?.total_cost" :precision="2" prefix="¥" />
            <div style="color: #999; font-size: 12px; margin-top: 4px">{{ result.period1?.start }} ~ {{ result.period1?.end }}</div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card>
            <a-statistic title="周期2总成本" :value="result.period2?.data?.total_cost" :precision="2" prefix="¥" />
            <div style="color: #999; font-size: 12px; margin-top: 4px">{{ result.period2?.start }} ~ {{ result.period2?.end }}</div>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card>
            <a-statistic title="成本变化" :value="result.total_change" :precision="2" prefix="¥" :value-style="{ color: result.total_change >= 0 ? '#ff4d4f' : '#52c41a' }">
              <template #suffix>
                <span style="font-size: 14px">{{ result.total_change >= 0 ? '↑' : '↓' }}</span>
              </template>
            </a-statistic>
          </a-card>
        </a-col>
        <a-col :span="6">
          <a-card>
            <a-statistic title="变化率" :value="Math.abs(result.total_change_rate)" :precision="1" suffix="%" :value-style="{ color: result.total_change_rate >= 0 ? '#ff4d4f' : '#52c41a' }">
              <template #prefix>
                <span>{{ result.total_change_rate >= 0 ? '+' : '-' }}</span>
              </template>
            </a-statistic>
          </a-card>
        </a-col>
      </a-row>

      <!-- 分项对比 -->
      <a-card title="分项成本对比" style="margin-bottom: 16px">
        <a-table :columns="itemColumns" :data-source="itemData" :pagination="false" size="small">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'change'">
              <span :style="{ color: record.change >= 0 ? '#ff4d4f' : '#52c41a' }">
                {{ record.change >= 0 ? '+' : '' }}¥{{ record.change?.toFixed(2) }}
              </span>
            </template>
            <template v-else-if="column.key === 'change_rate'">
              <span :style="{ color: record.change_rate >= 0 ? '#ff4d4f' : '#52c41a' }">
                {{ record.change_rate >= 0 ? '+' : '' }}{{ record.change_rate?.toFixed(1) }}%
              </span>
            </template>
          </template>
        </a-table>
      </a-card>

      <!-- 命名空间对比 -->
      <a-card title="命名空间成本对比">
        <div ref="chartRef" style="height: 400px"></div>
        <a-table :columns="nsColumns" :data-source="result.namespace_comparison" :pagination="{ pageSize: 10 }" size="small" style="margin-top: 16px">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'change'">
              <span :style="{ color: record.change >= 0 ? '#ff4d4f' : '#52c41a' }">
                {{ record.change >= 0 ? '+' : '' }}¥{{ record.change?.toFixed(2) }}
              </span>
            </template>
            <template v-else-if="column.key === 'change_rate'">
              <span :style="{ color: record.change_rate >= 0 ? '#ff4d4f' : '#52c41a' }">
                {{ record.change_rate >= 0 ? '+' : '' }}{{ record.change_rate?.toFixed(1) }}%
              </span>
            </template>
          </template>
        </a-table>
      </a-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import dayjs, { Dayjs } from 'dayjs'
import * as echarts from 'echarts'
import { costApi } from '@/services/cost'
import { k8sClusterApi } from '@/services/k8s'

const clusters = ref<any[]>([])
const loading = ref(false)
const result = ref<any>(null)
const chartRef = ref<HTMLElement>()
let chart: echarts.ECharts | null = null

const form = ref({
  cluster_id: undefined as number | undefined
})

const period1 = ref<[Dayjs, Dayjs]>([dayjs().subtract(14, 'day'), dayjs().subtract(7, 'day')])
const period2 = ref<[Dayjs, Dayjs]>([dayjs().subtract(7, 'day'), dayjs()])

const itemColumns = [
  { title: '项目', dataIndex: 'name', key: 'name' },
  { title: '周期1', dataIndex: 'period1', key: 'period1' },
  { title: '周期2', dataIndex: 'period2', key: 'period2' },
  { title: '变化', dataIndex: 'change', key: 'change' },
  { title: '变化率', dataIndex: 'change_rate', key: 'change_rate' }
]

const nsColumns = [
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace' },
  { title: '周期1成本', dataIndex: 'period1_cost', key: 'period1_cost' },
  { title: '周期2成本', dataIndex: 'period2_cost', key: 'period2_cost' },
  { title: '变化', dataIndex: 'change', key: 'change' },
  { title: '变化率', dataIndex: 'change_rate', key: 'change_rate' }
]

const itemData = computed(() => {
  if (!result.value) return []
  const p1 = result.value.period1?.data || {}
  const p2 = result.value.period2?.data || {}
  return [
    { name: '总成本', period1: `¥${p1.total_cost?.toFixed(2) || 0}`, period2: `¥${p2.total_cost?.toFixed(2) || 0}`, change: result.value.total_change, change_rate: result.value.total_change_rate },
    { name: 'CPU成本', period1: `¥${p1.cpu_cost?.toFixed(2) || 0}`, period2: `¥${p2.cpu_cost?.toFixed(2) || 0}`, change: result.value.cpu_change, change_rate: result.value.cpu_change_rate },
    { name: '内存成本', period1: `¥${p1.memory_cost?.toFixed(2) || 0}`, period2: `¥${p2.memory_cost?.toFixed(2) || 0}`, change: result.value.memory_change, change_rate: result.value.memory_change_rate },
    { name: '存储成本', period1: `¥${p1.storage_cost?.toFixed(2) || 0}`, period2: `¥${p2.storage_cost?.toFixed(2) || 0}`, change: result.value.storage_change, change_rate: result.value.storage_change_rate }
  ]
})

const setQuickPeriod = (type: string) => {
  if (type === 'week') {
    period1.value = [dayjs().subtract(14, 'day'), dayjs().subtract(7, 'day')]
    period2.value = [dayjs().subtract(7, 'day'), dayjs()]
  } else if (type === 'month') {
    period1.value = [dayjs().subtract(60, 'day'), dayjs().subtract(30, 'day')]
    period2.value = [dayjs().subtract(30, 'day'), dayjs()]
  }
}

const compare = async () => {
  if (!period1.value || !period2.value) {
    message.warning('请选择对比周期')
    return
  }
  loading.value = true
  try {
    const res: any = await costApi.getComparison({
      cluster_id: form.value.cluster_id,
      period1_start: period1.value[0].format('YYYY-MM-DD'),
      period1_end: period1.value[1].format('YYYY-MM-DD'),
      period2_start: period2.value[0].format('YYYY-MM-DD'),
      period2_end: period2.value[1].format('YYYY-MM-DD')
    })
    if (res?.code === 0) {
      result.value = res.data
      await nextTick()
      renderChart()
    }
  } finally {
    loading.value = false
  }
}

const renderChart = () => {
  if (!chartRef.value || !result.value?.namespace_comparison) return
  if (!chart) {
    chart = echarts.init(chartRef.value)
  }
  const data = result.value.namespace_comparison.slice(0, 10)
  chart.setOption({
    tooltip: { trigger: 'axis', axisPointer: { type: 'shadow' } },
    legend: { data: ['周期1', '周期2'] },
    grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
    xAxis: { type: 'category', data: data.map((d: any) => d.namespace) },
    yAxis: { type: 'value', name: '成本 (¥)' },
    series: [
      { name: '周期1', type: 'bar', data: data.map((d: any) => d.period1_cost?.toFixed(2)) },
      { name: '周期2', type: 'bar', data: data.map((d: any) => d.period2_cost?.toFixed(2)) }
    ]
  })
}

const fetchClusters = async () => {
  const res: any = await k8sClusterApi.list()
  if (res?.code === 0) {
    clusters.value = res.data?.items || []
  }
}

onMounted(async () => {
  await fetchClusters()
})
</script>
