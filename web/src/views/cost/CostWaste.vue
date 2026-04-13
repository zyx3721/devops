<template>
  <div>
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-select v-model:value="selectedCluster" style="width: 100%" placeholder="全部集群" allowClear @change="fetchData">
          <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</a-select-option>
        </a-select>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-card>
          <a-statistic title="总浪费成本" :value="waste.total_waste" :precision="2" prefix="¥" :value-style="{ color: '#ff4d4f' }" />
          <div style="color: #999; margin-top: 8px">占总成本 {{ waste.waste_percent?.toFixed(1) || 0 }}%</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="闲置资源" :value="waste.summary?.idle_count || 0" suffix="个" />
          <div style="color: #999; margin-top: 8px">浪费 ¥{{ waste.summary?.idle_cost?.toFixed(2) || 0 }}</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="超配资源" :value="waste.summary?.overprovisioned_count || 0" suffix="个" />
          <div style="color: #999; margin-top: 8px">浪费 ¥{{ waste.summary?.overprovisioned_cost?.toFixed(2) || 0 }}</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="僵尸资源" :value="waste.summary?.zombie_count || 0" suffix="个" />
          <div style="color: #999; margin-top: 8px">长期无流量</div>
        </a-card>
      </a-col>
    </a-row>

    <a-card>
      <a-tabs v-model:activeKey="activeTab">
        <a-tab-pane key="idle" :tab="`闲置资源 (${waste.idle_resources?.length || 0})`">
          <a-table :columns="columns" :data-source="waste.idle_resources" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'resource'">
                <div>{{ record.resource_name }}</div>
                <div style="color: #999; font-size: 12px">{{ record.namespace }} / {{ record.resource_type }}</div>
              </template>
              <template v-if="column.key === 'waste_cost'"><span style="color: #ff4d4f; font-weight: bold">¥{{ record.waste_cost?.toFixed(2) }}</span></template>
              <template v-if="column.key === 'usage'"><a-progress :percent="record.current_usage" size="small" status="exception" style="width: 80px" /></template>
              <template v-if="column.key === 'impact'"><a-tag :color="getImpactColor(record.impact)">{{ record.impact }}</a-tag></template>
            </template>
          </a-table>
        </a-tab-pane>
        <a-tab-pane key="over" :tab="`超配资源 (${waste.overprovisioned?.length || 0})`">
          <a-table :columns="columns" :data-source="waste.overprovisioned" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'resource'">
                <div>{{ record.resource_name }}</div>
                <div style="color: #999; font-size: 12px">{{ record.namespace }} / {{ record.resource_type }}</div>
              </template>
              <template v-if="column.key === 'waste_cost'"><span style="color: #faad14; font-weight: bold">¥{{ record.waste_cost?.toFixed(2) }}</span></template>
              <template v-if="column.key === 'usage'"><a-progress :percent="record.current_usage" size="small" status="active" style="width: 80px" /></template>
              <template v-if="column.key === 'impact'"><a-tag :color="getImpactColor(record.impact)">{{ record.impact }}</a-tag></template>
            </template>
          </a-table>
        </a-tab-pane>
        <a-tab-pane key="zombie" :tab="`僵尸资源 (${waste.zombie_resources?.length || 0})`">
          <a-table :columns="zombieColumns" :data-source="waste.zombie_resources" size="small">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'resource'">
                <div>{{ record.resource_name }}</div>
                <div style="color: #999; font-size: 12px">{{ record.namespace }} / {{ record.resource_type }}</div>
              </template>
              <template v-if="column.key === 'idle_days'"><a-tag color="red">{{ record.idle_days }} 天</a-tag></template>
            </template>
          </a-table>
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { costApi } from '@/services/cost'
import { k8sClusterApi } from '@/services/k8s'

const selectedCluster = ref<number>()
const clusters = ref<any[]>([])
const waste = ref<any>({ idle_resources: [], overprovisioned: [], zombie_resources: [], summary: {} })
const activeTab = ref('idle')

const columns = [
  { title: '集群', dataIndex: 'cluster_name', width: 120 },
  { title: '资源', key: 'resource', width: 250 },
  { title: '浪费成本', key: 'waste_cost', width: 100 },
  { title: '利用率', key: 'usage', width: 120 },
  { title: '影响', key: 'impact', width: 80 },
  { title: '建议', dataIndex: 'suggestion' }
]

const zombieColumns = [
  { title: '集群', dataIndex: 'cluster_name', width: 120 },
  { title: '资源', key: 'resource', width: 250 },
  { title: '闲置天数', key: 'idle_days', width: 100 },
  { title: '建议', dataIndex: 'suggestion' }
]

const fetchData = async () => {
  const [w, c]: any[] = await Promise.all([costApi.getWasteDetection(selectedCluster.value, 7), k8sClusterApi.list()])
  if (w?.code === 0) waste.value = w.data || {}
  if (c?.code === 0) clusters.value = c.data?.items || []
}

const getImpactColor = (i: string) => ({ high: 'red', medium: 'orange', low: 'blue' }[i] || 'default')

onMounted(fetchData)
</script>
