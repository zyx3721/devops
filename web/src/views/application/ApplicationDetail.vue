<template>
  <div class="app-detail">
    <div class="page-header">
      <div class="header-left">
        <a-button type="text" @click="goBack"><ArrowLeftOutlined /> 返回</a-button>
        <h1>{{ app?.display_name || app?.name || '应用详情' }}</h1>
        <a-tag v-if="app?.language" :color="langColors[app.language] || 'default'">{{ app.language }}</a-tag>
      </div>
      <div class="header-right">
        <!-- 发布窗口状态 -->
        <a-tooltip v-if="deployWindowStatus" :title="deployWindowStatus.message">
          <a-tag :color="deployWindowStatus.in_window ? 'green' : 'orange'">
            <ClockCircleOutlined /> {{ deployWindowStatus.in_window ? '发布窗口内' : '窗口外' }}
          </a-tag>
        </a-tooltip>
        <!-- 发布锁状态 -->
        <a-tooltip v-if="deployLockStatus?.locked" :title="`锁定者: ${deployLockStatus.locked_by}, 时间: ${fmtTime(deployLockStatus.locked_at)}`">
          <a-tag color="red"><LockOutlined /> 已锁定</a-tag>
        </a-tooltip>
        <a-dropdown>
          <a-button type="primary"><RocketOutlined /> 发起部署 <DownOutlined /></a-button>
          <template #overlay>
            <a-menu @click="handleDeployMenuClick">
              <a-menu-item key="normal">普通部署</a-menu-item>
              <a-menu-item key="emergency"><WarningOutlined /> 紧急部署</a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
        <a-button @click="showEditModal"><EditOutlined /> 编辑</a-button>
      </div>
    </div>

    <a-spin :spinning="loading">
      <a-tabs v-model:activeKey="activeTab">
        <!-- 基本信息 -->
        <a-tab-pane key="info" tab="基本信息">
          <a-card :bordered="false">
            <a-descriptions :column="2" bordered>
              <a-descriptions-item label="应用名称">{{ app?.name }}</a-descriptions-item>
              <a-descriptions-item label="显示名称">{{ app?.display_name || '-' }}</a-descriptions-item>
              <a-descriptions-item label="语言"><a-tag v-if="app?.language" :color="langColors[app.language]">{{ app.language }}</a-tag><span v-else>-</span></a-descriptions-item>
              <a-descriptions-item label="框架">{{ app?.framework || '-' }}</a-descriptions-item>
              <a-descriptions-item label="团队">{{ app?.team || '-' }}</a-descriptions-item>
              <a-descriptions-item label="负责人">{{ app?.owner || '-' }}</a-descriptions-item>
              <a-descriptions-item label="Git 仓库" :span="2"><a v-if="app?.git_repo" :href="app.git_repo" target="_blank">{{ app.git_repo }}</a><span v-else>-</span></a-descriptions-item>
              <a-descriptions-item label="描述" :span="2">{{ app?.description || '-' }}</a-descriptions-item>
            </a-descriptions>
          </a-card>
        </a-tab-pane>

        <!-- 环境配置 -->
        <a-tab-pane key="envs" tab="环境配置">
          <a-card :bordered="false">
            <template #extra><a-button type="primary" size="small" @click="showEnvModal()"><PlusOutlined /> 添加</a-button></template>
            <a-table :columns="envColumns" :data-source="envs" row-key="id" :pagination="false">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'env_name'"><a-tag :color="envColors[record.env_name]">{{ record.env_name }}</a-tag></template>
                <template v-if="column.key === 'action'">
                  <a-button type="link" size="small" @click="showEnvModal(record)">编辑</a-button>
                  <a-popconfirm title="确定删除？" @confirm="deleteEnv(record.id)"><a-button type="link" size="small" danger>删除</a-button></a-popconfirm>
                </template>
              </template>
            </a-table>
          </a-card>
        </a-tab-pane>

        <!-- 部署记录 -->
        <a-tab-pane key="deploys" tab="部署记录">
          <a-card :bordered="false">
            <div style="margin-bottom: 16px">
              <a-space>
                <a-select v-model:value="deployFilter.env_name" placeholder="环境" allow-clear style="width: 100px">
                  <a-select-option v-for="e in ['dev','test','staging','prod']" :key="e" :value="e">{{ e }}</a-select-option>
                </a-select>
                <a-select v-model:value="deployFilter.status" placeholder="状态" allow-clear style="width: 100px">
                  <a-select-option v-for="s in ['success','failed','running','pending']" :key="s" :value="s">{{ statusText[s] }}</a-select-option>
                </a-select>
                <a-button type="primary" @click="fetchDeploys">查询</a-button>
              </a-space>
            </div>
            <a-table :columns="deployColumns" :data-source="deploys" row-key="id" :loading="deploysLoading" :pagination="deployPagination" @change="onDeployPageChange">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'created_at'">{{ fmtTime(record.created_at) }}</template>
                <template v-if="column.key === 'env_name'"><a-tag :color="envColors[record.env_name]">{{ record.env_name }}</a-tag></template>
                <template v-if="column.key === 'status'"><a-badge :status="statusType[record.status]" :text="statusText[record.status]" /></template>
                <template v-if="column.key === 'duration'">{{ record.duration ? `${record.duration}s` : '-' }}</template>
                <template v-if="column.key === 'action'"><a-button type="link" size="small" @click="viewDeploy(record)">详情</a-button></template>
              </template>
            </a-table>
          </a-card>
        </a-tab-pane>

        <!-- 发布请求 -->
        <a-tab-pane key="requests" tab="发布请求">
          <a-card :bordered="false">
            <a-table :columns="requestColumns" :data-source="requests" row-key="id" :loading="requestsLoading" :pagination="requestPagination" @change="onRequestPageChange">
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'created_at'">{{ fmtTime(record.created_at) }}</template>
                <template v-if="column.key === 'env_name'"><a-tag :color="envColors[record.env_name]">{{ record.env_name }}</a-tag></template>
                <template v-if="column.key === 'status'"><a-badge :status="statusType[record.status]" :text="statusText[record.status]" /></template>
                <template v-if="column.key === 'action'">
                  <a-space v-if="record.status === 'pending' && record.need_approval">
                    <a-button type="link" size="small" @click="approveRequest(record.id)">通过</a-button>
                    <a-button type="link" size="small" danger @click="rejectRequest(record.id)">拒绝</a-button>
                  </a-space>
                  <a-button v-else type="link" size="small" @click="viewDeploy(record)">详情</a-button>
                </template>
              </template>
            </a-table>
          </a-card>
        </a-tab-pane>
      </a-tabs>
    </a-spin>

    <!-- 编辑应用 -->
    <a-modal v-model:open="editModalVisible" title="编辑应用" @ok="saveApp" :confirmLoading="saving" width="600px">
      <a-form :model="editForm" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12"><a-form-item label="应用名称"><a-input v-model:value="editForm.name" disabled /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="显示名称"><a-input v-model:value="editForm.display_name" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="8"><a-form-item label="语言"><a-select v-model:value="editForm.language" allow-clear>
            <a-select-option v-for="l in ['go','java','python','nodejs']" :key="l" :value="l">{{ l }}</a-select-option>
          </a-select></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="框架"><a-input v-model:value="editForm.framework" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="团队"><a-input v-model:value="editForm.team" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12"><a-form-item label="负责人"><a-input v-model:value="editForm.owner" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="Git 仓库"><a-input v-model:value="editForm.git_repo" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="描述"><a-textarea v-model:value="editForm.description" :rows="2" /></a-form-item>
      </a-form>
    </a-modal>

    <!-- 环境配置 -->
    <a-modal v-model:open="envModalVisible" :title="envForm.id ? '编辑环境' : '添加环境'" @ok="saveEnv" :confirmLoading="savingEnv" width="500px">
      <a-form :model="envForm" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="8"><a-form-item label="环境" required><a-select v-model:value="envForm.env_name" :disabled="!!envForm.id">
            <a-select-option v-for="e in ['dev','test','staging','prod']" :key="e" :value="e">{{ e }}</a-select-option>
          </a-select></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="分支"><a-input v-model:value="envForm.branch" placeholder="main" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="副本数"><a-input-number v-model:value="envForm.replicas" :min="1" style="width:100%" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12"><a-form-item label="Jenkins Job"><a-input v-model:value="envForm.jenkins_job" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="K8s Namespace"><a-input v-model:value="envForm.k8s_namespace" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="K8s Deployment"><a-input v-model:value="envForm.k8s_deployment" /></a-form-item>
      </a-form>
    </a-modal>

    <!-- 发起部署 -->
    <a-modal v-model:open="deployModalVisible" :title="deployForm.is_emergency ? '紧急部署' : '发起部署'" @ok="submitDeploy" :confirmLoading="deploying" width="500px">
      <a-alert v-if="deployForm.is_emergency" type="warning" show-icon style="margin-bottom: 16px">
        <template #message>紧急部署将绕过审批流程和发布窗口限制，请谨慎操作</template>
      </a-alert>
      <a-form :model="deployForm" layout="vertical">
        <a-form-item label="环境" required><a-select v-model:value="deployForm.env_name" @change="checkDeployStatus">
          <a-select-option v-for="e in envs" :key="e.env_name" :value="e.env_name">{{ e.env_name }}</a-select-option>
        </a-select></a-form-item>
        <!-- 审批状态提示 -->
        <a-alert v-if="needApproval && !deployForm.is_emergency" type="info" show-icon style="margin-bottom: 12px">
          <template #message>该环境需要审批，提交后将进入审批流程</template>
        </a-alert>
        <!-- 发布窗口提示 -->
        <a-alert v-if="!deployWindowStatus?.in_window && !deployForm.is_emergency" type="warning" show-icon style="margin-bottom: 12px">
          <template #message>当前不在发布窗口内，下次可发布时间: {{ deployWindowStatus?.next_window || '-' }}</template>
        </a-alert>
        <a-form-item label="版本/镜像标签"><a-input v-model:value="deployForm.image_tag" placeholder="v1.0.0" /></a-form-item>
        <a-form-item label="分支"><a-input v-model:value="deployForm.branch" placeholder="main" /></a-form-item>
        <a-form-item label="发布说明"><a-textarea v-model:value="deployForm.description" :rows="2" /></a-form-item>
        <a-form-item v-if="deployForm.is_emergency" label="紧急原因" required>
          <a-textarea v-model:value="deployForm.emergency_reason" :rows="2" placeholder="请说明紧急发布的原因" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 部署详情 -->
    <a-drawer v-model:open="deployDetailVisible" title="部署详情" :width="450">
      <a-descriptions v-if="currentDeploy" :column="1" bordered size="small">
        <a-descriptions-item label="环境"><a-tag :color="envColors[currentDeploy.env_name]">{{ currentDeploy.env_name }}</a-tag></a-descriptions-item>
        <a-descriptions-item label="状态"><a-badge :status="statusType[currentDeploy.status]" :text="statusText[currentDeploy.status]" /></a-descriptions-item>
        <a-descriptions-item label="版本">{{ currentDeploy.version || currentDeploy.image_tag || '-' }}</a-descriptions-item>
        <a-descriptions-item label="分支">{{ currentDeploy.branch || '-' }}</a-descriptions-item>
        <a-descriptions-item label="操作人">{{ currentDeploy.operator || '-' }}</a-descriptions-item>
        <a-descriptions-item label="开始时间">{{ fmtTime(currentDeploy.started_at) }}</a-descriptions-item>
        <a-descriptions-item label="结束时间">{{ fmtTime(currentDeploy.finished_at) }}</a-descriptions-item>
        <a-descriptions-item label="耗时">{{ currentDeploy.duration ? `${currentDeploy.duration}s` : '-' }}</a-descriptions-item>
        <a-descriptions-item v-if="currentDeploy.error_msg" label="错误"><span style="color:#ff4d4f">{{ currentDeploy.error_msg }}</span></a-descriptions-item>
      </a-descriptions>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { ArrowLeftOutlined, RocketOutlined, EditOutlined, PlusOutlined, ClockCircleOutlined, LockOutlined, WarningOutlined, DownOutlined } from '@ant-design/icons-vue'
