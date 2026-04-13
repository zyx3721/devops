<template>
  <div class="deployment-management">
    <a-page-header
      title="Deployment 管理"
      :sub-title="clusterName"
      @back="() => $router.push('/k8s/clusters')"
    />

    <a-card :bordered="false">
      <a-row :gutter="16" style="margin-bottom: 16px">
        <a-col :span="6">
          <a-select
            v-model:value="selectedNamespace"
            placeholder="选择命名空间"
            style="width: 100%"
          >
            <a-select-option value="">全部命名空间</a-select-option>
            <a-select-option v-for="ns in namespaces" :key="ns.name" :value="ns.name">
              {{ ns.name }}
            </a-select-option>
          </a-select>
        </a-col>
        <a-col :span="6">
          <a-input-search
            v-model:value="searchText"
            placeholder="搜索 Deployment"
            allow-clear
          />
        </a-col>
        <a-col :span="12" style="text-align: right">
          <a-space>
            <a-dropdown v-if="selectedRowKeys.length > 0">
              <a-button type="primary">
                批量操作 ({{ selectedRowKeys.length }}) <DownOutlined />
              </a-button>
              <template #overlay>
                <a-menu @click="handleBatchAction">
                  <a-menu-item key="restart">
                    <SyncOutlined /> 批量重启
                  </a-menu-item>
                  <a-menu-item key="scale">
                    <ExpandOutlined /> 批量扩缩容
                  </a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
            <a-button @click="loadDeployments" :loading="loading">
              <template #icon><ReloadOutlined /></template>
              刷新
            </a-button>
          </a-space>
        </a-col>
      </a-row>

      <a-table
        :columns="columns"
        :data-source="filteredDeployments"
        :loading="loading"
        :pagination="{ pageSize: 20 }"
        row-key="name"
        :row-selection="{ selectedRowKeys, onChange: onSelectChange }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <a @click="showDetail(record)">{{ record.name }}</a>
          </template>
          <template v-else-if="column.key === 'ready'">
            <a-tag :color="record.ready === record.replicas ? 'green' : 'orange'">
              {{ record.ready }}/{{ record.replicas }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'images'">
            <div v-for="(img, idx) in record.images" :key="idx" class="image-tag">
              <a-tooltip :title="img">
                <a-tag>{{ getImageShortName(img) }}</a-tag>
              </a-tooltip>
            </div>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space :size="4">
              <a-tooltip title="查看 YAML">
                <a-button type="link" size="small" @click="showDetail(record)">YAML</a-button>
              </a-tooltip>
              <a-dropdown>
                <a-button type="link" size="small">
                  更多 <DownOutlined />
                </a-button>
                <template #overlay>
                  <a-menu @click="({ key }) => handleMenuClick(key, record)">
                    <a-menu-item key="pods">
                      <AppstoreOutlined /> 查看 Pods
                    </a-menu-item>
                    <a-menu-item key="scale">
                      <ExpandOutlined /> 扩缩容
                    </a-menu-item>
                    <a-menu-item key="image">
                      <EditOutlined /> 更新镜像
                    </a-menu-item>
                    <a-menu-item key="restart">
                      <SyncOutlined /> 重启
                    </a-menu-item>
                    <a-menu-item key="rollback">
                      <RollbackOutlined /> 回滚
                    </a-menu-item>
                  </a-menu>
                </template>
              </a-dropdown>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 批量扩缩容弹窗 -->
    <a-modal
      v-model:open="batchScaleModalVisible"
      title="批量扩缩容"
      @ok="handleBatchScale"
      :confirm-loading="batchScaleLoading"
    >
      <a-form layout="vertical">
        <a-form-item label="选中的 Deployment">
          <div>
            <a-tag v-for="name in selectedRowKeys" :key="name" style="margin: 2px">{{ name }}</a-tag>
          </div>
        </a-form-item>
        <a-form-item label="目标副本数">
          <a-input-number v-model:value="batchTargetReplicas" :min="0" :max="100" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 扩缩容弹窗 -->
    <a-modal
      v-model:open="scaleModalVisible"
      title="扩缩容"
      @ok="handleScale"
      :confirm-loading="scaleLoading"
    >
      <a-form layout="vertical">
        <a-form-item label="Deployment">
          <a-input :value="currentDeployment?.name" disabled />
        </a-form-item>
        <a-form-item label="当前副本数">
          <a-input :value="currentDeployment?.replicas" disabled />
        </a-form-item>
        <a-form-item label="目标副本数">
          <a-input-number v-model:value="targetReplicas" :min="0" :max="100" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 更新镜像弹窗 -->
    <a-modal
      v-model:open="imageModalVisible"
      title="更新镜像"
      @ok="handleUpdateImage"
      :confirm-loading="imageLoading"
      width="600px"
    >
      <a-form layout="vertical">
        <a-form-item label="Deployment">
          <a-input :value="currentDeployment?.name" disabled />
        </a-form-item>
        <a-form-item label="容器">
          <a-select v-model:value="selectedContainer" style="width: 100%">
            <a-select-option v-for="c in containers" :key="c" :value="c">{{ c }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="新镜像">
          <a-input v-model:value="newImage" placeholder="输入新镜像地址" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 回滚弹窗 -->
    <a-modal
      v-model:open="rollbackModalVisible"
      title="回滚版本"
      @ok="handleRollback"
      :confirm-loading="rollbackLoading"
      width="700px"
    >
      <a-form layout="vertical">
        <a-form-item label="Deployment">
          <a-input :value="currentDeployment?.name" disabled />
        </a-form-item>
        <a-form-item label="选择版本">
          <a-table
            :columns="revisionColumns"
            :data-source="revisions"
            :loading="revisionsLoading"
            :pagination="false"
            row-key="revision"
            :row-selection="{ type: 'radio', selectedRowKeys: selectedRevision, onChange: onRevisionSelect }"
            size="small"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 详情抽屉 -->
    <a-drawer
      v-model:open="detailVisible"
      title="Deployment 详情"
      width="700"
      placement="right"
    >
      <template v-if="deploymentDetail">
        <a-descriptions :column="2" bordered size="small">
          <a-descriptions-item label="名称">{{ deploymentDetail.name }}</a-descriptions-item>
          <a-descriptions-item label="命名空间">{{ deploymentDetail.namespace }}</a-descriptions-item>
          <a-descriptions-item label="就绪状态">{{ deploymentDetail.ready }}</a-descriptions-item>
          <a-descriptions-item label="策略">{{ deploymentDetail.strategy }}</a-descriptions-item>
          <a-descriptions-item label="创建时间" :span="2">{{ deploymentDetail.created_at }}</a-descriptions-item>
        </a-descriptions>

        <a-divider>镜像</a-divider>
        <div v-for="(img, idx) in deploymentDetail.images" :key="idx">
          <a-tag>{{ img }}</a-tag>
        </div>

        <a-divider>更新进度</a-divider>
        <a-descriptions :column="2" bordered size="small" v-if="updateProgress">
          <a-descriptions-item label="总副本">{{ updateProgress.replicas }}</a-descriptions-item>
          <a-descriptions-item label="已更新">{{ updateProgress.updated_replicas }}</a-descriptions-item>
          <a-descriptions-item label="就绪">{{ updateProgress.ready_replicas }}</a-descriptions-item>
          <a-descriptions-item label="可用">{{ updateProgress.available_replicas }}</a-descriptions-item>
          <a-descriptions-item label="状态" :span="2">
            <a-tag :color="updateProgress.status === 'Complete' ? 'green' : 'blue'">
              {{ updateProgress.status }}
            </a-tag>
          </a-descriptions-item>
        </a-descriptions>

        <a-divider>标签</a-divider>
        <div>
          <a-tag v-for="(v, k) in deploymentDetail.labels" :key="k">{{ k }}={{ v }}</a-tag>
        </div>
      </template>
    </a-drawer>

    <!-- Pods 抽屉 -->
    <a-drawer
      v-model:open="podsDrawerVisible"
      :title="`Pods - ${currentDeployment?.name}`"
      width="900"
      placement="right"
    >
      <a-table
        :columns="podColumns"
        :data-source="deploymentPods"
        :loading="podsLoading"
        :pagination="false"
        row-key="name"
        size="small"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="getPodStatusColor(record.status)">{{ record.status }}</a-tag>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showPodLogs(record)">日志</a-button>
              <a-button type="link" size="small" @click="showPodTerminal(record)">终端</a-button>
              <a-popconfirm title="确定删除此 Pod？" @confirm="deletePod(record)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-drawer>

    <!-- Pod 日志抽屉 -->
    <a-drawer
      v-model:open="logsDrawerVisible"
      :title="`日志 - ${currentPod?.name}`"
      width="80%"
      placement="right"
      :destroy-on-close="true"
    >
      <PodLogs
        v-if="logsDrawerVisible && currentPod"
        :cluster-id="clusterId"
        :namespace="currentPod.namespace || selectedNamespace"
        :pod-name="currentPod.name"
      />
    </a-drawer>

    <!-- Pod 终端抽屉 -->
    <a-drawer
      v-model:open="terminalDrawerVisible"
      :title="`终端 - ${currentPod?.name}`"
      width="80%"
      placement="right"
      :destroy-on-close="true"
    >
      <PodTerminal
        v-if="terminalDrawerVisible && currentPod"
        :cluster-id="clusterId"
        :namespace="currentPod.namespace || selectedNamespace"
        :pod-name="currentPod.name"
      />
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import {
  ReloadOutlined,
  ExpandOutlined,
  EditOutlined,
  SyncOutlined,
  RollbackOutlined,
  AppstoreOutlined,
  DownOutlined
} from '@ant-design/icons-vue'
import { k8sResourceApi, k8sDeploymentApi, k8sClusterApi, k8sPodApi } from '@/services/k8s'
import type { K8sDeploymentDetail, K8sRevisionInfo, K8sUpdateProgress, K8sNamespace, K8sPodDetail } from '@/services/k8s'
import PodLogs from './components/PodLogs.vue'
import PodTerminal from './components/PodTerminal.vue'

const route = useRoute()
const clusterId = computed(() => Number(route.params.id))
const clusterName = ref('')

const loading = ref(false)
const namespaces = ref<K8sNamespace[]>([])
const selectedNamespace = ref('')
const searchText = ref('')
const deployments = ref<K8sDeploymentDetail[]>([])

const filteredDeployments = computed(() => {
  if (!searchText.value) return deployments.value
  return deployments.value.filter(d => d.name.toLowerCase().includes(searchText.value.toLowerCase()))
})

// 批量选择
const selectedRowKeys = ref<string[]>([])
const batchScaleModalVisible = ref(false)
const batchScaleLoading = ref(false)
const batchTargetReplicas = ref(1)

const onSelectChange = (keys: string[]) => {
  selectedRowKeys.value = keys
}

const handleBatchAction = async ({ key }: { key: string }) => {
  if (selectedRowKeys.value.length === 0) {
    message.warning('请先选择 Deployment')
    return
  }
  
  if (key === 'restart') {
    Modal.confirm({
      title: '批量重启',
      content: `确定要重启选中的 ${selectedRowKeys.value.length} 个 Deployment 吗？`,
      onOk: async () => {
        let success = 0
        let failed = 0
        for (const name of selectedRowKeys.value) {
          const deploy = deployments.value.find(d => d.name === name)
          const ns = deploy?.namespace || selectedNamespace.value
          try {
            await k8sDeploymentApi.restart(clusterId.value, ns, name)
            success++
          } catch {
            failed++
          }
        }
        if (failed === 0) {
          message.success(`成功重启 ${success} 个 Deployment`)
        } else {
          message.warning(`重启完成：成功 ${success} 个，失败 ${failed} 个`)
        }
        selectedRowKeys.value = []
        loadDeployments()
      }
    })
  } else if (key === 'scale') {
    batchTargetReplicas.value = 1
    batchScaleModalVisible.value = true
  }
}

const handleBatchScale = async () => {
  if (selectedRowKeys.value.length === 0) return
  
  batchScaleLoading.value = true
  let success = 0
  let failed = 0
  
  for (const name of selectedRowKeys.value) {
    const deploy = deployments.value.find(d => d.name === name)
    const ns = deploy?.namespace || selectedNamespace.value
    try {
      await k8sDeploymentApi.scale(clusterId.value, ns, name, batchTargetReplicas.value)
      success++
    } catch {
      failed++
    }
  }
  
  batchScaleLoading.value = false
  batchScaleModalVisible.value = false
  
  if (failed === 0) {
    message.success(`成功扩缩容 ${success} 个 Deployment`)
  } else {
    message.warning(`扩缩容完成：成功 ${success} 个，失败 ${failed} 个`)
  }
  
  selectedRowKeys.value = []
  loadDeployments()
}

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '命名空间', dataIndex: 'namespace', key: 'namespace', width: 130 },
  { title: '副本', key: 'ready', width: 80, align: 'center' as const },
  { title: '镜像', key: 'images', ellipsis: true },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 130, fixed: 'right' as const }
]

