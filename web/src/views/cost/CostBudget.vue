<template>
  <div>
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-select v-model:value="selectedCluster" style="width: 100%" placeholder="全部集群" allowClear @change="fetchData">
          <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</a-select-option>
        </a-select>
      </a-col>
      <a-col :span="4">
        <a-button type="primary" @click="showModal = true">添加预算</a-button>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6"><a-statistic title="总预算" :value="budgets.total_budget" :precision="2" prefix="¥" /></a-col>
      <a-col :span="6"><a-statistic title="已使用" :value="budgets.total_used" :precision="2" prefix="¥" /></a-col>
      <a-col :span="6"><a-statistic title="超预算" :value="budgets.over_budget" suffix="个" :value-style="{ color: budgets.over_budget > 0 ? '#ff4d4f' : '#52c41a' }" /></a-col>
      <a-col :span="6"><a-statistic title="风险预警" :value="budgets.at_risk" suffix="个" :value-style="{ color: budgets.at_risk > 0 ? '#faad14' : '#52c41a' }" /></a-col>
    </a-row>

    <a-table :columns="columns" :data-source="budgets.items" size="small">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'namespace'">{{ record.namespace || '集群级别' }}</template>
        <template v-if="column.key === 'budget'">¥{{ record.monthly_budget?.toFixed(2) }}</template>
        <template v-if="column.key === 'used'">¥{{ record.current_cost?.toFixed(2) }}</template>
        <template v-if="column.key === 'usage'">
          <a-progress :percent="record.usage_percent" size="small" :status="getStatus(record.status)" style="width: 120px" />
        </template>
        <template v-if="column.key === 'status'">
          <a-tag :color="getStatusColor(record.status)">{{ getStatusText(record.status) }}</a-tag>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="showModal" title="添加预算" @ok="save" :confirm-loading="saving">
      <a-form :model="form" layout="vertical">
        <a-form-item label="集群" required>
          <a-select v-model:value="form.cluster_id" style="width: 100%">
            <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="命名空间（留空表示集群级别）">
          <a-input v-model:value="form.namespace" placeholder="可选" />
        </a-form-item>
        <a-form-item label="月度预算（元）" required>
          <a-input-number v-model:value="form.monthly_budget" :min="0" style="width: 100%" />
        </a-form-item>
        <a-form-item label="告警阈值（%）">
          <a-input-number v-model:value="form.alert_threshold" :min="0" :max="100" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { costApi } from '@/services/cost'
import { k8sClusterApi } from '@/services/k8s'

const selectedCluster = ref<number>()
const clusters = ref<any[]>([])
const budgets = ref<any>({ items: [] })
const showModal = ref(false)
const saving = ref(false)
const form = ref({ cluster_id: undefined as number | undefined, namespace: '', monthly_budget: 1000, alert_threshold: 80 })

const columns = [
  { title: '命名空间', key: 'namespace' },
  { title: '月度预算', key: 'budget' },
  { title: '当前成本', key: 'used' },
  { title: '使用率', key: 'usage', width: 180 },
  { title: '状态', key: 'status', width: 100 }
]

const fetchData = async () => {
  const [b, c]: any[] = await Promise.all([costApi.getBudgets(selectedCluster.value), k8sClusterApi.list()])
  if (b?.code === 0) budgets.value = b.data || { items: [] }
  if (c?.code === 0) clusters.value = c.data?.items || []
}

const save = async () => {
  if (!form.value.cluster_id) { message.warning('请选择集群'); return }
  saving.value = true
  const res: any = await costApi.saveBudget(form.value as any)
  if (res?.code === 0) { message.success('保存成功'); showModal.value = false; fetchData() }
  saving.value = false
}

const getStatus = (s: string) => s === 'exceeded' ? 'exception' : s === 'warning' ? 'active' : 'success'
const getStatusColor = (s: string) => ({ normal: 'green', warning: 'orange', exceeded: 'red' }[s] || 'default')
const getStatusText = (s: string) => ({ normal: '正常', warning: '预警', exceeded: '超支' }[s] || s)

onMounted(fetchData)
</script>