import { applicationApi, type Application, type ApplicationEnv, type DeployRecord } from '@/services/application'

const route = useRoute()
const router = useRouter()
const appId = Number(route.params.id)

const loading = ref(false)
const saving = ref(false)
const savingEnv = ref(false)
const deploying = ref(false)
const deploysLoading = ref(false)
const requestsLoading = ref(false)
const activeTab = ref('info')

const app = ref<Application | null>(null)
const envs = ref<ApplicationEnv[]>([])
const deploys = ref<DeployRecord[]>([])
const requests = ref<DeployRecord[]>([])
const currentDeploy = ref<DeployRecord | null>(null)

// 发布窗口和锁状态
const deployWindowStatus = ref<{ in_window: boolean; message: string; next_window?: string } | null>(null)
const deployLockStatus = ref<{ locked: boolean; locked_by?: string; locked_at?: string } | null>(null)
const needApproval = ref(false)

const editModalVisible = ref(false)
const envModalVisible = ref(false)
const deployModalVisible = ref(false)
const deployDetailVisible = ref(false)

const editForm = reactive<Partial<Application>>({})
const envForm = reactive<Partial<ApplicationEnv>>({ replicas: 1 })
const deployForm = reactive({ env_name: '', image_tag: '', branch: '', description: '', is_emergency: false, emergency_reason: '' })
const deployFilter = reactive({ env_name: '', status: '' })
const deployPagination = reactive({ current: 1, pageSize: 10, total: 0 })
const requestPagination = reactive({ current: 1, pageSize: 10, total: 0 })

