<template>
  <div class="pod-management">
    <a-card title="Pod 管理">
      <template #extra>
        <a-space>
          <a-select v-model:value="selectedCluster" placeholder="选择集群" style="width: 200px" @change="loadNamespaces">
            <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">
              {{ c.name }}
            </a-select-option>
          </a-select>
          <a-select v-model:value="selectedNamespace" placeholder="选择命名空间" style="width: 200px" @change="loadPods">
            <a-select-option v-for="ns in namespaces" :key="ns.name" :value="ns.name">
              {{ ns.name }}
            </a-select-option>
          </a-select>
          <a-button @click="loadPods" :loading="loading">
            <template #icon><ReloadOutlined /></template>
            刷新
          </a-button>
        </a-space>
      </template>

      <!-- 批量操作栏 -->
      <div v-if="selectedRowKeys.length > 0" class="batch-actions">
        <a-space>
          <span>已选择 {{ selectedRowKeys.length }} 项</span>
          <a-popconfirm title="确定批量删除选中的 Pod？" @confirm="batchDelete">
            <a-button danger size="small">
              <template #icon><DeleteOutlined /></template>
              批量删除
            </a-button>
          </a-popconfirm>
          <a-button size="small" @click="clearSelection">取消选择</a-button>
        </a-space>
      </div>

      <a-table 
        :columns="columns" 
        :data-source="pods" 
        :loading="loading" 
        row-key="name"
        :row-selection="{ selectedRowKeys, onChange: onSelectChange }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <a @click="showPodDetail(record)">{{ record.name }}</a>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">{{ record.status }}</a-tag>
          </template>
          <template v-else-if="column.key === 'containers'">
            <a-tag v-for="c in record.containers" :key="c.name" style="margin: 2px">
              {{ c.name }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showLogs(record)">日志</a-button>
              <a-button type="link" size="small" @click="showTerminal(record)">终端</a-button>
              <a-popconfirm title="确定删除此 Pod？" @confirm="deletePod(record)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- Pod 详情抽屉 -->
    <a-drawer v-model:open="detailDrawerVisible" title="Pod 详情" width="600" placement="right">
      <a-descriptions :column="1" bordered size="small" v-if="currentPod">
        <a-descriptions-item label="名称">{{ currentPod.name }}</a-descriptions-item>
        <a-descriptions-item label="命名空间">{{ currentPod.namespace }}</a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-tag :color="getStatusColor(currentPod.status)">{{ currentPod.status }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="就绪">{{ currentPod.ready }}</a-descriptions-item>
        <a-descriptions-item label="重启次数">{{ currentPod.restarts }}</a-descriptions-item>
        <a-descriptions-item label="运行时间">{{ currentPod.age }}</a-descriptions-item>
        <a-descriptions-item label="IP">{{ currentPod.ip }}</a-descriptions-item>
        <a-descriptions-item label="节点">{{ currentPod.node }}</a-descriptions-item>
        <a-descriptions-item label="创建时间">{{ currentPod.created_at }}</a-descriptions-item>
      </a-descriptions>

      <a-divider>容器</a-divider>
      <a-table :columns="containerColumns" :data-source="currentPod?.containers" :pagination="false" size="small">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'ready'">
            <a-tag :color="record.ready ? 'green' : 'red'">{{ record.ready ? '就绪' : '未就绪' }}</a-tag>
          </template>
        </template>
      </a-table>
    </a-drawer>

    <!-- 日志抽屉 -->
    <a-drawer v-model:open="logsDrawerVisible" :title="`日志 - ${currentPod?.name}`" width="80%" placement="right">
      <PodLogs 
        v-if="logsDrawerVisible && currentPod"
        :cluster-id="selectedCluster"
        :namespace="selectedNamespace"
        :pod-name="currentPod.name"
        style="height: calc(100vh - 120px)"
      />
    </a-drawer>

    <!-- 终端抽屉 -->
    <a-drawer v-model:open="terminalDrawerVisible" :title="`终端 - ${currentPod?.name}`" width="80%" placement="right" :destroy-on-close="true">
      <PodTerminal
        v-if="terminalDrawerVisible && currentPod"
        :cluster-id="selectedCluster"
        :namespace="selectedNamespace"
        :pod-name="currentPod.name"
        style="height: calc(100vh - 120px)"
      />
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { ReloadOutlined, DeleteOutlined } from '@ant-design/icons-vue'
import { k8sClusterApi, k8sResourceApi, k8sPodApi, type K8sCluster, type K8sNamespace, type K8sPodDetail } from '@/services/k8s'
import PodLogs from './components/PodLogs.vue'
import PodTerminal from './components/PodTerminal.vue'

const route = useRoute()
const loading = ref(false)
const clusters = ref<K8sCluster[]>([])
const namespaces = ref<K8sNamespace[]>([])
const pods = ref<K8sPodDetail[]>([])
const selectedCluster = ref<number>(0)
const selectedNamespace = ref('')
const currentPod = ref<K8sPodDetail | null>(null)
const detailDrawerVisible = ref(false)
const logsDrawerVisible = ref(false)
const terminalDrawerVisible = ref(false)

// 批量选择
const selectedRowKeys = ref<string[]>([])

const onSelectChange = (keys: string[]) => {
  selectedRowKeys.value = keys
}

const clearSelection = () => {
  selectedRowKeys.value = []
}

const batchDelete = async () => {
  if (selectedRowKeys.value.length === 0) return
  
  const total = selectedRowKeys.value.length
  let success = 0
  let failed = 0
  
  for (const podName of selectedRowKeys.value) {
    try {
      await k8sPodApi.delete(selectedCluster.value, selectedNamespace.value, podName)
      success++
    } catch {
      failed++
    }
  }
  
  if (failed === 0) {
    message.success(`成功删除 ${success} 个 Pod`)
  } else {
    message.warning(`删除完成：成功 ${success} 个，失败 ${failed} 个`)
  }
  
  clearSelection()
  loadPods()
}

const columns = [
  { title: '名称', key: 'name', dataIndex: 'name' },
  { title: '状态', key: 'status', dataIndex: 'status' },
  { title: '就绪', dataIndex: 'ready' },
  { title: '重启', dataIndex: 'restarts' },
  { title: '运行时间', dataIndex: 'age' },
  { title: 'IP', dataIndex: 'ip' },
  { title: '节点', dataIndex: 'node' },
  { title: '容器', key: 'containers' },
  { title: '操作', key: 'action', width: 180 }
]

const containerColumns = [
  { title: '名称', dataIndex: 'name' },
  { title: '镜像', dataIndex: 'image' },
  { title: '状态', key: 'ready' },
  { title: '重启', dataIndex: 'restart_count' }
]

const getStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    Running: 'green', Pending: 'orange', Succeeded: 'blue', Failed: 'red', Unknown: 'default'
  }
  return colors[status] || 'default'
}

