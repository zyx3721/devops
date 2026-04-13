<template>
  <div class="git-repos">
    <a-card title="Git 仓库管理" :bordered="false">
      <template #extra>
        <a-button type="primary" @click="showAddModal">
          <template #icon><PlusOutlined /></template>
          添加仓库
        </a-button>
      </template>

      <!-- 搜索栏 -->
      <a-row :gutter="16" style="margin-bottom: 16px">
        <a-col :span="6">
          <a-input v-model:value="searchForm.name" placeholder="仓库名称" allowClear @pressEnter="loadData" />
        </a-col>
        <a-col :span="4">
          <a-select v-model:value="searchForm.provider" placeholder="提供商" allowClear style="width: 100%">
            <a-select-option value="github">GitHub</a-select-option>
            <a-select-option value="gitlab">GitLab</a-select-option>
            <a-select-option value="gitee">Gitee</a-select-option>
            <a-select-option value="custom">自建</a-select-option>
          </a-select>
        </a-col>
        <a-col :span="4">
          <a-button type="primary" @click="loadData">查询</a-button>
          <a-button style="margin-left: 8px" @click="resetSearch">重置</a-button>
        </a-col>
      </a-row>

      <!-- 仓库列表 -->
      <a-table
        :columns="columns"
        :data-source="repos"
        :loading="loading"
        :pagination="pagination"
        @change="handleTableChange"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'provider'">
            <a-tag :color="getProviderColor(record.provider)">
              {{ getProviderLabel(record.provider) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'url'">
            <a :href="record.url" target="_blank" rel="noopener">
              {{ record.url }}
              <LinkOutlined />
            </a>
          </template>
          <template v-else-if="column.key === 'credential'">
            <span v-if="record.credential_name">{{ record.credential_name }}</span>
            <span v-else class="text-gray">未配置</span>
          </template>
          <template v-else-if="column.key === 'webhook'">
            <a-tooltip v-if="record.webhook_url" :title="record.webhook_url">
              <a-button size="small" @click="copyWebhook(record)">
                <CopyOutlined /> 复制
              </a-button>
            </a-tooltip>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="testConnection(record)">
                <ApiOutlined /> 测试
              </a-button>
              <a-button type="link" size="small" @click="showEditModal(record)">
                编辑
              </a-button>
              <a-popconfirm title="确定删除该仓库？" @confirm="deleteRepo(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 添加/编辑弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      :title="isEdit ? '编辑仓库' : '添加仓库'"
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
        <a-form-item label="仓库名称" name="name">
          <a-input v-model:value="form.name" placeholder="请输入仓库名称" />
        </a-form-item>
        <a-form-item label="仓库地址" name="url">
          <a-input v-model:value="form.url" placeholder="https://github.com/user/repo.git" />
        </a-form-item>
        <a-form-item label="提供商" name="provider">
          <a-select v-model:value="form.provider" placeholder="自动检测">
            <a-select-option value="">自动检测</a-select-option>
            <a-select-option value="github">GitHub</a-select-option>
            <a-select-option value="gitlab">GitLab</a-select-option>
            <a-select-option value="gitee">Gitee</a-select-option>
            <a-select-option value="custom">自建</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="默认分支" name="default_branch">
          <a-input v-model:value="form.default_branch" placeholder="main" />
        </a-form-item>
        <a-form-item label="认证凭证" name="credential_id">
          <a-select v-model:value="form.credential_id" placeholder="选择凭证（可选）" allowClear>
            <a-select-option v-for="cred in credentials" :key="cred.id" :value="cred.id">
              {{ cred.name }} ({{ cred.type }})
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="描述" name="description">
          <a-textarea v-model:value="form.description" placeholder="仓库描述" :rows="3" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, LinkOutlined, CopyOutlined, ApiOutlined } from '@ant-design/icons-vue'
import { gitRepoApi, pipelineApi } from '@/services/pipeline'

interface GitRepo {
  id: number
  name: string
  url: string
  provider: string
  default_branch: string
  credential_id?: number
  credential_name?: string
  webhook_url: string
  description: string
  created_at: string
}

interface Credential {
  id: number
  name: string
  type: string
}

const loading = ref(false)
const submitting = ref(false)
const modalVisible = ref(false)
const isEdit = ref(false)
const repos = ref<GitRepo[]>([])
const credentials = ref<Credential[]>([])
const formRef = ref()

const searchForm = reactive({
  name: '',
  provider: undefined as string | undefined,
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

const form = reactive({
  id: 0,
  name: '',
  url: '',
  provider: '',
  default_branch: 'main',
  credential_id: undefined as number | undefined,
  description: ''
})

const rules = {
  name: [{ required: true, message: '请输入仓库名称' }],
  url: [{ required: true, message: '请输入仓库地址' }]
}

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 150 },
  { title: '仓库地址', dataIndex: 'url', key: 'url', ellipsis: true },
  { title: '提供商', dataIndex: 'provider', key: 'provider', width: 100 },
  { title: '默认分支', dataIndex: 'default_branch', key: 'default_branch', width: 100 },
  { title: '凭证', dataIndex: 'credential_name', key: 'credential', width: 120 },
  { title: 'Webhook', key: 'webhook', width: 100 },
  { title: '操作', key: 'action', width: 180, fixed: 'right' }
]

const getProviderColor = (provider: string) => {
  const colors: Record<string, string> = {
    github: 'purple',
    gitlab: 'orange',
    gitee: 'red',
    custom: 'blue'
  }
  return colors[provider] || 'default'
}

const getProviderLabel = (provider: string) => {
  const labels: Record<string, string> = {
    github: 'GitHub',
    gitlab: 'GitLab',
    gitee: 'Gitee',
    custom: '自建'
  }
  return labels[provider] || provider
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await gitRepoApi.list({
      name: searchForm.name || undefined,
      provider: searchForm.provider,
      page: pagination.current,
      page_size: pagination.pageSize
    })
    if (res?.data) {
      repos.value = res.data.items || []
      pagination.total = res.data.total || 0
    }
  } catch (error) {
    console.error('加载仓库列表失败:', error)
  } finally {
    loading.value = false
  }
}

