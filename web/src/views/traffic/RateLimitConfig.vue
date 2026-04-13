<template>
  <div class="traffic-config">
    <a-page-header title="限流配置" sub-title="多维度流量控制，保护服务稳定性">
      <template #extra>
        <a-select v-model:value="selectedAppId" placeholder="选择应用" style="width: 200px" @change="onAppChange" show-search option-filter-prop="label">
          <a-select-option v-for="app in apps" :key="app.id" :value="app.id" :label="app.display_name || app.name">
            {{ app.display_name || app.name }}
          </a-select-option>
        </a-select>
        <a-button type="primary" @click="showModal()" :disabled="!selectedAppId"><PlusOutlined /> 添加规则</a-button>
      </template>
    </a-page-header>

    <a-alert v-if="!selectedAppId" type="info" show-icon style="margin-bottom: 16px">
      <template #message>请先选择一个应用来管理其限流规则</template>
    </a-alert>

    <!-- 限流策略说明 -->
    <a-row :gutter="16" style="margin-bottom: 16px" v-if="selectedAppId">
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="QPS 限流" :value="stats.qps_rules" suffix="条">
            <template #prefix><ThunderboltOutlined style="color: #1890ff" /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="并发限流" :value="stats.concurrent_rules" suffix="条">
            <template #prefix><TeamOutlined style="color: #52c41a" /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="热点限流" :value="stats.hotspot_rules" suffix="条">
            <template #prefix><FireOutlined style="color: #fa8c16" /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="已启用" :value="stats.enabled_rules" suffix="条">
            <template #prefix><CheckCircleOutlined style="color: #52c41a" /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <a-card :bordered="false" v-if="selectedAppId">
      <a-table :columns="columns" :data-source="rules" :loading="loading" row-key="id" size="middle">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <div>
              <span>{{ record.name }}</span>
              <div style="color: #999; font-size: 12px">{{ record.description }}</div>
            </div>
          </template>
          <template v-if="column.key === 'resource'">
            <a-tag color="blue">{{ record.resource_type }}</a-tag>
            <code style="margin-left: 4px">{{ record.resource }}</code>
          </template>
          <template v-if="column.key === 'strategy'">
            <a-tag :color="getStrategyColor(record.strategy)">{{ getStrategyText(record.strategy) }}</a-tag>
          </template>
          <template v-if="column.key === 'threshold'">
            <span v-if="record.strategy === 'qps'">{{ record.threshold }} req/s</span>
            <span v-else-if="record.strategy === 'concurrent'">{{ record.threshold }} 并发</span>
            <span v-else-if="record.strategy === 'token_bucket'">{{ record.threshold }} tokens/s (容量: {{ record.burst }})</span>
            <span v-else-if="record.strategy === 'leaky_bucket'">{{ record.threshold }} req/s (队列: {{ record.queue_size }})</span>
            <span v-else>{{ record.threshold }}</span>
          </template>
          <template v-if="column.key === 'control_behavior'">
            <a-tag>{{ getBehaviorText(record.control_behavior) }}</a-tag>
          </template>
          <template v-if="column.key === 'enabled'">
            <a-switch v-model:checked="record.enabled" size="small" @change="toggleRule(record)" />
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showModal(record)">编辑</a-button>
              <a-popconfirm title="确定删除？" @confirm="deleteRule(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 编辑弹窗 -->
    <a-modal v-model:open="modalVisible" :title="editingRule ? '编辑限流规则' : '添加限流规则'" @ok="saveRule" :confirm-loading="saving" width="650px">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="规则名称" required>
          <a-input v-model:value="form.name" placeholder="如：用户接口限流" />
        </a-form-item>
        <a-form-item label="规则描述">
          <a-input v-model:value="form.description" placeholder="规则用途说明" />
        </a-form-item>
        
        <a-divider>资源配置</a-divider>
        <a-form-item label="资源类型" required>
          <a-radio-group v-model:value="form.resource_type">
            <a-radio value="api">API 接口</a-radio>
            <a-radio value="service">服务</a-radio>
            <a-radio value="method">方法</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="资源标识" required>
          <a-input v-model:value="form.resource" placeholder="/api/v1/users 或 UserService.getUser" />
        </a-form-item>
        <a-form-item label="请求方法" v-if="form.resource_type === 'api'">
          <a-select v-model:value="form.method" placeholder="全部" allow-clear>
            <a-select-option value="">全部</a-select-option>
            <a-select-option value="GET">GET</a-select-option>
            <a-select-option value="POST">POST</a-select-option>
            <a-select-option value="PUT">PUT</a-select-option>
            <a-select-option value="DELETE">DELETE</a-select-option>
          </a-select>
        </a-form-item>

        <a-divider>限流策略</a-divider>
        <a-form-item label="限流类型" required>
          <a-select v-model:value="form.strategy" @change="onStrategyChange">
            <a-select-option value="qps">QPS 限流 (每秒请求数)</a-select-option>
            <a-select-option value="concurrent">并发限流 (同时处理数)</a-select-option>
            <a-select-option value="token_bucket">令牌桶 (平滑限流)</a-select-option>
            <a-select-option value="leaky_bucket">漏桶 (匀速排队)</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="阈值" required>
          <a-input-number v-model:value="form.threshold" :min="1" :max="1000000" style="width: 100%" />
          <div style="color: #999; font-size: 12px">
            <span v-if="form.strategy === 'qps'">每秒允许通过的请求数</span>
            <span v-else-if="form.strategy === 'concurrent'">允许同时处理的请求数</span>
            <span v-else-if="form.strategy === 'token_bucket'">每秒生成的令牌数</span>
            <span v-else-if="form.strategy === 'leaky_bucket'">每秒处理的请求数</span>
          </div>
        </a-form-item>
        <a-form-item label="突发容量" v-if="form.strategy === 'token_bucket'">
          <a-input-number v-model:value="form.burst" :min="0" :max="100000" style="width: 100%" />
          <div style="color: #999; font-size: 12px">令牌桶最大容量，允许短时间突发流量</div>
        </a-form-item>
        <a-form-item label="队列大小" v-if="form.strategy === 'leaky_bucket'">
          <a-input-number v-model:value="form.queue_size" :min="0" :max="10000" style="width: 100%" />
          <div style="color: #999; font-size: 12px">等待队列大小，超出则直接拒绝</div>
        </a-form-item>
        <a-form-item label="超限行为">
          <a-select v-model:value="form.control_behavior">
            <a-select-option value="reject">直接拒绝</a-select-option>
            <a-select-option value="warm_up">预热 (Warm Up)</a-select-option>
            <a-select-option value="queue">排队等待</a-select-option>
            <a-select-option value="warm_up_queue">预热+排队</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="预热时长" v-if="form.control_behavior?.includes('warm_up')">
          <a-input-number v-model:value="form.warm_up_period" :min="1" :max="3600" style="width: 100%" addon-after="秒" />
          <div style="color: #999; font-size: 12px">系统冷启动时逐渐增加阈值的时间</div>
        </a-form-item>
        <a-form-item label="最大等待" v-if="form.control_behavior?.includes('queue')">
          <a-input-number v-model:value="form.max_queue_time" :min="0" :max="60000" style="width: 100%" addon-after="毫秒" />
          <div style="color: #999; font-size: 12px">请求在队列中最大等待时间</div>
        </a-form-item>

        <a-divider>高级配置</a-divider>
        <a-form-item label="限流维度">
          <a-checkbox-group v-model:value="form.limit_dimensions">
            <a-checkbox value="ip">按 IP</a-checkbox>
            <a-checkbox value="user">按用户</a-checkbox>
            <a-checkbox value="api_key">按 API Key</a-checkbox>
            <a-checkbox value="header">按请求头</a-checkbox>
          </a-checkbox-group>
        </a-form-item>
        <a-form-item label="限流 Header" v-if="form.limit_dimensions?.includes('header')">
          <a-input v-model:value="form.limit_header" placeholder="X-Tenant-Id" />
        </a-form-item>
        <a-form-item label="返回状态码">
          <a-select v-model:value="form.rejected_code">
            <a-select-option :value="429">429 Too Many Requests</a-select-option>
            <a-select-option :value="503">503 Service Unavailable</a-select-option>
            <a-select-option :value="403">403 Forbidden</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="form.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, ThunderboltOutlined, TeamOutlined, FireOutlined, CheckCircleOutlined } from '@ant-design/icons-vue'
