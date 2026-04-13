<template>
  <div class="cost-analysis">
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-select v-model:value="selectedCluster" style="width: 100%" placeholder="全部集群" allowClear @change="fetchData">
          <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</a-select-option>
        </a-select>
      </a-col>
      <a-col :span="8">
        <a-range-picker v-model:value="dateRange" format="YYYY-MM-DD" @change="fetchData" />
      </a-col>
    </a-row>

    <a-tabs v-model:activeKey="activeTab" @change="fetchData">
      <!-- 应用维度 -->
      <a-tab-pane key="app" tab="应用成本">
        <a-row :gutter="16" style="margin-bottom: 16px">
          <a-col :span="6"><a-statistic title="应用总数" :value="appData.total_apps" /></a-col>
          <a-col :span="6"><a-statistic title="总成本" :value="appData.total_cost" :precision="2" prefix="¥" /></a-col>
          <a-col :span="6"><a-statistic title="平均成本" :value="appData.avg_cost" :precision="2" prefix="¥" /></a-col>
          <a-col :span="6">
            <div style="color: #999">Top 应用</div>
            <a-tag v-for="app in appData.top_cost_apps" :key="app" color="blue">{{ app }}</a-tag>
          </a-col>
        </a-row>
        <a-table :columns="appColumns" :data-source="appData.items" :loading="loading" size="small">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'app'">
              <div>{{ record.app_name }}</div>
              <div style="color: #999; font-size: 12px">{{ record.namespace }}</div>
            </template>
            <template v-else-if="column.key === 'cpu'">
              <div>{{ record.cpu_usage?.toFixed(2) }} / {{ record.cpu_request?.toFixed(2) }} 核</div>
              <a-progress :percent="record.cpu_usage_rate" size="small" :status="getUsageStatus(record.cpu_usage_rate)" style="width: 80px" />
            </template>
            <template v-else-if="column.key === 'memory'">
              <div>{{ record.memory_usage?.toFixed(2) }} / {{ record.memory_request?.toFixed(2) }} GB</div>
              <a-progress :percent="record.memory_usage_rate" size="small" :status="getUsageStatus(record.memory_usage_rate)" style="width: 80px" />
            </template>
            <template v-else-if="column.key === 'cost'">
              <div style="font-weight: bold">¥{{ record.total_cost?.toFixed(2) }}</div>
              <div style="color: #999; font-size: 12px">{{ record.percentage?.toFixed(1) }}%</div>
            </template>
            <template v-else-if="column.key === 'efficiency'">
              <a-tag :color="getEfficiencyColor(record.efficiency)">{{ record.efficiency?.toFixed(0) }}%</a-tag>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <!-- 团队维度 -->
      <a-tab-pane key="team" tab="团队成本">
        <a-row :gutter="16" style="margin-bottom: 16px">
          <a-col :span="6"><a-statistic title="团队总数" :value="teamData.total_teams" /></a-col>
          <a-col :span="6"><a-statistic title="总成本" :value="teamData.total_cost" :precision="2" prefix="¥" /></a-col>
          <a-col :span="6"><a-statistic title="公共成本" :value="teamData.shared_cost" :precision="2" prefix="¥" :value-style="{ color: '#faad14' }" /></a-col>
        </a-row>
        <a-table :columns="teamColumns" :data-source="teamData.items" :loading="loading" size="small">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'cost'">
              <div style="font-weight: bold">¥{{ record.total_cost?.toFixed(2) }}</div>
              <div style="color: #999; font-size: 12px">{{ record.percentage?.toFixed(1) }}%</div>
            </template>
            <template v-else-if="column.key === 'budget'">
              <template v-if="record.monthly_budget > 0">
                <a-progress :percent="record.budget_used" size="small" :status="record.budget_used > 100 ? 'exception' : record.budget_used > 80 ? 'active' : 'success'" style="width: 80px" />
              </template>
              <span v-else style="color: #999">未设置</span>
            </template>
            <template v-else-if="column.key === 'efficiency'">
              <a-tag :color="getEfficiencyColor(record.avg_efficiency)">{{ record.avg_efficiency?.toFixed(0) }}%</a-tag>
            </template>
            <template v-else-if="column.key === 'wasted'">
              <span style="color: #ff4d4f">¥{{ record.wasted_cost?.toFixed(2) }}</span>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <!-- 环境维度 -->
      <a-tab-pane key="env" tab="环境成本">
        <a-row :gutter="16" style="margin-bottom: 16px">
          <a-col :span="12">
            <div ref="envChartRef" style="height: 300px"></div>
          </a-col>
          <a-col :span="12">
            <a-table :columns="envColumns" :data-source="envData.items" :loading="loading" size="small" :pagination="false">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'env'">
                  <a-tag :color="getEnvColor(record.environment)">{{ getEnvName(record.environment) }}</a-tag>
                </template>
                <template v-else-if="column.key === 'cost'">
                  <div style="font-weight: bold">¥{{ record.total_cost?.toFixed(2) }}</div>
                  <div style="color: #999; font-size: 12px">{{ record.percentage?.toFixed(1) }}%</div>
                </template>
              </template>
            </a-table>
          </a-col>
        </a-row>
      </a-tab-pane>

      <!-- 节点成本 -->
      <a-tab-pane key="node" tab="节点成本">
        <a-row :gutter="16" style="margin-bottom: 16px">
          <a-col :span="4"><a-statistic title="节点总数" :value="nodeData.total_nodes" /></a-col>
          <a-col :span="4"><a-statistic title="总 CPU" :value="nodeData.total_cpu" :precision="0" suffix="核" /></a-col>
          <a-col :span="4"><a-statistic title="总内存" :value="nodeData.total_memory" :precision="0" suffix="GB" /></a-col>
          <a-col :span="4"><a-statistic title="平均 CPU 利用率" :value="nodeData.avg_cpu_usage" :precision="1" suffix="%" /></a-col>
          <a-col :span="4"><a-statistic title="平均内存利用率" :value="nodeData.avg_memory_usage" :precision="1" suffix="%" /></a-col>
          <a-col :span="4"><a-statistic title="低利用率节点" :value="nodeData.underutil_nodes" :value-style="{ color: nodeData.underutil_nodes > 0 ? '#faad14' : '#52c41a' }" /></a-col>
        </a-row>
        <a-table :columns="nodeColumns" :data-source="nodeData.items" :loading="loading" size="small" :scroll="{ x: 1400 }">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'node'">
              <div>{{ record.node_name }}</div>
              <div style="color: #999; font-size: 12px">{{ record.node_ip }}</div>
            </template>
            <template v-else-if="column.key === 'type'">
              <a-tag :color="record.node_type === 'master' ? 'purple' : 'blue'">{{ record.node_type }}</a-tag>
              <div style="color: #999; font-size: 12px">{{ record.instance_type }}</div>
            </template>
            <template v-else-if="column.key === 'cpu'">
              <div>{{ record.cpu_usage?.toFixed(2) }} / {{ record.cpu_allocatable?.toFixed(0) }} 核</div>
              <a-progress :percent="Math.min(record.cpu_usage_rate, 100)" size="small" :status="getUsageStatus(record.cpu_usage_rate)" style="width: 80px" />
            </template>
            <template v-else-if="column.key === 'memory'">
              <div>{{ record.memory_usage?.toFixed(1) }} / {{ record.memory_allocatable?.toFixed(0) }} GB</div>
              <a-progress :percent="Math.min(record.memory_usage_rate, 100)" size="small" :status="getUsageStatus(record.memory_usage_rate)" style="width: 80px" />
            </template>
            <template v-else-if="column.key === 'pods'">
              <div>{{ record.pod_count }} / {{ record.pod_capacity }}</div>
            </template>
            <template v-else-if="column.key === 'cost'">
              <span style="font-weight: bold">¥{{ record.estimated_cost?.toFixed(2) }}/h</span>
            </template>
            <template v-else-if="column.key === 'efficiency'">
              <a-tag :color="getEfficiencyColor(record.efficiency)">{{ record.efficiency?.toFixed(0) }}%</a-tag>
            </template>
            <template v-else-if="column.key === 'status'">
              <a-tag :color="record.status === 'Ready' ? 'green' : 'red'">{{ record.status }}</a-tag>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <!-- 成本分摊 -->
      <a-tab-pane key="allocation" tab="成本分摊">
        <a-row :gutter="16" style="margin-bottom: 16px">
          <a-col :span="4">
            <a-select v-model:value="allocationGroupBy" style="width: 100%" @change="fetchAllocation">
              <a-select-option value="team">按团队</a-select-option>
              <a-select-option value="namespace">按命名空间</a-select-option>
              <a-select-option value="app">按应用</a-select-option>
            </a-select>
          </a-col>
          <a-col :span="4">
            <a-checkbox v-model:checked="includeShared" @change="fetchAllocation">分摊公共成本</a-checkbox>
          </a-col>
          <a-col :span="4">
            <a-button type="primary" @click="exportAllocation">导出报表</a-button>
          </a-col>
        </a-row>
        <a-row :gutter="16" style="margin-bottom: 16px">
          <a-col :span="6"><a-statistic title="总成本" :value="allocationData.total_cost" :precision="2" prefix="¥" /></a-col>
          <a-col :span="6"><a-statistic title="直接成本" :value="allocationData.direct_cost" :precision="2" prefix="¥" /></a-col>
          <a-col :span="6"><a-statistic title="公共成本" :value="allocationData.shared_cost" :precision="2" prefix="¥" /></a-col>
          <a-col :span="6"><a-statistic title="未分配" :value="allocationData.unallocated_cost" :precision="2" prefix="¥" :value-style="{ color: '#faad14' }" /></a-col>
        </a-row>
        <a-table :columns="allocationColumns" :data-source="allocationData.items" :loading="loading" size="small">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'direct'">¥{{ record.direct_cost?.toFixed(2) }}</template>
            <template v-else-if="column.key === 'shared'">¥{{ record.shared_cost?.toFixed(2) }}</template>
            <template v-else-if="column.key === 'total'">
              <span style="font-weight: bold">¥{{ record.total_cost?.toFixed(2) }}</span>
            </template>
            <template v-else-if="column.key === 'percentage'">{{ record.percentage?.toFixed(1) }}%</template>
          </template>
        </a-table>
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import dayjs, { Dayjs } from 'dayjs'
import * as echarts from 'echarts'
import { costApi } from '@/services/cost'
import { k8sClusterApi } from '@/services/k8s'

