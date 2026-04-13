<template>
  <div class="application-list">
    <div class="page-header">
      <h1>应用管理</h1>
      <a-button type="primary" @click="showAppModal()">
        <template #icon><PlusOutlined /></template>
        添加应用
      </a-button>
    </div>

    <!-- 统计卡片 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-card>
          <a-statistic title="应用总数" :value="stats.app_count" :value-style="{ color: '#1890ff' }">
            <template #prefix><AppstoreOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="今日部署" :value="stats.today_deploys" :value-style="{ color: '#52c41a' }">
            <template #prefix><RocketOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="本周部署" :value="stats.week_deploys" :value-style="{ color: '#722ed1' }">
            <template #prefix><BarChartOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="成功率" :value="stats.success_rate" suffix="%" :precision="1" :value-style="{ color: stats.success_rate >= 90 ? '#52c41a' : '#fa8c16' }">
            <template #prefix><CheckCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 筛选 -->
    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-form layout="inline">
        <a-form-item label="应用名">
          <a-input v-model:value="filter.name" placeholder="搜索应用" allow-clear style="width: 150px" @pressEnter="fetchApps" />
        </a-form-item>
        <a-form-item label="团队">
          <a-select v-model:value="filter.team" placeholder="全部" allow-clear style="width: 120px">
            <a-select-option v-for="team in teams" :key="team" :value="team">{{ team }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="语言">
          <a-select v-model:value="filter.language" placeholder="全部" allow-clear style="width: 100px">
            <a-select-option value="go">Go</a-select-option>
            <a-select-option value="java">Java</a-select-option>
            <a-select-option value="python">Python</a-select-option>
            <a-select-option value="nodejs">Node.js</a-select-option>
            <a-select-option value="php">PHP</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="filter.status" placeholder="全部" allow-clear style="width: 90px">
            <a-select-option value="active">启用</a-select-option>
            <a-select-option value="inactive">禁用</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="fetchApps">查询</a-button>
          <a-button style="margin-left: 8px" @click="resetFilter">重置</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 应用列表 -->
    <a-card :bordered="false">
      <a-table :columns="columns" :data-source="apps" :loading="loading" row-key="id" :pagination="pagination" @change="onTableChange">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <div>
              <a @click="viewApp(record)">{{ record.name }}</a>
              <div v-if="record.display_name" class="sub-text">{{ record.display_name }}</div>
            </div>
          </template>
          <template v-if="column.key === 'language'">
            <a-tag v-if="record.language" :color="getLangColor(record.language)">{{ record.language }}</a-tag>
            <span v-else>-</span>
          </template>
          <template v-if="column.key === 'team'">
            <a-tag v-if="record.team" color="blue">{{ record.team }}</a-tag>
            <span v-else>-</span>
          </template>
          <template v-if="column.key === 'status'">
            <a-badge :status="record.status === 'active' ? 'success' : 'default'" :text="record.status === 'active' ? '启用' : '禁用'" />
          </template>
          <template v-if="column.key === 'jenkins'">
            <span v-if="record.jenkins_job_name">{{ record.jenkins_job_name }}</span>
            <span v-else class="text-muted">未配置</span>
          </template>
          <template v-if="column.key === 'k8s'">
            <span v-if="record.k8s_deployment">{{ record.k8s_namespace }}/{{ record.k8s_deployment }}</span>
            <span v-else class="text-muted">未配置</span>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="viewApp(record)">详情</a-button>
              <a-button type="link" size="small" @click="goTraffic(record)">流量治理</a-button>
              <a-button type="link" size="small" @click="showAppModal(record)">编辑</a-button>
              <a-popconfirm title="确定删除？" @confirm="deleteApp(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 应用编辑弹窗 -->
    <a-modal v-model:open="appModalVisible" :title="editingAppId ? '编辑应用' : '添加应用'" @ok="saveApp" :confirm-loading="savingApp" width="700px">
      <a-form :model="editingApp" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="应用名称" required>
              <a-input v-model:value="editingApp.name" placeholder="如：user-service" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="显示名称">
              <a-input v-model:value="editingApp.display_name" placeholder="如：用户服务" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="开发语言">
              <a-select v-model:value="editingApp.language" placeholder="选择语言" allow-clear>
                <a-select-option value="go">Go</a-select-option>
                <a-select-option value="java">Java</a-select-option>
                <a-select-option value="python">Python</a-select-option>
                <a-select-option value="nodejs">Node.js</a-select-option>
                <a-select-option value="php">PHP</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="框架">
              <a-input v-model:value="editingApp.framework" placeholder="如：gin, spring" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="团队">
              <a-auto-complete v-model:value="editingApp.team" :options="teamOptions" placeholder="所属团队" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="负责人">
              <a-input v-model:value="editingApp.owner" placeholder="负责人姓名" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="Git 仓库">
              <a-input v-model:value="editingApp.git_repo" placeholder="https://github.com/..." />
            </a-form-item>
          </a-col>
        </a-row>
        <a-divider orientation="left">Jenkins 配置</a-divider>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="Jenkins 实例">
              <a-select v-model:value="editingApp.jenkins_instance_id" placeholder="选择实例" allow-clear>
                <a-select-option v-for="inst in jenkinsInstances" :key="inst.id" :value="inst.id">{{ inst.name }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="Jenkins Job">
              <a-input v-model:value="editingApp.jenkins_job_name" placeholder="Job 名称" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-divider orientation="left">K8s 配置</a-divider>
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="K8s 集群">
              <a-select v-model:value="editingApp.k8s_cluster_id" placeholder="选择集群" allow-clear>
                <a-select-option v-for="cluster in k8sClusters" :key="cluster.id" :value="cluster.id">{{ cluster.name }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="Namespace">
              <a-input v-model:value="editingApp.k8s_namespace" placeholder="default" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="Deployment">
              <a-input v-model:value="editingApp.k8s_deployment" placeholder="deployment 名称" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="描述">
          <a-textarea v-model:value="editingApp.description" :rows="2" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 应用详情抽屉 -->
    <a-drawer v-model:open="detailVisible" :title="currentApp?.display_name || currentApp?.name" :width="700">
      <template v-if="currentApp">
        <a-descriptions :column="2" bordered size="small">
          <a-descriptions-item label="应用名称">{{ currentApp.name }}</a-descriptions-item>
          <a-descriptions-item label="显示名称">{{ currentApp.display_name || '-' }}</a-descriptions-item>
          <a-descriptions-item label="开发语言"><a-tag v-if="currentApp.language" :color="getLangColor(currentApp.language)">{{ currentApp.language }}</a-tag><span v-else>-</span></a-descriptions-item>
          <a-descriptions-item label="框架">{{ currentApp.framework || '-' }}</a-descriptions-item>
          <a-descriptions-item label="团队">{{ currentApp.team || '-' }}</a-descriptions-item>
          <a-descriptions-item label="负责人">{{ currentApp.owner || '-' }}</a-descriptions-item>
          <a-descriptions-item label="Git 仓库" :span="2"><a v-if="currentApp.git_repo" :href="currentApp.git_repo" target="_blank">{{ currentApp.git_repo }}</a><span v-else>-</span></a-descriptions-item>
          <a-descriptions-item label="Jenkins Job">{{ currentApp.jenkins_job_name || '-' }}</a-descriptions-item>
          <a-descriptions-item label="K8s 部署">{{ currentApp.k8s_namespace && currentApp.k8s_deployment ? `${currentApp.k8s_namespace}/${currentApp.k8s_deployment}` : '-' }}</a-descriptions-item>
        </a-descriptions>

        <a-divider orientation="left">环境配置</a-divider>
        <a-table :columns="envColumns" :data-source="currentEnvs" row-key="id" size="small" :pagination="false">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'env_name'">
              <a-tag :color="getEnvColor(record.env_name)">{{ record.env_name }}</a-tag>
            </template>
          </template>
        </a-table>

        <a-divider orientation="left">最近部署</a-divider>
        <a-table :columns="deployColumns" :data-source="currentDeploys" row-key="id" size="small" :pagination="{ pageSize: 5 }">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'created_at'">{{ formatTime(record.created_at) }}</template>
            <template v-if="column.key === 'env_name'"><a-tag :color="getEnvColor(record.env_name)">{{ record.env_name }}</a-tag></template>
            <template v-if="column.key === 'status'">
              <a-badge :status="getDeployStatusType(record.status)" :text="getDeployStatusText(record.status)" />
            </template>
            <template v-if="column.key === 'duration'">{{ record.duration ? `${record.duration}s` : '-' }}</template>
          </template>
        </a-table>
      </template>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { PlusOutlined, AppstoreOutlined, RocketOutlined, BarChartOutlined, CheckCircleOutlined } from '@ant-design/icons-vue'
import { applicationApi, type Application, type ApplicationEnv, type DeployRecord, type AppStats } from '@/services/application'
import { jenkinsInstanceApi } from '@/services/jenkins'
import { k8sClusterApi } from '@/services/k8s'

const router = useRouter()
const loading = ref(false)
const savingApp = ref(false)
const appModalVisible = ref(false)
const detailVisible = ref(false)
const editingAppId = ref<number | undefined>(undefined)

const apps = ref<Application[]>([])
const teams = ref<string[]>([])
const stats = ref<AppStats>({ app_count: 0, team_stats: [], lang_stats: [], today_deploys: 0, week_deploys: 0, success_rate: 0 })
const jenkinsInstances = ref<any[]>([])
const k8sClusters = ref<any[]>([])
const currentApp = ref<Application | null>(null)
const currentEnvs = ref<ApplicationEnv[]>([])
const currentDeploys = ref<DeployRecord[]>([])

const filter = reactive({ name: '', team: '', language: '', status: '' })
const pagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true })

const editingApp = reactive<Partial<Application>>({
  name: '', display_name: '', description: '', git_repo: '', language: '', framework: '',
  team: '', owner: '', status: 'active', jenkins_instance_id: undefined, jenkins_job_name: '',
  k8s_cluster_id: undefined, k8s_namespace: '', k8s_deployment: ''
})

const teamOptions = computed(() => teams.value.map(t => ({ value: t })))

const columns = [
  { title: '应用名称', dataIndex: 'name', key: 'name' },
  { title: '语言', dataIndex: 'language', key: 'language', width: 90 },
  { title: '团队', dataIndex: 'team', key: 'team', width: 100 },
  { title: 'Jenkins Job', key: 'jenkins', width: 150 },
  { title: 'K8s 部署', key: 'k8s', width: 180 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '操作', key: 'action', width: 260 }
]

const envColumns = [
  { title: '环境', dataIndex: 'env_name', key: 'env_name', width: 80 },
  { title: '分支', dataIndex: 'branch', key: 'branch' },
  { title: 'Jenkins Job', dataIndex: 'jenkins_job', key: 'jenkins_job' },
  { title: 'Namespace', dataIndex: 'k8s_namespace', key: 'k8s_namespace' },
  { title: '副本数', dataIndex: 'replicas', key: 'replicas', width: 70 }
]

const deployColumns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 150 },
  { title: '环境', dataIndex: 'env_name', key: 'env_name', width: 80 },
  { title: '版本', dataIndex: 'version', key: 'version' },
  { title: '操作人', dataIndex: 'operator', key: 'operator', width: 80 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '耗时', dataIndex: 'duration', key: 'duration', width: 70 }
]

