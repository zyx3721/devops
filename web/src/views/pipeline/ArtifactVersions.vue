<template>
  <div class="artifact-versions">
    <!-- 面包屑导航 -->
    <a-breadcrumb style="margin-bottom: 16px">
      <a-breadcrumb-item>
        <router-link to="/pipeline/artifacts">制品管理</router-link>
      </a-breadcrumb-item>
      <a-breadcrumb-item>{{ artifactInfo.name || '制品详情' }}</a-breadcrumb-item>
      <a-breadcrumb-item>版本管理</a-breadcrumb-item>
    </a-breadcrumb>

    <!-- 制品基本信息 -->
    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-descriptions :column="4" v-if="artifactInfo.id">
        <a-descriptions-item label="制品名称">
          {{ artifactInfo.name }}
        </a-descriptions-item>
        <a-descriptions-item label="制品类型">
          <a-tag>{{ artifactInfo.type }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="仓库">
          {{ artifactInfo.repository }}
        </a-descriptions-item>
        <a-descriptions-item label="版本总数">
          <a-badge :count="versions.length" :number-style="{ backgroundColor: '#1890ff' }" />
        </a-descriptions-item>
      </a-descriptions>
    </a-card>

    <a-row :gutter="16">
      <!-- 左侧：版本列表（时间线样式） -->
      <a-col :xs="24" :md="10" :lg="8">
        <a-card title="版本列表" :bordered="false">
          <a-input-search
            v-model:value="searchKeyword"
            placeholder="搜索版本号或标签"
            style="margin-bottom: 16px"
            @search="filterVersions"
          />
          
          <a-timeline v-if="filteredVersions.length > 0">
            <a-timeline-item
              v-for="version in filteredVersions"
              :key="version.id"
              :color="getVersionColor(version)"
            >
              <div
                class="version-item"
                :class="{ active: currentVersion?.id === version.id }"
                @click="selectVersion(version)"
              >
                <div class="version-header">
                  <span class="version-number">{{ version.version }}</span>
                  <a-tag
                    v-if="version.scan_status"
                    :color="getScanStatusColor(version.scan_status)"
                    size="small"
                  >
                    {{ getScanStatusText(version.scan_status) }}
                  </a-tag>
                </div>
                <div class="version-tags" v-if="version.tags && version.tags.length > 0">
                  <a-tag v-for="tag in version.tags" :key="tag" size="small">
                    {{ tag }}
                  </a-tag>
                </div>
                <div class="version-meta">
                  <span class="version-size">{{ version.size_human }}</span>
                  <span class="version-time">{{ formatTime(version.created_at) }}</span>
                </div>
              </div>
            </a-timeline-item>
          </a-timeline>

          <a-empty v-else description="暂无版本" />
        </a-card>
      </a-col>

      <!-- 右侧：版本详情 -->
      <a-col :xs="24" :md="14" :lg="16">
        <a-card title="版本详情" :bordered="false" v-if="currentVersion">
          <template #extra>
            <a-space>
              <a-button
                type="primary"
                @click="showCompareModal"
                :disabled="selectedVersions.length !== 2"
              >
                <template #icon><DiffOutlined /></template>
                对比版本 ({{ selectedVersions.length }}/2)
              </a-button>
              <a-popconfirm
                title="确定删除该版本？"
                ok-text="确定"
                cancel-text="取消"
                @confirm="deleteVersion(currentVersion.id)"
              >
                <a-button danger>
                  <template #icon><DeleteOutlined /></template>
                  删除
                </a-button>
              </a-popconfirm>
            </a-space>
          </template>

          <a-descriptions :column="2" bordered>
            <a-descriptions-item label="版本号" :span="2">
              {{ currentVersion.version }}
            </a-descriptions-item>
            <a-descriptions-item label="标签" :span="2">
              <a-tag v-for="tag in currentVersion.tags" :key="tag">
                {{ tag }}
              </a-tag>
              <span v-if="!currentVersion.tags || currentVersion.tags.length === 0" class="text-gray">
                无标签
              </span>
            </a-descriptions-item>
            <a-descriptions-item label="大小">
              {{ currentVersion.size_human }}
            </a-descriptions-item>
            <a-descriptions-item label="校验和">
              <a-typography-text copyable>{{ currentVersion.checksum }}</a-typography-text>
            </a-descriptions-item>
            <a-descriptions-item label="扫描状态">
              <a-tag :color="getScanStatusColor(currentVersion.scan_status)">
                {{ getScanStatusText(currentVersion.scan_status) }}
              </a-tag>
              <a-button
                v-if="currentVersion.scan_status === 'passed' || currentVersion.scan_status === 'failed'"
                type="link"
                size="small"
                @click="viewScanResults"
              >
                查看报告
              </a-button>
            </a-descriptions-item>
            <a-descriptions-item label="扫描结果" v-if="currentVersion.scan_result">
              <a-space>
                <a-tag color="red">严重: {{ currentVersion.scan_result.critical || 0 }}</a-tag>
                <a-tag color="orange">高危: {{ currentVersion.scan_result.high || 0 }}</a-tag>
                <a-tag color="gold">中危: {{ currentVersion.scan_result.medium || 0 }}</a-tag>
                <a-tag color="green">低危: {{ currentVersion.scan_result.low || 0 }}</a-tag>
              </a-space>
            </a-descriptions-item>
            <a-descriptions-item label="创建时间" :span="2">
              {{ formatTime(currentVersion.created_at) }}
            </a-descriptions-item>
          </a-descriptions>

          <!-- 元数据 -->
          <a-divider>元数据</a-divider>
          <a-descriptions :column="1" bordered v-if="currentVersion.metadata">
            <a-descriptions-item
              v-for="(value, key) in currentVersion.metadata"
              :key="key"
              :label="key"
            >
              {{ value }}
            </a-descriptions-item>
          </a-descriptions>
          <a-empty v-else description="暂无元数据" :image="simpleImage" />

          <!-- 选择对比 -->
          <a-divider>版本对比</a-divider>
          <a-checkbox
            :checked="isVersionSelected(currentVersion.id)"
            @change="toggleVersionSelection(currentVersion.id)"
          >
            选择此版本用于对比
          </a-checkbox>
          <span class="text-gray" style="margin-left: 12px">
            (选择两个版本后点击"对比版本"按钮)
          </span>
        </a-card>

        <a-empty v-else description="请选择一个版本查看详情" />
      </a-col>
    </a-row>

    <!-- 版本对比弹窗 -->
    <a-modal
      v-model:open="compareModalVisible"
      title="版本对比"
      width="800px"
      :footer="null"
    >
      <a-spin :spinning="comparing">
        <div v-if="compareResult">
          <a-row :gutter="16" style="margin-bottom: 16px">
            <a-col :span="12">
              <a-card size="small" title="版本 1">
                <p><strong>版本号:</strong> {{ compareResult.version1.version }}</p>
                <p><strong>大小:</strong> {{ compareResult.version1.size_human }}</p>
                <p><strong>创建时间:</strong> {{ formatTime(compareResult.version1.created_at) }}</p>
              </a-card>
            </a-col>
            <a-col :span="12">
              <a-card size="small" title="版本 2">
                <p><strong>版本号:</strong> {{ compareResult.version2.version }}</p>
                <p><strong>大小:</strong> {{ compareResult.version2.size_human }}</p>
                <p><strong>创建时间:</strong> {{ formatTime(compareResult.version2.created_at) }}</p>
              </a-card>
            </a-col>
          </a-row>

          <a-divider>差异信息</a-divider>
          
          <a-descriptions :column="1" bordered>
            <a-descriptions-item label="大小变化">
              <span :class="compareResult.size_diff > 0 ? 'text-red' : 'text-green'">
                {{ compareResult.size_diff > 0 ? '+' : '' }}{{ formatSize(compareResult.size_diff) }}
              </span>
            </a-descriptions-item>
            <a-descriptions-item label="标签变化">
              <div v-if="compareResult.tags_added && compareResult.tags_added.length > 0">
                <span class="text-green">新增: </span>
                <a-tag v-for="tag in compareResult.tags_added" :key="tag" color="green">
                  {{ tag }}
                </a-tag>
              </div>
              <div v-if="compareResult.tags_removed && compareResult.tags_removed.length > 0">
                <span class="text-red">移除: </span>
                <a-tag v-for="tag in compareResult.tags_removed" :key="tag" color="red">
                  {{ tag }}
                </a-tag>
              </div>
              <span v-if="!compareResult.tags_added?.length && !compareResult.tags_removed?.length" class="text-gray">
                无变化
              </span>
            </a-descriptions-item>
            <a-descriptions-item label="元数据变化">
              <pre v-if="compareResult.metadata_diff" class="diff-content">{{ compareResult.metadata_diff }}</pre>
              <span v-else class="text-gray">无变化</span>
            </a-descriptions-item>
          </a-descriptions>
        </div>
      </a-spin>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Empty } from 'ant-design-vue'
