<template>
  <div class="k8s-clusters">
    <div class="page-header">
      <h1>K8s 集群管理</h1>
      <a-button type="primary" @click="showModal()">
        <template #icon><PlusOutlined /></template>
        新增集群
      </a-button>
    </div>

    <a-table :columns="columns" :data-source="clusters" :loading="loading" :pagination="pagination" @change="handleTableChange" row-key="id">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <a-tag :color="record.status === 'active' ? 'green' : 'red'">
            {{ record.status === 'active' ? '启用' : '禁用' }}
          </a-tag>
        </template>
        <template v-if="column.key === 'is_default'">
          <a-tag v-if="record.is_default" color="blue">默认</a-tag>
          <span v-else>-</span>
        </template>
        <template v-if="column.key === 'name'">
          <a @click="goToResources(record)">{{ record.name }}</a>
        </template>
        <template v-if="column.key === 'feishu_apps'">
          <a-tag v-for="app in record.feishu_apps" :key="app.id" color="purple" style="margin: 2px;">
            {{ app.name }}
          </a-tag>
          <span v-if="!record.feishu_apps?.length">-</span>
        </template>
        <template v-if="column.key === 'action'">
          <a-space :size="4">
            <a @click="testConnection(record)" :class="{ 'testing': testingIds.has(record.id) }">
              <LoadingOutlined v-if="testingIds.has(record.id)" />
              测试连接
            </a>
            <a-divider type="vertical" />
            <a @click="goToResources(record)">资源</a>
            <a-divider type="vertical" />
            <a @click="showModal(record)">编辑</a>
            <a-divider type="vertical" />
            <a @click="setDefault(record)" v-if="!record.is_default">设为默认</a>
            <a-divider type="vertical" v-if="!record.is_default" />
            <a-popconfirm title="确定删除？" @confirm="handleDelete(record.id)">
              <a style="color: #ff4d4f">删除</a>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="modalVisible" :title="editingId ? '编辑集群' : '新增集群'" @ok="handleSubmit" :confirm-loading="submitting" width="600px">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="名称" required>
          <a-input v-model:value="form.name" placeholder="请输入名称" />
        </a-form-item>
        <a-form-item label="Kubeconfig" required>
          <a-textarea v-model:value="form.kubeconfig" placeholder="请输入 Kubeconfig 内容" :rows="6" />
        </a-form-item>
        <a-form-item label="飞书应用">
          <a-select v-model:value="form.feishu_app_ids" mode="multiple" placeholder="请选择飞书应用（可选）" :options="feishuAppOptions" allow-clear />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="form.description" placeholder="请输入描述" :rows="2" />
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

    <a-modal v-model:open="feishuModalVisible" title="绑定飞书应用" @ok="handleBindFeishu" :confirm-loading="feishuSubmitting">
      <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="飞书应用">
          <a-select v-model:value="selectedFeishuApps" mode="multiple" placeholder="请选择飞书应用" :options="feishuAppOptions" />
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
import { k8sClusterApi, type FeishuAppSimple } from '@/services/k8s'
import { feishuAppApi } from '@/services/feishu'
import type { K8sCluster } from '@/types'

interface K8sClusterWithApps extends K8sCluster {
  feishu_apps?: FeishuAppSimple[]
}

const router = useRouter()
const loading = ref(false)
const submitting = ref(false)
const modalVisible = ref(false)
const editingId = ref<number | null>(null)
const clusters = ref<K8sClusterWithApps[]>([])
const testingIds = ref<Set<number>>(new Set())

const feishuModalVisible = ref(false)
const feishuSubmitting = ref(false)
const selectedFeishuApps = ref<number[]>([])
const feishuAppOptions = ref<{ label: string; value: number }[]>([])
const currentClusterId = ref<number | null>(null)

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0
})