const langColors: Record<string, string> = { go: 'cyan', java: 'orange', python: 'blue', nodejs: 'green', php: 'purple' }
const envColors: Record<string, string> = { dev: 'blue', test: 'cyan', staging: 'orange', prod: 'red' }

const getLangColor = (lang: string) => langColors[lang] || 'default'
const getEnvColor = (env: string) => envColors[env] || 'default'
const formatTime = (time: string) => time ? time.replace('T', ' ').substring(0, 19) : '-'

const getDeployStatusType = (status: string) => {
  const map: Record<string, string> = { pending: 'default', running: 'processing', success: 'success', failed: 'error', cancelled: 'warning' }
  return map[status] || 'default'
}
const getDeployStatusText = (status: string) => {
  const map: Record<string, string> = { pending: '等待中', running: '运行中', success: '成功', failed: '失败', cancelled: '已取消' }
  return map[status] || status
}

const fetchApps = async () => {
  loading.value = true
  try {
    const response = await applicationApi.list({ page: pagination.current, page_size: pagination.pageSize, ...filter })
    if (response.code === 0 && response.data) {
      apps.value = response.data.list || []
      pagination.total = response.data.total
    }
  } catch (error) { console.error('获取应用列表失败', error) }
  finally { loading.value = false }
}

const fetchStats = async () => {
  try {
    const response = await applicationApi.getStats()
    if (response.code === 0 && response.data) { stats.value = response.data }
  } catch (error) { console.error('获取统计失败', error) }
}