const loadClusters = async () => {
  try {
    const res = await k8sClusterApi.list()
    clusters.value = res.data?.items || []
    
    // 优先使用路由参数中的 clusterId
    const routeClusterId = Number(route.params.id)
    if (routeClusterId) {
      selectedCluster.value = routeClusterId
      loadNamespaces()
    } else if (clusters.value.length > 0 && !selectedCluster.value) {
      selectedCluster.value = clusters.value[0].id
      loadNamespaces()
    }
  } catch {}
}

const loadNamespaces = async () => {
  if (!selectedCluster.value) return
  try {
    const res = await k8sResourceApi.getNamespaces(selectedCluster.value)
    namespaces.value = res.data || []
    
    // 优先使用路由 query 中的 namespace
    const queryNamespace = route.query.namespace as string
    if (queryNamespace && namespaces.value.some(ns => ns.name === queryNamespace)) {
      selectedNamespace.value = queryNamespace
    } else if (namespaces.value.length > 0) {
      selectedNamespace.value = namespaces.value[0].name
    }
    loadPods()
  } catch {}
}

const loadPods = async () => {
  if (!selectedCluster.value || !selectedNamespace.value) return
  loading.value = true
  try {
    const res = await k8sPodApi.list(selectedCluster.value, selectedNamespace.value)
    pods.value = res.data || []
  } finally {
    loading.value = false
  }
}

const showPodDetail = (pod: K8sPodDetail) => {
  currentPod.value = pod
  detailDrawerVisible.value = true
}

const showLogs = (pod: K8sPodDetail) => {
  currentPod.value = pod
  logsDrawerVisible.value = true
}

const showTerminal = (pod: K8sPodDetail) => {
  currentPod.value = pod
  terminalDrawerVisible.value = true
}

const deletePod = async (pod: K8sPodDetail) => {
  try {
    await k8sPodApi.delete(selectedCluster.value, selectedNamespace.value, pod.name)
    message.success('删除成功')
    loadPods()
  } catch {}
}

onMounted(() => {
  loadClusters()
})

// 处理路由 query 参数，自动打开终端或日志
watch(() => pods.value, (newPods) => {
  if (newPods.length > 0) {
    const queryPod = route.query.pod as string
    const queryAction = route.query.action as string
    if (queryPod && queryAction) {
      const pod = newPods.find(p => p.name === queryPod)
      if (pod) {
        currentPod.value = pod
        if (queryAction === 'terminal') {
          terminalDrawerVisible.value = true
        } else if (queryAction === 'logs') {
          logsDrawerVisible.value = true
        }
      }
    }
  }
}, { once: true })
</script>

<style scoped>
.pod-management {
  padding: 16px;
}
.batch-actions {
  margin-bottom: 16px;
  padding: 12px 16px;
  background: #e6f7ff;
  border: 1px solid #91d5ff;
  border-radius: 4px;
}
</style>