const langColors: Record<string, string> = { go: 'cyan', java: 'orange', python: 'blue', nodejs: 'green' }
const envColors: Record<string, string> = { dev: 'blue', test: 'cyan', staging: 'orange', prod: 'red' }
const statusType: Record<string, string> = { pending: 'warning', approved: 'processing', running: 'processing', success: 'success', failed: 'error', rejected: 'error', cancelled: 'default' }
const statusText: Record<string, string> = { pending: '待审批', approved: '已通过', running: '运行中', success: '成功', failed: '失败', rejected: '已拒绝', cancelled: '已取消' }

const envColumns = [
  { title: '环境', dataIndex: 'env_name', key: 'env_name', width: 80 },
  { title: '分支', dataIndex: 'branch', key: 'branch', width: 100 },
  { title: 'Jenkins Job', dataIndex: 'jenkins_job', key: 'jenkins_job' },
  { title: 'K8s Namespace', dataIndex: 'k8s_namespace', key: 'k8s_namespace' },
  { title: 'Deployment', dataIndex: 'k8s_deployment', key: 'k8s_deployment' },
  { title: '副本', dataIndex: 'replicas', key: 'replicas', width: 60 },
  { title: '操作', key: 'action', width: 100 }
]
const deployColumns = [
  { title: '时间', key: 'created_at', width: 150 },
  { title: '环境', key: 'env_name', width: 80 },
  { title: '版本', dataIndex: 'version', width: 100 },
  { title: '分支', dataIndex: 'branch', width: 100 },
  { title: '操作人', dataIndex: 'operator', width: 80 },
  { title: '状态', key: 'status', width: 80 },
  { title: '耗时', key: 'duration', width: 60 },
  { title: '操作', key: 'action', width: 60 }
]
const requestColumns = [
  { title: '时间', key: 'created_at', width: 150 },
  { title: '环境', key: 'env_name', width: 80 },
  { title: '版本', dataIndex: 'image_tag', width: 100 },
  { title: '申请人', dataIndex: 'operator', width: 80 },
  { title: '状态', key: 'status', width: 90 },
  { title: '说明', dataIndex: 'description', ellipsis: true },
  { title: '操作', key: 'action', width: 100 }
]