const fetchTeams = async () => {
  try {
    const response = await applicationApi.getTeams()
    if (response.code === 0 && response.data) { teams.value = response.data }
  } catch (error) { console.error('获取团队列表失败', error) }
}

const fetchJenkinsInstances = async () => {
  try {
    const response = await jenkinsInstanceApi.list()
    if (response.code === 0 && response.data) { jenkinsInstances.value = response.data.items || [] }
  } catch (error) { console.error('获取 Jenkins 实例失败', error) }
}

const fetchK8sClusters = async () => {
  try {
    const response = await k8sClusterApi.list()
    if (response.code === 0 && response.data) { k8sClusters.value = response.data.items || [] }
  } catch (error) { console.error('获取 K8s 集群失败', error) }
}

const onTableChange = (pag: any) => { pagination.current = pag.current; pagination.pageSize = pag.pageSize; fetchApps() }
const resetFilter = () => { filter.name = ''; filter.team = ''; filter.language = ''; filter.status = ''; pagination.current = 1; fetchApps() }

const showAppModal = (app?: Application) => {
  if (app) {
    editingAppId.value = app.id
    Object.assign(editingApp, app)
  } else {
    editingAppId.value = undefined
    Object.assign(editingApp, { name: '', display_name: '', description: '', git_repo: '', language: '', framework: '', team: '', owner: '', status: 'active', jenkins_instance_id: undefined, jenkins_job_name: '', k8s_cluster_id: undefined, k8s_namespace: '', k8s_deployment: '' })
  }
  appModalVisible.value = true
}

