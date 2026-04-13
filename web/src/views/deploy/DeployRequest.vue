<template>
  <div class="deploy-request">
    <a-card title="发布管理">
      <!-- 搜索栏 -->
      <a-form layout="inline" :model="searchForm" class="search-form">
        <a-form-item label="应用">
          <a-input v-model:value="searchForm.app_name" placeholder="应用名称" allow-clear style="width: 150px" />
        </a-form-item>
        <a-form-item label="环境">
          <a-select v-model:value="searchForm.env_name" placeholder="选择环境" allow-clear style="width: 120px">
            <a-select-option value="dev">开发</a-select-option>
            <a-select-option value="test">测试</a-select-option>
            <a-select-option value="staging">预发</a-select-option>
            <a-select-option value="prod">生产</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="searchForm.status" placeholder="选择状态" allow-clear style="width: 120px">
            <a-select-option value="pending">待审批</a-select-option>
            <a-select-option value="approved">已通过</a-select-option>
            <a-select-option value="rejected">已拒绝</a-select-option>
            <a-select-option value="running">执行中</a-select-option>
            <a-select-option value="success">成功</a-select-option>
            <a-select-option value="failed">失败</a-select-option>
            <a-select-option value="cancelled">已取消</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" @click="handleSearch">查询</a-button>
            <a-button @click="handleReset">重置</a-button>
            <a-button type="primary" @click="showCreateModal">创建发布</a-button>
          </a-space>
        </a-form-item>
      </a-form>

      <!-- 表格 -->
      <a-table
        :columns="columns"
        :data-source="records"
        :loading="loading"
        :pagination="pagination"
        row-key="id"
        @change="handleTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="getStatusColor(record.status)">{{ getStatusText(record.status) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'env_name'">
            <a-tag :color="getEnvColor(record.env_name)">{{ record.env_name }}</a-tag>
          </template>
          <template v-else-if="column.key === 'deploy_type'">
            <a-tag :color="record.deploy_type === 'rollback' ? 'orange' : 'blue'">
              {{ record.deploy_type === 'rollback' ? '回滚' : '部署' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'deploy_method'">
            <a-tag :color="record.deploy_method === 'k8s' ? 'green' : 'blue'">
              {{ record.deploy_method === 'k8s' ? 'K8s' : 'Jenkins' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'duration'">
            {{ record.duration > 0 ? `${record.duration}秒` : '-' }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showDetail(record)">详情</a-button>
              <template v-if="record.status === 'pending' && record.need_approval">
                <a-button type="link" size="small" @click="handleApprove(record)">通过</a-button>
                <a-button type="link" size="small" danger @click="handleReject(record)">拒绝</a-button>
              </template>
              <template v-if="record.status === 'pending' && !record.need_approval">
                <a-button type="link" size="small" @click="handleExecute(record)">执行</a-button>
              </template>
              <template v-if="record.status === 'approved'">
                <a-button type="link" size="small" @click="handleExecute(record)">执行</a-button>
              </template>
              <template v-if="record.status === 'pending'">
                <a-button type="link" size="small" @click="handleCancel(record)">取消</a-button>
              </template>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 创建发布弹窗 -->
    <a-modal v-model:open="createModalVisible" title="创建发布" @ok="handleCreate" :confirm-loading="createLoading">
      <a-form :model="createForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="应用" required>
          <a-select v-model:value="createForm.application_id" placeholder="选择应用" show-search :filter-option="filterOption">
            <a-select-option v-for="app in applications" :key="app.id" :value="app.id">{{ app.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="环境" required>
          <a-select v-model:value="createForm.env_name" placeholder="选择环境">
            <a-select-option value="dev">开发</a-select-option>
            <a-select-option value="test">测试</a-select-option>
            <a-select-option value="staging">预发</a-select-option>
            <a-select-option value="prod">生产</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="部署方式">
          <a-select v-model:value="createForm.deploy_method" placeholder="选择部署方式">
            <a-select-option value="jenkins">Jenkins</a-select-option>
            <a-select-option value="k8s">K8s</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="分支">
          <a-input v-model:value="createForm.branch" placeholder="如: main, develop" />
        </a-form-item>
        <a-form-item label="版本/镜像">
          <a-input v-model:value="createForm.image_tag" placeholder="镜像标签或版本号" />
        </a-form-item>
        <a-form-item label="说明">
          <a-textarea v-model:value="createForm.description" :rows="3" placeholder="发布说明" />
        </a-form-item>
      </a-form>
      <a-alert v-if="createForm.env_name === 'prod'" type="warning" message="生产环境需要审批后才能执行" show-icon style="margin-top: 16px" />
    </a-modal>

    <!-- 详情弹窗 -->
    <a-modal v-model:open="detailModalVisible" title="部署详情" :footer="null" width="700px">
      <a-descriptions :column="2" bordered v-if="currentRecord">
        <a-descriptions-item label="应用">{{ currentRecord.app_name }}</a-descriptions-item>
        <a-descriptions-item label="环境">
          <a-tag :color="getEnvColor(currentRecord.env_name)">{{ currentRecord.env_name }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="类型">
          <a-tag :color="currentRecord.deploy_type === 'rollback' ? 'orange' : 'blue'">
            {{ currentRecord.deploy_type === 'rollback' ? '回滚' : '部署' }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="方式">
          <a-tag :color="currentRecord.deploy_method === 'k8s' ? 'green' : 'blue'">
            {{ currentRecord.deploy_method === 'k8s' ? 'K8s' : 'Jenkins' }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-tag :color="getStatusColor(currentRecord.status)">{{ getStatusText(currentRecord.status) }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="需要审批">{{ currentRecord.need_approval ? '是' : '否' }}</a-descriptions-item>
        <a-descriptions-item label="分支">{{ currentRecord.branch || '-' }}</a-descriptions-item>
        <a-descriptions-item label="镜像">{{ currentRecord.image_tag || '-' }}</a-descriptions-item>
        <a-descriptions-item label="操作人">{{ currentRecord.operator }}</a-descriptions-item>
        <a-descriptions-item label="创建时间">{{ currentRecord.created_at }}</a-descriptions-item>
        <a-descriptions-item label="审批人" v-if="currentRecord.approver_name">{{ currentRecord.approver_name }}</a-descriptions-item>
        <a-descriptions-item label="审批时间" v-if="currentRecord.approved_at">{{ currentRecord.approved_at }}</a-descriptions-item>
        <a-descriptions-item label="开始时间" v-if="currentRecord.started_at">{{ currentRecord.started_at }}</a-descriptions-item>
        <a-descriptions-item label="完成时间" v-if="currentRecord.finished_at">{{ currentRecord.finished_at }}</a-descriptions-item>
        <a-descriptions-item label="耗时" v-if="currentRecord.duration">{{ currentRecord.duration }}秒</a-descriptions-item>
        <a-descriptions-item label="说明" :span="2">{{ currentRecord.description || '-' }}</a-descriptions-item>
        <a-descriptions-item label="拒绝原因" :span="2" v-if="currentRecord.reject_reason">
          <span style="color: red">{{ currentRecord.reject_reason }}</span>
        </a-descriptions-item>
        <a-descriptions-item label="错误信息" :span="2" v-if="currentRecord.error_msg">
          <span style="color: red">{{ currentRecord.error_msg }}</span>
        </a-descriptions-item>
      </a-descriptions>

      <a-divider v-if="approvalRecords.length > 0">审批记录</a-divider>
      <a-timeline v-if="approvalRecords.length > 0">
        <a-timeline-item v-for="record in approvalRecords" :key="record.id" :color="record.action === 'approve' ? 'green' : 'red'">
          <p><strong>{{ record.approver_name }}</strong> {{ record.action === 'approve' ? '通过' : '拒绝' }}</p>
          <p style="color: #999">{{ record.created_at }}</p>
          <p v-if="record.comment">{{ record.comment }}</p>
        </a-timeline-item>
      </a-timeline>
    </a-modal>

    <!-- 拒绝原因弹窗 -->
    <a-modal v-model:open="rejectModalVisible" title="拒绝原因" @ok="confirmReject" :confirm-loading="rejectLoading">
      <a-textarea v-model:value="rejectReason" :rows="3" placeholder="请输入拒绝原因" />
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { deployApi, type DeployRecord, type ApprovalRecord } from '../../services/deploy'
import { applicationApi, type Application } from '../../services/application'

const loading = ref(false)
const records = ref<DeployRecord[]>([])
const applications = ref<Application[]>([])
const pagination = reactive({ current: 1, pageSize: 20, total: 0 })

const searchForm = reactive({
  app_name: '',
  env_name: '',
  status: ''
})

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '应用', dataIndex: 'app_name', key: 'app_name' },
  { title: '环境', dataIndex: 'env_name', key: 'env_name', width: 80 },
  { title: '类型', dataIndex: 'deploy_type', key: 'deploy_type', width: 80 },
  { title: '方式', dataIndex: 'deploy_method', key: 'deploy_method', width: 80 },
  { title: '分支', dataIndex: 'branch', key: 'branch' },
  { title: '状态', dataIndex: 'status', key: 'status', width: 90 },
  { title: '操作人', dataIndex: 'operator', key: 'operator', width: 100 },
  { title: '耗时', dataIndex: 'duration', key: 'duration', width: 80 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 200 }
]

// 创建发布
const createModalVisible = ref(false)
const createLoading = ref(false)
const createForm = reactive({
  application_id: undefined as number | undefined,
  env_name: '',
  deploy_method: 'jenkins',
  branch: '',
  image_tag: '',
  description: ''
})

// 详情
const detailModalVisible = ref(false)
const currentRecord = ref<DeployRecord | null>(null)
const approvalRecords = ref<ApprovalRecord[]>([])

// 拒绝
const rejectModalVisible = ref(false)
const rejectLoading = ref(false)
const rejectReason = ref('')
const rejectingId = ref<number>(0)

const getStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    pending: 'orange',
    approved: 'blue',
    rejected: 'red',
    running: 'processing',
    success: 'green',
    failed: 'red',
    cancelled: 'default'
  }
  return colors[status] || 'default'
}

const getStatusText = (status: string) => {
  const texts: Record<string, string> = {
    pending: '待审批',
    approved: '已通过',
    rejected: '已拒绝',
    running: '执行中',
    success: '成功',
    failed: '失败',
    cancelled: '已取消'
  }
  return texts[status] || status
}

const getEnvColor = (env: string) => {
  const colors: Record<string, string> = {
    dev: 'green',
    test: 'blue',
    staging: 'orange',
    prod: 'red'
  }
  return colors[env] || 'default'
}

const filterOption = (input: string, option: any) => {
  return option.children[0].children.toLowerCase().indexOf(input.toLowerCase()) >= 0
}

const fetchRecords = async () => {
  loading.value = true
  try {
    const res = await deployApi.listRecords({
      page: pagination.current,
      page_size: pagination.pageSize,
      ...searchForm
    })
    if (res.code === 0) {
      records.value = res.data.list || []
      pagination.total = res.data.total
    }
  } finally {
    loading.value = false
  }
}

const fetchApplications = async () => {
  const res = await applicationApi.list({ page_size: 1000 })
  if (res.code === 0) {
    applications.value = res.data.list || []
  }
}

const handleSearch = () => {
  pagination.current = 1
  fetchRecords()
}

const handleReset = () => {
  searchForm.app_name = ''
  searchForm.env_name = ''
  searchForm.status = ''
  handleSearch()
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  fetchRecords()
}

const showCreateModal = () => {
  createForm.application_id = undefined
  createForm.env_name = ''
  createForm.deploy_method = 'jenkins'
  createForm.branch = ''
  createForm.image_tag = ''
  createForm.description = ''
  createModalVisible.value = true
}

const handleCreate = async () => {
  if (!createForm.application_id || !createForm.env_name) {
    message.error('请选择应用和环境')
    return
  }
  createLoading.value = true
  try {
    const res = await deployApi.createDeploy(createForm as any)
    if (res.code === 0) {
      message.success('创建成功')
      createModalVisible.value = false
      fetchRecords()
    } else {
      message.error(res.message)
    }
  } finally {
    createLoading.value = false
  }
}

const showDetail = async (record: DeployRecord) => {
  const res = await deployApi.getRecord(record.id)
  if (res.code === 0) {
    currentRecord.value = res.data.record
    approvalRecords.value = res.data.approvals || []
    detailModalVisible.value = true
  }
}

const handleApprove = (record: DeployRecord) => {
  Modal.confirm({
    title: '确认审批通过？',
    content: `应用: ${record.app_name}, 环境: ${record.env_name}`,
    onOk: async () => {
      const res = await deployApi.approveDeploy(record.id)
      if (res.code === 0) {
        message.success('审批通过')
        fetchRecords()
      } else {
        message.error(res.message)
      }
    }
  })
}

const handleReject = (record: DeployRecord) => {
  rejectingId.value = record.id
  rejectReason.value = ''
  rejectModalVisible.value = true
}

const confirmReject = async () => {
  if (!rejectReason.value) {
    message.error('请输入拒绝原因')
    return
  }
  rejectLoading.value = true
  try {
    const res = await deployApi.rejectDeploy(rejectingId.value, rejectReason.value)
    if (res.code === 0) {
      message.success('已拒绝')
      rejectModalVisible.value = false
      fetchRecords()
    } else {
      message.error(res.message)
    }
  } finally {
    rejectLoading.value = false
  }
}

const handleCancel = (record: DeployRecord) => {
  Modal.confirm({
    title: '确认取消？',
    content: `应用: ${record.app_name}, 环境: ${record.env_name}`,
    onOk: async () => {
      const res = await deployApi.cancelDeploy(record.id)
      if (res.code === 0) {
        message.success('已取消')
        fetchRecords()
      } else {
        message.error(res.message)
      }
    }
  })
}

const handleExecute = (record: DeployRecord) => {
  Modal.confirm({
    title: '确认执行发布？',
    content: `应用: ${record.app_name}, 环境: ${record.env_name}`,
    onOk: async () => {
      const res = await deployApi.executeDeploy(record.id)
      if (res.code === 0) {
        message.success('已开始执行')
        fetchRecords()
      } else {
        message.error(res.message)
      }
    }
  })
}

onMounted(() => {
  fetchRecords()
  fetchApplications()
})
</script>

<style scoped>
.deploy-request {
  padding: 16px;
}
.search-form {
  margin-bottom: 16px;
}
</style>