const form = reactive({
  name: '',
  kubeconfig: '',
  description: '',
  status: 'active',
  is_default: false,
  feishu_app_ids: [] as number[]
})

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '飞书应用', key: 'feishu_apps', width: 200 },
  { title: '状态', key: 'status' },
  { title: '默认', key: 'is_default' },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at' },
  { title: '操作', key: 'action', width: 340 }
]

const goToResources = (record: K8sCluster) => {
  router.push(`/k8s/clusters/${record.id}/resources`)
}

const fetchClusters = async () => {
  loading.value = true
  try {
    const response = await k8sClusterApi.getClusters({
      page: pagination.current,
      page_size: pagination.pageSize
    })
    if (response.code === 0 && response.data) {
      const items = response.data.items
      // 获取每个集群绑定的飞书应用
      for (const item of items) {
        try {
          const appsRes = await k8sClusterApi.getFeishuApps(item.id)
          if (appsRes.code === 0) {
            (item as K8sClusterWithApps).feishu_apps = appsRes.data
          }
        } catch {}
      }
      clusters.value = items
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

const showModal = async (record?: K8sClusterWithApps) => {
  await fetchFeishuApps()
  if (record) {
    editingId.value = record.id
    Object.assign(form, {
      name: record.name,
      kubeconfig: '',
      description: record.description,
      status: record.status,
      is_default: record.is_default,
      feishu_app_ids: record.feishu_apps?.map(app => app.id) || []
    })
  } else {
    editingId.value = null
    Object.assign(form, {
      name: '',
      kubeconfig: '',
      description: '',
      status: 'active',
      is_default: false,
      feishu_app_ids: []
    })
  }
  modalVisible.value = true
}

const showFeishuModal = async (record: K8sClusterWithApps) => {
  currentClusterId.value = record.id
  selectedFeishuApps.value = record.feishu_apps?.map(app => app.id) || []
  await fetchFeishuApps()
  feishuModalVisible.value = true
}

const handleBindFeishu = async () => {
  if (!currentClusterId.value) return
  feishuSubmitting.value = true
  try {
    await k8sClusterApi.bindFeishuApps(currentClusterId.value, selectedFeishuApps.value)
    message.success('绑定成功')
    feishuModalVisible.value = false
    fetchClusters()
  } catch (error: any) {
    message.error(error.message || '绑定失败')
  } finally {
    feishuSubmitting.value = false
  }
}

const handleSubmit = async () => {
  if (!form.name || !form.kubeconfig) {
    message.error('请填写必填项')
    return
  }

  submitting.value = true
  try {
    let clusterId: number
    if (editingId.value) {
      await k8sClusterApi.updateCluster(editingId.value, form)
      clusterId = editingId.value
      message.success('更新成功')
    } else {
      const res = await k8sClusterApi.createCluster(form)
      clusterId = res.data?.id || 0
      message.success('创建成功')
    }
    // 绑定飞书应用
    if (clusterId && form.feishu_app_ids) {
      await k8sClusterApi.bindFeishuApps(clusterId, form.feishu_app_ids)
    }
    modalVisible.value = false
    fetchClusters()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: number) => {
  try {
    await k8sClusterApi.deleteCluster(id)
    message.success('删除成功')
    fetchClusters()
  } catch (error: any) {
    message.error(error.message || '删除失败')
  }
}

const setDefault = async (record: K8sCluster) => {
  try {
    await k8sClusterApi.setDefaultCluster(record.id)
    message.success('设置成功')
    fetchClusters()
  } catch (error: any) {
    message.error(error.message || '设置失败')
  }
}

const testConnection = async (record: K8sCluster) => {
  if (testingIds.value.has(record.id)) return
  testingIds.value.add(record.id)
  try {
    const response = await k8sClusterApi.testConnection(record.id)
    if (response.data?.connected) {
      message.success(`连接成功！K8s 版本: ${response.data.server_version || '未知'}，节点数: ${response.data.node_count || 0}，响应时间: ${response.data.response_time_ms}ms`)
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
  fetchClusters()
}

onMounted(() => {
  fetchClusters()
})
</script>

<style scoped>
.k8s-clusters {
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
