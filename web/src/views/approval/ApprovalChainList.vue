<template>
  <div class="approval-chain-list">
    <!-- 搜索栏 -->
    <a-card :bordered="false" class="search-card">
      <a-form layout="inline" :model="searchForm">
        <a-form-item label="应用">
          <a-select
            v-model:value="searchForm.app_id"
            placeholder="全部应用"
            allow-clear
            style="width: 200px"
          >
            <a-select-option v-for="app in appList" :key="app.id" :value="app.id">
              {{ app.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="环境">
          <a-select
            v-model:value="searchForm.env"
            placeholder="全部环境"
            allow-clear
            style="width: 150px"
          >
            <a-select-option value="prod">生产环境</a-select-option>
            <a-select-option value="staging">预发环境</a-select-option>
            <a-select-option value="test">测试环境</a-select-option>
            <a-select-option value="dev">开发环境</a-select-option>
            <a-select-option value="*">通配符(*)</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" @click="handleSearch">查询</a-button>
            <a-button @click="handleReset">重置</a-button>
          </a-space>
        </a-form-item>
        <a-form-item style="margin-left: auto">
          <a-button type="primary" @click="handleCreate">
            <template #icon><PlusOutlined /></template>
            新建审批链
          </a-button>
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
          <template v-if="column.key === 'app_name'">
            {{ record.app_id === 0 ? '全局' : record.app_name || '-' }}
          </template>
          <template v-else-if="column.key === 'env'">
            <a-tag :color="getEnvTagColor(record.env)">
              {{ getEnvLabel(record.env) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'nodes_count'">
            {{ record.nodes?.length || 0 }}
          </template>
          <template v-else-if="column.key === 'timeout_minutes'">
            {{ record.timeout_minutes }}分钟
          </template>
          <template v-else-if="column.key === 'enabled'">
            <a-switch v-model:checked="record.enabled" @change="handleToggleEnabled(record)" />
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatDate(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="handleEdit(record)">编辑</a-button>
              <a-button type="link" size="small" @click="handleDesign(record)">设计</a-button>
              <a-button type="link" size="small" @click="handleTest(record)" :disabled="!record.nodes?.length">测试</a-button>
              <a-button type="link" size="small" danger @click="handleDelete(record)">删除</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 新建/编辑对话框 -->
    <a-modal
      v-model:open="dialogVisible"
      :title="dialogTitle"
      width="600px"
      :confirm-loading="submitting"
      @ok="handleSubmit"
    >
      <a-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        :label-col="{ span: 5 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="名称" name="name">
          <a-input v-model:value="formData.name" placeholder="请输入审批链名称" />
        </a-form-item>
        <a-form-item label="应用" name="app_id">
          <a-select v-model:value="formData.app_id" placeholder="选择应用（0表示全局）">
            <a-select-option :value="0">全局（所有应用）</a-select-option>
            <a-select-option v-for="app in appList" :key="app.id" :value="app.id">
              {{ app.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="环境" name="env">
          <a-select v-model:value="formData.env" placeholder="选择环境">
            <a-select-option value="*">所有环境(*)</a-select-option>
            <a-select-option value="prod">生产环境</a-select-option>
            <a-select-option value="staging">预发环境</a-select-option>
            <a-select-option value="test">测试环境</a-select-option>
            <a-select-option value="dev">开发环境</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="超时时间" name="timeout_minutes">
          <a-input-number v-model:value="formData.timeout_minutes" :min="1" :max="1440" />
          <span style="margin-left: 8px">分钟</span>
        </a-form-item>
        <a-form-item label="超时动作" name="timeout_action">
          <a-select v-model:value="formData.timeout_action">
            <a-select-option value="auto_reject">自动拒绝</a-select-option>
            <a-select-option value="auto_approve">自动通过</a-select-option>
            <a-select-option value="auto_cancel">自动取消</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="formData.description" :rows="3" placeholder="请输入描述" />
        </a-form-item>
        <a-form-item label="启用" name="enabled">
          <a-switch v-model:checked="formData.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { useRouter } from 'vue-router'
import type { FormInstance, Rule } from 'ant-design-vue/es/form'
import type { TablePaginationConfig } from 'ant-design-vue'
import {
  getChainList,
  createChain,
  updateChain,
  deleteChain,
  testChain,
  type ApprovalChain
} from '@/services/approvalChain'
import { applicationApi } from '@/services/application'
import dayjs from 'dayjs'

const router = useRouter()

// 搜索表单
const searchForm = reactive({
  app_id: undefined as number | undefined,
  env: undefined as string | undefined
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
const tableData = ref<ApprovalChain[]>([])
const appList = ref<any[]>([])

// 表格列
const columns = [
  { title: '审批链名称', dataIndex: 'name', key: 'name', ellipsis: true },
  { title: '应用', key: 'app_name', width: 150 },
  { title: '环境', key: 'env', width: 100 },
  { title: '节点数', key: 'nodes_count', width: 80, align: 'center' as const },
  { title: '超时时间', key: 'timeout_minutes', width: 100 },
  { title: '状态', key: 'enabled', width: 80, align: 'center' as const },
  { title: '创建时间', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 180, fixed: 'right' as const }
]

// 对话框
const dialogVisible = ref(false)
const dialogTitle = ref('新建审批链')
const submitting = ref(false)
const formRef = ref<FormInstance>()
const formData = reactive({
  id: 0,
  name: '',
  app_id: 0,
  env: '*',
  timeout_minutes: 60,
  timeout_action: 'auto_reject',
  description: '',
  enabled: true
})

const formRules: Record<string, Rule[]> = {
  name: [{ required: true, message: '请输入审批链名称', trigger: 'blur' }],
  env: [{ required: true, message: '请选择环境', trigger: 'change' }]
}

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const res = await getChainList({
      page: pagination.current,
      page_size: pagination.pageSize,
      app_id: searchForm.app_id,
      env: searchForm.env
    })
    tableData.value = res.data.list || []
    pagination.total = res.data.total || 0
  } catch (error) {
    console.error('加载审批链列表失败:', error)
  } finally {
    loading.value = false
  }
}

// 加载应用列表
const loadAppList = async () => {
  try {
    const res = await applicationApi.list({ page: 1, page_size: 1000 })
    appList.value = res.data.list || []
  } catch (error) {
    console.error('加载应用列表失败:', error)
  }
}

// 搜索
const handleSearch = () => {
  pagination.current = 1
  loadData()
}

// 重置
const handleReset = () => {
  searchForm.app_id = undefined
  searchForm.env = undefined
  handleSearch()
}

// 表格变化
const handleTableChange = (pag: TablePaginationConfig) => {
  pagination.current = pag.current || 1
  pagination.pageSize = pag.pageSize || 10
  loadData()
}

// 新建
const handleCreate = () => {
  dialogTitle.value = '新建审批链'
  Object.assign(formData, {
    id: 0,
    name: '',
    app_id: 0,
    env: '*',
    timeout_minutes: 60,
    timeout_action: 'auto_reject',
    description: '',
    enabled: true
  })
  dialogVisible.value = true
}

// 编辑
const handleEdit = (row: ApprovalChain) => {
  dialogTitle.value = '编辑审批链'
  Object.assign(formData, {
    id: row.id,
    name: row.name,
    app_id: row.app_id,
    env: row.env,
    timeout_minutes: row.timeout_minutes,
    timeout_action: row.timeout_action,
    description: row.description,
    enabled: row.enabled
  })
  dialogVisible.value = true
}

// 设计（跳转到节点设计页面）
const handleDesign = (row: ApprovalChain) => {
  router.push(`/approval/chains/${row.id}/design`)
}

// 删除
const handleDelete = (row: ApprovalChain) => {
  Modal.confirm({
    title: '提示',
    content: `确定要删除审批链"${row.name}"吗？`,
    okType: 'danger',
    onOk: async () => {
      try {
        await deleteChain(row.id)
        message.success('删除成功')
        loadData()
      } catch (error: any) {
        message.error(error.message || '删除失败')
      }
    }
  })
}

// 测试审批链
const handleTest = (row: ApprovalChain) => {
  if (!row.nodes?.length) {
    message.warning('请先添加审批节点')
    return
  }
  Modal.confirm({
    title: '测试审批链',
    content: `将为审批链"${row.name}"创建一个测试审批实例，确定继续？`,
    onOk: async () => {
      try {
        const res = await testChain(row.id)
        message.success('测试实例创建成功')
        // 跳转到审批实例详情
        router.push(`/approval/instances/${res.data.id}`)
      } catch (error: any) {
        message.error(error.message || '创建测试实例失败')
      }
    }
  })
}

// 切换启用状态
const handleToggleEnabled = async (row: ApprovalChain) => {
  try {
    await updateChain(row.id, { enabled: row.enabled })
    message.success(row.enabled ? '已启用' : '已禁用')
  } catch (error: any) {
    row.enabled = !row.enabled
    message.error(error.message || '操作失败')
  }
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
    submitting.value = true
    if (formData.id) {
      await updateChain(formData.id, formData)
      message.success('更新成功')
    } else {
      await createChain(formData)
      message.success('创建成功')
    }
    dialogVisible.value = false
    loadData()
  } catch (error: any) {
    if (error.errorFields) return // 表单验证错误
    message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

// 工具函数
const formatDate = (date: string) => {
  if (!date) return '-'
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

const getEnvLabel = (env: string) => {
  const map: Record<string, string> = {
    '*': '所有',
    prod: '生产',
    staging: '预发',
    test: '测试',
    dev: '开发'
  }
  return map[env] || env
}

const getEnvTagColor = (env: string) => {
  const map: Record<string, string> = {
    prod: 'red',
    staging: 'orange',
    test: 'blue',
    dev: 'green',
    '*': 'default'
  }
  return map[env] || 'default'
}

onMounted(() => {
  loadData()
  loadAppList()
})
</script>

<style scoped>
.approval-chain-list {
  padding: 16px;
}

.search-card {
  margin-bottom: 16px;
}

.search-card :deep(.ant-form) {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
}
</style>