// Pods 相关
const podsDrawerVisible = ref(false)
const podsLoading = ref(false)
const deploymentPods = ref<K8sPodDetail[]>([])
const currentPod = ref<K8sPodDetail | null>(null)
const logsDrawerVisible = ref(false)
const terminalDrawerVisible = ref(false)

const podColumns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '状态', key: 'status', width: 100 },
  { title: '就绪', dataIndex: 'ready', width: 80 },
  { title: '重启', dataIndex: 'restarts', width: 80 },
  { title: 'IP', dataIndex: 'ip', width: 120 },
  { title: '节点', dataIndex: 'node', width: 150 },
  { title: '操作', key: 'action', width: 180 }
]

const getPodStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    Running: 'green', Pending: 'orange', Succeeded: 'blue', Failed: 'red', Unknown: 'default'
  }
  return colors[status] || 'default'
}

// 扩缩容
const scaleModalVisible = ref(false)
const scaleLoading = ref(false)
const currentDeployment = ref<K8sDeploymentDetail | null>(null)
const targetReplicas = ref(1)

// 更新镜像
const imageModalVisible = ref(false)
const imageLoading = ref(false)
const containers = ref<string[]>([])
const selectedContainer = ref('')
const newImage = ref('')

// 回滚
const rollbackModalVisible = ref(false)
const rollbackLoading = ref(false)
const revisionsLoading = ref(false)
const revisions = ref<K8sRevisionInfo[]>([])
const selectedRevision = ref<number[]>([])

