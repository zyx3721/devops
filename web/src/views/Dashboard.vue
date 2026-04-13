<template>
  <div class="dashboard">
    <div class="page-header">
      <h1>仪表盘</h1>
      <a-tag :color="getHealthColor(healthOverview.status)" style="font-size: 14px; padding: 4px 12px">
        {{ getHealthText(healthOverview.status) }}
      </a-tag>
    </div>

    <!-- 统计卡片 - 第一行 -->
    <a-row :gutter="[16, 16]">
      <a-col :xs="12" :sm="8" :md="4">
        <a-card hoverable @click="$router.push('/pipeline/list')">
          <a-statistic title="流水线" :value="pipelineStats.total" :value-style="{ color: '#1890ff' }">
            <template #prefix><RocketOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="12" :sm="8" :md="4">
        <a-card hoverable @click="$router.push('/k8s/clusters')">
          <a-statistic title="K8s 集群" :value="stats.k8sClusters" :value-style="{ color: '#722ed1' }">
            <template #prefix><CloudOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="12" :sm="8" :md="4">
        <a-card hoverable @click="$router.push('/applications')">
          <a-statistic title="应用" :value="stats.applications || 0" :value-style="{ color: '#13c2c2' }">
            <template #prefix><AppstoreOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="12" :sm="8" :md="4">
        <a-card hoverable @click="$router.push('/alert/overview')">
          <a-statistic title="今日告警" :value="stats.alertsToday" :value-style="{ color: stats.alertsToday > 0 ? '#fa8c16' : '#52c41a' }">
            <template #prefix><AlertOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="12" :sm="8" :md="4">
        <a-card hoverable @click="$router.push('/security/overview')">
          <a-statistic title="安全风险" :value="securityStats.highRisk" :value-style="{ color: securityStats.highRisk > 0 ? '#ff4d4f' : '#52c41a' }">
            <template #prefix><SafetyCertificateOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :xs="12" :sm="8" :md="4">
        <a-card hoverable @click="$router.push('/approval/pending')">
          <a-statistic title="待审批" :value="stats.pendingApprovals || 0" :value-style="{ color: '#eb2f96' }">
            <template #prefix><AuditOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <!-- 快捷操作 - 移到更靠前的位置 -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :span="24">
        <a-card title="快捷操作" :bordered="false" size="small">
          <a-space wrap size="middle">
            <a-button type="primary" @click="$router.push('/pipeline/create')"><PlusOutlined /> 新建流水线</a-button>
            <a-button @click="$router.push('/pipeline/list')"><RocketOutlined /> 流水线</a-button>
            <a-button @click="$router.push('/k8s/clusters')"><CloudOutlined /> K8s 集群</a-button>
            <a-button @click="$router.push('/logs/center')"><FileSearchOutlined /> 日志中心</a-button>
            <a-button @click="$router.push('/cost/overview')"><DollarOutlined /> 成本管理</a-button>
            <a-button @click="$router.push('/security/overview')"><SafetyCertificateOutlined /> 安全中心</a-button>
            <a-button @click="$router.push('/alert/overview')"><AlertOutlined /> 告警管理</a-button>
            <a-button @click="$router.push('/approval/pending')"><AuditOutlined /> 待审批</a-button>
          </a-space>
        </a-card>
      </a-col>
    </a-row>

    <!-- 流水线和成本概览 -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <!-- 流水线执行统计 -->
      <a-col :xs="24" :lg="8">
        <a-card title="流水线执行" :bordered="false">
          <template #extra>
            <a-button type="link" size="small" @click="$router.push('/pipeline/list')">查看全部</a-button>
          </template>
          <div class="pipeline-stats">
            <a-row :gutter="16">
              <a-col :span="8">
                <a-statistic title="今日执行" :value="pipelineStats.todayRuns" :value-style="{ fontSize: '24px' }" />
              </a-col>
              <a-col :span="8">
                <a-statistic title="成功率" :value="pipelineStats.successRate" suffix="%" :value-style="{ fontSize: '24px', color: '#52c41a' }" />
              </a-col>
              <a-col :span="8">
                <a-statistic title="平均耗时" :value="pipelineStats.avgDuration" suffix="分" :value-style="{ fontSize: '24px' }" />
              </a-col>
            </a-row>
            <a-divider style="margin: 16px 0" />
            <div class="recent-runs">
              <div v-for="run in recentPipelineRuns" :key="run.id" class="run-item">
                <div class="run-info">
                  <a-badge :status="getRunStatus(run.status)" />
                  <span class="run-name">{{ run.pipeline_name }}</span>
                </div>
                <span class="run-time">{{ formatTime(run.created_at) }}</span>
              </div>
              <a-empty v-if="recentPipelineRuns.length === 0" description="暂无执行记录" :image="Empty.PRESENTED_IMAGE_SIMPLE" />
            </div>
          </div>
        </a-card>
      </a-col>

      <!-- 成本概览 -->
      <a-col :xs="24" :lg="8">
        <a-card title="成本概览" :bordered="false">
          <template #extra>
            <a-button type="link" size="small" @click="$router.push('/cost/overview')">查看详情</a-button>
          </template>
          <div class="cost-overview">
            <div class="cost-main">
              <div class="cost-value">¥{{ formatCost(costStats.monthCost) }}</div>
              <div class="cost-label">本月成本</div>
              <div class="cost-trend" :class="costStats.trend > 0 ? 'up' : 'down'">
                <template v-if="costStats.trend > 0">
                  <ArrowUpOutlined /> +{{ costStats.trend }}%
                </template>
                <template v-else>
                  <ArrowDownOutlined /> {{ costStats.trend }}%
                </template>
                <span class="trend-label">环比上月</span>
              </div>
            </div>
            <a-divider style="margin: 16px 0" />
            <a-row :gutter="16">
              <a-col :span="12">
                <div class="cost-item">
                  <span class="cost-item-label">优化空间</span>
                  <span class="cost-item-value savings">¥{{ formatCost(costStats.savings) }}</span>
                </div>
              </a-col>
              <a-col :span="12">
                <div class="cost-item">
                  <span class="cost-item-label">闲置资源</span>
                  <span class="cost-item-value warning">{{ costStats.idleResources }}</span>
                </div>
              </a-col>
            </a-row>
          </div>
        </a-card>
      </a-col>

      <!-- 安全概览 -->
      <a-col :xs="24" :lg="8">
        <a-card title="安全概览" :bordered="false">
          <template #extra>
            <a-button type="link" size="small" @click="$router.push('/security/overview')">查看详情</a-button>
          </template>
          <div class="security-overview">
            <a-row :gutter="16">
              <a-col :span="8">
                <div class="security-item critical">
                  <div class="security-value">{{ securityStats.critical }}</div>
                  <div class="security-label">严重</div>
                </div>
              </a-col>
              <a-col :span="8">
                <div class="security-item high">
                  <div class="security-value">{{ securityStats.high }}</div>
                  <div class="security-label">高危</div>
                </div>
              </a-col>
              <a-col :span="8">
                <div class="security-item medium">
                  <div class="security-value">{{ securityStats.medium }}</div>
                  <div class="security-label">中危</div>
                </div>
              </a-col>
            </a-row>
            <a-divider style="margin: 16px 0" />
            <div class="security-actions">
              <a-button size="small" @click="$router.push('/security/image-scan')">
                <ScanOutlined /> 镜像扫描
              </a-button>
              <a-button size="small" @click="$router.push('/security/config-check')">
                <SettingOutlined /> 配置检查
              </a-button>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 健康状态和告警 -->
    <a-row :gutter="[16, 16]" style="margin-top: 16px">
      <a-col :xs="24" :lg="8">
        <a-card title="健康状态" :bordered="false" class="health-card">
          <template #extra>
            <a-button type="link" size="small" @click="$router.push('/healthcheck')">查看详情</a-button>
          </template>
          <div class="health-overview">
            <div class="health-ring">
              <a-progress type="circle" :percent="healthPercent" :stroke-color="getHealthColor(healthOverview.status)" :width="100">
                <template #format>
                  <div class="health-center">
                    <div class="health-value">{{ healthOverview.healthy }}/{{ healthOverview.total }}</div>
                    <div class="health-label">健康</div>
                  </div>
                </template>
              </a-progress>
            </div>
            <div class="health-stats">
              <div class="health-item">
                <span class="dot green"></span>
                <span>健康: {{ healthOverview.healthy }}</span>
              </div>
              <div class="health-item">
                <span class="dot red"></span>
                <span>异常: {{ healthOverview.unhealthy }}</span>
              </div>
              <div class="health-item">
                <span class="dot gray"></span>
                <span>未知: {{ healthOverview.unknown }}</span>
              </div>
            </div>
          </div>
        </a-card>
      </a-col>

      <a-col :xs="24" :lg="8">
        <a-card title="最近告警" :bordered="false">
          <template #extra>
            <a-button type="link" size="small" @click="$router.push('/alert/history')">更多</a-button>
          </template>
          <a-list :data-source="recentAlerts" :loading="loadingAlerts" size="small">
            <template #renderItem="{ item }">
              <a-list-item>
                <a-list-item-meta>
                  <template #avatar>
                    <a-tag :color="getLevelColor(item.level)" size="small">{{ getLevelText(item.level) }}</a-tag>
                  </template>
                  <template #title>{{ item.title || item.type }}</template>
                  <template #description>{{ formatTime(item.created_at) }}</template>
                </a-list-item-meta>
              </a-list-item>
            </template>
            <template #empty>
              <a-empty description="暂无告警" :image="Empty.PRESENTED_IMAGE_SIMPLE" />
            </template>
          </a-list>
        </a-card>
      </a-col>

      <a-col :xs="24" :lg="8">
        <a-card title="最近操作" :bordered="false">
          <template #extra>
            <a-button type="link" size="small" @click="$router.push('/audit/logs')">更多</a-button>
          </template>
          <a-list :data-source="recentAudits" :loading="loadingAudits" size="small">
            <template #renderItem="{ item }">
              <a-list-item>
                <a-list-item-meta>
                  <template #avatar>
                    <a-avatar size="small" style="background-color: #1890ff">{{ item.username?.charAt(0)?.toUpperCase() }}</a-avatar>
                  </template>
                  <template #title>
                    <span>{{ item.username }}</span>
                    <a-tag size="small" style="margin-left: 8px">{{ getActionText(item.action) }}</a-tag>
                  </template>
                  <template #description>
                    {{ item.resource }} {{ item.resource_id ? `#${item.resource_id}` : '' }}
                    <span style="margin-left: 8px; color: #999">{{ formatTime(item.created_at) }}</span>
                  </template>
                </a-list-item-meta>
              </a-list-item>
            </template>
            <template #empty>
              <a-empty description="暂无操作记录" :image="Empty.PRESENTED_IMAGE_SIMPLE" />
            </template>
          </a-list>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Empty } from 'ant-design-vue'