const fmtTime = (t: string) => t ? t.replace('T', ' ').substring(0, 19) : '-'
const goBack = () => router.push('/applications')

const fetchApp = async () => {
  loading.value = true
  try {
    const res = await applicationApi.get(appId)
    if (res.code === 0 && res.data) { app.value = res.data.app || res.data; envs.value = res.data.envs || [] }
  } catch (e) { console.error(e) }
  finally { loading.value = false }
}

const fetchDeploys = async () => {
  deploysLoading.value = true
  try {
    const res = await applicationApi.listDeploys(appId, { page: deployPagination.current, page_size: deployPagination.pageSize, ...deployFilter })
    if (res.code === 0 && res.data) { deploys.value = res.data.list || []; deployPagination.total = res.data.total }
  } catch (e) { console.error(e) }
  finally { deploysLoading.value = false }
}

const fetchRequests = async () => {
  requestsLoading.value = true
  try {
    const res = await applicationApi.listDeploys(appId, { page: requestPagination.current, page_size: requestPagination.pageSize, status: 'pending' })
    if (res.code === 0 && res.data) { requests.value = res.data.list || []; requestPagination.total = res.data.total }
  } catch (e) { console.error(e) }
  finally { requestsLoading.value = false }
}

const onDeployPageChange = (p: any) => { deployPagination.current = p.current; fetchDeploys() }
const onRequestPageChange = (p: any) => { requestPagination.current = p.current; fetchRequests() }