const loadCredentials = async () => {
  try {
    const res = await pipelineApi.getCredentials()
    if (res?.data) {
      credentials.value = res.data || []
    }
  } catch (error) {
    console.error('加载凭证列表失败:', error)
  }
}

const resetSearch = () => {
  searchForm.name = ''
  searchForm.provider = undefined
  pagination.current = 1
  loadData()
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  loadData()
}

const showAddModal = () => {
  isEdit.value = false
  Object.assign(form, {
    id: 0,
    name: '',
    url: '',
    provider: '',
    default_branch: 'main',
    credential_id: undefined,
    description: ''
  })
  modalVisible.value = true
}

const showEditModal = (record: GitRepo) => {
  isEdit.value = true
  Object.assign(form, {
    id: record.id,
    name: record.name,
    url: record.url,
    provider: record.provider,
    default_branch: record.default_branch,
    credential_id: record.credential_id,
    description: record.description
  })
  modalVisible.value = true
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
    submitting.value = true

    const data = {
      name: form.name,
      url: form.url,
      provider: form.provider || undefined,
      default_branch: form.default_branch,
      credential_id: form.credential_id,
      description: form.description
    }

    if (isEdit.value) {
      await gitRepoApi.update(form.id, data)
      message.success('更新成功')
    } else {
      await gitRepoApi.create(data)
      message.success('创建成功')
    }

    modalVisible.value = false
    loadData()
  } catch (error: any) {
    if (error?.errorFields) return
    message.error(error?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const deleteRepo = async (id: number) => {
  try {
    await gitRepoApi.delete(id)
    message.success('删除成功')
    loadData()
  } catch (error: any) {
    message.error(error?.message || '删除失败')
  }
}

const testConnection = async (record: GitRepo) => {
  try {
    message.loading({ content: '测试连接中...', key: 'test' })
    const res = await gitRepoApi.testConnection({
      url: record.url,
      credential_id: record.credential_id
    })
    if (res?.data?.success) {
      message.success({ content: '连接成功', key: 'test' })
    } else {
      message.error({ content: res?.data?.message || '连接失败', key: 'test' })
    }
  } catch (error: any) {
    message.error({ content: error?.message || '测试失败', key: 'test' })
  }
}

const copyWebhook = (record: GitRepo) => {
  navigator.clipboard.writeText(record.webhook_url)
  message.success('Webhook URL 已复制')
}

onMounted(() => {
  loadData()
  loadCredentials()
})
</script>

<style scoped>
.git-repos {
  padding: 0;
}
.text-gray {
  color: #999;
}
</style>
