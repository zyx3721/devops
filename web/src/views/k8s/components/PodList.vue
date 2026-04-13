<template>
  <div class="pod-list">
    <div class="filter-bar">
      <el-select
        v-model="selectedStatus"
        placeholder="状态过滤"
        clearable
        style="width: 150px; margin-bottom: 16px"
        @change="handleStatusChange"
      >
        <el-option label="全部状态" value="" />
        <el-option label="Running" value="Running" />
        <el-option label="Pending" value="Pending" />
        <el-option label="Failed" value="Failed" />
        <el-option label="Succeeded" value="Succeeded" />
        <el-option label="Unknown" value="Unknown" />
      </el-select>
    </div>

    <el-table
      :data="filteredPods"
      stripe
      style="width: 100%"
      v-loading="loading"
    >
      <el-table-column prop="name" label="名称" min-width="200" />
      <el-table-column prop="namespace" label="命名空间" width="120" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="getStatusType(row.status)">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="ready" label="就绪" width="80" />
      <el-table-column prop="restarts" label="重启次数" width="100" />
      <el-table-column prop="node" label="节点" min-width="150" />
      <el-table-column prop="ip" label="IP" width="130" />
      <el-table-column prop="age" label="运行时间" width="100" />
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="handleViewLogs(row)">日志</el-button>
          <el-button size="small" @click="handleViewDetail(row)">详情</el-button>
          <el-button size="small" type="danger" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessageBox } from 'element-plus'

interface ContainerInfo {
  name: string
  image: string
  ready: boolean
  state: string
  restart_count: number
}

interface PodInfo {
  name: string
  namespace: string
  status: string
  ready: string
  restarts: number
  age: string
  ip: string
  node: string
  containers: ContainerInfo[]
  labels: Record<string, string>
  created_at: string
}

interface Props {
  pods: PodInfo[]
  loading?: boolean
  searchKeyword?: string
  statusFilter?: string
}

interface Emits {
  (e: 'viewLogs', pod: PodInfo): void
  (e: 'viewDetail', pod: PodInfo): void
  (e: 'delete', pod: PodInfo): void
  (e: 'statusChange', status: string): void
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  searchKeyword: '',
  statusFilter: ''
})

const emit = defineEmits<Emits>()

const selectedStatus = ref(props.statusFilter)

const filteredPods = computed(() => {
  let result = props.pods

  // 按名称搜索
  if (props.searchKeyword) {
    const keyword = props.searchKeyword.toLowerCase()
    result = result.filter(p => p.name.toLowerCase().includes(keyword))
  }

  // 按状态过滤
  if (selectedStatus.value) {
    result = result.filter(p => p.status === selectedStatus.value)
  }

  return result
})

const getStatusType = (status: string) => {
  switch (status) {
    case 'Running':
      return 'success'
    case 'Pending':
      return 'warning'
    case 'Failed':
      return 'danger'
    case 'Succeeded':
      return 'info'
    default:
      return ''
  }
}

const handleStatusChange = (status: string) => {
  emit('statusChange', status)
}

const handleViewLogs = (pod: PodInfo) => {
  emit('viewLogs', pod)
}

const handleViewDetail = (pod: PodInfo) => {
  emit('viewDetail', pod)
}

const handleDelete = async (pod: PodInfo) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除 Pod "${pod.name}" 吗？`,
      '确认操作',
      { type: 'warning' }
    )
    emit('delete', pod)
  } catch {
    // 用户取消
  }
}
</script>

<style scoped>
.pod-list {
  width: 100%;
}

.filter-bar {
  margin-bottom: 16px;
}
</style>