const saveApp = async () => {
  if (!editingApp.name) { message.error('请填写应用名称'); return }
  savingApp.value = true
  try {
    const response = editingAppId.value
      ? await applicationApi.update(editingAppId.value, editingApp)
      : await applicationApi.create(editingApp)
    if (response.code === 0) {
      message.success(editingAppId.value ? '更新成功' : '添加成功')
      appModalVisible.value = false
      fetchApps()
      fetchStats()
      fetchTeams()
    } else { message.error(response.message || '保存失败') }
  } catch (error: any) { message.error(error.message || '保存失败') }
  finally { savingApp.value = false }
}

const deleteApp = async (id: number) => {
  try {
    const response = await applicationApi.delete(id)
    if (response.code === 0) { message.success('删除成功'); fetchApps(); fetchStats() }
    else { message.error(response.message || '删除失败') }
  } catch (error: any) { message.error(error.message || '删除失败') }
}

const viewApp = (app: Application) => {
  router.push(`/applications/${app.id}`)
}

const goTraffic = (app: Application) => {
  router.push(`/applications/${app.id}/traffic`)
}

onMounted(() => {
  fetchApps()
  fetchStats()
  fetchTeams()
  fetchJenkinsInstances()
  fetchK8sClusters()
})
</script>

<style scoped>
.application-list { padding: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h1 { font-size: 20px; font-weight: 500; margin: 0; }
.sub-text { color: #999; font-size: 12px; }
.text-muted { color: #999; }
</style>