const selectedCluster = ref<number>()
const dateRange = ref<[Dayjs, Dayjs]>([dayjs().subtract(30, 'day'), dayjs()])
const clusters = ref<any[]>([])
const loading = ref(false)
const activeTab = ref('app')

const appData = ref<any>({ items: [], top_cost_apps: [] })
const teamData = ref<any>({ items: [] })
const envData = ref<any>({ items: [] })
const nodeData = ref<any>({ items: [], total_nodes: 0, total_cpu: 0, total_memory: 0, avg_cpu_usage: 0, avg_memory_usage: 0, underutil_nodes: 0 })
const allocationData = ref<any>({ items: [] })
const allocationGroupBy = ref('team')
const includeShared = ref(true)

const envChartRef = ref<HTMLElement>()
let envChart: echarts.ECharts | null = null

const appColumns = [
  { title: '应用', key: 'app', width: 200 },
  { title: '资源数', dataIndex: 'resource_count', width: 80 },
  { title: 'CPU', key: 'cpu', width: 180 },
  { title: '内存', key: 'memory', width: 180 },
  { title: '成本', key: 'cost', width: 120 },
  { title: '效率', key: 'efficiency', width: 80 }
]

const teamColumns = [
  { title: '团队', dataIndex: 'team_name', width: 150 },
  { title: '应用数', dataIndex: 'app_count', width: 80 },
  { title: '资源数', dataIndex: 'resource_count', width: 80 },
  { title: '成本', key: 'cost', width: 120 },
  { title: '预算使用', key: 'budget', width: 120 },
  { title: '效率', key: 'efficiency', width: 80 },
  { title: '浪费', key: 'wasted', width: 100 }
]

