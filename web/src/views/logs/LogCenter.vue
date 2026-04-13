<template>
  <div class="log-center-page">
    <!-- 顶部工具栏 -->
    <div class="log-toolbar">
      <a-select v-model:value="selectedCluster" placeholder="选择集群" @change="onClusterChange" style="width: 200px">
        <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">
          {{ cluster.name }}
        </a-select-option>
      </a-select>
      <a-select v-model:value="selectedNamespace" placeholder="选择命名空间" @change="onNamespaceChange" style="width: 200px">
        <a-select-option v-for="ns in namespaces" :key="ns" :value="ns">
          {{ ns }}
        </a-select-option>
      </a-select>
      <a-switch 
        v-model:checked="multiPodMode" 
        checked-children="多Pod模式" 
        un-checked-children="单Pod模式"
      />
      <a-button type="primary" @click="refreshPods">
        <template #icon><ReloadOutlined /></template>
        刷新
      </a-button>
      <a-button @click="openSettings">
        <template #icon><SettingOutlined /></template>
        设置
      </a-button>
      <a-button @click="openExport">
        <template #icon><DownloadOutlined /></template>
        导出
      </a-button>
    </div>

    <div class="log-main-content">
      <!-- 左侧 Pod 列表 -->
      <div class="log-pod-list">
        <div class="log-pod-list-header">
          <span>Pod 列表 {{ multiPodMode && selectedPods.length > 0 ? `(${selectedPods.length}/10)` : '' }}</span>
          <a-input v-model:value="podFilter" placeholder="搜索 Pod" size="small" allow-clear style="width: 150px" />
        </div>
        <div class="log-pod-list-scroll">
          <!-- 多 Pod 模式 -->
          <template v-if="multiPodMode">
            <div v-for="pod in filteredPods" :key="pod.name" 
                 :class="['log-pod-item', { active: selectedPods.includes(pod.name) }]"
                 @click="togglePodSelection(pod)">
              <a-checkbox 
                :checked="selectedPods.includes(pod.name)" 
                :disabled="!selectedPods.includes(pod.name) && selectedPods.length >= 10"
                @click.stop
                @change="(e: any) => togglePodSelection(pod, e.target.checked)"
              />
              <MonitorOutlined :style="{ color: getPodStatusIconColor(pod.status), fontSize: '16px' }" />
              <span class="log-pod-name">{{ pod.name }}</span>
              <a-tag size="small" :color="getPodStatusColor(pod.status)">{{ pod.status }}</a-tag>
              <div 
                v-if="selectedPods.includes(pod.name)" 
                class="log-pod-color-indicator" 
                :style="{ backgroundColor: getPodColor(pod.name) }"
              />
            </div>
          </template>
          <!-- 单 Pod 模式 -->
          <template v-else>
            <div v-for="pod in filteredPods" :key="pod.name" 
                 :class="['log-pod-item', { active: selectedPod === pod.name }]"
                 @click="selectPod(pod)">
              <MonitorOutlined :style="{ color: getPodStatusIconColor(pod.status), fontSize: '16px' }" />
              <span class="log-pod-name">{{ pod.name }}</span>
              <a-tag size="small" :color="getPodStatusColor(pod.status)">{{ pod.status }}</a-tag>
            </div>
          </template>
          <a-empty v-if="filteredPods.length === 0" description="暂无 Pod" />
        </div>
      </div>

      <!-- 右侧日志展示 -->
      <div class="log-viewer-wrapper">
        <!-- 多 Pod 模式 -->
        <LogViewer 
          v-if="multiPodMode && selectedPods.length > 0"
          :cluster-id="selectedCluster"
          :namespace="selectedNamespace"
          :pod-names="selectedPods"
          :pod-colors="podColorMap"
          :highlight-rules="highlightRules"
          multi-pod
        />
        <!-- 单 Pod 模式 -->
        <LogViewer 
          v-else-if="!multiPodMode && selectedPod"
          :cluster-id="selectedCluster"
          :namespace="selectedNamespace"
          :pod-name="selectedPod"
          :container="selectedContainer"
          :highlight-rules="highlightRules"
          @container-change="onContainerChange"
        />
        <a-empty v-else :description="multiPodMode ? '请选择 Pod 查看日志（最多10个）' : '请选择一个 Pod 查看日志'" />
      </div>
    </div>

    <!-- 设置抽屉 -->
    <a-drawer v-model:open="settingsVisible" title="日志设置" width="400px">
      <HighlightConfig v-model:rules="highlightRules" />
    </a-drawer>

    <!-- 导出对话框 -->
    <a-modal v-model:open="exportVisible" title="导出日志" width="500px">
      <LogExport 
        :cluster-id="selectedCluster || 0"
        :namespace="selectedNamespace"
        :pod-name="multiPodMode ? '' : selectedPod"
        :pod-names="multiPodMode ? selectedPods : []"
        @close="exportVisible = false"
      />
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ReloadOutlined, SettingOutlined, DownloadOutlined, MonitorOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { k8sApi } from '@/services/k8s'
import { logApi, type HighlightRule } from '@/services/logs'
import LogViewer from './LogViewer.vue'
import HighlightConfig from './components/HighlightConfig.vue'
import LogExport from './LogExport.vue'

