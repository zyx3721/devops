<template>
  <div class="approval-rules">
    <a-card title="审批规则管理">
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <template #icon><PlusOutlined /></template>
          新建规则
        </a-button>
      </template>

      <a-table :columns="columns" :data-source="rules" :loading="loading" row-key="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'app_id'">
            {{ record.app_id === 0 ? '全局' : `应用 #${record.app_id}` }}
          </template>
          <template v-else-if="column.key === 'env'">
            <a-tag :color="getEnvColor(record.env)">{{ record.env }}</a-tag>
          </template>
          <template v-else-if="column.key === 'need_approval'">
            <a-tag :color="record.need_approval ? 'orange' : 'green'">
              {{ record.need_approval ? '需要审批' : '无需审批' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'enabled'">
            <a-switch :checked="record.enabled" @change="toggleEnabled(record)" />
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showEditModal(record)">编辑</a-button>
              <a-popconfirm title="确定删除此规则？" @confirm="deleteRule(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 创建/编辑弹窗 -->
    <a-modal v-model:open="modalVisible" :title="isEdit ? '编辑规则' : '新建规则'" @ok="handleSubmit" :confirm-loading="submitting">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="应用">
          <a-select v-model:value="form.app_id" placeholder="选择应用（留空为全局）" allow-clear>
            <a-select-option :value="0">全局规则</a-select-option>
            <a-select-option v-for="app in applications" :key="app.id" :value="app.id">
              {{ app.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="环境" required>
          <a-select v-model:value="form.env" placeholder="选择环境">
            <a-select-option value="dev">开发环境</a-select-option>
            <a-select-option value="test">测试环境</a-select-option>
            <a-select-option value="staging">预发环境</a-select-option>
            <a-select-option value="prod">生产环境</a-select-option>
            <a-select-option value="production">生产环境(production)</a-select-option>
            <a-select-option value="*">所有环境</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="需要审批">
          <a-switch v-model:checked="form.need_approval" />
        </a-form-item>
        <a-form-item label="审批人">
          <a-select v-model:value="form.approvers" mode="multiple" placeholder="选择审批人" allow-clear>
            <a-select-option v-for="user in users" :key="user.id" :value="user.id.toString()">
              {{ user.username }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="超时时间">
          <a-input-number v-model:value="form.timeout_minutes" :min="5" :max="1440" addon-after="分钟" style="width: 100%" />
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="form.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { approvalRuleApi, type ApprovalRule } from '@/services/approval'
import { applicationApi } from '@/services/application'
import { userApi } from '@/services/user'

const loading = ref(false)
const submitting = ref(false)
const modalVisible = ref(false)
const isEdit = ref(false)
const rules = ref<ApprovalRule[]>([])
const applications = ref<any[]>([])
const users = ref<any[]>([])

const form = reactive({
  id: 0,
  app_id: 0,
  env: '',
  need_approval: true,
  approvers: [] as string[],
  timeout_minutes: 30,
  enabled: true
})

const columns = [
  { title: '应用', key: 'app_id', dataIndex: 'app_id' },
  { title: '环境', key: 'env', dataIndex: 'env' },
  { title: '审批要求', key: 'need_approval', dataIndex: 'need_approval' },
  { title: '超时时间', dataIndex: 'timeout_minutes', customRender: ({ text }: any) => `${text} 分钟` },
  { title: '状态', key: 'enabled', dataIndex: 'enabled' },
  { title: '操作', key: 'action', width: 150 }
]

const getEnvColor = (env: string) => {
  const colors: Record<string, string> = {
    dev: 'blue', test: 'cyan', staging: 'orange', prod: 'red', production: 'red', '*': 'purple'
  }
  return colors[env] || 'default'
}

const loadRules = async () => {
  loading.value = true
  try {
    const res = await approvalRuleApi.list()
    rules.value = res.data || []
  } finally {
    loading.value = false
  }
}

const loadApplications = async () => {
  try {
    const res = await applicationApi.list({ page: 1, page_size: 1000 })
    applications.value = res.data?.list || []
  } catch {}
}

const loadUsers = async () => {
  try {
    const res = await userApi.list({ page: 1, page_size: 1000 })
    users.value = res.data?.list || []
  } catch {}
}

const showCreateModal = () => {
  isEdit.value = false
  Object.assign(form, { id: 0, app_id: 0, env: '', need_approval: true, approvers: [], timeout_minutes: 30, enabled: true })
  modalVisible.value = true
}

const showEditModal = (record: ApprovalRule) => {
  isEdit.value = true
  Object.assign(form, {
    ...record,
    approvers: record.approvers ? record.approvers.split(',').filter(Boolean) : []
  })
  modalVisible.value = true
}

const handleSubmit = async () => {
  if (!form.env) {
    message.error('请选择环境')
    return
  }
  submitting.value = true
  try {
    const data = { ...form, approvers: form.approvers.join(',') }
    if (isEdit.value) {
      await approvalRuleApi.update(form.id, data)
      message.success('更新成功')
    } else {
      await approvalRuleApi.create(data)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadRules()
  } finally {
    submitting.value = false
  }
}

const toggleEnabled = async (record: ApprovalRule) => {
  try {
    await approvalRuleApi.update(record.id, { ...record, enabled: !record.enabled })
    message.success('更新成功')
    loadRules()
  } catch {}
}

const deleteRule = async (id: number) => {
  try {
    await approvalRuleApi.delete(id)
    message.success('删除成功')
    loadRules()
  } catch {}
}

onMounted(() => {
  loadRules()
  loadApplications()
  loadUsers()
})
</script>

<style scoped>
.approval-rules {
  padding: 16px;
}
</style>