const envColumns = [
  { title: '环境', key: 'env', width: 100 },
  { title: '命名空间', dataIndex: 'namespace_count', width: 100 },
  { title: '应用数', dataIndex: 'app_count', width: 80 },
  { title: '成本', key: 'cost', width: 120 }
]

const nodeColumns = [
  { title: '节点', key: 'node', width: 180 },
  { title: '类型', key: 'type', width: 120 },
  { title: 'CPU', key: 'cpu', width: 160 },
  { title: '内存', key: 'memory', width: 160 },
  { title: 'Pod', key: 'pods', width: 100 },
  { title: '估算成本', key: 'cost', width: 100 },
  { title: '效率', key: 'efficiency', width: 80 },
  { title: '状态', key: 'status', width: 80 }
]

const allocationColumns = [
  { title: '名称', dataIndex: 'name', width: 150 },
  { title: '直接成本', key: 'direct', width: 120 },
  { title: '分摊成本', key: 'shared', width: 120 },
  { title: '总成本', key: 'total', width: 120 },
  { title: '占比', key: 'percentage', width: 80 },
  { title: '资源数', dataIndex: 'resource_count', width: 80 }
]

const fetchData = async () => {
  loading.value = true
  try {
    const params = {
      cluster_id: selectedCluster.value,
      start_time: dateRange.value?.[0]?.format('YYYY-MM-DD'),
      end_time: dateRange.value?.[1]?.format('YYYY-MM-DD')
    }

    if (activeTab.value === 'app') {
      const res: any = await (costApi as any).getAppCost(params)
      if (res?.code === 0) appData.value = res.data || { items: [], top_cost_apps: [] }
    } else if (activeTab.value === 'team') {
      const res: any = await (costApi as any).getTeamCost(params)
      if (res?.code === 0) teamData.value = res.data || { items: [] }
    } else if (activeTab.value === 'env') {
      const res: any = await (costApi as any).getEnvCost(params)
      if (res?.code === 0) {
        envData.value = res.data || { items: [] }
        await nextTick()
        renderEnvChart()
      }
    } else if (activeTab.value === 'node') {
      if (!selectedCluster.value) {
        message.warning('请先选择集群')
        loading.value = false
        return
      }
      const res: any = await (costApi as any).getNodeCost(selectedCluster.value)
      if (res?.code === 0) nodeData.value = res.data || { items: [] }
    } else if (activeTab.value === 'allocation') {
      await fetchAllocation()
    }
  } finally {
    loading.value = false
  }
}

