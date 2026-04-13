<template>
  <div>
    <a-card title="成本单价配置" style="max-width: 600px">
      <a-alert message="配置各类资源的单价，用于计算成本" type="info" show-icon style="margin-bottom: 24px" />
      
      <a-form :model="form" layout="vertical">
        <a-form-item label="选择集群" required>
          <a-select v-model:value="selectedCluster" style="width: 100%" placeholder="请选择集群" @change="fetchConfig">
            <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</a-select-option>
          </a-select>
        </a-form-item>

        <a-divider />

        <a-form-item label="CPU 单价（元/核/小时）">
          <a-input-number v-model:value="form.cpu_price_per_core" :min="0" :step="0.01" :precision="4" style="width: 100%" />
          <div style="color: #999; font-size: 12px; margin-top: 4px">参考：阿里云约 0.08-0.15 元/核/小时</div>
        </a-form-item>

        <a-form-item label="内存单价（元/GB/小时）">
          <a-input-number v-model:value="form.memory_price_per_gb" :min="0" :step="0.01" :precision="4" style="width: 100%" />
          <div style="color: #999; font-size: 12px; margin-top: 4px">参考：阿里云约 0.03-0.06 元/GB/小时</div>
        </a-form-item>

        <a-form-item label="存储单价（元/GB/月）">
          <a-input-number v-model:value="form.storage_price_per_gb" :min="0" :step="0.1" :precision="2" style="width: 100%" />
          <div style="color: #999; font-size: 12px; margin-top: 4px">参考：SSD云盘约 0.5-1.0 元/GB/月</div>
        </a-form-item>

        <a-form-item>
          <a-button type="primary" @click="save" :loading="saving" :disabled="!selectedCluster">保存配置</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <a-card title="云厂商参考价格" style="max-width: 600px; margin-top: 16px">
      <a-descriptions :column="1" size="small">
        <a-descriptions-item label="阿里云 ECS">CPU: 0.08-0.15 元/核/时，内存: 0.03-0.06 元/GB/时</a-descriptions-item>
        <a-descriptions-item label="腾讯云 CVM">CPU: 0.06-0.12 元/核/时，内存: 0.02-0.05 元/GB/时</a-descriptions-item>
        <a-descriptions-item label="华为云 ECS">CPU: 0.07-0.14 元/核/时，内存: 0.03-0.05 元/GB/时</a-descriptions-item>
        <a-descriptions-item label="AWS EC2">CPU: 0.10-0.20 元/核/时，内存: 0.04-0.08 元/GB/时</a-descriptions-item>
      </a-descriptions>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { costApi } from '@/services/cost'
import { k8sClusterApi } from '@/services/k8s'

const selectedCluster = ref<number>()
const clusters = ref<any[]>([])
const saving = ref(false)
const form = ref({
  cpu_price_per_core: 0.1,
  memory_price_per_gb: 0.05,
  storage_price_per_gb: 0.5
})

const fetchClusters = async () => {
  const res: any = await k8sClusterApi.list()
  if (res?.code === 0) clusters.value = res.data?.items || []
}

const fetchConfig = async () => {
  if (!selectedCluster.value) return
  const res: any = await costApi.getConfig(selectedCluster.value)
  if (res?.code === 0 && res.data) {
    form.value = {
      cpu_price_per_core: res.data.cpu_price_per_core,
      memory_price_per_gb: res.data.memory_price_per_gb,
      storage_price_per_gb: res.data.storage_price_per_gb
    }
  }
}

const save = async () => {
  if (!selectedCluster.value) { message.warning('请先选择集群'); return }
  saving.value = true
  const res: any = await costApi.saveConfig(selectedCluster.value, form.value)
  if (res?.code === 0) message.success('保存成功')
  saving.value = false
}

onMounted(fetchClusters)
</script>
