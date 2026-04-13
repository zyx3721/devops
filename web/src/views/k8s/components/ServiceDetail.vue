<template>
  <el-dialog
    v-model="visible"
    :title="`Service 详情: ${service?.name || ''}`"
    width="800px"
    @close="handleClose"
  >
    <div v-if="service" class="service-detail">
      <el-descriptions :column="2" border>
        <el-descriptions-item label="名称">{{ service.name }}</el-descriptions-item>
        <el-descriptions-item label="命名空间">{{ service.namespace }}</el-descriptions-item>
        <el-descriptions-item label="类型">
          <el-tag>{{ service.type }}</el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="Cluster IP">{{ service.cluster_ip }}</el-descriptions-item>
        <el-descriptions-item label="External IP">
          {{ service.external_ip || '-' }}
        </el-descriptions-item>
        <el-descriptions-item label="创建时间">{{ service.created_at }}</el-descriptions-item>
      </el-descriptions>

      <el-divider content-position="left">端口映射</el-divider>
      <el-table :data="service.ports" border style="width: 100%">
        <el-table-column prop="name" label="名称" width="120">
          <template #default="{ row }">
            {{ row.name || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="protocol" label="协议" width="80" />
        <el-table-column prop="port" label="端口" width="80" />
        <el-table-column prop="target_port" label="目标端口" width="100" />
        <el-table-column prop="node_port" label="Node Port" width="100">
          <template #default="{ row }">
            {{ row.node_port || '-' }}
          </template>
        </el-table-column>
      </el-table>

      <el-divider content-position="left">选择器</el-divider>
      <el-tag
        v-for="(value, key) in service.selector"
        :key="key"
        style="margin-right: 8px; margin-bottom: 8px"
      >
        {{ key }}: {{ value }}
      </el-tag>
      <div v-if="!service.selector || Object.keys(service.selector).length === 0">
        <el-empty description="无选择器" :image-size="60" />
      </div>

      <el-divider content-position="left">Endpoints</el-divider>
      <el-table :data="service.endpoints" border style="width: 100%">
        <el-table-column prop="ip" label="IP" width="150" />
        <el-table-column prop="node_name" label="节点" min-width="200" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.ready ? 'success' : 'danger'">
              {{ row.ready ? '就绪' : '未就绪' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!service.endpoints || service.endpoints.length === 0">
        <el-empty description="无 Endpoints" :image-size="60" />
      </div>

      <el-divider content-position="left">Labels</el-divider>
      <el-tag
        v-for="(value, key) in service.labels"
        :key="key"
        type="info"
        style="margin-right: 8px; margin-bottom: 8px"
      >
        {{ key }}: {{ value }}
      </el-tag>
      <div v-if="!service.labels || Object.keys(service.labels).length === 0">
        <el-empty description="无 Labels" :image-size="60" />
      </div>
    </div>

    <template #footer>
      <el-button @click="handleClose">关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

interface ServicePort {
  name: string
  protocol: string
  port: number
  target_port: string
  node_port?: number
}

interface EndpointInfo {
  ip: string
  node_name: string
  ready: boolean
}

interface ServiceDetail {
  name: string
  namespace: string
  type: string
  cluster_ip: string
  external_ip: string
  ports: ServicePort[]
  age: string
  selector: Record<string, string>
  created_at: string
  labels: Record<string, string>
  annotations: Record<string, string>
  endpoints: EndpointInfo[]
}

interface Props {
  modelValue: boolean
  service: ServiceDetail | null
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const visible = ref(props.modelValue)

watch(() => props.modelValue, (newVal) => {
  visible.value = newVal
})

watch(visible, (newVal) => {
  emit('update:modelValue', newVal)
})

const handleClose = () => {
  visible.value = false
}
</script>

<style scoped>
.service-detail {
  max-height: 600px;
  overflow-y: auto;
}

.el-divider {
  margin: 24px 0 16px 0;
}
</style>
