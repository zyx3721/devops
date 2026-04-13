<template>
  <div class="credentials-page">
    <a-page-header title="凭证管理" sub-title="管理流水线使用的凭证信息">
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <PlusOutlined /> 新建凭证
        </a-button>
      </template>
    </a-page-header>

    <a-card :bordered="false">
      <a-table
        :columns="columns"
        :data-source="credentials"
        :loading="loading"
        :pagination="false"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <a-tag :color="getTypeColor(record.type)">{{ getTypeLabel(record.type) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatDate(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showEditModal(record)">编辑</a-button>
              <a-popconfirm
                title="确定要删除此凭证吗？"
                @confirm="handleDelete(record.id)"
              >
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 创建/编辑弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      :title="isEdit ? '编辑凭证' : '新建凭证'"
      :confirm-loading="submitting"
      @ok="handleSubmit"
      width="600px"
    >
      <a-form
        ref="formRef"
        :model="form"
        :rules="rules"
        :label-col="{ span: 5 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="凭证名称" name="name">
          <a-input v-model:value="form.name" placeholder="请输入凭证名称" />
        </a-form-item>
        <a-form-item label="凭证类型" name="type">
          <a-select v-model:value="form.type" placeholder="请选择凭证类型" @change="onTypeChange">
            <a-select-option value="username_password">用户名/密码</a-select-option>
            <a-select-option value="token">访问令牌</a-select-option>
            <a-select-option value="ssh_key">SSH 密钥</a-select-option>
            <a-select-option value="docker_config">Docker 配置</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="form.description" placeholder="凭证描述（可选）" :rows="2" />
        </a-form-item>

        <!-- 用户名/密码类型 -->
        <template v-if="form.type === 'username_password'">
          <a-form-item label="用户名" name="username" :rules="[{ required: true, message: '请输入用户名' }]">
            <a-input v-model:value="credentialData.username" placeholder="请输入用户名" />
          </a-form-item>
          <a-form-item label="密码" name="password" :rules="[{ required: !isEdit, message: '请输入密码' }]">
            <a-input-password v-model:value="credentialData.password" :placeholder="isEdit ? '留空则不修改' : '请输入密码'" />
          </a-form-item>
        </template>

        <!-- 访问令牌类型 -->
        <template v-if="form.type === 'token'">
          <a-form-item label="令牌" name="token" :rules="[{ required: !isEdit, message: '请输入令牌' }]">
            <a-input-password v-model:value="credentialData.token" :placeholder="isEdit ? '留空则不修改' : '请输入访问令牌'" />
          </a-form-item>
        </template>

        <!-- SSH 密钥类型 -->
        <template v-if="form.type === 'ssh_key'">
          <a-form-item label="私钥" name="ssh_key" :rules="[{ required: !isEdit, message: '请输入私钥' }]">
            <a-textarea
              v-model:value="credentialData.ssh_key"
              :placeholder="isEdit ? '留空则不修改' : '请粘贴 SSH 私钥内容'"
              :rows="6"
              style="font-family: monospace"
            />
          </a-form-item>
          <a-form-item label="密钥密码" name="passphrase">
            <a-input-password v-model:value="credentialData.passphrase" placeholder="密钥密码（如有）" />
          </a-form-item>
        </template>

        <!-- Docker 配置类型 -->
        <template v-if="form.type === 'docker_config'">
          <a-form-item label="仓库地址" name="registry" :rules="[{ required: true, message: '请输入仓库地址' }]">
            <a-input v-model:value="credentialData.registry" placeholder="如: registry.example.com" />
          </a-form-item>
          <a-form-item label="用户名" name="docker_username" :rules="[{ required: true, message: '请输入用户名' }]">
            <a-input v-model:value="credentialData.username" placeholder="请输入用户名" />
          </a-form-item>
          <a-form-item label="密码" name="docker_password" :rules="[{ required: !isEdit, message: '请输入密码' }]">
            <a-input-password v-model:value="credentialData.password" :placeholder="isEdit ? '留空则不修改' : '请输入密码'" />
          </a-form-item>
        </template>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { pipelineApi } from '@/services/pipeline'
import dayjs from 'dayjs'

interface Credential {
  id: number
  name: string
  type: string
  description: string
  created_at: string
  updated_at: string
}

interface CredentialData {
  username: string
  password: string
  token: string
  ssh_key: string
  passphrase: string
  registry: string
}

const loading = ref(false)
const submitting = ref(false)
const modalVisible = ref(false)
const isEdit = ref(false)
const credentials = ref<Credential[]>([])
const formRef = ref()

const form = reactive({
  id: 0,
  name: '',
  type: 'username_password',
  description: ''
})

const credentialData = reactive<CredentialData>({
  username: '',
  password: '',
  token: '',
  ssh_key: '',
  passphrase: '',
  registry: ''
})

const rules = {
  name: [{ required: true, message: '请输入凭证名称' }],
  type: [{ required: true, message: '请选择凭证类型' }]
}

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '类型', dataIndex: 'type', key: 'type', width: 150 },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', key: 'action', width: 150 }
]

const getTypeColor = (type: string) => {
  const colors: Record<string, string> = {
    username_password: 'blue',
    token: 'green',
    ssh_key: 'orange',
    docker_config: 'purple'
  }
  return colors[type] || 'default'
}

const getTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    username_password: '用户名/密码',
    token: '访问令牌',
    ssh_key: 'SSH 密钥',
    docker_config: 'Docker 配置'
  }
  return labels[type] || type
}