const showEditModal = () => { if (app.value) Object.assign(editForm, app.value); editModalVisible.value = true }
const saveApp = async () => {
  saving.value = true
  try {
    const res = await applicationApi.update(appId, editForm)
    if (res.code === 0) { message.success('保存成功'); editModalVisible.value = false; fetchApp() }
  } catch (e: any) { message.error(e.message || '保存失败') }
  finally { saving.value = false }
}

const showEnvModal = (env?: ApplicationEnv) => {
  if (env) Object.assign(envForm, env)
  else Object.assign(envForm, { id: undefined, env_name: '', branch: '', replicas: 1, jenkins_job: '', k8s_namespace: '', k8s_deployment: '' })
  envModalVisible.value = true
}
const saveEnv = async () => {
  if (!envForm.env_name) { message.error('请选择环境'); return }
  savingEnv.value = true
  try {
    const res = envForm.id ? await applicationApi.updateEnv(appId, envForm.id, envForm) : await applicationApi.createEnv(appId, envForm)
    if (res.code === 0) { message.success('保存成功'); envModalVisible.value = false; fetchApp() }
  } catch (e: any) { message.error(e.message || '保存失败') }
  finally { savingEnv.value = false }
}
const deleteEnv = async (id: number) => {
  try { const res = await applicationApi.deleteEnv(appId, id); if (res.code === 0) { message.success('删除成功'); fetchApp() } }
  catch (e: any) { message.error(e.message || '删除失败') }
}

const showDeployModal = (isEmergency = false) => { 
  Object.assign(deployForm, { 
    env_name: envs.value[0]?.env_name || '', 
    image_tag: '', 
    branch: '', 
    description: '',
    is_emergency: isEmergency,
    emergency_reason: ''
  })
  if (deployForm.env_name) checkDeployStatus()
  deployModalVisible.value = true 
}

const handleDeployMenuClick = ({ key }: { key: string }) => {
  showDeployModal(key === 'emergency')
}

const checkDeployStatus = async () => {
  if (!deployForm.env_name) return
  try {
    // 检查发布窗口状态
    const windowRes = await applicationApi.getDeployWindowStatus(appId, deployForm.env_name)
    if (windowRes.code === 0) deployWindowStatus.value = windowRes.data
    // 检查发布锁状态
    const lockRes = await applicationApi.getDeployLockStatus(appId, deployForm.env_name)
    if (lockRes.code === 0) deployLockStatus.value = lockRes.data
    // 检查是否需要审批
    const approvalRes = await applicationApi.checkApprovalRequired(appId, deployForm.env_name)
    if (approvalRes.code === 0) needApproval.value = approvalRes.data?.required || false
  } catch (e) { console.error(e) }
}

const submitDeploy = async () => {
  if (!deployForm.env_name) { message.error('请选择环境'); return }
  if (deployForm.is_emergency && !deployForm.emergency_reason) { message.error('请填写紧急原因'); return }
  deploying.value = true
  try {
    const payload = {
      ...deployForm,
      app_id: appId
    }
    const res = deployForm.is_emergency 
      ? await applicationApi.emergencyDeploy(appId, payload)
      : await applicationApi.deploy(appId, payload)
    if (res.code === 0) { 
      message.success(deployForm.is_emergency ? '紧急部署已提交' : (needApproval.value ? '部署申请已提交，等待审批' : '部署已提交'))
      deployModalVisible.value = false
      fetchDeploys()
      fetchRequests() 
    }
  } catch (e: any) { message.error(e.message || '部署失败') }
  finally { deploying.value = false }
}

const viewDeploy = (r: DeployRecord) => { currentDeploy.value = r; deployDetailVisible.value = true }
const approveRequest = async (id: number) => {
  try { message.success('审批通过'); fetchRequests(); fetchDeploys() } catch (e: any) { message.error(e.message) }
}
const rejectRequest = async (id: number) => {
  try { message.success('已拒绝'); fetchRequests() } catch (e: any) { message.error(e.message) }
}

onMounted(() => { fetchApp(); fetchDeploys(); fetchRequests(); checkDeployStatus() })
</script>

<style scoped>
.app-detail { padding: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.header-left { display: flex; align-items: center; gap: 12px; }
.header-left h1 { font-size: 20px; font-weight: 500; margin: 0; }
.header-right { display: flex; gap: 8px; }
</style>