const fetchAllocation = async () => {
  loading.value = true
  try {
    const res: any = await (costApi as any).getCostAllocation({
      cluster_id: selectedCluster.value,
      start_time: dateRange.value?.[0]?.format('YYYY-MM-DD') || '',
      end_time: dateRange.value?.[1]?.format('YYYY-MM-DD') || '',
      group_by: allocationGroupBy.value,
      include_shared: includeShared.value
    })
    if (res?.code === 0) allocationData.value = res.data || { items: [] }
  } finally {
    loading.value = false
  }
}

const fetchClusters = async () => {
  const res: any = await k8sClusterApi.list()
  if (res?.code === 0) clusters.value = res.data?.items || []
}

const renderEnvChart = () => {
  if (!envChartRef.value || !envData.value.items?.length) return
  if (!envChart) envChart = echarts.init(envChartRef.value)
  envChart.setOption({
    tooltip: { trigger: 'item', formatter: '{b}: ¥{c} ({d}%)' },
    legend: { orient: 'vertical', left: 'left' },
    series: [{
      type: 'pie',
      radius: ['40%', '70%'],
      data: envData.value.items.map((item: any) => ({
        name: getEnvName(item.environment),
        value: item.total_cost?.toFixed(2)
      }))
    }]
  })
}

const exportAllocation = () => {
  message.info('导出功能开发中')
}

const getUsageStatus = (rate: number) => rate < 30 ? 'exception' : rate < 60 ? 'active' : 'success'
const getEfficiencyColor = (e: number) => e < 30 ? 'red' : e < 60 ? 'orange' : 'green'
const getEnvColor = (e: string) => ({ dev: 'blue', test: 'cyan', staging: 'orange', prod: 'red', other: 'default' }[e] || 'default')
const getEnvName = (e: string) => ({ dev: '开发', test: '测试', staging: '预发', prod: '生产', other: '其他' }[e] || e)

onMounted(async () => {
  await fetchClusters()
  fetchData()
})
</script>
