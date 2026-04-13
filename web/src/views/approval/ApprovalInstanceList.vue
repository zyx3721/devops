<template>
  <div class="approval-instance-list">
    <!-- 搜索栏 -->
    <a-card :bordered="false" class="search-card">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="状态">
          <a-select
            v-model:value="searchForm.status"
            placeholder="全部状态"
            allow-clear
            style="width: 150px"
          >
            <a-select-option value="pending">待审批</a-select-option>
            <a-select-option value="approved">已通过</a-select-option>
            <a-select-option value="rejected">已拒绝</a-select-option>
            <a-select-option value="cancelled">已取消</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" @click="handleSearch">查询</a-button>
            <a-button @click="handleReset">重置</a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 列表 -->
    <a-card :bordered="false" style="margin-top: 16px">
      <a-table
        :columns="columns"
        :data-source="tableData"
        :loading="loading"
        :pagination="paginationConfig"
        row-key="id"
        @change="handleTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="getStatusTagColor(record.status)">
              {{ getStatusLabel(record.status) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'current_node_order'">
            <span v-if="record.status === 'pending'">第 {{ record.current_node_order }} 节点</span>
            <span v-else>-</span>
          </template>
          <template v-else-if="column.key === 'started_at'">
            {{ formatDate(record.started_at) }}
          </template>
          <template v-else-if="column.key === 'finished_at'">
            {{ formatDate(record.finished_at) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="handleView(record)">查看</a-button>
              <a-button
                v-if="record.status === 'pending'"
                type="link"
                size="small"
                danger
                @click="handleCancel(record)"
              >取消</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 详情对话框 -->
    <a-modal
      v-model:open="detailDialogVisible"
      title="审批实例详情"
      width="800px"
      :footer="null"
    >
      <ApprovalInstanceDetail
        v-if="selectedInstance"
        :instance="selectedInstance"
        @refresh="loadInstanceDetail"
      />
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import type { TablePaginationConfig } from 'ant-design-vue'
import {
  getInstanceList,
  getInstance,
  cancelInstance,
  type ApprovalInstance
} from '@/services/approvalChain'
import ApprovalInstanceDetail from './ApprovalInstanceDetail.vue'
import dayjs from 'dayjs'

// 搜索表单
const searchForm = reactive({
  status: undefined as string | undefined
})

// 分页
const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0
})

const paginationConfig = computed<TablePaginationConfig>(() => ({
  current: pagination.current,
  pageSize: pagination.pageSize,
  total: pagination.total,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`
}))

// 数据
const loading = ref(false)
const tableData = ref<ApprovalInstance[]>([])
const detailDialogVisible = ref(false)
const selectedInstance = ref<ApprovalInstance | null>(null)

// 表格列
const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '审批链', dataIndex: 'chain_name', key: 'chain_name', ellipsis: true },
  { title: '部署记录', dataIndex: 'record_id', key: 'record_id', width: 100 },
  { title: '状态', key: 'status', width: 100 },
  { title: '当前节点', key: 'current_node_order', width: 100 },
  { title: '开始时间', key: 'started_at', width: 170 },
  { title: '完成时间', key: 'finished_at', width: 170 },
  { title: '操作', key: 'action', width: 150, fixed: 'right' as const }
]

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await getInstanceList({
      page: pagination.current,
      page_size: pagination.pageSize,
      status: searchForm.status
    })
    tableData.value = res.data.list || []
    pagination.total = res.data.total || 0
  } catch (error) {
    console.error('加载审批实例列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载实例详情
const loadInstanceDetail = async () => {
  if (!selectedInstance.value) return
  try {
    const res = await getInstance(selectedInstance.value.id)
    selectedInstance.value = res.data
  } catch (error) {
    console.error('加载实例详情失败:', error)
  }
}

// 搜索
const handleSearch = () => {
  pagination.current = 1
  loadData()
}

// 重置
const handleReset = () => {
  searchForm.status = undefined
  handleSearch()
}

// 表格变化
const handleTableChange = (pag: TablePaginationConfig) => {
  pagination.current = pag.current || 1
  pagination.pageSize = pag.pageSize || 10
  loadData()
}

// 查看详情
const handleView = async (row: ApprovalInstance) => {
  try {
    const res = await getInstance(row.id)
    selectedInstance.value = res.data
    detailDialogVisible.value = true
  } catch (error: any) {
    message.error(error.message || '加载详情失败')
  }
}

// 取消
const handleCancel = (row: ApprovalInstance) => {
  Modal.confirm({
    title: '取消审批',
    content: '确定要取消该审批实例吗？',
    okType: 'danger',
    onOk: async () => {
      try {
        await cancelInstance(row.id, '用户取消')
        message.success('取消成功')
        loadData()
      } catch (error: any) {
        message.error(error.message || '取消失败')
      }
    }
  })
}

// 工具函数
const formatDate = (date: string) => {
  if (!date) return '-'
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

const getStatusLabel = (status: string) => {
  const map: Record<string, string> = {
    pending: '待审批',
    approved: '已通过',
    rejected: '已拒绝',
    cancelled: '已取消'
  }
  return map[status] || status
}

const getStatusTagColor = (status: string) => {
  const map: Record<string, string> = {
    pending: 'processing',
    approved: 'success',
    rejected: 'error',
    cancelled: 'default'
  }
  return map[status] || 'default'
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.approval-instance-list {
  padding: 16px;
}

.search-card {
  margin-bottom: 16px;
}
</style>