const revisionColumns = [
  { title: '版本', dataIndex: 'revision', key: 'revision', width: 80 },
  { title: '镜像', dataIndex: 'image', key: 'image' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '变更原因', dataIndex: 'change_cause', key: 'change_cause' }
]

// 详情
const detailVisible = ref(false)
const deploymentDetail = ref<K8sDeploymentDetail | null>(null)
const updateProgress = ref<K8sUpdateProgress | null>(null)

const getImageShortName = (image: string) => {
  const parts = image.split('/')
  return parts[parts.length - 1]
}

// 下拉菜单点击处理
const handleMenuClick = (key: string, record: K8sDeploymentDetail) => {
  switch (key) {
    case 'pods':
      showPodsDrawer(record)
      break
    case 'scale':
      showScaleModal(record)
      break
    case 'image':
      showImageModal(record)
      break
    case 'restart':
      handleRestart(record)
      break
    case 'rollback':
      showRollbackModal(record)
      break
  }
}

const loadClusterInfo = async () => {
  try {
    const res = await k8sClusterApi.getCluster(clusterId.value)
    if (res.code === 0) {
      clusterName.value = res.data.name
    }
  } catch (e) {
    console.error(e)
  }
}

const loadNamespaces = async () => {
  try {
    const res = await k8sResourceApi.getNamespaces(clusterId.value)
    if (res.code === 0) {
      namespaces.value = res.data || []
    }
  } catch (e) {
    console.error(e)
  }
}

