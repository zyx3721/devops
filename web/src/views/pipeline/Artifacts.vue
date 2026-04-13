<template>
  <div class="artifacts">
    <!-- 制品库管理 -->
    <a-card title="制品库" :bordered="false" style="margin-bottom: 16px">
      <template #extra>
        <a-space>
          <a-button @click="refreshAllStatus">
            <template #icon><ReloadOutlined /></template>
            刷新状态
          </a-button>
          <a-button type="primary" @click="showRegistryModal()">
            <template #icon><PlusOutlined /></template>
            添加制品库
          </a-button>
        </a-space>
      </template>

      <a-table 
        :dataSource="registries" 
        :loading="registryLoading" 
        rowKey="id" 
        :pagination="false" 
        size="small"
      >
        <a-table-column title="名称" dataIndex="name" :width="150" />
        <a-table-column title="类型" dataIndex="type" :width="120">
          <template #default="{ record }">
            <a-tag :color="getRegistryTypeColor(record.type)">
              {{ getRegistryTypeLabel(record.type) }}
            </a-tag>
          </template>
        </a-table-column>
        <a-table-column title="地址" dataIndex="url" ellipsis />
        
        <!-- 增强的状态列 -->
        <a-table-column title="连接状态" :width="180">
          <template #default="{ record }">
            <a-space>
              <a-badge 
                :status="getStatusBadge(record.connection_status)" 
                :text="getStatusText(record.connection_status)" 
              />
              <a-tooltip v-if="record.last_error" :title="record.last_error">
                <ExclamationCircleOutlined style="color: #ff4d4f" />
              </a-tooltip>
              <a-button 
                type="link" 
                size="small" 
                @click="showConnectionHistory(record)"
              >
                历史
              </a-button>
            </a-space>
          </template>
        </a-table-column>
        
        <!-- 最后检查时间 -->
        <a-table-column title="最后检查" :width="160">
          <template #default="{ record }">
            <span v-if="record.last_check_at">
              {{ formatTime(record.last_check_at) }}
            </span>
            <span v-else class="text-gray">未检查</span>
          </template>
        </a-table-column>
        
        <a-table-column title="操作" :width="200">
          <template #default="{ record }">
            <a-space>
              <a-button 
                type="link" 
                size="small" 
                :loading="testingRegistry === record.id"
                @click="testRegistry(record)"
              >
                <template #icon><ApiOutlined /></template>
                测试
              </a-button>
              <a-button type="link" size="small" @click="editRegistry(record)">
                编辑
              </a-button>
              <a-popconfirm title="确定删除?" @confirm="deleteRegistry(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </a-table-column>
      </a-table>
    </a-card>

    <!-- 构建制品列表 -->
    <a-card title="构建制品" :bordered="false">
      <!-- 搜索栏 -->
      <a-row :gutter="16" style="margin-bottom: 16px">
        <a-col :span="6">
          <a-select v-model:value="searchForm.pipeline_id" placeholder="选择流水线" allowClear style="width: 100%" @change="loadData">
            <a-select-option v-for="p in pipelines" :key="p.id" :value="p.id">
              {{ p.name }}
            </a-select-option>
          </a-select>
        </a-col>
        <a-col :span="4">
          <a-select v-model:value="searchForm.type" placeholder="制品类型" allowClear style="width: 100%" @change="loadData">
            <a-select-option value="docker_image">Docker 镜像</a-select-option>
            <a-select-option value="helm_chart">Helm Chart</a-select-option>
            <a-select-option value="binary">二进制文件</a-select-option>
            <a-select-option value="archive">压缩包</a-select-option>
          </a-select>
        </a-col>
        <a-col :span="4">
          <a-button type="primary" @click="loadData">查询</a-button>
          <a-button style="margin-left: 8px" @click="resetSearch">重置</a-button>
        </a-col>
      </a-row>

      <!-- 制品列表 -->
      <a-table
        :columns="columns"
        :data-source="artifacts"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <a-tag :color="getTypeColor(record.type)">
              {{ getTypeLabel(record.type) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'path'">
            <a-tooltip :title="record.path">
              <span class="path-text">{{ record.path }}</span>
            </a-tooltip>
            <a-button type="link" size="small" @click="copyPath(record.path)">
              <CopyOutlined />
            </a-button>
          </template>
          <template v-else-if="column.key === 'git'">
            <div v-if="record.git_commit">
              <a-tag color="blue">{{ record.git_branch }}</a-tag>
              <span class="commit-sha">{{ record.git_commit?.substring(0, 8) }}</span>
            </div>
            <span v-else class="text-gray">-</span>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showDetail(record)">
                详情
              </a-button>
              <a-popconfirm title="确定删除该制品？" @confirm="deleteArtifact(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 添加/编辑制品库弹窗 -->
    <a-modal 
      v-model:open="registryModalVisible" 
      :title="editingRegistry ? '编辑制品库' : '添加制品库'" 
      @ok="saveRegistry" 
      :confirmLoading="registrySaving"
    >
      <a-form :model="registryForm" :label-col="{ span: 5 }" :wrapper-col="{ span: 19 }">
        <a-form-item label="名称" required>
          <a-input v-model:value="registryForm.name" placeholder="如: Harbor 生产环境" />
        </a-form-item>
        <a-form-item label="类型" required>
          <a-select v-model:value="registryForm.type" placeholder="选择制品库类型">
            <a-select-option value="harbor">Harbor</a-select-option>
            <a-select-option value="nexus">Nexus</a-select-option>
            <a-select-option value="dockerhub">Docker Hub</a-select-option>
            <a-select-option value="acr">阿里云 ACR</a-select-option>
            <a-select-option value="ecr">AWS ECR</a-select-option>
            <a-select-option value="gcr">Google GCR</a-select-option>
            <a-select-option value="custom">自定义</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="地址" required>
          <a-input v-model:value="registryForm.url" placeholder="https://harbor.example.com" />
        </a-form-item>
        <a-form-item label="用户名">
          <a-input v-model:value="registryForm.username" placeholder="用户名" />
        </a-form-item>
        <a-form-item label="密码">
          <a-input-password v-model:value="registryForm.password" placeholder="密码或 Token" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="registryForm.description" placeholder="描述信息" :rows="2" />
        </a-form-item>
        <a-form-item label="默认库">
          <a-switch v-model:checked="registryForm.is_default" />
          <span style="margin-left: 8px; color: #999">设为默认制品库</span>
        </a-form-item>
        <a-form-item label="启用监控">
          <a-switch v-model:checked="registryForm.enable_monitoring" />
          <span style="margin-left: 8px; color: #999">定期检查连接状态</span>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 连接历史记录抽屉 -->
    <a-drawer
      v-model:open="historyDrawerVisible"
      title="连接历史记录"
      width="600"
      :footer-style="{ textAlign: 'right' }"
    >
      <template #extra>
        <a-button @click="loadConnectionHistory">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </template>

      <a-spin :spinning="historyLoading">
        <a-timeline v-if="connectionHistory.length > 0">
          <a-timeline-item
            v-for="item in connectionHistory"
            :key="item.id"
            :color="item.status === 'success' ? 'green' : 'red'"
          >
            <template #dot>
              <CheckCircleOutlined v-if="item.status === 'success'" style="font-size: 16px" />
              <CloseCircleOutlined v-else style="font-size: 16px" />
            </template>
            <div class="history-item">
              <div class="history-header">
                <a-tag :color="item.status === 'success' ? 'success' : 'error'">
                  {{ item.status === 'success' ? '连接成功' : '连接失败' }}
                </a-tag>
                <span class="history-time">{{ formatTime(item.checked_at) }}</span>
              </div>
              <div v-if="item.response_time" class="history-detail">
                响应时间: {{ item.response_time }}ms
              </div>
              <div v-if="item.error_message" class="history-error">
                <a-alert :message="item.error_message" type="error" show-icon />
              </div>
            </div>
          </a-timeline-item>
        </a-timeline>
        <a-empty v-else description="暂无历史记录" />
      </a-spin>
    </a-drawer>

    <!-- 详情弹窗 -->
    <a-modal v-model:open="detailVisible" title="制品详情" :footer="null" width="600px">
      <a-descriptions :column="1" bordered v-if="currentArtifact">
        <a-descriptions-item label="名称">{{ currentArtifact.name }}</a-descriptions-item>
        <a-descriptions-item label="类型">
          <a-tag :color="getTypeColor(currentArtifact.type)">
            {{ getTypeLabel(currentArtifact.type) }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="路径/地址">
          <span>{{ currentArtifact.path }}</span>
          <a-button type="link" size="small" @click="copyPath(currentArtifact.path)">
            <CopyOutlined />
          </a-button>
        </a-descriptions-item>
        <a-descriptions-item label="大小">{{ currentArtifact.size_human }}</a-descriptions-item>
        <a-descriptions-item label="校验和" v-if="currentArtifact.checksum">
          {{ currentArtifact.checksum }}
        </a-descriptions-item>
        <a-descriptions-item label="流水线">{{ currentArtifact.pipeline_name }}</a-descriptions-item>
        <a-descriptions-item label="Git 分支" v-if="currentArtifact.git_branch">
          {{ currentArtifact.git_branch }}
        </a-descriptions-item>
        <a-descriptions-item label="Git 提交" v-if="currentArtifact.git_commit">
          {{ currentArtifact.git_commit }}
        </a-descriptions-item>
        <a-descriptions-item label="创建时间">
          {{ formatTime(currentArtifact.created_at) }}
        </a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { message, notification } from 'ant-design-vue'
import {
  CopyOutlined,
  PlusOutlined,
  ReloadOutlined,
  ApiOutlined,
  ExclamationCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons-vue'
import request from '@/services/api'
import { pipelineApi } from '@/services/pipeline'
import dayjs from 'dayjs'

interface Artifact {
  id: number
  pipeline_run_id: number
  pipeline_id?: number
  pipeline_name: string
  name: string
  type: string
  path: string
  size: number
  size_human: string
  checksum: string
  git_commit: string
  git_branch: string
  created_at: string
}

interface Pipeline {
  id: number
  name: string
}

interface Registry {
  id: number
  name: string
  type: string
  url: string
  username: string
  description: string
  is_default: boolean
  connection_status: 'connected' | 'disconnected' | 'checking' | 'unknown'
  last_check_at?: string
  last_error?: string
  enable_monitoring?: boolean
}

interface ConnectionHistory {
  id: number
  registry_id: number
  status: 'success' | 'failed'
  checked_at: string
  response_time?: number
  error_message?: string
}

const loading = ref(false)
const registryLoading = ref(false)
const registrySaving = ref(false)
const detailVisible = ref(false)
const registryModalVisible = ref(false)
const historyDrawerVisible = ref(false)
const historyLoading = ref(false)
const testingRegistry = ref<number | null>(null)

const artifacts = ref<Artifact[]>([])
const pipelines = ref<Pipeline[]>([])
const registries = ref<Registry[]>([])
const currentArtifact = ref<Artifact | null>(null)
const editingRegistry = ref<Registry | null>(null)
const currentRegistry = ref<Registry | null>(null)
const connectionHistory = ref<ConnectionHistory[]>([])

let statusCheckInterval: number | null = null

const registryForm = reactive({
  name: '',
  type: undefined as string | undefined,
  url: '',
  username: '',
  password: '',
  description: '',
  is_default: false,
  enable_monitoring: true,
})

const searchForm = reactive({
  pipeline_id: undefined as number | undefined,
  type: undefined as string | undefined,
  page: 1,
  page_size: 10
})

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`
})

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 200 },
  { title: '类型', dataIndex: 'type', key: 'type', width: 120 },
  { title: '路径/地址', dataIndex: 'path', key: 'path', ellipsis: true },
  { title: '大小', dataIndex: 'size_human', key: 'size', width: 100 },
  { title: 'Git 信息', key: 'git', width: 180 },
  { title: '流水线', dataIndex: 'pipeline_name', key: 'pipeline', width: 150 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180,
    customRender: ({ text }: { text: string }) => formatTime(text) },
  { title: '操作', key: 'action', width: 120, fixed: 'right' }
]

const getStatusBadge = (status: string) => {
  const statusMap: Record<string, any> = {
    connected: 'success',
    disconnected: 'error',
    checking: 'processing',
    unknown: 'default',
  }
  return statusMap[status] || 'default'
}

const getStatusText = (status: string) => {
  const textMap: Record<string, string> = {
    connected: '已连接',
    disconnected: '连接失败',
    checking: '检查中',
    unknown: '未知',
  }
  return textMap[status] || '未知'
}

const getTypeColor = (type: string) => {
  const colors: Record<string, string> = {
    docker_image: 'blue',
    helm_chart: 'green',
    binary: 'orange',
    archive: 'purple'
  }
  return colors[type] || 'default'
}

const getTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    docker_image: 'Docker 镜像',
    helm_chart: 'Helm Chart',
    binary: '二进制文件',
    archive: '压缩包'
  }
  return labels[type] || type
}

const getRegistryTypeColor = (type: string) => {
  const colors: Record<string, string> = {
    harbor: 'blue',
    nexus: 'green',
    dockerhub: 'cyan',
    acr: 'orange',
    ecr: 'gold',
    gcr: 'red',
    custom: 'default'
  }
  return colors[type] || 'default'
}

const getRegistryTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    harbor: 'Harbor',
    nexus: 'Nexus',
    dockerhub: 'Docker Hub',
    acr: '阿里云 ACR',
    ecr: 'AWS ECR',
    gcr: 'Google GCR',
    custom: '自定义'
  }
  return labels[type] || type
}

const formatTime = (time: string) => {
  return time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await request.get('/artifacts', {
      params: {
        pipeline_id: searchForm.pipeline_id,
        type: searchForm.type,
        page: pagination.current,
        page_size: pagination.pageSize
      }
    })
    if (res?.data) {
      artifacts.value = res.data.items || []
      pagination.total = res.data.total || 0
    }
  } catch (error) {
    console.error('加载制品列表失败:', error)
  } finally {
    loading.value = false
  }
}

const loadPipelines = async () => {
  try {
    const res = await pipelineApi.list({ page_size: 100 })
    pipelines.value = res?.data?.items || []
  } catch (error) {
    console.error('加载流水线列表失败:', error)
  }
}

const loadRegistries = async () => {
  registryLoading.value = true
  try {
    const res = await request.get('/artifact/repositories')
    registries.value = res?.data?.items || []
  } catch (error) {
    console.error('加载制品库列表失败:', error)
  } finally {
    registryLoading.value = false
  }
}

const showRegistryModal = (registry?: Registry) => {
  if (registry) {
    editingRegistry.value = registry
    Object.assign(registryForm, {
      name: registry.name,
      type: registry.type,
      url: registry.url,
      username: registry.username,
      password: '',
      description: registry.description,
      is_default: registry.is_default,
      enable_monitoring: registry.enable_monitoring ?? true,
    })
  } else {
    editingRegistry.value = null
    Object.assign(registryForm, {
      name: '',
      type: undefined,
      url: '',
      username: '',
      password: '',
      description: '',
      is_default: false,
      enable_monitoring: true,
    })
  }
  registryModalVisible.value = true
}

const editRegistry = (registry: Registry) => {
  showRegistryModal(registry)
}

const saveRegistry = async () => {
  if (!registryForm.name || !registryForm.type || !registryForm.url) {
    message.warning('请填写完整信息')
    return
  }
  registrySaving.value = true
  try {
    if (editingRegistry.value) {
      await request.put(`/artifact/repositories/${editingRegistry.value.id}`, registryForm)
      message.success('更新成功')
    } else {
      await request.post('/artifact/repositories', registryForm)
      message.success('添加成功')
    }
    registryModalVisible.value = false
    loadRegistries()
  } catch (error: any) {
    message.error(error?.message || '保存失败')
  } finally {
    registrySaving.value = false
  }
}

const deleteRegistry = async (id: number) => {
  try {
    await request.delete(`/artifact/repositories/${id}`)
    message.success('删除成功')
    loadRegistries()
  } catch (error: any) {
    message.error(error?.message || '删除失败')
  }
}

const testRegistry = async (registry: Registry) => {
  testingRegistry.value = registry.id
  try {
    const res = await request.post(`/artifact/repositories/${registry.id}/test`)
    if (res?.data?.connected) {
      message.success(`连接成功 (响应时间: ${res.data.response_time}ms)`)
      // 发送成功通知
      notification.success({
        message: '连接测试成功',
        description: `制品库 "${registry.name}" 连接正常`,
        duration: 3,
      })
    } else {
      message.error('连接失败: ' + (res?.data?.error || '未知错误'))
      // 发送失败通知
      notification.error({
        message: '连接测试失败',
        description: `制品库 "${registry.name}" 连接失败: ${res?.data?.error || '未知错误'}`,
        duration: 5,
      })
    }
    loadRegistries()
  } catch (error: any) {
    message.error('测试失败: ' + (error?.message || '未知错误'))
    notification.error({
      message: '连接测试失败',
      description: `制品库 "${registry.name}" 测试失败: ${error?.message || '未知错误'}`,
      duration: 5,
    })
  } finally {
    testingRegistry.value = null
  }
}

const refreshAllStatus = async () => {
  registryLoading.value = true
  try {
    await request.post('/artifact/repositories/refresh-status')
    message.success('状态刷新成功')
    loadRegistries()
  } catch (error: any) {
    message.error('刷新失败: ' + (error?.message || '未知错误'))
  } finally {
    registryLoading.value = false
  }
}

const showConnectionHistory = (registry: Registry) => {
  currentRegistry.value = registry
  historyDrawerVisible.value = true
  loadConnectionHistory()
}

const loadConnectionHistory = async () => {
  if (!currentRegistry.value) return
  
  historyLoading.value = true
  try {
    const res = await request.get(`/artifact/repositories/${currentRegistry.value.id}/history`, {
      params: {
        page: 1,
        page_size: 50,
      },
    })
    connectionHistory.value = res?.data?.items || []
  } catch (error) {
    console.error('加载连接历史失败:', error)
    message.error('加载连接历史失败')
  } finally {
    historyLoading.value = false
  }
}

const resetSearch = () => {
  searchForm.pipeline_id = undefined
  searchForm.type = undefined
  pagination.current = 1
  loadData()
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  loadData()
}

const showDetail = (record: Artifact) => {
  currentArtifact.value = record
  detailVisible.value = true
}

const deleteArtifact = async (id: number) => {
  try {
    await request.delete(`/artifacts/${id}`)
    message.success('删除成功')
    loadData()
  } catch (error: any) {
    message.error(error?.message || '删除失败')
  }
}

const copyPath = (path: string) => {
  navigator.clipboard.writeText(path)
  message.success('已复制到剪贴板')
}

// 定期检查连接状态
const startStatusCheck = () => {
  // 每 5 分钟检查一次
  statusCheckInterval = window.setInterval(() => {
    loadRegistries()
  }, 5 * 60 * 1000)
}

const stopStatusCheck = () => {
  if (statusCheckInterval) {
    clearInterval(statusCheckInterval)
    statusCheckInterval = null
  }
}

onMounted(() => {
  loadData()
  loadPipelines()
  loadRegistries()
  startStatusCheck()
})

onUnmounted(() => {
  stopStatusCheck()
})
</script>

<style scoped>
.artifacts {
  padding: 0;
}

.path-text {
  max-width: 300px;
  display: inline-block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}

.commit-sha {
  font-family: monospace;
  color: #666;
  margin-left: 8px;
}

.text-gray {
  color: #999;
}

.history-item {
  padding: 8px 0;
}

.history-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.history-time {
  color: #999;
  font-size: 12px;
}

.history-detail {
  color: #666;
  font-size: 12px;
  margin-bottom: 4px;
}

.history-error {
  margin-top: 8px;
}
</style>
