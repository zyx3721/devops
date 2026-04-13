<template>
  <div class="bluegreen-list">
    <a-page-header title="蓝绿部署" sub-title="管理蓝绿部署任务" />

    <!-- Istio 依赖提示 -->
    <a-alert style="margin-bottom: 16px" type="info" show-icon>
      <template #message>
        <span>蓝绿部署基于 <a-typography-text strong>Istio</a-typography-text> 实现流量切换</span>
      </template>
      <template #description>
        <span>请确保目标 K8s 集群已安装 Istio，且应用已注入 Envoy Sidecar。蓝绿部署通过 VirtualService 实现 100% 流量瞬时切换。</span>
      </template>
    </a-alert>

    <!-- 筛选 -->
    <a-card style="margin-bottom: 16px">
      <a-form layout="inline">
        <a-form-item label="应用">
          <a-select v-model:value="filters.application_id" style="width: 180px" placeholder="全部应用" allow-clear show-search>
            <a-select-option v-for="app in applications" :key="app.id" :value="app.id">{{ app.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="环境">
          <a-select v-model:value="filters.env_name" style="width: 120px" placeholder="全部" allow-clear>
            <a-select-option value="dev">开发</a-select-option>
            <a-select-option value="test">测试</a-select-option>
            <a-select-option value="staging">预发</a-select-option>
            <a-select-option value="prod">生产</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="filters.status" style="width: 120px" placeholder="全部" allow-clear>
            <a-select-option value="pending">待切换</a-select-option>
            <a-select-option value="switched">已切换</a-select-option>
            <a-select-option value="rolled_back">已回滚</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" @click="fetchList">查询</a-button>
            <a-button @click="resetFilters">重置</a-button>
            <a-button type="primary" @click="showCreateModal = true">
              <PlusOutlined /> 新建蓝绿部署
            </a-button>
          </a-space>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 列表 -->
    <a-card>
      <a-table :columns="columns" :data-source="list" :loading="loading" :pagination="pagination" @change="handleTableChange" row-key="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'app_name'">
            <a-tag color="blue">{{ record.app_name }}</a-tag>
          </template>
          <template v-else-if="column.key === 'env_name'">
            <a-tag :color="getEnvColor(record.env_name)">{{ getEnvLabel(record.env_name) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-badge :status="getStatusBadge(record.status)" :text="getStatusText(record.status)" />
          </template>
          <template v-else-if="column.key === 'active_version'">
            <a-tag :color="record.active_version === 'blue' ? 'blue' : 'green'">
              {{ record.active_version === 'blue' ? '蓝版本' : '绿版本' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'versions'">
            <a-space direction="vertical" size="small">
              <div><a-tag color="blue">蓝:</a-tag> <a-typography-text code>{{ record.blue_image_tag }}</a-typography-text></div>
              <div><a-tag color="green">绿:</a-tag> <a-typography-text code>{{ record.green_image_tag || '-' }}</a-typography-text></div>
            </a-space>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="viewDetail(record)">详情</a-button>
              <template v-if="record.status === 'pending'">
                <a-popconfirm title="确定切换流量到新版本?" @confirm="switchVersion(record)">
                  <a-button type="link" size="small" style="color: #52c41a">切换</a-button>
                </a-popconfirm>
                <a-popconfirm title="确定回滚?" @confirm="rollbackDeploy(record)">
                  <a-button type="link" size="small" danger>回滚</a-button>
                </a-popconfirm>
              </template>
              <template v-else-if="record.status === 'switched'">
                <a-popconfirm title="确定清理旧版本资源?" @confirm="cleanup(record)">
                  <a-button type="link" size="small">清理</a-button>
                </a-popconfirm>
              </template>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 新建蓝绿部署弹窗 -->
    <a-modal v-model:open="showCreateModal" title="新建蓝绿部署" width="600px" @ok="handleCreate" :confirm-loading="creating">
      <a-form :model="createForm" :label-col="{ span: 5 }" :wrapper-col="{ span: 18 }">
        <a-form-item label="应用" required>
          <a-select v-model:value="createForm.application_id" placeholder="选择应用" show-search>
            <a-select-option v-for="app in applications" :key="app.id" :value="app.id">{{ app.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="环境" required>
          <a-select v-model:value="createForm.env_name">
            <a-select-option value="dev">开发</a-select-option>
            <a-select-option value="test">测试</a-select-option>
            <a-select-option value="staging">预发</a-select-option>
            <a-select-option value="prod">生产</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="新版本镜像" required>
          <a-input v-model:value="createForm.green_image_tag" placeholder="新版本镜像标签，如 v1.2.0" />
        </a-form-item>
        <a-form-item label="副本数">
          <a-input-number v-model:value="createForm.replicas" :min="1" :max="100" style="width: 100%" />
        </a-form-item>
        <a-form-item label="发布说明">
          <a-textarea v-model:value="createForm.description" :rows="3" placeholder="本次发布说明" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 详情抽屉 -->
    <a-drawer v-model:open="showDetail" title="蓝绿部署详情" width="650" placement="right">
      <template v-if="currentDeploy">
        <a-descriptions :column="1" bordered size="small">
          <a-descriptions-item label="应用">{{ currentDeploy.app_name }}</a-descriptions-item>
          <a-descriptions-item label="环境">{{ getEnvLabel(currentDeploy.env_name) }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-badge :status="getStatusBadge(currentDeploy.status)" :text="getStatusText(currentDeploy.status)" />
          </a-descriptions-item>
          <a-descriptions-item label="当前版本">
            <a-tag :color="currentDeploy.active_version === 'blue' ? 'blue' : 'green'">
              {{ currentDeploy.active_version === 'blue' ? '蓝版本' : '绿版本' }}
            </a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="蓝版本镜像">
            <a-typography-text code>{{ currentDeploy.blue_image_tag }}</a-typography-text>
          </a-descriptions-item>
          <a-descriptions-item label="绿版本镜像">
            <a-typography-text code>{{ currentDeploy.green_image_tag || '-' }}</a-typography-text>
          </a-descriptions-item>
          <a-descriptions-item label="副本数">{{ currentDeploy.replicas }}</a-descriptions-item>
          <a-descriptions-item label="创建时间">{{ formatTime(currentDeploy.created_at) }}</a-descriptions-item>
          <a-descriptions-item label="切换时间">{{ formatTime(currentDeploy.switched_at) }}</a-descriptions-item>
          <a-descriptions-item label="操作人">{{ currentDeploy.operator }}</a-descriptions-item>
          <a-descriptions-item label="说明">{{ currentDeploy.description || '-' }}</a-descriptions-item>
        </a-descriptions>

        <a-divider>操作历史</a-divider>
        <a-timeline>
          <a-timeline-item v-for="(event, idx) in currentDeploy.events || []" :key="idx" :color="event.type === 'error' ? 'red' : 'blue'">
            <p>{{ event.message }}</p>
            <p style="color: #999; font-size: 12px">{{ formatTime(event.time) }}</p>
          </a-timeline-item>
        </a-timeline>
      </template>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { applicationApi } from '@/services/application'
import { blueGreenApi } from '@/services/bluegreen'

const loading = ref(false)
const creating = ref(false)
const showCreateModal = ref(false)
const showDetail = ref(false)

const applications = ref<any[]>([])
const list = ref<any[]>([])
const currentDeploy = ref<any>(null)

const filters = reactive({
  application_id: undefined as number | undefined,
  env_name: '',
  status: ''
})

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0
})

const createForm = reactive({
  application_id: undefined as number | undefined,
  env_name: 'dev',
  green_image_tag: '',
  replicas: 2,
  description: ''
})

const columns = [
  { title: '应用', key: 'app_name', dataIndex: 'app_name' },
  { title: '环境', key: 'env_name', dataIndex: 'env_name', width: 80 },
  { title: '版本', key: 'versions', width: 200 },
  { title: '当前版本', key: 'active_version', width: 100 },
  { title: '状态', key: 'status', dataIndex: 'status', width: 100 },
  { title: '创建时间', key: 'created_at', dataIndex: 'created_at', width: 160 },
  { title: '操作', key: 'action', width: 180 }
]

const fetchApplications = async () => {
  try {
    const res = await applicationApi.list({ page: 1, page_size: 1000 })
    if (res?.data) {
      applications.value = res.data.list || []
    }
  } catch (error) {
    console.error('获取应用列表失败')
  }
}

const fetchList = async () => {
  loading.value = true
  try {
    const res = await blueGreenApi.list({
      page: pagination.current,
      page_size: pagination.pageSize,
      ...filters
    })
    if (res?.data) {
      list.value = res.data.list || []
      pagination.total = res.data.total || 0
    }
  } catch (error) {
    message.error('获取列表失败')
  } finally {
    loading.value = false
  }
}

const resetFilters = () => {
  filters.application_id = undefined
  filters.env_name = ''
  filters.status = ''
  pagination.current = 1
  fetchList()
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  fetchList()
}

const handleCreate = async () => {
  if (!createForm.application_id || !createForm.green_image_tag) {
    message.error('请填写必要信息')
    return
  }
  creating.value = true
  try {
    const res = await blueGreenApi.start({
      application_id: createForm.application_id,
      env_name: createForm.env_name,
      green_image_tag: createForm.green_image_tag,
      replicas: createForm.replicas,
      description: createForm.description
    })
    if (res?.data) {
      message.success('蓝绿部署已创建')
      showCreateModal.value = false
      resetCreateForm()
      fetchList()
    }
  } catch (error) {
    message.error('创建失败')
  } finally {
    creating.value = false
  }
}

const resetCreateForm = () => {
  createForm.application_id = undefined
  createForm.env_name = 'dev'
  createForm.green_image_tag = ''
  createForm.replicas = 2
  createForm.description = ''
}

const viewDetail = (record: any) => {
  currentDeploy.value = record
  showDetail.value = true
}

const switchVersion = async (record: any) => {
  try {
    const res = await blueGreenApi.switch(record.id)
    if (res) {
      message.success('流量切换成功')
      fetchList()
    }
  } catch (error) {
    message.error('切换失败')
  }
}

const rollbackDeploy = async (record: any) => {
  try {
    const res = await blueGreenApi.rollback(record.id)
    if (res) {
      message.success('回滚成功')
      fetchList()
    }
  } catch (error) {
    message.error('回滚失败')
  }
}

const cleanup = async (record: any) => {
  try {
    const res = await blueGreenApi.cleanup(record.id)
    if (res) {
      message.success('清理成功')
      fetchList()
    }
  } catch (error) {
    message.error('清理失败')
  }
}

const getEnvColor = (env: string) => {
  const map: Record<string, string> = { dev: 'green', test: 'blue', staging: 'orange', prod: 'red' }
  return map[env] || 'default'
}

const getEnvLabel = (env: string) => {
  const map: Record<string, string> = { dev: '开发', test: '测试', staging: '预发', prod: '生产' }
  return map[env] || env
}

const getStatusBadge = (status: string) => {
  const map: Record<string, 'processing' | 'success' | 'warning' | 'error' | 'default'> = {
    pending: 'processing',
    switched: 'success',
    rolled_back: 'warning',
    failed: 'error'
  }
  return map[status] || 'default'
}

const getStatusText = (status: string) => {
  const map: Record<string, string> = {
    pending: '待切换',
    switched: '已切换',
    rolled_back: '已回滚',
    failed: '失败'
  }
  return map[status] || status
}

const formatTime = (time: string) => {
  if (!time) return '-'
  return time.replace('T', ' ').substring(0, 19)
}

onMounted(() => {
  fetchApplications()
  fetchList()
})
</script>

<style scoped>
.bluegreen-list {
  padding: 16px;
}
</style>
