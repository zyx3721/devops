<template>
  <div class="jenkins-instances">
    <div class="page-header">
      <h1>Jenkins 实例管理</h1>
      <a-button type="primary" @click="showModal()">
        <template #icon><PlusOutlined /></template>
        新增实例
      </a-button>
    </div>

    <a-table :columns="columns" :data-source="instances" :loading="loading" :pagination="pagination" @change="handleTableChange" row-key="id">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'name'">
          <a @click="goToJobs(record)">{{ record.name }}</a>
        </template>
        <template v-if="column.key === 'status'">
          <a-tag :color="record.status === 'active' ? 'green' : 'red'">
            {{ record.status === 'active' ? '启用' : '禁用' }}
          </a-tag>
        </template>
        <template v-if="column.key === 'is_default'">
          <a-tag v-if="record.is_default" color="blue">默认</a-tag>
          <span v-else>-</span>
        </template>
        <template v-if="column.key === 'feishu_apps'">
          <a-tag v-for="app in record.feishu_apps" :key="app.id" color="purple" style="margin: 2px;">
            {{ app.name }}
          </a-tag>
          <span v-if="!record.feishu_apps?.length">-</span>
        </template>
        <template v-if="column.key === 'action'">
          <span style="white-space: nowrap;">
            <a @click="testConnection(record)" :class="{ 'testing': testingIds.has(record.id) }">
              <LoadingOutlined v-if="testingIds.has(record.id)" />
              测试连接
            </a>
            <a-divider type="vertical" />
            <a @click="showModal(record)">编辑</a>
            <a-divider type="vertical" />
            <a @click="setDefault(record)" v-if="!record.is_default">设为默认</a>
            <a-divider type="vertical" v-if="!record.is_default" />
            <a-popconfirm title="确定删除？" @confirm="handleDelete(record.id)">
              <a style="color: #ff4d4f">删除</a>
            </a-popconfirm>
          </span>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="modalVisible" :title="editingId ? '编辑实例' : '新增实例'" @ok="handleSubmit" :confirm-loading="submitting">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="名称" required>
          <a-input v-model:value="form.name" placeholder="请输入名称" />
        </a-form-item>
        <a-form-item label="URL" required>
          <a-input v-model:value="form.url" placeholder="请输入 Jenkins URL" />
        </a-form-item>
        <a-form-item label="用户名">
          <a-input v-model:value="form.username" placeholder="请输入用户名" />
        </a-form-item>
        <a-form-item label="API Token">
          <a-input-password v-model:value="form.api_token" placeholder="请输入 API Token" />
        </a-form-item>
        <a-form-item label="飞书应用">
          <a-select v-model:value="form.feishu_app_ids" mode="multiple" placeholder="请选择飞书应用（可选）" :options="feishuAppOptions" allow-clear />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="form.description" placeholder="请输入描述" :rows="3" />
        </a-form-item>
        <a-form-item label="状态" required>
          <a-select v-model:value="form.status">
            <a-select-option value="active">启用</a-select-option>
            <a-select-option value="inactive">禁用</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="设为默认">
          <a-switch v-model:checked="form.is_default" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { PlusOutlined, LoadingOutlined } from '@ant-design/icons-vue'
import { jenkinsInstanceApi, type FeishuAppSimple } from '@/services/jenkins'
import { feishuAppApi } from '@/services/feishu'
import type { JenkinsInstance } from '@/types'

interface JenkinsInstanceWithApps extends JenkinsInstance {
  feishu_apps?: FeishuAppSimple[]
}

const router = useRouter()
const loading = ref(false)
const submitting = ref(false)
const modalVisible = ref(false)
const editingId = ref<number | null>(null)
const instances = ref<JenkinsInstanceWithApps[]>([])
const feishuAppOptions = ref<{ label: string; value: number }[]>([])
const testingIds = ref<Set<number>>(new Set())

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0
})

const form = reactive({
  name: '',
  url: '',
  username: '',
  api_token: '',
  description: '',
  status: 'active',
  is_default: false,
  feishu_app_ids: [] as number[]
})

