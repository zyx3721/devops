<template>
  <div class="log-compare">
    <!-- 对比配置 -->
    <a-card class="config-card">
      <a-form :model="compareForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
        <a-row :gutter="20">
          <a-col :span="8">
            <a-form-item label="集群">
              <a-select v-model:value="compareForm.cluster_id" placeholder="选择集群" @change="onClusterChange" style="width: 100%">
                <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">{{ cluster.name }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="命名空间">
              <a-select v-model:value="compareForm.namespace" placeholder="选择命名空间" @change="onNamespaceChange" style="width: 100%">
                <a-select-option v-for="ns in namespaces" :key="ns" :value="ns">{{ ns }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="对比类型">
              <a-radio-group v-model:value="compareForm.compare_type">
                <a-radio value="pod">Pod 对比</a-radio>
                <a-radio value="time_range">时间段对比</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-col>
        </a-row>

        <!-- Pod 对比 -->
        <template v-if="compareForm.compare_type === 'pod'">
          <a-row :gutter="20">
            <a-col :span="12">
              <a-form-item label="左侧 Pod">
                <a-select v-model:value="compareForm.left_pod_name" placeholder="选择 Pod" style="width: 100%">
                  <a-select-option v-for="pod in pods" :key="pod.name" :value="pod.name">{{ pod.name }}</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="右侧 Pod">
                <a-select v-model:value="compareForm.right_pod_name" placeholder="选择 Pod" style="width: 100%">
                  <a-select-option v-for="pod in pods" :key="pod.name" :value="pod.name">{{ pod.name }}</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
        </template>

        <!-- 时间段对比 -->
        <template v-else>
          <a-row :gutter="20">
            <a-col :span="8">
              <a-form-item label="Pod">
                <a-select v-model:value="compareForm.left_pod_name" placeholder="选择 Pod" style="width: 100%">
                  <a-select-option v-for="pod in pods" :key="pod.name" :value="pod.name">{{ pod.name }}</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="左侧时间">
                <a-range-picker
                  v-model:value="leftTimeRange"
                  show-time
                  format="YYYY-MM-DD HH:mm:ss"
                  style="width: 100%"
                />
              </a-form-item>
            </a-col>
            <a-col :span="8">
              <a-form-item label="右侧时间">
                <a-range-picker
                  v-model:value="rightTimeRange"
                  show-time
                  format="YYYY-MM-DD HH:mm:ss"
                  style="width: 100%"
                />
              </a-form-item>
            </a-col>
          </a-row>
        </template>

        <a-form-item :wrapper-col="{ span: 24, offset: 0 }">
          <a-button type="primary" @click="doCompare" :loading="loading">开始对比</a-button>
          <a-button @click="exportResult" :disabled="!result" style="margin-left: 8px">导出结果</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 对比统计 -->
    <a-row :gutter="20" style="margin-top: 20px" v-if="result">
      <a-col :span="6">
        <a-card :body-style="{ padding: '12px 24px' }">
          <a-statistic title="左侧日志数" :value="result.total_left" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :body-style="{ padding: '12px 24px' }">
          <a-statistic title="右侧日志数" :value="result.total_right" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :body-style="{ padding: '12px 24px' }">
          <a-statistic title="新增" :value="result.added_count" :value-style="{ color: '#67C23A' }" />
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card :body-style="{ padding: '12px 24px' }">
          <a-statistic title="删除" :value="result.removed_count" :value-style="{ color: '#F56C6C' }" />
        </a-card>
      </a-col>
    </a-row>

    <!-- 对比视图 -->
    <div class="compare-view" v-if="result">
      <div class="compare-panel left-panel">
        <div class="panel-header">
          <span>{{ compareForm.compare_type === 'pod' ? compareForm.left_pod_name : '左侧时间段' }}</span>
          <a-tag size="small">{{ result.total_left }} 行</a-tag>
        </div>
        <div class="panel-content" ref="leftPanelRef" @scroll="syncScroll('left')">
          <div 
            v-for="line in result.left_lines" 
            :key="line.line_number"
            :class="['log-line', `diff-${line.diff_type}`]"
          >
            <span class="line-number">{{ line.line_number }}</span>
            <span class="timestamp">{{ formatTimestamp(line.timestamp) }}</span>
            <span :class="['level', `level-${line.level?.toLowerCase()}`]">[{{ line.level }}]</span>
            <span class="content">{{ line.content }}</span>
          </div>
        </div>
      </div>

      <div class="compare-panel right-panel">
        <div class="panel-header">
          <span>{{ compareForm.compare_type === 'pod' ? compareForm.right_pod_name : '右侧时间段' }}</span>
          <a-tag size="small">{{ result.total_right }} 行</a-tag>
        </div>
        <div class="panel-content" ref="rightPanelRef" @scroll="syncScroll('right')">
          <div 
            v-for="line in result.right_lines" 
            :key="line.line_number"
            :class="['log-line', `diff-${line.diff_type}`]"
          >
            <span class="line-number">{{ line.line_number }}</span>
            <span class="timestamp">{{ formatTimestamp(line.timestamp) }}</span>
            <span :class="['level', `level-${line.level?.toLowerCase()}`]">[{{ line.level }}]</span>
            <span class="content">{{ line.content }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import dayjs, { Dayjs } from 'dayjs'
import { k8sApi } from '@/services/k8s'
import { logApi } from '@/services/logs'

interface Cluster {
  id: number
  name: string
}

interface Pod {
  name: string
  status: string
}

interface CompareLine {
  line_number: number
  timestamp: string
  content: string
  level: string
  diff_type: string
  pod_name: string
}

interface CompareResult {
  left_lines: CompareLine[]
  right_lines: CompareLine[]
  total_left: number
  total_right: number
  added_count: number
  removed_count: number
  same_count: number
}

const clusters = ref<Cluster[]>([])
const namespaces = ref<string[]>([])
const pods = ref<Pod[]>([])
const result = ref<CompareResult | null>(null)
const loading = ref(false)

const compareForm = reactive({
  cluster_id: null as number | null,
  namespace: '',
  compare_type: 'pod',
  left_pod_name: '',
  right_pod_name: '',
  container: ''
})

const leftTimeRange = ref<[Dayjs, Dayjs] | null>(null)
const rightTimeRange = ref<[Dayjs, Dayjs] | null>(null)

const leftPanelRef = ref<HTMLElement | null>(null)
const rightPanelRef = ref<HTMLElement | null>(null)
let isSyncing = false

const loadClusters = async () => {
  try {
    const res = await k8sApi.getClusters()
    clusters.value = res.data || []
  } catch (error) {
    message.error('加载集群列表失败')
  }
}

const onClusterChange = async () => {
  compareForm.namespace = ''
  compareForm.left_pod_name = ''
  compareForm.right_pod_name = ''
  namespaces.value = []
  pods.value = []
  
  if (!compareForm.cluster_id) return
  
  try {
    const res = await k8sApi.getNamespaces(compareForm.cluster_id)
    namespaces.value = res.data || []
  } catch (error) {
    message.error('加载命名空间失败')
  }
}

const onNamespaceChange = async () => {
  compareForm.left_pod_name = ''
  compareForm.right_pod_name = ''
  pods.value = []
  
  if (!compareForm.cluster_id || !compareForm.namespace) return
  
  try {
    const res = await k8sApi.getPods(compareForm.cluster_id, compareForm.namespace)
    pods.value = (res.data || []).map((pod: any) => ({
      name: pod.name,
      status: pod.status
    }))
  } catch (error) {
    message.error('加载 Pod 列表失败')
  }
}

const doCompare = async () => {
  if (!compareForm.cluster_id || !compareForm.namespace) {
    message.warning('请选择集群和命名空间')
    return
  }

  if (compareForm.compare_type === 'pod') {
    if (!compareForm.left_pod_name || !compareForm.right_pod_name) {
      message.warning('请选择要对比的 Pod')
      return
    }
  } else {
    if (!compareForm.left_pod_name || !leftTimeRange.value || !rightTimeRange.value) {
      message.warning('请选择 Pod 和时间范围')
      return
    }
  }

  loading.value = true
  try {
    const params: any = {
      cluster_id: compareForm.cluster_id,
      namespace: compareForm.namespace,
      compare_type: compareForm.compare_type,
      left_pod_name: compareForm.left_pod_name,
      container: compareForm.container
    }

    if (compareForm.compare_type === 'pod') {
      params.right_pod_name = compareForm.right_pod_name
    } else {
      params.left_start_time = leftTimeRange.value![0].toISOString()
      params.left_end_time = leftTimeRange.value![1].toISOString()
      params.right_start_time = rightTimeRange.value![0].toISOString()
      params.right_end_time = rightTimeRange.value![1].toISOString()
    }

    const res = await logApi.compareLogs(params)
    result.value = res.data
  } catch (error) {
    message.error('对比失败')
  } finally {
    loading.value = false
  }
}

const syncScroll = (source: 'left' | 'right') => {
  if (isSyncing) return
  isSyncing = true

  const sourcePanel = source === 'left' ? leftPanelRef.value : rightPanelRef.value
  const targetPanel = source === 'left' ? rightPanelRef.value : leftPanelRef.value

  if (sourcePanel && targetPanel) {
    targetPanel.scrollTop = sourcePanel.scrollTop
  }

  setTimeout(() => {
    isSyncing = false
  }, 50)
}

const formatTimestamp = (ts: string) => {
  if (!ts) return ''
  try {
    return new Date(ts).toLocaleTimeString('zh-CN', { 
      hour12: false,
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit'
    })
  } catch {
    return ts.substring(11, 19)
  }
}

const exportResult = () => {
  if (!result.value) return

  const data = JSON.stringify(result.value, null, 2)
  const blob = new Blob([data], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `log-compare-${new Date().toISOString().slice(0, 10)}.json`
  a.click()
  URL.revokeObjectURL(url)
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.log-compare {
  padding: 20px;
}

.config-card {
  margin-bottom: 20px;
}

.compare-view {
  display: flex;
  gap: 20px;
  margin-top: 20px;
}

.compare-panel {
  flex: 1;
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  overflow: hidden;
}

.panel-header {
  padding: 10px 15px;
  background: #fafafa;
  border-bottom: 1px solid #d9d9d9;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 500;
}

.panel-content {
  height: 500px;
  overflow: auto;
  background: #1e1e1e;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
}

.log-line {
  display: flex;
  padding: 3px 10px;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-all;
  border-left: 3px solid transparent;
}

.log-line.diff-added {
  background: rgba(103, 194, 58, 0.15);
  border-left-color: #67C23A;
}

.log-line.diff-removed {
  background: rgba(245, 108, 108, 0.15);
  border-left-color: #F56C6C;
}

.line-number {
  color: #858585;
  margin-right: 10px;
  min-width: 30px;
  text-align: right;
  flex-shrink: 0;
}

.timestamp {
  color: #6a9955;
  margin-right: 8px;
  flex-shrink: 0;
}

.level {
  margin-right: 8px;
  flex-shrink: 0;
}

.level-error, .level-fatal {
  color: #f14c4c;
}

.level-warn, .level-warning {
  color: #cca700;
}

.level-info {
  color: #3794ff;
}

.content {
  flex: 1;
}
</style>
