<template>
  <div class="approval-history">
    <a-card title="审批历史">
      <!-- 筛选条件 -->
      <a-form layout="inline" style="margin-bottom: 16px">
        <a-form-item label="环境">
          <a-select v-model:value="filters.env" placeholder="全部" allow-clear style="width: 120px" @change="loadRecords">
            <a-select-option value="dev">开发</a-select-option>
            <a-select-option value="test">测试</a-select-option>
            <a-select-option value="staging">预发</a-select-option>
            <a-select-option value="prod">生产</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="filters.status" placeholder="全部" allow-clear style="width: 120px" @change="loadRecords">
            <a-select-option value="pending">待审批</a-select-option>
            <a-select-option value="approved">已通过</a-select-option>
            <a-select-option value="rejected">已拒绝</a-select-option>
            <a-select-option value="cancelled">已取消</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="时间范围">
          <a-range-picker v-model:value="filters.dateRange" @change="loadRecords" />
        </a-form-item>
        <a-form-item>
          <a-button @click="loadRecords">查询</a-button>
        </a-form-item>
        <a-form-item>
          <a-dropdown>
            <a-button><DownloadOutlined /> 导出 <DownOutlined /></a-button>
            <template #overlay>
              <a-menu @click="handleExport">
                <a-menu-item key="excel"><FileExcelOutlined /> 导出 Excel</a-menu-item>
                <a-menu-item key="csv"><FileTextOutlined /> 导出 CSV</a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </a-form-item>
      </a-form>

      <a-table 
        :columns="columns" 
        :data-source="records" 
        :loading="loading" 
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'env_name'">
            <a-tag :color="getEnvColor(record.env_name)">{{ record.env_name }}</a-tag>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">{{ getStatusText(record.status) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'approved_at'">
            {{ formatTime(record.approved_at) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-button type="link" size="small" @click="showDetail(record)">详情</a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 详情弹窗 -->
    <a-modal v-model:open="detailModalVisible" title="审批详情" :footer="null" width="800px">
      <a-descriptions :column="2" bordered size="small">
        <a-descriptions-item label="应用名称">{{ currentRecord?.app_name }}</a-descriptions-item>
        <a-descriptions-item label="环境">{{ currentRecord?.env_name }}</a-descriptions-item>
        <a-descriptions-item label="版本">{{ currentRecord?.version }}</a-descriptions-item>
        <a-descriptions-item label="镜像标签">{{ currentRecord?.image_tag }}</a-descriptions-item>
        <a-descriptions-item label="分支">{{ currentRecord?.branch }}</a-descriptions-item>
        <a-descriptions-item label="Commit">{{ currentRecord?.commit_id?.substring(0, 8) }}</a-descriptions-item>
        <a-descriptions-item label="申请人">{{ currentRecord?.operator }}</a-descriptions-item>
        <a-descriptions-item label="申请时间">{{ formatTime(currentRecord?.created_at) }}</a-descriptions-item>
        <a-descriptions-item label="审批人">{{ currentRecord?.approver_name || '-' }}</a-descriptions-item>
        <a-descriptions-item label="审批时间">{{ formatTime(currentRecord?.approved_at) }}</a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-tag :color="getStatusColor(currentRecord?.status)">{{ getStatusText(currentRecord?.status) }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="拒绝原因" v-if="currentRecord?.reject_reason">
          {{ currentRecord?.reject_reason }}
        </a-descriptions-item>
        <a-descriptions-item label="发布说明" :span="2">{{ currentRecord?.description || '-' }}</a-descriptions-item>
      </a-descriptions>

      <a-divider>审批记录</a-divider>
      <a-timeline>
        <a-timeline-item v-for="item in approvalRecords" :key="item.id" :color="item.action === 'approve' ? 'green' : 'red'">
          <p><strong>{{ item.approver_name }}</strong> {{ item.action === 'approve' ? '通过' : '拒绝' }}了审批</p>
          <p style="color: #999">{{ formatTime(item.created_at) }}</p>
          <p v-if="item.comment">{{ item.comment }}</p>
        </a-timeline-item>
      </a-timeline>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { DownloadOutlined, DownOutlined, FileExcelOutlined, FileTextOutlined } from '@ant-design/icons-vue'
import { approvalApi, type ApprovalRecord } from '@/services/approval'
import dayjs, { Dayjs } from 'dayjs'

const loading = ref(false)
const exporting = ref(false)
const records = ref<any[]>([])
const currentRecord = ref<any>(null)
const detailModalVisible = ref(false)
const approvalRecords = ref<ApprovalRecord[]>([])

const filters = reactive({
  env: undefined as string | undefined,
  status: undefined as string | undefined,
  dateRange: null as [Dayjs, Dayjs] | null
})

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
  showSizeChanger: true,
  showTotal: (total: number) => `共 ${total} 条`
})

const columns = [
  { title: '应用', dataIndex: 'app_name' },
  { title: '环境', key: 'env_name', dataIndex: 'env_name' },
  { title: '版本', dataIndex: 'version' },
  { title: '状态', key: 'status', dataIndex: 'status' },
  { title: '申请人', dataIndex: 'operator' },
  { title: '审批人', dataIndex: 'approver_name' },
  { title: '申请时间', key: 'created_at', dataIndex: 'created_at' },
  { title: '审批时间', key: 'approved_at', dataIndex: 'approved_at' },
  { title: '操作', key: 'action', width: 100 }
]

const getEnvColor = (env: string) => {
  const colors: Record<string, string> = {
    dev: 'blue', test: 'cyan', staging: 'orange', prod: 'red', production: 'red'
  }
  return colors[env] || 'default'
}

const getStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    pending: 'orange', approved: 'green', rejected: 'red', cancelled: 'default'
  }
  return colors[status] || 'default'
}

