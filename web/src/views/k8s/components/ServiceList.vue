<template>
  <div class="service-list">
    <el-table
      :data="filteredServices"
      stripe
      style="width: 100%"
      v-loading="loading"
    >
      <el-table-column prop="name" label="名称" min-width="150" />
      <el-table-column prop="namespace" label="命名空间" width="120" />
      <el-table-column prop="type" label="类型" width="120">
        <template #default="{ row }">
          <el-tag>{{ row.type }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="cluster_ip" label="Cluster IP" width="140" />
      <el-table-column prop="external_ip" label="External IP" width="140">
        <template #default="{ row }">
          {{ row.external_ip || '-' }}
        </template>
      </el-table-column>
      <el-table-column label="端口" min-width="200">
        <template #default="{ row }">
          <div v-for="(port, index) in row.ports" :key="index" class="port-item">
            {{ port.port }}{{ port.node_port ? ':' + port.node_port : '' }}/{{ port.protocol }}
            <span v-if="port.name" class="port-name">{{ port.name }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="age" label="运行时间" width="100" />
      <el-table-column label="操作" width="120" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="handleViewDetail(row)">详情</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface ServicePort {
  name: string
  protocol: string
  port: number
  target_port: string
  node_port?: number
}

interface ServiceInfo {
  name: string
  namespace: string
  type: string
  cluster_ip: string
  external_ip: string
  ports: ServicePort[]
  age: string
  selector: Record<string, string>
  created_at: string
}

interface Props {
  services: ServiceInfo[]
  loading?: boolean
  searchKeyword?: string
}

interface Emits {
  (e: 'viewDetail', service: ServiceInfo): void
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  searchKeyword: ''
})

const emit = defineEmits<Emits>()

const filteredServices = computed(() => {
  if (!props.searchKeyword) {
    return props.services
  }
  const keyword = props.searchKeyword.toLowerCase()
  return props.services.filter(s => 
    s.name.toLowerCase().includes(keyword)
  )
})

const handleViewDetail = (service: ServiceInfo) => {
  emit('viewDetail', service)
}
</script>

<style scoped>
.service-list {
  width: 100%;
}

.port-item {
  font-size: 12px;
  color: var(--el-text-color-regular);
  line-height: 1.5;
}

.port-name {
  color: var(--el-text-color-secondary);
  margin-left: 4px;
}
</style>