interface Cluster {
  id: number
  name: string
}

interface Pod {
  name: string
  status: string
  containers: string[]
}

// Pod 颜色列表
const POD_COLORS = [
  '#409EFF', '#67C23A', '#E6A23C', '#F56C6C', '#909399',
  '#00CED1', '#FF69B4', '#9370DB', '#20B2AA', '#FF6347'
]

const clusters = ref<Cluster[]>([])
const namespaces = ref<string[]>([])
const pods = ref<Pod[]>([])
const selectedCluster = ref<number | null>(null)
const selectedNamespace = ref('')
const selectedPod = ref('')
const selectedPods = ref<string[]>([])
const selectedContainer = ref('')
const podFilter = ref('')
const settingsVisible = ref(false)
const exportVisible = ref(false)
const highlightRules = ref<HighlightRule[]>([])
const multiPodMode = ref(false)

// Pod 颜色映射
const podColorMap = computed(() => {
  const map: Record<string, string> = {}
  selectedPods.value.forEach((podName, index) => {
    map[podName] = POD_COLORS[index % POD_COLORS.length]
  })
  return map
})

const filteredPods = computed(() => {
  if (!podFilter.value) return pods.value
  return pods.value.filter(pod => 
    pod.name.toLowerCase().includes(podFilter.value.toLowerCase())
  )
})

const getPodStatusColor = (status: string) => {
  switch (status) {
    case 'Running': return 'success'
    case 'Pending': return 'warning'
    case 'Failed': return 'error'
    default: return 'default'
  }
}

const getPodStatusIconColor = (status: string) => {
  switch (status) {
    case 'Running': return '#67C23A'
    case 'Pending': return '#E6A23C'
    case 'Failed': return '#F56C6C'
    default: return '#909399'
  }
}

const getPodColor = (podName: string) => {
  return podColorMap.value[podName] || '#409EFF'
}

const openSettings = () => {
  settingsVisible.value = true
}

const openExport = () => {
  exportVisible.value = true
}

const loadClusters = async () => {
  try {
    const res = await k8sApi.getClusters()
    clusters.value = res.data || []
    if (clusters.value.length > 0 && !selectedCluster.value) {
      selectedCluster.value = clusters.value[0].id
      await loadNamespaces()
    }
  } catch (error) {
    console.error('加载集群列表失败', error)
    message.error('加载集群列表失败')
  }
}

const loadNamespaces = async () => {
  if (!selectedCluster.value) return
  try {
    const res = await k8sApi.getNamespaces(selectedCluster.value)
    namespaces.value = res.data || []
    if (namespaces.value.length > 0 && !selectedNamespace.value) {
      selectedNamespace.value = namespaces.value[0]
      await loadPods()
    }
  } catch (error) {
    console.error('加载命名空间失败', error)
    message.error('加载命名空间失败')
  }
}

