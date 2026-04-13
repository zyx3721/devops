<template>
  <div class="deploy-history">
    <div class="page-header">
      <h1>部署记录</h1>
    </div>

    <!-- 统计卡片 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-card>
          <a-statistic title="今日部署" :value="todayCount" :value-style="{ color: '#1890ff' }">
            <template #prefix><RocketOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="成功" :value="successCount" :value-style="{ color: '#52c41a' }">
            <template #prefix><CheckCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="失败" :value="failedCount" :value-style="{ color: '#cf1322' }">
            <template #prefix><CloseCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="运行中" :value="runningCount" :value-style="{ color: '#fa8c16' }">
            <template #prefix><SyncOutlined spin /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 筛选 -->
    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-form layout="inline">
        <a-form-item label="应用">
          <a-input v-model:value="filter.app_name" placeholder="应用名称" allow-clear style="width: 150px" @pressEnter="fetchRecords" />
        </a-form-item>
        <a-form-item label="环境">
          <a-select v-model:value="filter.env" placeholder="全部" allow-clear style="width: 100px">
            <a-select-option value="dev">dev</a-select-option>
            <a-select-option value="test">test</a-select-option>
            <a-select-option value="staging">staging</a-select-option>
            <a-select-option value="prod">prod</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="filter.status" placeholder="全部" allow-clear style="width: 100px">
            <a-select-option value="pending">等待中</a-select-option>
            <a-select-option value="running">运行中</a-select-option>
            <a-select-option value="success">成功</a-select-option>
            <a-select-option value="failed">失败</a-select-option>
            <a-select-option value="cancelled">已取消</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="fetchRecords">查询</a-button>
          <a-button style="margin-left: 8px" @click="resetFilter">重置</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 部署记录列表 -->
    <a-card :bordered="false">
      <a-table :columns="columns" :data-source="records" :loading="loading" row-key="id" :pagination="pagination" @change="onTableChange">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          <template v-if="column.key === 'app_name'">
            <a @click="goToApp(record)">{{ record.app_name }}</a>
          </template>
          <template v-if="column.key === 'env_name'">
            <a-tag :color="getEnvColor(record.env_name)">{{ record.env_name }}</a-tag>
          </template>
          <template v-if="column.key === 'deploy_type'">
            <a-tag :color="getDeployTypeColor(record.deploy_type)">{{ getDeployTypeLabel(record.deploy_type) }}</a-tag>
          </template>
          <template v-if="column.key === 'status'">
            <a-badge :status="getStatusType(record.status)" :text="getStatusText(record.status)" />
          </template>
          <template v-if="column.key === 'duration'">
            {{ record.duration ? `${record.duration}s` : '-' }}
          </template>
          <template v-if="column.key === 'jenkins_url'">
            <a v-if="record.jenkins_url" :href="record.jenkins_url" target="_blank">
              #{{ record.jenkins_build }}
            </a>
            <span v-else>-</span>
          </template>
          <template v-if="column.key === 'action'">
            <a-button type="link" size="small" @click="showDetail(record)">详情</a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 详情抽屉 -->
    <a-drawer v-model:open="detailVisible" title="部署详情" :width="550">
      <template v-if="currentRecord">
        <a-descriptions :column="2" bordered size="small">
          <a-descriptions-item label="应用">{{ currentRecord.app_name }}</a-descriptions-item>
          <a-descriptions-item label="环境"><a-tag :color="getEnvColor(currentRecord.env_name)">{{ currentRecord.env_name }}</a-tag></a-descriptions-item>
          <a-descriptions-item label="版本">{{ currentRecord.version || '-' }}</a-descriptions-item>
          <a-descriptions-item label="分支">{{ currentRecord.branch || '-' }}</a-descriptions-item>
          <a-descriptions-item label="Commit">{{ currentRecord.commit_id ? currentRecord.commit_id.substring(0, 8) : '-' }}</a-descriptions-item>
          <a-descriptions-item label="类型"><a-tag :color="getDeployTypeColor(currentRecord.deploy_type)">{{ getDeployTypeLabel(currentRecord.deploy_type) }}</a-tag></a-descriptions-item>
          <a-descriptions-item label="状态"><a-badge :status="getStatusType(currentRecord.status)" :text="getStatusText(currentRecord.status)" /></a-descriptions-item>
          <a-descriptions-item label="耗时">{{ currentRecord.duration ? `${currentRecord.duration}s` : '-' }}</a-descriptions-item>
          <a-descriptions-item label="操作人">{{ currentRecord.operator || '-' }}</a-descriptions-item>
          <a-descriptions-item label="Jenkins 构建">
            <a v-if="currentRecord.jenkins_url" :href="currentRecord.jenkins_url" target="_blank">#{{ currentRecord.jenkins_build }}</a>
            <span v-else>-</span>
          </a-descriptions-item>
          <a-descriptions-item label="开始时间" :span="2">{{ formatTime(currentRecord.started_at) }}</a-descriptions-item>
          <a-descriptions-item label="结束时间" :span="2">{{ formatTime(currentRecord.finished_at) }}</a-descriptions-item>
        </a-descriptions>
        <template v-if="currentRecord.error_msg">
          <a-divider orientation="left">错误信息</a-divider>
          <a-alert type="error" :message="currentRecord.error_msg" />
        </template>
      </template>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { RocketOutlined, CheckCircleOutlined, CloseCircleOutlined, SyncOutlined } from '@ant-design/icons-vue'