const loadDeployments = async () => {
  loading.value = true
  try {
    const res = await k8sDeploymentApi.list(clusterId.value, selectedNamespace.value)
    if (res.code === 0) {
      deployments.value = res.data || []
    }
  } catch (e) {
    message.error('加载 Deployment 列表失败')
  } finally {
    loading.value = false
  }
}

const showScaleModal = (record: K8sDeploymentDetail) => {
  currentDeployment.value = record
  targetReplicas.value = record.replicas
  scaleModalVisible.value = true
}

const handleScale = async () => {
  if (!currentDeployment.value) return
  scaleLoading.value = true
  try {
    const res = await k8sDeploymentApi.scale(
      clusterId.value,
      currentDeployment.value.namespace || selectedNamespace.value,
      currentDeployment.value.name,
      targetReplicas.value
    )
    if (res.code === 0) {
      message.success('扩缩容成功')
      scaleModalVisible.value = false
      loadDeployments()
    } else {
      message.error(res.message || '扩缩容失败')
    }
  } catch (e) {
    message.error('扩缩容失败')
  } finally {
    scaleLoading.value = false
  }
}

const showImageModal = async (record: K8sDeploymentDetail) => {
  currentDeployment.value = record
  // 从镜像中提取容器名（简化处理，实际应从详情获取）
  containers.value = record.images.map((_, idx) => `container-${idx}`)
  // 获取详情以获取真实容器名
  try {
    const res = await k8sDeploymentApi.get(clusterId.value, record.namespace || selectedNamespace.value, record.name)
    if (res.code === 0 && res.data) {
      // 假设 selector 中有容器信息，这里简化处理
      containers.value = record.images.map((img, idx) => img.split('/').pop()?.split(':')[0] || `container-${idx}`)
    }
  } catch (e) {
    console.error(e)
  }
  selectedContainer.value = containers.value[0] || ''
  newImage.value = ''
  imageModalVisible.value = true
}