const loadPods = async () => {
  if (!selectedCluster.value || !selectedNamespace.value) return
  try {
    const res = await k8sApi.getPods(selectedCluster.value, selectedNamespace.value)
    pods.value = (res.data || []).map((pod: any) => ({
      name: pod.name,
      status: pod.status,
      // containers 可能是对象数组，需要提取 name
      containers: (pod.containers || []).map((c: any) => typeof c === 'string' ? c : c.name)
    }))
  } catch (error) {
    console.error('加载 Pod 列表失败', error)
    message.error('加载 Pod 列表失败')
  }
}

const loadHighlightRules = async () => {
  try {
    const res = await logApi.getHighlightRules()
    highlightRules.value = res.data || []
  } catch (error) {
    console.error('加载染色规则失败', error)
  }
}

const onClusterChange = async () => {
  selectedNamespace.value = ''
  selectedPod.value = ''
  selectedPods.value = []
  pods.value = []
  await loadNamespaces()
}

const onNamespaceChange = async () => {
  selectedPod.value = ''
  selectedPods.value = []
  await loadPods()
}

const selectPod = (pod: Pod) => {
  selectedPod.value = pod.name
  // containers 可能是对象数组或字符串数组
  const firstContainer = pod.containers[0]
  selectedContainer.value = typeof firstContainer === 'string' ? firstContainer : (firstContainer?.name || '')
}

const togglePodSelection = (pod: Pod, checked?: boolean) => {
  const isSelected = selectedPods.value.includes(pod.name)
  const shouldSelect = checked !== undefined ? checked : !isSelected
  
  if (shouldSelect && !isSelected) {
    if (selectedPods.value.length >= 10) {
      message.warning('最多只能选择 10 个 Pod')
      return
    }
    selectedPods.value.push(pod.name)
  } else if (!shouldSelect && isSelected) {
    selectedPods.value = selectedPods.value.filter(p => p !== pod.name)
  }
}

const onContainerChange = (container: string) => {
  selectedContainer.value = container
}

const refreshPods = () => {
  loadPods()
}

// 切换模式时清空选择
watch(multiPodMode, () => {
  selectedPod.value = ''
  selectedPods.value = []
})

onMounted(() => {
  loadClusters()
  loadHighlightRules()
})
</script>

<style scoped>
.log-center-page {
  height: calc(100vh - 190px);
  display: flex;
  flex-direction: column;
  background: #f0f2f5;
}

.log-toolbar {
  padding: 16px;
  background: #ffffff;
  border-bottom: 1px solid #d9d9d9;
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.log-main-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.log-pod-list {
  width: 320px;
  min-width: 320px;
  border-right: 1px solid #d9d9d9;
  background: #ffffff;
  display: flex;
  flex-direction: column;
}

.log-pod-list-header {
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 500;
  font-size: 14px;
  color: rgba(0, 0, 0, 0.85);
  flex-shrink: 0;
}

.log-pod-list-scroll {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.log-pod-item {
  padding: 12px 16px;
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  border-bottom: 1px solid #f0f0f0;
  position: relative;
  transition: background-color 0.3s;
}

.log-pod-item:hover {
  background: #fafafa;
}

.log-pod-item.active {
  background: #e6f7ff;
}

.log-pod-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  color: rgba(0, 0, 0, 0.85);
}

.log-pod-color-indicator {
  width: 3px;
  height: 100%;
  position: absolute;
  left: 0;
  top: 0;
}

.log-viewer-wrapper {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

/* Responsive layout adjustments */
@media (max-width: 1200px) {
  .log-pod-list {
    width: 280px;
    min-width: 280px;
  }
  
  .log-toolbar {
    flex-wrap: wrap;
    gap: 8px;
  }
}
</style>
