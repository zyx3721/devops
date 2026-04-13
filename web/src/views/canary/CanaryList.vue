<template>
  <div class="canary-list">
    <a-page-header title="灰度发布" sub-title="管理所有灰度发布任务" />

    <!-- Istio 依赖提示 -->
    <a-alert style="margin-bottom: 16px" type="info" show-icon>
      <template #message>
        <span>灰度发布基于 <a-typography-text strong>Istio</a-typography-text> 实现流量控制</span>
      </template>
      <template #description>
        <span>请确保目标 K8s 集群已安装 Istio，且应用已注入 Envoy Sidecar。灰度发布通过 VirtualService 按权重分配流量到不同版本。</span>
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
            <a-select-option value="canary_running">进行中</a-select-option>
            <a-select-option value="success">已完成</a-select-option>
            <a-select-option value="rolled_back">已回滚</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-space>
            <a-button type="primary" @click="fetchList">查询</a-button>
            <a-button @click="resetFilters">重置</a-button>
            <a-button type="primary" @click="showCreateModal = true">
              <PlusOutlined /> 新建灰度
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
          <template v-else-if="column.key === 'progress'">
            <a-progress :percent="record.canary_percent || 0" :status="record.status === 'canary_running' ? 'active' : 'normal'" size="small" style="width: 120px" />
          </template>
          <template v-else-if="column.key === 'image_tag'">
            <a-typography-text code>{{ record.image_tag }}</a-typography-text>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="viewDetail(record)">详情</a-button>
              <template v-if="record.status === 'canary_running'">
                <a-button type="link" size="small" @click="adjustCanary(record)">调整</a-button>
                <a-popconfirm title="确定全量发布?" @confirm="promoteCanary(record)">
                  <a-button type="link" size="small" style="color: #52c41a">全量</a-button>
                </a-popconfirm>
                <a-popconfirm title="确定回滚?" @confirm="rollbackCanary(record)">
                  <a-button type="link" size="small" danger>回滚</a-button>
                </a-popconfirm>
              </template>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 新建灰度弹窗 -->
    <a-modal v-model:open="showCreateModal" title="新建灰度发布" width="650px" @ok="handleCreate" :confirm-loading="creating">
      <a-form :model="createForm" :label-col="{ span: 5 }" :wrapper-col="{ span: 18 }">
        <a-form-item label="应用" required>
          <a-select v-model:value="createForm.application_id" placeholder="选择应用" show-search @change="onAppChange">
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
        <a-form-item label="镜像标签" required>
          <a-input v-model:value="createForm.image_tag" placeholder="新版本镜像标签，如 v1.2.0" />
        </a-form-item>
        <a-form-item label="灰度策略">
          <a-radio-group v-model:value="createForm.canary_strategy">
            <a-radio-button value="weight">按权重</a-radio-button>
            <a-radio-button value="header">按Header</a-radio-button>
            <a-radio-button value="cookie">按Cookie</a-radio-button>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="流量比例" v-if="createForm.canary_strategy === 'weight'">
          <a-row :gutter="16">
            <a-col :span="16">
              <a-slider v-model:value="createForm.canary_percent" :min="1" :max="100" :marks="{ 10: '10%', 30: '30%', 50: '50%', 100: '100%' }" />
            </a-col>
            <a-col :span="8">
              <a-input-number v-model:value="createForm.canary_percent" :min="1" :max="100" :formatter="(v: number) => `${v}%`" style="width: 100%" />
            </a-col>
          </a-row>
          <div style="color: #999; font-size: 12px; margin-top: 4px">按流量百分比随机分配到灰度版本</div>
        </a-form-item>
        <a-form-item label="Header名称" v-if="createForm.canary_strategy === 'header'">
          <a-input v-model:value="createForm.canary_header" placeholder="如: X-Canary" />
        </a-form-item>
        <a-form-item label="Header值" v-if="createForm.canary_strategy === 'header'">
          <a-input v-model:value="createForm.canary_header_value" placeholder="如: always" />
          <div style="color: #999; font-size: 12px; margin-top: 4px">请求头 {{ createForm.canary_header }}={{ createForm.canary_header_value }} 时路由到灰度版本</div>
        </a-form-item>
        <a-form-item label="Cookie名称" v-if="createForm.canary_strategy === 'cookie'">
          <a-input v-model:value="createForm.canary_cookie" placeholder="如: canary" />
          <div style="color: #999; font-size: 12px; margin-top: 4px">Cookie {{ createForm.canary_cookie }}=always 时路由到灰度版本</div>
        </a-form-item>
        <a-form-item label="发布说明">
          <a-textarea v-model:value="createForm.description" :rows="3" placeholder="本次发布说明" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 详情抽屉 -->
    <a-drawer v-model:open="showDetail" title="灰度发布详情" width="700" placement="right">
      <template v-if="currentCanary">
        <CanaryDetail :record="currentCanary" @refresh="refreshDetail" @close="showDetail = false" />
      </template>
    </a-drawer>

    <!-- 调整灰度弹窗 -->
    <a-modal v-model:open="showAdjustModal" title="调整灰度比例" @ok="handleAdjust" :confirm-loading="adjusting">
      <a-form :label-col="{ span: 6 }">
        <a-form-item label="当前比例">
          <a-tag color="blue">{{ adjustForm.current_percent }}%</a-tag>
        </a-form-item>
        <a-form-item label="新比例">
          <a-slider v-model:value="adjustForm.new_percent" :min="1" :max="100" :marks="{ 10: '10%', 30: '30%', 50: '50%', 100: '100%' }" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { applicationApi } from '@/services/application'