import { applicationApi, type DeployRecord } from '@/services/application'

const router = useRouter()
const loading = ref(false)
const detailVisible = ref(false)
const records = ref<DeployRecord[]>([])
const currentRecord = ref<DeployRecord | null>(null)

const filter = reactive({ app_name: '', env: '', status: '' })
const pagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true })

const todayCount = computed(() => records.value.filter(r => isToday(r.created_at)).length)
const successCount = computed(() => records.value.filter(r => r.status === 'success').length)
const failedCount = computed(() => records.value.filter(r => r.status === 'failed').length)
const runningCount = computed(() => records.value.filter(r => r.status === 'running').length)

const columns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '应用', dataIndex: 'app_name', key: 'app_name', width: 150 },
  { title: '环境', dataIndex: 'env_name', key: 'env_name', width: 80 },
  { title: '版本', dataIndex: 'version', key: 'version', width: 120 },
  { title: '类型', dataIndex: 'deploy_type', key: 'deploy_type', width: 80 },
  { title: '操作人', dataIndex: 'operator', key: 'operator', width: 80 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 90 },
  { title: '耗时', dataIndex: 'duration', key: 'duration', width: 70 },
  { title: 'Jenkins', key: 'jenkins_url', width: 80 },
  { title: '操作', key: 'action', width: 70 }
]

const envColors: Record<string, string> = { dev: 'blue', test: 'cyan', staging: 'orange', prod: 'red' }
const deployTypeLabels: Record<string, string> = { deploy: '部署', rollback: '回滚', restart: '重启', scale: '扩缩容' }
const deployTypeColors: Record<string, string> = { deploy: 'blue', rollback: 'orange', restart: 'cyan', scale: 'purple' }
const statusTypes: Record<string, string> = { pending: 'default', running: 'processing', success: 'success', failed: 'error', cancelled: 'warning' }
const statusTexts: Record<string, string> = { pending: '等待中', running: '运行中', success: '成功', failed: '失败', cancelled: '已取消' }

const getEnvColor = (env: string) => envColors[env] || 'default'
const getDeployTypeLabel = (type: string) => deployTypeLabels[type] || type
const getDeployTypeColor = (type: string) => deployTypeColors[type] || 'default'
const getStatusType = (status: string) => statusTypes[status] || 'default'
const getStatusText = (status: string) => statusTexts[status] || status
const formatTime = (time: string | undefined) => time ? time.replace('T', ' ').substring(0, 19) : '-'
const isToday = (time: string) => time && time.substring(0, 10) === new Date().toISOString().substring(0, 10)

const fetchRecords = async () => {
  loading.value = true
  try {
    const response = await applicationApi.listAllDeploys({ page: pagination.current, page_size: pagination.pageSize, ...filter })
    if (response.code === 0 && response.data) {
      records.value = response.data.list || []
      pagination.total = response.data.total
    }
  } catch (error) { console.error('获取部署记录失败', error) }
  finally { loading.value = false }
}

const onTableChange = (pag: any) => { pagination.current = pag.current; pagination.pageSize = pag.pageSize; fetchRecords() }
const resetFilter = () => { filter.app_name = ''; filter.env = ''; filter.status = ''; pagination.current = 1; fetchRecords() }
const showDetail = (record: DeployRecord) => { currentRecord.value = record; detailVisible.value = true }
const goToApp = (record: DeployRecord) => { if (record.application_id) router.push('/applications') }

onMounted(() => { fetchRecords() })
</script>

<style scoped>
.deploy-history { padding: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h1 { font-size: 20px; font-weight: 500; margin: 0; }
</style>