const getStatusText = (status: string) => {
  const texts: Record<string, string> = {
    pending: '待审批', approved: '已通过', rejected: '已拒绝', cancelled: '已取消'
  }
  return texts[status] || status
}

const formatTime = (time: string) => {
  return time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'
}

const loadRecords = async () => {
  loading.value = true
  try {
    const params: any = {
      page: pagination.current,
      page_size: pagination.pageSize,
      env: filters.env,
      status: filters.status
    }
    if (filters.dateRange) {
      params.start_time = filters.dateRange[0].format('YYYY-MM-DD')
      params.end_time = filters.dateRange[1].format('YYYY-MM-DD')
    }
    const res = await approvalApi.getHistory(params)
    records.value = res.data?.list || []
    pagination.total = res.data?.total || 0
  } finally {
    loading.value = false
  }
}

const handleExport = async ({ key }: { key: string }) => {
  exporting.value = true
  try {
    const params: any = {
      format: key,
      env: filters.env,
      status: filters.status
    }
    if (filters.dateRange) {
      params.start_time = filters.dateRange[0].format('YYYY-MM-DD')
      params.end_time = filters.dateRange[1].format('YYYY-MM-DD')
    }
    const res = await approvalApi.exportHistory(params)
    // 创建下载链接
    const blob = new Blob([res], { type: key === 'excel' ? 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' : 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `approval-history-${dayjs().format('YYYYMMDD')}.${key === 'excel' ? 'xlsx' : 'csv'}`
    a.click()
    URL.revokeObjectURL(url)
    message.success('导出成功')
  } catch (e: any) {
    message.error(e.message || '导出失败')
  } finally {
    exporting.value = false
  }
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  loadRecords()
}

const showDetail = async (record: any) => {
  currentRecord.value = record
  detailModalVisible.value = true
  try {
    const res = await approvalApi.getRecords(record.id)
    approvalRecords.value = res.data || []
  } catch {
    approvalRecords.value = []
  }
}

onMounted(() => {
  loadRecords()
})
</script>

<style scoped>
.approval-history {
  padding: 16px;
}
</style>
