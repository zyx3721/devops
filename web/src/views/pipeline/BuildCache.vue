<template>
  <div class="build-cache">
    <!-- 统计卡片 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :xs="24" :sm="12" :md="6">
        <a-card :bordered="false">
          <a-statistic
            title="缓存总数"
            :value="stats.total_count"
            :value-style="{ color: '#1890ff' }"
          >
            <template #prefix>
              <DatabaseOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :md="6">
        <a-card :bordered="false">
          <a-statistic
            title="总大小"
            :value="stats.total_size_human"
            :value-style="{ color: '#52c41a' }"
          >
            <template #prefix>
              <HddOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :md="6">
        <a-card :bordered="false">
          <a-statistic
            title="总命中次数"
            :value="stats.total_hits"
            :value-style="{ color: '#faad14' }"
          >
            <template #prefix>
              <ThunderboltOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="24" :sm="12" :md="6">
        <a-card :bordered="false">
          <a-statistic
            title="命中率"
            :value="hitRate"
            suffix="%"
            :precision="1"
            :value-style="{ color: '#eb2f96' }"
          >
            <template #prefix>
              <RiseOutlined />
            </template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 主内容卡片 -->
    <a-card title="构建缓存" :bordered="false">
      <!-- 筛选和操作栏 -->
      <a-row :gutter="16" style="margin-bottom: 16px">
        <a-col :xs="24" :sm="12" :md="8">
          <a-select
            v-model:value="searchForm.pipeline_id"
            placeholder="选择流水线"
            allowClear
            style="width: 100%"
            @change="loadData"
            show-search
            :filter-option="filterPipeline"
          >
            <a-select-option v-for="p in pipelines" :key="p.id" :value="p.id">
              {{ p.name }}
            </a-select-option>
          </a-select>
        </a-col>
        <a-col :xs="24" :sm="12" :md="8">
          <a-space>
            <a-button type="primary" @click="loadData">
              <template #icon><SearchOutlined /></template>
              查询
            </a-button>
            <a-button @click="resetSearch">
              <template #icon><ReloadOutlined /></template>
              重置
            </a-button>
          </a-space>
        </a-col>
        <a-col :xs="24" :sm="24" :md="8" style="text-align: right">
          <a-space>
            <a-button
              type="primary"
              danger
              @click="cleanExpired"
              :loading="cleaning"
              :disabled="selectedRowKeys.length === 0 && !stats.total_count"
            >
              <template #icon><DeleteOutlined /></template>
              清理过期缓存
            </a-button>
            <a-button
              danger
              @click="batchDelete"
              :disabled="selectedRowKeys.length === 0"
            >
              <template #icon><DeleteOutlined /></template>
              批量删除 ({{ selectedRowKeys.length }})
            </a-button>
          </a-space>
        </a-col>
      </a-row>

      <!-- 缓存列表表格 -->
      <a-table
        :columns="columns"
        :data-source="caches"
        :loading="loading"
        :pagination="pagination"
        :row-selection="rowSelection"
        @change="handleTableChange"
        row-key="id"
        :scroll="{ x: 1200 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'cache_key'">
            <a-tooltip :title="record.cache_key">
              <span class="cache-key">{{ record.cache_key }}</span>
            </a-tooltip>
            <a-button type="link" size="small" @click="copyCacheKey(record.cache_key)">
              <CopyOutlined />
            </a-button>
          </template>
          <template v-else-if="column.key === 'size'">
            <a-tag color="blue">{{ record.size_human }}</a-tag>
          </template>
          <template v-else-if="column.key === 'hit_count'">
            <a-badge
              :count="record.hit_count"
              :number-style="{ backgroundColor: record.hit_count > 0 ? '#52c41a' : '#d9d9d9' }"
            />
          </template>
          <template v-else-if="column.key === 'last_used_at'">
            <span v-if="record.last_used_at">
              {{ formatTime(record.last_used_at) }}
            </span>
            <span v-else class="text-gray">从未使用</span>
          </template>
          <template v-else-if="column.key === 'expires_at'">
            <span v-if="record.expires_at">
              <a-tag :color="isExpired(record.expires_at) ? 'red' : 'green'">
                {{ formatTime(record.expires_at) }}
              </a-tag>
            </span>
            <span v-else class="text-gray">永不过期</span>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showDetail(record)">
                详情
              </a-button>
              <a-popconfirm
                title="确定删除该缓存？"
                ok-text="确定"
                cancel-text="取消"
                @confirm="deleteCache(record.id)"
              >
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 详情弹窗 -->
    <a-modal
      v-model:open="detailVisible"
      title="缓存详情"
      :footer="null"
      width="600px"
    >
      <a-descriptions :column="1" bordered v-if="currentCache">
        <a-descriptions-item label="缓存键">
          {{ currentCache.cache_key }}
        </a-descriptions-item>
        <a-descriptions-item label="流水线ID">
          {{ currentCache.pipeline_id }}
        </a-descriptions-item>
        <a-descriptions-item label="存储路径">
          {{ currentCache.storage_path }}
        </a-descriptions-item>
        <a-descriptions-item label="大小">
          {{ currentCache.size_human }}
        </a-descriptions-item>
        <a-descriptions-item label="命中次数">
          <a-badge
            :count="currentCache.hit_count"
            :number-style="{ backgroundColor: currentCache.hit_count > 0 ? '#52c41a' : '#d9d9d9' }"
          />
        </a-descriptions-item>
        <a-descriptions-item label="最后使用时间">
          {{ currentCache.last_used_at ? formatTime(currentCache.last_used_at) : '从未使用' }}
        </a-descriptions-item>
        <a-descriptions-item label="过期时间">
          {{ currentCache.expires_at ? formatTime(currentCache.expires_at) : '永不过期' }}
        </a-descriptions-item>
        <a-descriptions-item label="创建时间">
          {{ formatTime(currentCache.created_at) }}
        </a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import {
  DatabaseOutlined,
  HddOutlined,
  ThunderboltOutlined,
  RiseOutlined,
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  CopyOutlined,
} from '@ant-design/icons-vue'
import request from '@/services/api'
import dayjs from 'dayjs'