import { 
  RocketOutlined, CloudOutlined, AlertOutlined, AppstoreOutlined,
  SafetyCertificateOutlined, AuditOutlined, ArrowUpOutlined, ArrowDownOutlined,
  ScanOutlined, SettingOutlined, PlusOutlined, FileSearchOutlined, DollarOutlined
} from '@ant-design/icons-vue'
import { dashboardApi, type DashboardStats, type HealthOverview, type RecentAlert, type RecentAudit } from '@/services/dashboard'
import request from '@/services/api'

const stats = ref<DashboardStats & { applications?: number; pendingApprovals?: number }>({ 
  jenkinsInstances: 0, k8sClusters: 0, users: 0, healthChecks: 0, alertsToday: 0, auditsToday: 0 
})
const healthOverview = ref<HealthOverview>({ status: 'unknown', healthy: 0, unhealthy: 0, unknown: 0, total: 0 })
const recentAlerts = ref<RecentAlert[]>([])
const recentAudits = ref<RecentAudit[]>([])
const loadingAlerts = ref(false)
const loadingAudits = ref(false)

// 流水线统计
const pipelineStats = ref({
  total: 0,
  todayRuns: 0,
  successRate: 0,
  avgDuration: 0
})
const recentPipelineRuns = ref<any[]>([])