const formatDate = (date: string) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

const loadCredentials = async () => {
  loading.value = true
  try {
    const res = await pipelineApi.getCredentials()
    credentials.value = res?.data || []
  } catch (error) {
    console.error('加载凭证失败:', error)
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  form.id = 0
  form.name = ''
  form.type = 'username_password'
  form.description = ''
  credentialData.username = ''
  credentialData.password = ''
  credentialData.token = ''
  credentialData.ssh_key = ''
  credentialData.passphrase = ''
  credentialData.registry = ''
}

const showCreateModal = () => {
  resetForm()
  isEdit.value = false
  modalVisible.value = true
}

const showEditModal = (record: Credential) => {
  resetForm()
  form.id = record.id
  form.name = record.name
  form.type = record.type
  form.description = record.description
  isEdit.value = true
  modalVisible.value = true
}

const onTypeChange = () => {
  // 切换类型时清空凭证数据
  credentialData.username = ''
  credentialData.password = ''
  credentialData.token = ''
  credentialData.ssh_key = ''
  credentialData.passphrase = ''
  credentialData.registry = ''
}

const buildCredentialJson = () => {
  switch (form.type) {
    case 'username_password':
      return JSON.stringify({
        username: credentialData.username,
        password: credentialData.password
      })
    case 'token':
      return JSON.stringify({ token: credentialData.token })
    case 'ssh_key':
      return JSON.stringify({
        ssh_key: credentialData.ssh_key,
        passphrase: credentialData.passphrase
      })
    case 'docker_config':
      return JSON.stringify({
        registry: credentialData.registry,
        username: credentialData.username,
        password: credentialData.password
      })
    default:
      return ''
  }
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  const data = buildCredentialJson()
  if (!isEdit.value && !data) {
    message.error('请填写凭证信息')
    return
  }

  submitting.value = true
  try {
    if (isEdit.value) {
      await pipelineApi.updateCredential(form.id, {
        name: form.name,
        type: form.type,
        description: form.description,
        data: data || undefined
      })
      message.success('更新成功')
    } else {
      await pipelineApi.createCredential({
        name: form.name,
        type: form.type,
        description: form.description,
        data
      })
      message.success('创建成功')
    }
    modalVisible.value = false
    loadCredentials()
  } catch (error: any) {
    message.error(error?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: number) => {
  try {
    await pipelineApi.deleteCredential(id)
    message.success('删除成功')
    loadCredentials()
  } catch (error: any) {
    message.error(error?.message || '删除失败')
  }
}

onMounted(() => {
  loadCredentials()
})
</script>

<style scoped>
.credentials-page {
  padding: 0;
}
</style>
