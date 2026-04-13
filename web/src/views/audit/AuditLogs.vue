<template>
  <div class="audit-logs">
    <div class="page-header">
      <h1>操作审计</h1>
    </div>

    <!-- 统计卡片 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-card>
          <a-statistic title="今日操作" :value="stats.today_count" :value-style="{ color: '#1890ff' }">
            <template #prefix><AuditOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="本周操作" :value="stats.week_count" :value-style="{ color: '#52c41a' }">
            <template #prefix><BarChartOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="活跃用户" :value="stats.user_stats?.length || 0" :value-style="{ color: '#722ed1' }">
            <template #prefix><UserOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="资源类型" :value="stats.resource_stats?.length || 0" :value-style="{ color: '#fa8c16' }">
            <template #prefix><AppstoreOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 筛选条件 -->
    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-form layout="inline">
        <a-form-item label="用户">
          <a-input v-model:value="filter.username" placeholder="用户名" allow-clear style="width: 120px" />
        </a-form-item>
        <a-form-item label="操作">
          <a-select v-model:value="filter.action" placeholder="全部" allow-clear style="width: 100px">
            <a-select-option value="create">创建</a-select-option>
            <a-select-option value="update">更新</a-select-option>
            <a-select-option value="delete">删除</a-select-option>
            <a-select-option value="trigger">触发</a-select-option>
            <a-select-option value="test">测试</a-select-option>
            <a-select-option value="send">发送</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="资源">
          <a-select v-model:value="filter.resource" placeholder="全部" allow-clear style="width: 120px">
            <a-select-option value="jenkins">Jenkins</a-select-option>
            <a-select-option value="k8s">K8s</a-select-option>
            <a-select-option value="feishu">飞书</a-select-option>
            <a-select-option value="dingtalk">钉钉</a-select-option>
            <a-select-option value="wechatwork">企业微信</a-select-option>
            <a-select-option value="oa">OA</a-select-option>
            <a-select-option value="user">用户</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="filter.status" placeholder="全部" allow-clear style="width: 90px">
            <a-select-option value="success">成功</a-select-option>
            <a-select-option value="failed">失败</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="时间">
          <a-range-picker v-model:value="dateRange" style="width: 220px" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="fetchLogs">查询</a-button>
          <a-button style="margin-left: 8px" @click="resetFilter">重置</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 日志列表 -->
    <a-card :bordered="false">
      <a-table :columns="columns" :data-source="logs" :loading="loading" row-key="id" :pagination="pagination" @change="onTableChange">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          <template v-if="column.key === 'username'">
            <a-tag v-if="record.username">{{ record.username }}</a-tag>
            <span v-else>-</span>
          </template>
          <template v-if="column.key === 'action'">
            <a-tag :color="getActionColor(record.action)">{{ getActionLabel(record.action) }}</a-tag>
          </template>
          <template v-if="column.key === 'resource'">
            <a-tag :color="getResourceColor(record.resource)">{{ getResourceLabel(record.resource) }}</a-tag>
          </template>
          <template v-if="column.key === 'status'">
            <a-badge :status="record.status === 'success' ? 'success' : 'error'" :text="record.status === 'success' ? '成功' : '失败'" />
          </template>
          <template v-if="column.key === 'detail'">
            <a-button type="link" size="small" @click="showDetail(record)">查看</a-button>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 详情抽屉 -->
    <a-drawer v-model:open="detailVisible" title="操作详情" :width="600">
      <a-descriptions :column="1" bordered size="small" v-if="currentLog">
        <a-descriptions-item label="操作时间">{{ formatTime(currentLog.created_at) }}</a-descriptions-item>
        <a-descriptions-item label="操作用户">{{ currentLog.username || '-' }}</a-descriptions-item>
        <a-descriptions-item label="操作类型"><a-tag :color="getActionColor(currentLog.action)">{{ getActionLabel(currentLog.action) }}</a-tag></a-descriptions-item>
        <a-descriptions-item label="资源类型"><a-tag :color="getResourceColor(currentLog.resource)">{{ getResourceLabel(currentLog.resource) }}</a-tag></a-descriptions-item>
        <a-descriptions-item label="资源名称">{{ currentLog.resource_name || '-' }}</a-descriptions-item>
        <a-descriptions-item label="状态"><a-badge :status="currentLog.status === 'success' ? 'success' : 'error'" :text="currentLog.status === 'success' ? '成功' : '失败'" /></a-descriptions-item>
        <a-descriptions-item label="IP 地址">{{ currentLog.ip_address || '-' }}</a-descriptions-item>
        <a-descriptions-item label="User Agent">{{ currentLog.user_agent || '-' }}</a-descriptions-item>
        <a-descriptions-item v-if="currentLog.error_msg" label="错误信息"><a-typography-text type="danger">{{ currentLog.error_msg }}</a-typography-text></a-descriptions-item>
      </a-descriptions>
      <a-divider orientation="left">请求详情</a-divider>
      <pre class="detail-json">{{ formatDetail(currentLog?.detail) }}</pre>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { AuditOutlined, BarChartOutlined, UserOutlined, AppstoreOutlined } from '@ant-design/icons-vue'