interface BuildCache {
  id: number
  pipeline_id: number
  cache_key: string
  storage_path: string
  size: number
  size_human: string
  hit_count: number
  last_used_at: string | null
  expires_at: string | null
  created_at: string
}

interface CacheStats {
  total_count: number
  total_size: number
  total_size_human: string
  total_hits: number
}

interface Pipeline {
  id: number
  name: string
}

const loading = ref(false)
const cleaning = ref(false)
const detailVisible = ref(false)
const caches = ref<BuildCache[]>([])
const pipelines = ref<Pipeline[]>([])
const currentCache = ref<BuildCache | null>(null)
const selectedRowKeys = ref<number[]>([])

const stats = reactive<CacheStats>({
  total_count: 0,
  total_size: 0,
  total_size_human: '0 B',
  total_hits: 0,
})

const searchForm = reactive({
  pipeline_id: undefined as number | undefined,
  page: 1,
  page_size: 10,
})

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`,
})

const columns = [
  { title: '缓存键', dataIndex: 'cache_key', key: 'cache_key', width: 250, ellipsis: true },
  { title: '流水线ID', dataIndex: 'pipeline_id', key: 'pipeline_id', width: 100 },
  { title: '大小', key: 'size', width: 100 },
  { title: '命中次数', key: 'hit_count', width: 100 },
  { title: '最后使用时间', key: 'last_used_at', width: 180 },
  { title: '过期时间', key: 'expires_at', width: 180 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180,
    customRender: ({ text }: { text: string }) => formatTime(text) },
  { title: '操作', key: 'action', width: 150, fixed: 'right' as const },
]

const rowSelection = {
  selectedRowKeys: selectedRowKeys,
  onChange: (keys: number[]) => {
    selectedRowKeys.value = keys
  },
}

const hitRate = computed(() => {
  if (stats.total_count === 0) return 0
  return (stats.total_hits / stats.total_count) * 100
})

const formatTime = (time: string) => {
  return time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'
}

const isExpired = (expiresAt: string) => {
  return dayjs(expiresAt).isBefore(dayjs())
}

const formatSize = (bytes: number): string => {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

const filterPipeline = (input: string, option: any) => {
  return option.children[0].children.toLowerCase().includes(input.toLowerCase())
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await request.get('/build-caches', {
      params: {
        pipeline_id: searchForm.pipeline_id,
        page: pagination.current,
        page_size: pagination.pageSize,
      },
    })
    if (res?.data) {
      caches.value = res.data.items || []
      pagination.total = res.data.total || 0
      
      // 格式化大小
      caches.value.forEach(cache => {
        cache.size_human = formatSize(cache.size)
      })
    }
  } catch (error) {
    console.error('加载缓存列表失败:', error)
  } finally {
    loading.value = false
  }
}

const loadStats = async () => {
  try {
    const res = await request.get('/build/cache/stats', {
      params: {
        pipeline_id: searchForm.pipeline_id,
      },
    })
    if (res?.data) {
      Object.assign(stats, res.data)
      stats.total_size_human = formatSize(stats.total_size)
    }
  } catch (error) {
    console.error('加载缓存统计失败:', error)
  }
}

const loadPipelines = async () => {
  try {
    const res = await request.get('/pipelines', {
      params: { page_size: 100 },
    })
    pipelines.value = res?.data?.items || []
  } catch (error) {
    console.error('加载流水线列表失败:', error)
  }
}

const resetSearch = () => {
  searchForm.pipeline_id = undefined
  pagination.current = 1
  loadData()
  loadStats()
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  loadData()
}

const showDetail = (record: BuildCache) => {
  currentCache.value = record
  detailVisible.value = true
}

const deleteCache = async (id: number) => {
  try {
    await request.delete(`/build-caches/${id}`)
    message.success('删除成功')
    loadData()
    loadStats()
  } catch (error: any) {
    message.error(error?.message || '删除失败')
  }
}

const batchDelete = () => {
  if (selectedRowKeys.value.length === 0) {
    message.warning('请选择要删除的缓存')
    return
  }
  
  const modal = {
    title: '批量删除确认',
    content: `确定要删除选中的 ${selectedRowKeys.value.length} 个缓存吗？`,
    okText: '确定',
    cancelText: '取消',
    onOk: async () => {
      try {
        await Promise.all(
          selectedRowKeys.value.map(id => request.delete(`/build-caches/${id}`))
        )
        message.success('批量删除成功')
        selectedRowKeys.value = []
        loadData()
        loadStats()
      } catch (error: any) {
        message.error(error?.message || '批量删除失败')
      }
    },
  }
  
  // 使用 Ant Design Vue 的 Modal.confirm
  import('ant-design-vue').then(({ Modal }) => {
    Modal.confirm(modal)
  })
}

const cleanExpired = async () => {
  cleaning.value = true
  try {
    const res = await request.post('/build/cache/clean')
    const count = res?.data?.cleaned_count || 0
    message.success(`清理完成，共清理 ${count} 个过期缓存`)
    loadData()
    loadStats()
  } catch (error: any) {
    message.error(error?.message || '清理失败')
  } finally {
    cleaning.value = false
  }
}

const copyCacheKey = (key: string) => {
  navigator.clipboard.writeText(key)
  message.success('已复制到剪贴板')
}

onMounted(() => {
  loadData()
  loadStats()
  loadPipelines()
})
</script>

<style scoped>
.build-cache {
  padding: 0;
}

.cache-key {
  max-width: 200px;
  display: inline-block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  vertical-align: middle;
}

.text-gray {
  color: #999;
}

:deep(.ant-statistic-title) {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.65);
}

:deep(.ant-statistic-content) {
  font-size: 24px;
  font-weight: 600;
}

@media (max-width: 768px) {
  :deep(.ant-col) {
    margin-bottom: 16px;
  }
}
</style>