// 成本统计
const costStats = ref({
  monthCost: 0,
  trend: 0,
  savings: 0,
  idleResources: 0
})

// 安全统计
const securityStats = ref({
  highRisk: 0,
  critical: 0,
  high: 0,
  medium: 0
})

const healthPercent = computed(() => {
  if (healthOverview.value.total === 0) return 100
  return Math.round((healthOverview.value.healthy / healthOverview.value.total) * 100)
})

const getHealthColor = (status: string) => ({ healthy: '#52c41a', unhealthy: '#ff4d4f', unknown: '#d9d9d9' }[status] || '#d9d9d9')
const getHealthText = (status: string) => ({ healthy: '系统健康', unhealthy: '存在异常', unknown: '状态未知' }[status] || '状态未知')
const getLevelColor = (level: string) => ({ info: 'blue', warning: 'orange', error: 'red', critical: 'magenta' }[level] || 'default')
const getLevelText = (level: string) => ({ info: '信息', warning: '警告', error: '错误', critical: '严重' }[level] || level)
const getActionText = (action: string) => ({ create: '创建', update: '更新', delete: '删除' }[action] || action)
const getRunStatus = (status: string) => ({ success: 'success', running: 'processing', failed: 'error', pending: 'default' }[status] || 'default')

const formatTime = (time: string) => {
  if (!time) return '-'
  const date = new Date(time)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
  return time.replace('T', ' ').substring(5, 16)
}