const handleUpdateImage = async () => {
  if (!currentDeployment.value || !selectedContainer.value || !newImage.value) {
    message.warning('请填写完整信息')
    return
  }
  imageLoading.value = true
  try {
    const res = await k8sDeploymentApi.updateImage(
      clusterId.value,
      currentDeployment.value.namespace || selectedNamespace.value,
      currentDeployment.value.name,
      selectedContainer.value,
      newImage.value
    )
    if (res.code === 0) {
      message.success('镜像更新成功')
      imageModalVisible.value = false
      loadDeployments()
    } else {
      message.error(res.message || '镜像更新失败')
    }
  } catch (e) {
    message.error('镜像更新失败')
  } finally {
    imageLoading.value = false
  }
}

const handleRestart = (record: K8sDeploymentDetail) => {
  Modal.confirm({
    title: '确认重启',
    content: `确定要重启 Deployment "${record.name}" 吗？`,
    onOk: async () => {
      try {
        const res = await k8sDeploymentApi.restart(clusterId.value, record.namespace || selectedNamespace.value, record.name)
        if (res.code === 0) {
          message.success('重启成功')
          loadDeployments()
        } else {
          message.error(res.message || '重启失败')
        }
      } catch (e) {
        message.error('重启失败')
      }
    }
  })
}

const showRollbackModal = async (record: K8sDeploymentDetail) => {
  currentDeployment.value = record
  selectedRevision.value = []
  rollbackModalVisible.value = true
  revisionsLoading.value = true
  try {
    const res = await k8sDeploymentApi.getRevisions(clusterId.value, record.namespace || selectedNamespace.value, record.name)
    if (res.code === 0) {
      revisions.value = res.data || []
    }
  } catch (e) {
    message.error('获取版本历史失败')
  } finally {
    revisionsLoading.value = false
  }
}

const onRevisionSelect = (keys: number[]) => {
  selectedRevision.value = keys
}