import request from '@/utils/request'
import { applicationApi, type Application } from '@/services/application'

const loading = ref(false)
const saving = ref(false)
const modalVisible = ref(false)
const selectedAppId = ref<number | undefined>()
const editingRule = ref<any>(null)

const apps = ref<Application[]>([])
const rules = ref<any[]>([])

const form = reactive({
  name: '',
  description: '',
  resource_type: 'api',
  resource: '',
  method: '',
  strategy: 'qps',
  threshold: 100,
  burst: 10,
  queue_size: 100,
  control_behavior: 'reject',
  warm_up_period: 10,
  max_queue_time: 500,
  limit_dimensions: [] as string[],
  limit_header: '',
  rejected_code: 429,
  enabled: true
})

const stats = computed(() => {
  const qps = rules.value.filter(r => r.strategy === 'qps').length
  const concurrent = rules.value.filter(r => r.strategy === 'concurrent').length
  const hotspot = rules.value.filter(r => r.strategy === 'hotspot').length
  const enabled = rules.value.filter(r => r.enabled).length
  return { qps_rules: qps, concurrent_rules: concurrent, hotspot_rules: hotspot, enabled_rules: enabled }
})

const columns = [
  { title: '规则名称', key: 'name', width: 200 },
  { title: '资源', key: 'resource' },
  { title: '策略', key: 'strategy', width: 120 },
  { title: '阈值', key: 'threshold', width: 180 },
  { title: '超限行为', key: 'control_behavior', width: 100 },
  { title: '启用', key: 'enabled', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const strategyMap: Record<string, { text: string; color: string }> = {
  qps: { text: 'QPS限流', color: 'blue' },
  concurrent: { text: '并发限流', color: 'green' },
  token_bucket: { text: '令牌桶', color: 'purple' },
  leaky_bucket: { text: '漏桶', color: 'orange' }
}
const getStrategyText = (s: string) => strategyMap[s]?.text || s
const getStrategyColor = (s: string) => strategyMap[s]?.color || 'default'

const behaviorMap: Record<string, string> = {
  reject: '直接拒绝',
  warm_up: '预热',
  queue: '排队',
  warm_up_queue: '预热+排队'
}
const getBehaviorText = (b: string) => behaviorMap[b] || b

const fetchApps = async () => {
  try {
    const response = await applicationApi.list({ page: 1, page_size: 1000 })
    if (response.code === 0 && response.data) {
      apps.value = (response.data.list || []).filter((a: Application) => a.k8s_deployment)
    }
  } catch (e) { console.error('获取应用列表失败', e) }
}

const fetchRules = async () => {
  if (!selectedAppId.value) return
  loading.value = true
  try {
    const res = await request.get(`/applications/${selectedAppId.value}/traffic/ratelimits`)
    rules.value = res.data?.items || []
  } catch (e) { console.error('获取限流规则失败', e) }
  finally { loading.value = false }
}

const onAppChange = () => { fetchRules() }
const onStrategyChange = () => {
  if (form.strategy === 'token_bucket') { form.burst = 10 }
  if (form.strategy === 'leaky_bucket') { form.queue_size = 100 }
}

const showModal = (record?: any) => {
  editingRule.value = record || null
  if (record) {
    Object.assign(form, record)
  } else {
    Object.assign(form, {
      name: '', description: '', resource_type: 'api', resource: '', method: '',
      strategy: 'qps', threshold: 100, burst: 10, queue_size: 100,
      control_behavior: 'reject', warm_up_period: 10, max_queue_time: 500,
      limit_dimensions: [], limit_header: '', rejected_code: 429, enabled: true
    })
  }
  modalVisible.value = true
}

const saveRule = async () => {
  if (!form.name || !form.resource) { message.warning('请填写必填项'); return }
  saving.value = true
  try {
    if (editingRule.value) {
      await request.put(`/applications/${selectedAppId.value}/traffic/ratelimits/${editingRule.value.id}`, form)
    } else {
      await request.post(`/applications/${selectedAppId.value}/traffic/ratelimits`, form)
    }
    message.success('保存成功')
    modalVisible.value = false
    fetchRules()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

const toggleRule = async (record: any) => {
  try {
    await request.put(`/applications/${selectedAppId.value}/traffic/ratelimits/${record.id}`, { enabled: record.enabled })
    message.success(record.enabled ? '已启用' : '已禁用')
  } catch (e) { record.enabled = !record.enabled; message.error('操作失败') }
}

const deleteRule = async (id: number) => {
  try {
    await request.delete(`/applications/${selectedAppId.value}/traffic/ratelimits/${id}`)
    message.success('删除成功')
    fetchRules()
  } catch (e) { message.error('删除失败') }
}

onMounted(() => { fetchApps() })
</script>

<style scoped>
.traffic-config { padding: 0; }
</style>