const formatCost = (cost: number) => {
  if (cost >= 10000) return (cost / 10000).toFixed(1) + '万'
  return cost.toFixed(0)
}

const fetchData = async () => {
  try {
    const [statsRes, healthRes] = await Promise.all([
      dashboardApi.getStats(),
      dashboardApi.getHealthOverview()
    ])
    if (statsRes.code === 0 && statsRes.data) stats.value = { ...stats.value, ...statsRes.data }
    if (healthRes.code === 0 && healthRes.data) healthOverview.value = healthRes.data
  } catch (error) { console.error('获取统计数据失败', error) }
}

const fetchAlerts = async () => {
  loadingAlerts.value = true
  try {
    const res = await dashboardApi.getRecentAlerts()
    if (res.code === 0 && res.data) recentAlerts.value = res.data
  } catch (error) { console.error('获取告警失败', error) }
  finally { loadingAlerts.value = false }
}

const fetchAudits = async () => {
  loadingAudits.value = true
  try {
    const res = await dashboardApi.getRecentAudits()
    if (res.code === 0 && res.data) recentAudits.value = res.data
  } catch (error) { console.error('获取审计失败', error) }
  finally { loadingAudits.value = false }
}

// 获取流水线统计
const fetchPipelineStats = async () => {
  try {
    // 获取流水线列表
    const pipelinesRes = await request.get('/pipelines', { params: { page: 1, page_size: 1 } })
    pipelineStats.value.total = pipelinesRes?.data?.total || 0

    // 获取今日执行记录
    const today = new Date().toISOString().split('T')[0]
    const runsRes = await request.get('/pipelines/runs', { params: { page: 1, page_size: 10 } })
    const runs = runsRes?.data?.items || []
    recentPipelineRuns.value = runs.slice(0, 5)

    // 计算今日执行数和成功率
    const todayRuns = runs.filter((r: any) => r.created_at?.startsWith(today))
    pipelineStats.value.todayRuns = todayRuns.length
    const successRuns = runs.filter((r: any) => r.status === 'success')
    pipelineStats.value.successRate = runs.length > 0 ? Math.round((successRuns.length / runs.length) * 100) : 0

    // 计算平均耗时
    const durations = runs.filter((r: any) => r.duration > 0).map((r: any) => r.duration)
    pipelineStats.value.avgDuration = durations.length > 0 ? Math.round(durations.reduce((a: number, b: number) => a + b, 0) / durations.length / 60) : 0
  } catch (error) {
    console.error('获取流水线统计失败', error)
  }
}