import {
  DiffOutlined,
  DeleteOutlined,
} from '@ant-design/icons-vue'
import request from '@/services/api'
import dayjs from 'dayjs'

const route = useRoute()
const router = useRouter()
const artifactId = computed(() => Number(route.params.artifactId))

const simpleImage = Empty.PRESENTED_IMAGE_SIMPLE

interface ArtifactVersion {
  id: number
  artifact_id: number
  version: string
  tags: string[]
  size: number
  size_human: string
  checksum: string
  scan_status: 'pending' | 'scanning' | 'passed' | 'failed' | 'warning'
  scan_result?: {
    critical: number
    high: number
    medium: number
    low: number
  }
  metadata: Record<string, any>
  created_at: string
}

const loading = ref(false)
const comparing = ref(false)
const compareModalVisible = ref(false)
const searchKeyword = ref('')
const versions = ref<ArtifactVersion[]>([])
const currentVersion = ref<ArtifactVersion | null>(null)
const selectedVersions = ref<number[]>([])
const compareResult = ref<any>(null)

const artifactInfo = reactive({
  id: 0,
  name: '',
  type: '',
  repository: '',
})

const filteredVersions = computed(() => {
  if (!searchKeyword.value) return versions.value
  
  const keyword = searchKeyword.value.toLowerCase()
  return versions.value.filter(v =>
    v.version.toLowerCase().includes(keyword) ||
    v.tags?.some(tag => tag.toLowerCase().includes(keyword))
  )
})