import { auditApi, type AuditLog, type AuditStats } from '@/services/audit'
import type { Dayjs } from 'dayjs'

const loading = ref(false)
const logs = ref<AuditLog[]>([])
const stats = ref<AuditStats>({ action_stats: [], resource_stats: [], user_stats: [], today_count: 0, week_count: 0 })
const detailVisible = ref(false)
const currentLog = ref<AuditLog | null>(null)
const dateRange = ref<[Dayjs, Dayjs] | null>(null)

const filter = reactive({ username: '', action: '', resource: '', status: '' })
const pagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true, showTotal: (total: number) => `共 ${total} 条` })

const columns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 170 },
  { title: '用户', dataIndex: 'username', key: 'username', width: 100 },
  { title: '操作', dataIndex: 'action', key: 'action', width: 80 },
  { title: '资源', dataIndex: 'resource', key: 'resource', width: 100 },
  { title: '资源名称', dataIndex: 'resource_name', key: 'resource_name', ellipsis: true },
  { title: 'IP', dataIndex: 'ip_address', key: 'ip_address', width: 130 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '详情', key: 'detail', width: 70 }
]

const actionLabels: Record<string, string> = { create: '创建', update: '更新', delete: '删除', trigger: '触发', test: '测试', send: '发送', login: '登录', logout: '登出' }
const actionColors: Record<string, string> = { create: 'green', update: 'blue', delete: 'red', trigger: 'orange', test: 'cyan', send: 'purple', login: 'geekblue', logout: 'default' }
const resourceLabels: Record<string, string> = { jenkins: 'Jenkins', jenkins_job: 'Jenkins Job', k8s: 'K8s', k8s_resource: 'K8s资源', feishu: '飞书', dingtalk: '钉钉', wechatwork: '企业微信', oa: 'OA', user: '用户' }
const resourceColors: Record<string, string> = { jenkins: 'red', jenkins_job: 'volcano', k8s: 'purple', k8s_resource: 'geekblue', feishu: 'blue', dingtalk: 'cyan', wechatwork: 'green', oa: 'orange', user: 'magenta' }

const getActionLabel = (action: string) => actionLabels[action] || action
const getActionColor = (action: string) => actionColors[action] || 'default'
const getResourceLabel = (resource: string) => resourceLabels[resource] || resource
const getResourceColor = (resource: string) => resourceColors[resource] || 'default'
const formatTime = (time: string) => time ? time.replace('T', ' ').substring(0, 19) : '-'

const formatDetail = (detail: string | undefined) => {
  if (!detail) return ''
  try { return JSON.stringify(JSON.parse(detail), null, 2) }
  catch { return detail }
}

const fetchLogs = async () => {
  loading.value = true
  try {
    const params: any = { page: pagination.current, page_size: pagination.pageSize, ...filter }
    if (dateRange.value) {
      params.start_time = dateRange.value[0].format('YYYY-MM-DD')
      params.end_time = dateRange.value[1].format('YYYY-MM-DD')
    }
    const response = await auditApi.list(params)
    if (response.code === 0 && response.data) {
      logs.value = response.data.list || []
      pagination.total = response.data.total
    }
  } catch (error) { console.error('获取审计日志失败', error) }
  finally { loading.value = false }
}

const fetchStats = async () => {
  try {
    const response = await auditApi.getStats()
    if (response.code === 0 && response.data) { stats.value = response.data }
  } catch (error) { console.error('获取统计失败', error) }
}

const onTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  fetchLogs()
}

const resetFilter = () => {
  filter.username = ''; filter.action = ''; filter.resource = ''; filter.status = ''
  dateRange.value = null
  pagination.current = 1
  fetchLogs()
}

const showDetail = (record: AuditLog) => { currentLog.value = record; detailVisible.value = true }

onMounted(() => { fetchLogs(); fetchStats() })
</script>

<style scoped>
.audit-logs { padding: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h1 { font-size: 20px; font-weight: 500; margin: 0; }
.detail-json { background: #1e1e1e; color: #d4d4d4; padding: 16px; border-radius: 6px; max-height: 400px; overflow: auto; font-size: 13px; font-family: monospace; }
</style>
