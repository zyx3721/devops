<template>
  <div class="log-search">
    <!-- 搜索表单 -->
    <a-card class="search-card">
      <template #title>
        <div class="card-header">
          <span>历史日志查询</span>
          <a-button type="link" @click="showSavedQueries = true">
            <template #icon><StarOutlined /></template>
            快捷查询
          </a-button>
        </div>
      </template>

      <a-form :model="searchForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }" @finish="handleSearch">
        <a-row :gutter="20">
          <a-col :span="8">
            <a-form-item label="集群">
              <a-select v-model:value="searchForm.cluster_id" placeholder="选择集群" @change="onClusterChange" style="width: 100%">
                <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">{{ cluster.name }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="命名空间">
              <a-select v-model:value="searchForm.namespace" placeholder="选择命名空间" @change="onNamespaceChange" style="width: 100%">
                <a-select-option v-for="ns in namespaces" :key="ns" :value="ns">{{ ns }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="Pod">
              <a-select v-model:value="searchForm.pod_names" mode="multiple" placeholder="选择 Pod" :max-tag-count="1" style="width: 100%">
                <a-select-option v-for="pod in pods" :key="pod.name" :value="pod.name">{{ pod.name }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="20">
          <a-col :span="8">
            <a-form-item label="时间范围">
              <a-range-picker
                v-model:value="timeRange"
                show-time
                format="YYYY-MM-DD HH:mm:ss"
                style="width: 100%"
              />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="关键词">
              <a-input v-model:value="searchForm.keyword" placeholder="搜索关键词" allow-clear>
                <template #addonAfter>
                  <a-checkbox v-model:checked="searchForm.use_regex">正则</a-checkbox>
                </template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="日志级别">
              <a-select v-model:value="searchForm.level" placeholder="全部级别" allow-clear style="width: 100%">
                <a-select-option value="DEBUG">DEBUG</a-select-option>
                <a-select-option value="INFO">INFO</a-select-option>
                <a-select-option value="WARN">WARN</a-select-option>
                <a-select-option value="ERROR">ERROR</a-select-option>
                <a-select-option value="FATAL">FATAL</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-form-item :wrapper-col="{ span: 24, offset: 0 }" style="text-align: right">
          <a-button type="primary" @click="handleSearch" :loading="loading">
            <template #icon><SearchOutlined /></template>
            搜索
          </a-button>
          <a-button @click="resetForm" style="margin-left: 8px">重置</a-button>
          <a-button type="default" @click="saveQuery" :disabled="!canSaveQuery" style="margin-left: 8px; border-color: #52c41a; color: #52c41a">
            <template #icon><StarOutlined /></template>
            保存查询
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 查询结果 -->
    <a-card class="result-card" v-if="hasSearched">
      <template #title>
        <div class="card-header">
          <span>查询结果 ({{ total }} 条)</span>
          <div>
            <a-button size="small" @click="exportResults" :disabled="results.length === 0">
              <template #icon><DownloadOutlined /></template>
              导出
            </a-button>
          </div>
        </div>
      </template>

      <!-- 长时间查询提示 -->
      <a-alert v-if="queryTime > 5000" type="warning" show-icon style="margin-bottom: 15px">
        <template #message>
          查询耗时 {{ (queryTime / 1000).toFixed(1) }} 秒，建议缩小时间范围或添加更多过滤条件
        </template>
      </a-alert>

      <div class="log-results">
        <div v-for="log in results" :key="log.id" class="log-item" @click="showLogContext(log)">
          <span class="timestamp">{{ formatTimestamp(log.timestamp) }}</span>
          <span v-if="log.pod_name" class="pod-name">[{{ log.pod_name }}]</span>
          <span :class="['level', `level-${log.level?.toLowerCase()}`]">[{{ log.level }}]</span>
          <span class="content" v-html="highlightKeyword(log.content)"></span>
        </div>
        <a-empty v-if="results.length === 0 && !loading" description="暂无数据" />
      </div>

      <!-- 分页 -->
      <a-pagination
        v-if="total > 0"
        v-model:current="currentPage"
        v-model:pageSize="pageSize"
        :total="total"
        :page-size-options="['50', '100', '200', '500']"
        show-size-changer
        show-quick-jumper
        @change="handleSearch"
        @showSizeChange="handleSearch"
        style="margin-top: 15px; text-align: right"
      />
    </a-card>

    <!-- 快捷查询抽屉 -->
    <a-drawer v-model:open="showSavedQueries" title="快捷查询" width="400px">
      <div class="saved-queries">
        <div v-for="query in savedQueries" :key="query.id" class="query-item">
          <div class="query-info">
            <div class="query-name">{{ query.name }}</div>
            <div class="query-desc">{{ query.description || '无描述' }}</div>
            <div class="query-meta">
              使用 {{ query.use_count }} 次 | {{ formatDate(query.created_at) }}
            </div>
          </div>
          <div class="query-actions">
            <a-button type="primary" size="small" @click="loadSavedQuery(query)">使用</a-button>
            <a-button type="primary" danger size="small" @click="deleteSavedQuery(query.id)">删除</a-button>
          </div>
        </div>
        <a-empty v-if="savedQueries.length === 0" description="暂无保存的查询" />
      </div>
    </a-drawer>

    <!-- 保存查询对话框 -->
    <a-modal v-model:open="showSaveDialog" title="保存查询" width="400px" @ok="confirmSaveQuery" :confirmLoading="saving">
      <a-form :model="saveForm" :label-col="{ span: 4 }" :wrapper-col="{ span: 20 }">
        <a-form-item label="名称" required>
          <a-input v-model:value="saveForm.name" placeholder="输入查询名称" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="saveForm.description" placeholder="输入描述（可选）" :rows="3" />
        </a-form-item>
        <a-form-item label="共享">
          <a-switch v-model:checked="saveForm.is_shared" />
          <span style="margin-left: 10px; color: rgba(0, 0, 0, 0.45)">其他用户可见</span>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 日志上下文 -->
    <LogContext
      v-model:open="showContext"
      :cluster-id="searchForm.cluster_id"
      :namespace="searchForm.namespace"
      :pod-name="contextLog?.pod_name || ''"
      :log="contextLog"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { SearchOutlined, StarOutlined, DownloadOutlined } from '@ant-design/icons-vue'
import { message, Modal } from 'ant-design-vue'
import dayjs, { Dayjs } from 'dayjs'
import { k8sApi } from '@/services/k8s'
import { logApi, type LogEntry } from '@/services/logs'
import LogContext from './components/LogContext.vue'

interface Cluster {
  id: number
  name: string
}

interface Pod {
  name: string
  status: string
}

interface SavedQuery {
  id: number
  name: string
  description: string
  query_params: Record<string, any>
  is_shared: boolean
  use_count: number
  created_at: string
}

const clusters = ref<Cluster[]>([])
const namespaces = ref<string[]>([])
const pods = ref<Pod[]>([])
const results = ref<LogEntry[]>([])
const savedQueries = ref<SavedQuery[]>([])

const searchForm = ref({
  cluster_id: null as number | null,
  namespace: '',
  pod_names: [] as string[],
  keyword: '',
  use_regex: false,
  level: undefined as string | undefined
})

const timeRange = ref<[Dayjs, Dayjs] | null>(null)
const currentPage = ref(1)
const pageSize = ref(100)
const total = ref(0)
const loading = ref(false)
const hasSearched = ref(false)
const queryTime = ref(0)

const showSavedQueries = ref(false)
const showSaveDialog = ref(false)
const showContext = ref(false)
const contextLog = ref<LogEntry | null>(null)
const saving = ref(false)

const saveForm = ref({
  name: '',
  description: '',
  is_shared: false
})

const canSaveQuery = computed(() => {
  return searchForm.value.cluster_id && searchForm.value.namespace
})

const loadClusters = async () => {
  try {
    const res = await k8sApi.getClusters()
    clusters.value = res.data || []
  } catch (error) {
    message.error('加载集群列表失败')
  }
}

const onClusterChange = async () => {
  searchForm.value.namespace = ''
  searchForm.value.pod_names = []
  namespaces.value = []
  pods.value = []
  
  if (!searchForm.value.cluster_id) return
  
  try {
    const res = await k8sApi.getNamespaces(searchForm.value.cluster_id)
    namespaces.value = res.data || []
  } catch (error) {
    message.error('加载命名空间失败')
  }
}

const onNamespaceChange = async () => {
  searchForm.value.pod_names = []
  pods.value = []
  
  if (!searchForm.value.cluster_id || !searchForm.value.namespace) return
  
  try {
    const res = await k8sApi.getPods(searchForm.value.cluster_id, searchForm.value.namespace)
    pods.value = (res.data || []).map((pod: any) => ({
      name: pod.name,
      status: pod.status
    }))
  } catch (error) {
    message.error('加载 Pod 列表失败')
  }
}

const handleSearch = async () => {
  if (!searchForm.value.cluster_id || !searchForm.value.namespace) {
    message.warning('请选择集群和命名空间')
    return
  }

  loading.value = true
  hasSearched.value = true
  const startTime = Date.now()

  try {
    const params: any = {
      cluster_id: searchForm.value.cluster_id,
      namespace: searchForm.value.namespace,
      page: currentPage.value,
      page_size: pageSize.value
    }

    if (searchForm.value.pod_names.length > 0) {
      params.pod_names = searchForm.value.pod_names
    }
    if (searchForm.value.keyword) {
      if (searchForm.value.use_regex) {
        params.regex = searchForm.value.keyword
      } else {
        params.keyword = searchForm.value.keyword
      }
    }
    if (searchForm.value.level) {
      params.level = searchForm.value.level
    }
    if (timeRange.value) {
      params.start_time = timeRange.value[0].toISOString()
      params.end_time = timeRange.value[1].toISOString()
    }

    const res = await logApi.query(params)
    results.value = res.data?.items || []
    total.value = res.data?.total || 0
    queryTime.value = Date.now() - startTime
  } catch (error) {
    message.error('查询失败')
    console.error(error)
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  searchForm.value = {
    cluster_id: null,
    namespace: '',
    pod_names: [],
    keyword: '',
    use_regex: false,
    level: undefined
  }
  timeRange.value = null
  results.value = []
  total.value = 0
  hasSearched.value = false
}

const saveQuery = () => {
  saveForm.value = {
    name: '',
    description: '',
    is_shared: false
  }
  showSaveDialog.value = true
}

const confirmSaveQuery = async () => {
  if (!saveForm.value.name) {
    message.warning('请输入查询名称')
    return
  }

  saving.value = true
  try {
    const queryParams = {
      ...searchForm.value,
      time_range: timeRange.value ? [
        timeRange.value[0].toISOString(),
        timeRange.value[1].toISOString()
      ] : null
    }

    await logApi.createSavedQuery({
      name: saveForm.value.name,
      description: saveForm.value.description,
      query_params: queryParams,
      is_shared: saveForm.value.is_shared
    })

    message.success('保存成功')
    showSaveDialog.value = false
    loadSavedQueries()
  } catch (error) {
    message.error('保存失败')
  } finally {
    saving.value = false
  }
}

const loadSavedQueries = async () => {
  try {
    const res = await logApi.getSavedQueries()
    savedQueries.value = res.data || []
  } catch (error) {
    console.error('加载快捷查询失败', error)
  }
}

const loadSavedQuery = async (query: SavedQuery) => {
  const params = query.query_params
  searchForm.value = {
    cluster_id: params.cluster_id,
    namespace: params.namespace,
    pod_names: params.pod_names || [],
    keyword: params.keyword || '',
    use_regex: params.use_regex || false,
    level: params.level || undefined
  }

  if (params.time_range) {
    timeRange.value = [dayjs(params.time_range[0]), dayjs(params.time_range[1])]
  }

  // 加载命名空间和 Pod
  await onClusterChange()
  if (searchForm.value.namespace) {
    await onNamespaceChange()
  }

  // 记录使用
  logApi.useSavedQuery(query.id)
  
  showSavedQueries.value = false
  handleSearch()
}

const deleteSavedQuery = async (id: number) => {
  try {
    await Modal.confirm({
      title: '提示',
      content: '确定删除此快捷查询？',
      onOk: async () => {
        await logApi.deleteSavedQuery(id)
        message.success('删除成功')
        loadSavedQueries()
      }
    })
  } catch (error) {
    // Cancelled
  }
}

const showLogContext = (log: LogEntry) => {
  contextLog.value = log
  showContext.value = true
}

const exportResults = () => {
  // 导出为 JSON
  const data = JSON.stringify(results.value, null, 2)
  const blob = new Blob([data], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `logs-${new Date().toISOString().slice(0, 10)}.json`
  a.click()
  URL.revokeObjectURL(url)
}

const formatTimestamp = (ts: string) => {
  if (!ts) return ''
  try {
    return new Date(ts).toLocaleString('zh-CN')
  } catch {
    return ts
  }
}

const formatDate = (ts: string) => {
  if (!ts) return ''
  try {
    return new Date(ts).toLocaleDateString('zh-CN')
  } catch {
    return ts
  }
}

const highlightKeyword = (content: string) => {
  if (!searchForm.value.keyword || !content) return content
  
  const escaped = content
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
  
  try {
    const regex = new RegExp(`(${escapeRegex(searchForm.value.keyword)})`, 'gi')
    return escaped.replace(regex, '<mark>$1</mark>')
  } catch {
    return escaped
  }
}

const escapeRegex = (str: string) => {
  return str.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

onMounted(() => {
  loadClusters()
  loadSavedQueries()
})
</script>

<style scoped>
.log-search {
  padding: 20px;
}

.search-card {
  margin-bottom: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.result-card {
  min-height: 400px;
}

.log-results {
  max-height: 500px;
  overflow: auto;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  background: #1e1e1e;
  border-radius: 4px;
  padding: 10px;
}

.log-item {
  padding: 4px 10px;
  color: #d4d4d4;
  cursor: pointer;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-item:hover {
  background: rgba(255, 255, 255, 0.05);
}

.timestamp {
  color: #6a9955;
  margin-right: 10px;
}

.pod-name {
  color: #569cd6;
  margin-right: 5px;
}

.level {
  margin-right: 10px;
  font-weight: 500;
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

.level-debug {
  color: #808080;
}

.content {
  flex: 1;
}

:deep(mark) {
  background: #613214;
  color: #fff;
  padding: 0 2px;
  border-radius: 2px;
}

.saved-queries {
  display: flex;
  flex-direction: column;
  gap: 15px;
}

.query-item {
  padding: 15px;
  background: #fafafa;
  border-radius: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border: 1px solid #f0f0f0;
}

.query-info {
  flex: 1;
}

.query-name {
  font-weight: 500;
  margin-bottom: 5px;
}

.query-desc {
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
  margin-bottom: 5px;
}

.query-meta {
  color: rgba(0, 0, 0, 0.25);
  font-size: 12px;
}

.query-actions {
  display: flex;
  gap: 8px;
}
</style>