const columns = [
  { title: '名称', key: 'name' },
  { title: 'URL', dataIndex: 'url', key: 'url', ellipsis: true },
  { title: '用户名', dataIndex: 'username', key: 'username' },
  { title: '飞书应用', key: 'feishu_apps', width: 180 },
  { title: '状态', key: 'status', width: 80 },
  { title: '默认', key: 'is_default', width: 70 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '操作', key: 'action', width: 240, fixed: 'right' }
]

const goToJobs = (record: JenkinsInstance) => {
  router.push(`/jenkins/instances/${record.id}/jobs`)
}

const fetchInstances = async () => {
  loading.value = true
  try {
    const response = await jenkinsInstanceApi.getInstances({
      page: pagination.current,
      page_size: pagination.pageSize
    })
    if (response.code === 0 && response.data) {
      const items = response.data.items
      // 获取每个实例绑定的飞书应用
      for (const item of items) {
        try {
          const appsRes = await jenkinsInstanceApi.getFeishuApps(item.id)
          if (appsRes.code === 0) {
            (item as JenkinsInstanceWithApps).feishu_apps = appsRes.data
          }
        } catch {}
      }
      instances.value = items
      pagination.total = response.data.total
    }
  } catch (error: any) {
    message.error(error.message || '获取列表失败')
  } finally {
    loading.value = false
  }
}

const fetchFeishuApps = async () => {
  try {
    const response = await feishuAppApi.list()
    if (response.code === 0 && response.data) {
      feishuAppOptions.value = response.data.list.map(app => ({
        label: `${app.name} (${app.project})`,
        value: app.id!
      }))
    }
  } catch {}
}

const showModal = async (record?: JenkinsInstanceWithApps) => {
  await fetchFeishuApps()
  if (record) {
    editingId.value = record.id
    Object.assign(form, {
      name: record.name,
      url: record.url,
      username: record.username,
      api_token: '',
      description: record.description,
      status: record.status,
      is_default: record.is_default,
      feishu_app_ids: record.feishu_apps?.map(app => app.id) || []
    })
  } else {
    editingId.value = null
    Object.assign(form, {
      name: '',
      url: '',
      username: '',
      api_token: '',
      description: '',
      status: 'active',
      is_default: false,
      feishu_app_ids: []
    })
  }
  modalVisible.value = true
}

const handleSubmit = async () => {
  if (!form.name || !form.url) {
    message.error('请填写必填项')
    return
  }

  submitting.value = true
  try {
    let instanceId: number
    if (editingId.value) {
      await jenkinsInstanceApi.updateInstance(editingId.value, form)
      instanceId = editingId.value
      message.success('更新成功')
    } else {
      const res = await jenkinsInstanceApi.createInstance(form)
      instanceId = res.data?.id || 0
      message.success('创建成功')
    }
    // 绑定飞书应用
    if (instanceId && form.feishu_app_ids) {
      await jenkinsInstanceApi.bindFeishuApps(instanceId, form.feishu_app_ids)
    }
    modalVisible.value = false
    fetchInstances()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: number) => {
  try {
    await jenkinsInstanceApi.deleteInstance(id)
    message.success('删除成功')
    fetchInstances()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

const setDefault = async (record: JenkinsInstance) => {
  try {
    await jenkinsInstanceApi.setDefaultInstance(record.id)
    message.success('设置成功')
    fetchInstances()
  } catch (error: any) {
    message.error(error.message || '设置失败')
  }
}

const testConnection = async (record: JenkinsInstance) => {
  if (testingIds.value.has(record.id)) return
  testingIds.value.add(record.id)
  try {
    const response = await jenkinsInstanceApi.testConnection(record.id)
    if (response.data?.connected) {
      message.success(`连接成功！Jenkins 版本: ${response.data.version || '未知'}，响应时间: ${response.data.response_time_ms}ms`)
    } else {
      message.error(`连接失败: ${response.data?.error || '未知错误'}`)
    }
  } catch (error: any) {
    message.error(error.message || '测试连接失败')
  } finally {
    testingIds.value.delete(record.id)
  }
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  fetchInstances()
}

onMounted(() => {
  fetchInstances()
})
</script>

<style scoped>
.jenkins-instances {
  padding: 0;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.page-header h1 {
  font-size: 20px;
  font-weight: 500;
  margin: 0;
}

.testing {
  color: #1890ff;
  cursor: wait;
}
</style>