import { canaryApi } from '@/services/canary'
import CanaryDetail from './CanaryDetail.vue'

const loading = ref(false)
const creating = ref(false)
const adjusting = ref(false)
const showCreateModal = ref(false)
const showDetail = ref(false)
const showAdjustModal = ref(false)

const applications = ref<any[]>([])
const list = ref<any[]>([])
const currentCanary = ref<any>(null)

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
  image_tag: '',
  canary_percent: 10,
  canary_strategy: 'weight' as 'weight' | 'header' | 'cookie',
  canary_header: 'X-Canary',
  canary_header_value: 'always',
  canary_cookie: 'canary',
  description: ''
})

const adjustForm = reactive({
  record_id: 0,
  current_percent: 0,
  new_percent: 30
})

const columns = [
  { title: '应用', key: 'app_name', dataIndex: 'app_name' },
  { title: '环境', key: 'env_name', dataIndex: 'env_name', width: 80 },
  { title: '镜像标签', key: 'image_tag', dataIndex: 'image_tag' },
  { title: '灰度进度', key: 'progress', width: 150 },
  { title: '状态', key: 'status', dataIndex: 'status', width: 100 },
  { title: '创建时间', key: 'created_at', dataIndex: 'created_at', width: 160 },
  { title: '操作', key: 'action', width: 200 }
]

const fetchApplications = async () => {
  try {
    const res = await applicationApi.list({ page: 1, page_size: 1000 })
    if (res?.code === 0) {
      applications.value = res.data?.list || []
    }
  } catch (error) {
    console.error('获取应用列表失败')
  }
}

const fetchList = async () => {
  loading.value = true
  try {
    const res = await canaryApi.list({
      page: pagination.current,
      page_size: pagination.pageSize,
      ...filters
    })
    if (res?.code === 0) {
      list.value = res.data?.list || []
      pagination.total = res.data?.total || 0
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

const onAppChange = () => {
  // 可以在这里加载应用的最新镜像标签
}

const handleCreate = async () => {
  if (!createForm.application_id || !createForm.image_tag) {
    message.error('请填写必要信息')
    return
  }
  creating.value = true
  try {
    const data: any = {
      application_id: createForm.application_id,
      env_name: createForm.env_name,
      image_tag: createForm.image_tag,
      canary_percent: createForm.canary_percent,
      description: createForm.description
    }
    // 根据策略添加对应参数
    if (createForm.canary_strategy === 'header') {
      data.canary_header = createForm.canary_header
      data.canary_header_value = createForm.canary_header_value
    } else if (createForm.canary_strategy === 'cookie') {
      data.canary_cookie = createForm.canary_cookie
    }
    const res = await canaryApi.start(data)
    if (res?.code === 0) {
      message.success('灰度发布已启动')
      showCreateModal.value = false
      resetCreateForm()
      fetchList()
    } else {
      message.error(res?.message || '启动失败')
    }
  } catch (error) {
    message.error('启动失败')
  } finally {
    creating.value = false
  }
}

const resetCreateForm = () => {
  createForm.application_id = undefined
  createForm.env_name = 'dev'
  createForm.image_tag = ''
  createForm.canary_percent = 10
  createForm.canary_strategy = 'weight'
  createForm.canary_header = 'X-Canary'
  createForm.canary_header_value = 'always'
  createForm.canary_cookie = 'canary'
  createForm.description = ''
}

const viewDetail = async (record: any) => {
  currentCanary.value = record
  showDetail.value = true
}

const refreshDetail = async () => {
  if (currentCanary.value) {
    try {
      const res = await canaryApi.getStatus(currentCanary.value.id)
      if (res?.code === 0) {
        currentCanary.value = { ...currentCanary.value, ...res.data }
      }
    } catch (error) {
      // ignore
    }
  }
  fetchList()
}

const adjustCanary = (record: any) => {
  adjustForm.record_id = record.id
  adjustForm.current_percent = record.canary_percent || 10
  adjustForm.new_percent = Math.min(adjustForm.current_percent + 20, 100)
  showAdjustModal.value = true
}

const handleAdjust = async () => {
  adjusting.value = true
  try {
    const res = await canaryApi.adjust(adjustForm.record_id, adjustForm.new_percent)
    if (res?.code === 0) {
      message.success('灰度比例已调整')
      showAdjustModal.value = false
      fetchList()
    } else {
      message.error(res?.message || '调整失败')
    }
  } catch (error) {
    message.error('调整失败')
  } finally {
    adjusting.value = false
  }
}

const promoteCanary = async (record: any) => {
  try {
    const res = await canaryApi.promote(record.id)
    if (res?.code === 0) {
      message.success('全量发布成功')
      fetchList()
    } else {
      message.error(res?.message || '全量发布失败')
    }
  } catch (error) {
    message.error('全量发布失败')
  }
}

const rollbackCanary = async (record: any) => {
  try {
    const res = await canaryApi.rollback(record.id)
    if (res?.code === 0) {
      message.success('回滚成功')
      fetchList()
    } else {
      message.error(res?.message || '回滚失败')
    }
  } catch (error) {
    message.error('回滚失败')
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
    canary_running: 'processing',
    success: 'success',
    rolled_back: 'warning',
    failed: 'error'
  }
  return map[status] || 'default'
}

const getStatusText = (status: string) => {
  const map: Record<string, string> = {
    canary_running: '进行中',
    success: '已完成',
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
.canary-list {
  padding: 16px;
}
</style>