const handleRollback = async () => {
  if (!currentDeployment.value || selectedRevision.value.length === 0) {
    message.warning('请选择要回滚的版本')
    return
  }
  rollbackLoading.value = true
  try {
    const res = await k8sDeploymentApi.rollback(
      clusterId.value,
      currentDeployment.value.namespace || selectedNamespace.value,
      currentDeployment.value.name,
      selectedRevision.value[0]
    )
    if (res.code === 0) {
      message.success('回滚成功')
      rollbackModalVisible.value = false
      loadDeployments()
    } else {
      message.error(res.message || '回滚失败')
    }
  } catch (e) {
    message.error('回滚失败')
  } finally {
    rollbackLoading.value = false
  }
}

const showDetail = async (record: K8sDeploymentDetail) => {
  detailVisible.value = true
  try {
    const ns = record.namespace || selectedNamespace.value
    const [detailRes, progressRes] = await Promise.all([
      k8sDeploymentApi.get(clusterId.value, ns, record.name),
      k8sDeploymentApi.getProgress(clusterId.value, ns, record.name)
    ])
    if (detailRes.code === 0) {
      deploymentDetail.value = detailRes.data
    }
    if (progressRes.code === 0) {
      updateProgress.value = progressRes.data
    }
  } catch (e) {
    message.error('获取详情失败')
  }
}

// Pods 相关方法
const showPodsDrawer = async (record: K8sDeploymentDetail) => {
  currentDeployment.value = record
  podsDrawerVisible.value = true
  podsLoading.value = true
  try {
    // 使用标签选择器获取 Deployment 的 Pods
    const labelSelector = Object.entries(record.selector || {})
      .map(([k, v]) => `${k}=${v}`)
      .join(',')
    // 使用 deployment 的 namespace，而不是 selectedNamespace
    const ns = record.namespace || selectedNamespace.value
    const res = await k8sPodApi.list(clusterId.value, ns, labelSelector)
    if (res.code === 0) {
      deploymentPods.value = res.data || []
    }
  } catch (e) {
    message.error('获取 Pods 失败')
  } finally {
    podsLoading.value = false
  }
}

const showPodLogs = (pod: K8sPodDetail) => {
  currentPod.value = pod
  logsDrawerVisible.value = true
}

const showPodTerminal = (pod: K8sPodDetail) => {
  currentPod.value = pod
  terminalDrawerVisible.value = true
}

const deletePod = async (pod: K8sPodDetail) => {
  try {
    const res = await k8sPodApi.delete(clusterId.value, selectedNamespace.value, pod.name)
    if (res.code === 0) {
      message.success('删除成功')
      // 刷新 Pods 列表
      if (currentDeployment.value) {
        showPodsDrawer(currentDeployment.value)
      }
    } else {
      message.error(res.message || '删除失败')
    }
  } catch (e) {
    message.error('删除失败')
  }
}

onMounted(() => {
  loadClusterInfo()
  loadNamespaces()
  loadDeployments()
})

// 监听命名空间变化，自动加载数据
watch(() => selectedNamespace.value, () => {
  loadDeployments()
})

// 处理路由 query 参数，自动打开镜像更新或回滚弹窗
watch(() => deployments.value, (newDeployments) => {
  if (newDeployments.length > 0) {
    const queryNamespace = route.query.namespace as string
    const queryDeployment = route.query.deployment as string
    const queryAction = route.query.action as string
    
    // 如果有 namespace 参数，先切换命名空间
    if (queryNamespace && queryNamespace !== selectedNamespace.value) {
      selectedNamespace.value = queryNamespace
      return
    }
    
    if (queryDeployment && queryAction) {
      const deployment = newDeployments.find(d => d.name === queryDeployment)
      if (deployment) {
        if (queryAction === 'image') {
          showImageModal(deployment)
        } else if (queryAction === 'rollback') {
          showRollbackModal(deployment)
        }
      }
    }
  }
}, { once: true })
</script>

<style scoped>
.deployment-management {
  padding: 0;
}
.image-tag {
  margin-bottom: 4px;
}
</style>