const formatTime = (time: string) => {
  return time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'
}

const formatSize = (bytes: number): string => {
  const abs = Math.abs(bytes)
  if (abs < 1024) return `${bytes} B`
  if (abs < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
  if (abs < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

const getVersionColor = (version: ArtifactVersion) => {
  switch (version.scan_status) {
    case 'passed':
      return 'green'
    case 'failed':
      return 'red'
    case 'warning':
      return 'orange'
    case 'scanning':
      return 'blue'
    default:
      return 'gray'
  }
}

const getScanStatusColor = (status: string) => {
  switch (status) {
    case 'passed':
      return 'success'
    case 'failed':
      return 'error'
    case 'warning':
      return 'warning'
    case 'scanning':
      return 'processing'
    default:
      return 'default'
  }
}

const getScanStatusText = (status: string) => {
  switch (status) {
    case 'passed':
      return '通过'
    case 'failed':
      return '失败'
    case 'warning':
      return '警告'
    case 'scanning':
      return '扫描中'
    case 'pending':
      return '待扫描'
    default:
      return '未知'
  }
}

const loadArtifactInfo = async () => {
  try {
    const res = await request.get(`/artifacts/${artifactId.value}`)
    if (res?.data) {
      Object.assign(artifactInfo, res.data)
    }
  } catch (error) {
    console.error('加载制品信息失败:', error)
  }
}

const loadVersions = async () => {
  loading.value = true
  try {
    const res = await request.get(`/artifacts/${artifactId.value}/versions`)
    versions.value = res?.data || []
    
    // 默认选择第一个版本
    if (versions.value.length > 0 && !currentVersion.value) {
      currentVersion.value = versions.value[0]
    }
  } catch (error) {
    console.error('加载版本列表失败:', error)
  } finally {
    loading.value = false
  }
}

const selectVersion = (version: ArtifactVersion) => {
  currentVersion.value = version
}

const filterVersions = () => {
  // 搜索已通过 computed 实现
}

const isVersionSelected = (versionId: number) => {
  return selectedVersions.value.includes(versionId)
}

const toggleVersionSelection = (versionId: number) => {
  const index = selectedVersions.value.indexOf(versionId)
  if (index > -1) {
    selectedVersions.value.splice(index, 1)
  } else {
    if (selectedVersions.value.length >= 2) {
      message.warning('最多只能选择两个版本进行对比')
      return
    }
    selectedVersions.value.push(versionId)
  }
}

const showCompareModal = async () => {
  if (selectedVersions.value.length !== 2) {
    message.warning('请选择两个版本进行对比')
    return
  }

  compareModalVisible.value = true
  comparing.value = true
  
  try {
    const res = await request.get('/artifact-versions/compare', {
      params: {
        v1: selectedVersions.value[0],
        v2: selectedVersions.value[1],
      },
    })
    compareResult.value = res?.data
  } catch (error: any) {
    message.error(error?.message || '对比失败')
  } finally {
    comparing.value = false
  }
}

const deleteVersion = async (versionId: number) => {
  try {
    await request.delete(`/artifact-versions/${versionId}`)
    message.success('删除成功')
    
    // 重新加载列表
    await loadVersions()
    
    // 如果删除的是当前选中的版本，清空选中
    if (currentVersion.value?.id === versionId) {
      currentVersion.value = versions.value.length > 0 ? versions.value[0] : null
    }
  } catch (error: any) {
    message.error(error?.message || '删除失败')
  }
}

const viewScanResults = () => {
  if (currentVersion.value) {
    router.push(`/pipeline/artifacts/${currentVersion.value.id}/scan`)
  }
}

onMounted(() => {
  loadArtifactInfo()
  loadVersions()
})
</script>

<style scoped>
.artifact-versions {
  padding: 0;
}

.version-item {
  padding: 8px 12px;
  border-radius: 4px;
  cursor: pointer;
  transition: all 0.3s;
  border: 1px solid transparent;
}

.version-item:hover {
  background: #f5f5f5;
}

.version-item.active {
  background: #e6f7ff;
  border-color: #1890ff;
}

.version-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.version-number {
  font-weight: 600;
  font-size: 14px;
  color: #1890ff;
}

.version-tags {
  margin-bottom: 4px;
}

.version-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
}

.version-size {
  color: #52c41a;
}

.text-gray {
  color: rgba(0, 0, 0, 0.45);
}

.text-red {
  color: #ff4d4f;
}

.text-green {
  color: #52c41a;
}

.diff-content {
  background: #f5f5f5;
  padding: 12px;
  border-radius: 4px;
  font-size: 12px;
  max-height: 300px;
  overflow: auto;
  margin: 0;
}

:deep(.ant-timeline-item-content) {
  margin-left: 0;
}

@media (max-width: 768px) {
  :deep(.ant-col) {
    margin-bottom: 16px;
  }
}
</style>