// 获取成本统计
const fetchCostStats = async () => {
  try {
    const res = await request.get('/cost/overview')
    if (res?.data) {
      costStats.value = {
        monthCost: res.data.month_cost || res.data.total_cost || 0,
        trend: res.data.trend || res.data.month_over_month || 0,
        savings: res.data.potential_savings || res.data.savings || 0,
        idleResources: res.data.idle_resources || res.data.idle_count || 0
      }
    }
  } catch (error) {
    console.error('获取成本统计失败', error)
  }
}

// 获取安全统计
const fetchSecurityStats = async () => {
  try {
    const res = await request.get('/security/overview')
    if (res?.data) {
      securityStats.value = {
        highRisk: (res.data.critical || 0) + (res.data.high || 0),
        critical: res.data.critical || 0,
        high: res.data.high || 0,
        medium: res.data.medium || 0
      }
    }
  } catch (error) {
    console.error('获取安全统计失败', error)
  }
}

onMounted(() => {
  fetchData()
  fetchAlerts()
  fetchAudits()
  fetchPipelineStats()
  fetchCostStats()
  fetchSecurityStats()
})
</script>

<style scoped>
.dashboard { padding: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h1 { font-size: 24px; font-weight: 500; margin: 0; }

/* 流水线统计 */
.pipeline-stats .recent-runs { max-height: 150px; overflow-y: auto; }
.run-item { display: flex; justify-content: space-between; align-items: center; padding: 6px 0; border-bottom: 1px solid #f0f0f0; }
.run-item:last-child { border-bottom: none; }
.run-info { display: flex; align-items: center; gap: 8px; }
.run-name { font-size: 13px; max-width: 180px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.run-time { font-size: 12px; color: #999; }

/* 成本概览 */
.cost-overview .cost-main { text-align: center; }
.cost-value { font-size: 32px; font-weight: 600; color: #1890ff; }
.cost-label { font-size: 14px; color: #999; margin-top: 4px; }
.cost-trend { font-size: 14px; margin-top: 8px; }
.cost-trend.up { color: #ff4d4f; }
.cost-trend.down { color: #52c41a; }
.trend-label { margin-left: 8px; color: #999; font-size: 12px; }
.cost-item { text-align: center; }
.cost-item-label { font-size: 12px; color: #999; display: block; }
.cost-item-value { font-size: 18px; font-weight: 500; }
.cost-item-value.savings { color: #52c41a; }
.cost-item-value.warning { color: #fa8c16; }

/* 安全概览 */
.security-overview .security-item { text-align: center; padding: 12px 0; }
.security-value { font-size: 28px; font-weight: 600; }
.security-label { font-size: 12px; color: #999; margin-top: 4px; }
.security-item.critical .security-value { color: #ff4d4f; }
.security-item.high .security-value { color: #fa8c16; }
.security-item.medium .security-value { color: #faad14; }
.security-actions { display: flex; gap: 8px; justify-content: center; }

/* 健康状态 */
.health-card .health-overview { display: flex; align-items: center; gap: 24px; }
.health-ring { flex-shrink: 0; }
.health-center { text-align: center; }
.health-value { font-size: 18px; font-weight: 600; color: #333; }
.health-label { font-size: 12px; color: #999; }
.health-stats { flex: 1; }
.health-item { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.health-item .dot { width: 8px; height: 8px; border-radius: 50%; }
.health-item .dot.green { background: #52c41a; }
.health-item .dot.red { background: #ff4d4f; }
.health-item .dot.gray { background: #d9d9d9; }

/* 列表高度限制 - 防止页面过长 */
:deep(.ant-list) {
  max-height: 320px;
  overflow-y: auto;
}

/* 优化滚动条样式 */
:deep(.ant-list::-webkit-scrollbar) {
  width: 6px;
}

:deep(.ant-list::-webkit-scrollbar-track) {
  background: #f0f0f0;
  border-radius: 3px;
}

:deep(.ant-list::-webkit-scrollbar-thumb) {
  background: #bfbfbf;
  border-radius: 3px;
}

:deep(.ant-list::-webkit-scrollbar-thumb:hover) {
  background: #999;
}

:deep(.ant-card) { cursor: default; }
:deep(.ant-card[class*="hoverable"]) { cursor: pointer; }
</style>
