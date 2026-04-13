<template>
  <div class="traffic-config">
    <a-page-header title="熔断配置" sub-title="服务熔断降级，防止故障扩散">
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
      <template #message>请先选择一个应用来管理其熔断规则</template>
    </a-alert>

    <!-- 熔断状态概览 -->
    <a-row :gutter="16" style="margin-bottom: 16px" v-if="selectedAppId">
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="慢调用熔断" :value="stats.slow_request" suffix="条">
            <template #prefix><ClockCircleOutlined style="color: #fa8c16" /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="异常比例熔断" :value="stats.error_ratio" suffix="条">
            <template #prefix><PercentageOutlined style="color: #f5222d" /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="异常数熔断" :value="stats.error_count" suffix="条">
            <template #prefix><ExclamationCircleOutlined style="color: #ff4d4f" /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="当前熔断中" :value="stats.open_breakers" suffix="个">
            <template #prefix><StopOutlined style="color: '#ff4d4f'" /></template>
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
              <div style="color: #999; font-size: 12px">{{ record.resource }}</div>
            </div>
          </template>
          <template v-if="column.key === 'strategy'">
            <a-tag :color="getStrategyColor(record.strategy)">{{ getStrategyText(record.strategy) }}</a-tag>
          </template>
          <template v-if="column.key === 'threshold'">
            <span v-if="record.strategy === 'slow_request'">
              RT > {{ record.slow_rt_threshold }}ms, 比例 ≥ {{ record.threshold }}%
            </span>
            <span v-else-if="record.strategy === 'error_ratio'">
              异常比例 ≥ {{ record.threshold }}%
            </span>
            <span v-else-if="record.strategy === 'error_count'">
              异常数 ≥ {{ record.threshold }}
            </span>
          </template>
          <template v-if="column.key === 'recovery'">
            <span>{{ record.recovery_timeout }}s 后探测</span>
            <span v-if="record.min_request_amount" style="color: #999"> (最小请求: {{ record.min_request_amount }})</span>
          </template>
          <template v-if="column.key === 'status'">
            <a-badge v-if="record.circuit_status === 'open'" status="error" text="熔断中" />
            <a-badge v-else-if="record.circuit_status === 'half_open'" status="warning" text="半开" />
            <a-badge v-else status="success" text="关闭" />
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
    <a-modal v-model:open="modalVisible" :title="editingRule ? '编辑熔断规则' : '添加熔断规则'" @ok="saveRule" :confirm-loading="saving" width="650px">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="规则名称" required>
          <a-input v-model:value="form.name" placeholder="如：订单服务熔断" />
        </a-form-item>
        <a-form-item label="资源标识" required>
          <a-input v-model:value="form.resource" placeholder="/api/v1/orders 或 OrderService" />
        </a-form-item>

        <a-divider>熔断策略</a-divider>
        <a-form-item label="熔断类型" required>
          <a-radio-group v-model:value="form.strategy" @change="onStrategyChange">
            <a-radio-button value="slow_request">慢调用比例</a-radio-button>
            <a-radio-button value="error_ratio">异常比例</a-radio-button>
            <a-radio-button value="error_count">异常数</a-radio-button>
          </a-radio-group>
        </a-form-item>

        <template v-if="form.strategy === 'slow_request'">
          <a-form-item label="慢调用阈值" required>
            <a-input-number v-model:value="form.slow_rt_threshold" :min="1" :max="60000" style="width: 100%" addon-after="毫秒" />
            <div style="color: #999; font-size: 12px">响应时间超过此值视为慢调用</div>
          </a-form-item>
          <a-form-item label="慢调用比例" required>
            <a-slider v-model:value="form.threshold" :min="0" :max="100" :marks="{ 0: '0%', 50: '50%', 100: '100%' }" />
            <div style="color: #999; font-size: 12px">慢调用比例达到此值触发熔断</div>
          </a-form-item>
        </template>

        <template v-if="form.strategy === 'error_ratio'">
          <a-form-item label="异常比例" required>
            <a-slider v-model:value="form.threshold" :min="0" :max="100" :marks="{ 0: '0%', 50: '50%', 100: '100%' }" />
            <div style="color: #999; font-size: 12px">异常比例达到此值触发熔断</div>
          </a-form-item>
        </template>

        <template v-if="form.strategy === 'error_count'">
          <a-form-item label="异常数" required>
            <a-input-number v-model:value="form.threshold" :min="1" :max="10000" style="width: 100%" />
            <div style="color: #999; font-size: 12px">异常数达到此值触发熔断</div>
          </a-form-item>
        </template>

        <a-divider>统计与恢复</a-divider>
        <a-form-item label="统计窗口">
          <a-input-number v-model:value="form.stat_interval" :min="1" :max="120" style="width: 100%" addon-after="秒" />
          <div style="color: #999; font-size: 12px">统计时间窗口长度</div>
        </a-form-item>
        <a-form-item label="最小请求数">
          <a-input-number v-model:value="form.min_request_amount" :min="1" :max="10000" style="width: 100%" />
          <div style="color: #999; font-size: 12px">触发熔断的最小请求数，避免小流量误判</div>
        </a-form-item>
        <a-form-item label="熔断时长">
          <a-input-number v-model:value="form.recovery_timeout" :min="1" :max="3600" style="width: 100%" addon-after="秒" />
          <div style="color: #999; font-size: 12px">熔断后多久进入半开状态进行探测</div>
        </a-form-item>
        <a-form-item label="探测请求数">
          <a-input-number v-model:value="form.probe_num" :min="1" :max="100" style="width: 100%" />
          <div style="color: #999; font-size: 12px">半开状态允许通过的探测请求数</div>
        </a-form-item>

        <a-divider>降级配置</a-divider>
        <a-form-item label="降级策略">
          <a-select v-model:value="form.fallback_strategy">
            <a-select-option value="return_error">返回错误</a-select-option>
            <a-select-option value="return_default">返回默认值</a-select-option>
            <a-select-option value="call_fallback">调用降级服务</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="降级返回值" v-if="form.fallback_strategy === 'return_default'">
          <a-textarea v-model:value="form.fallback_value" :rows="3" placeholder='{"code": -1, "message": "服务暂时不可用"}' />
        </a-form-item>
        <a-form-item label="降级服务" v-if="form.fallback_strategy === 'call_fallback'">
          <a-input v-model:value="form.fallback_service" placeholder="fallback-service:8080" />
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
import { PlusOutlined, ClockCircleOutlined, PercentageOutlined, ExclamationCircleOutlined, StopOutlined } from '@ant-design/icons-vue'
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
  resource: '',
  strategy: 'slow_request',
  slow_rt_threshold: 1000,
  threshold: 50,
  stat_interval: 10,
  min_request_amount: 5,
  recovery_timeout: 30,
  probe_num: 3,
  fallback_strategy: 'return_error',
  fallback_value: '',
  fallback_service: '',
  enabled: true
})

const stats = computed(() => {
  const slow = rules.value.filter(r => r.strategy === 'slow_request').length
  const ratio = rules.value.filter(r => r.strategy === 'error_ratio').length
  const count = rules.value.filter(r => r.strategy === 'error_count').length
  const open = rules.value.filter(r => r.circuit_status === 'open').length
  return { slow_request: slow, error_ratio: ratio, error_count: count, open_breakers: open }
})

const columns = [
  { title: '规则名称', key: 'name', width: 200 },
  { title: '策略', key: 'strategy', width: 120 },
  { title: '触发条件', key: 'threshold' },
  { title: '恢复配置', key: 'recovery', width: 180 },
  { title: '状态', key: 'status', width: 100 },
  { title: '启用', key: 'enabled', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const strategyMap: Record<string, { text: string; color: string }> = {
  slow_request: { text: '慢调用比例', color: 'orange' },
  error_ratio: { text: '异常比例', color: 'red' },
  error_count: { text: '异常数', color: 'volcano' }
}
const getStrategyText = (s: string) => strategyMap[s]?.text || s
const getStrategyColor = (s: string) => strategyMap[s]?.color || 'default'

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
    const res = await request.get(`/applications/${selectedAppId.value}/traffic/circuitbreakers`)
    rules.value = res.data?.items || []
  } catch (e) { console.error('获取熔断规则失败', e) }
  finally { loading.value = false }
}

const onAppChange = () => { fetchRules() }
const onStrategyChange = () => {
  if (form.strategy === 'slow_request') { form.threshold = 50; form.slow_rt_threshold = 1000 }
  else if (form.strategy === 'error_ratio') { form.threshold = 50 }
  else { form.threshold = 5 }
}

const showModal = (record?: any) => {
  editingRule.value = record || null
  if (record) {
    Object.assign(form, record)
  } else {
    Object.assign(form, {
      name: '', resource: '', strategy: 'slow_request', slow_rt_threshold: 1000,
      threshold: 50, stat_interval: 10, min_request_amount: 5, recovery_timeout: 30,
      probe_num: 3, fallback_strategy: 'return_error', fallback_value: '', fallback_service: '', enabled: true
    })
  }
  modalVisible.value = true
}

const saveRule = async () => {
  if (!form.name || !form.resource) { message.warning('请填写必填项'); return }
  saving.value = true
  try {
    let res
    if (editingRule.value) {
      res = await request.put(`/applications/${selectedAppId.value}/traffic/circuitbreakers/${editingRule.value.id}`, form)
    } else {
      res = await request.post(`/applications/${selectedAppId.value}/traffic/circuitbreakers`, form)
    }
    
    // 检查 K8s 同步状态
    if (res?.k8s_synced === false && res?.k8s_error) {
      message.warning(`规则已保存到数据库，但同步到 K8s 失败: ${res.k8s_error}`, 5)
    } else if (res?.k8s_synced === false) {
      message.warning('规则已保存到数据库，但未同步到 K8s（可能 Istio 未安装）', 5)
    } else {
      message.success('保存成功并已同步到 K8s')
    }
    
    modalVisible.value = false
    fetchRules()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

const toggleRule = async (record: any) => {
  try {
    await request.put(`/applications/${selectedAppId.value}/traffic/circuitbreakers/${record.id}`, { enabled: record.enabled })
    message.success(record.enabled ? '已启用' : '已禁用')
  } catch (e) { record.enabled = !record.enabled; message.error('操作失败') }
}

const deleteRule = async (id: number) => {
  try {
    await request.delete(`/applications/${selectedAppId.value}/traffic/circuitbreakers/${id}`)
    message.success('删除成功')
    fetchRules()
  } catch (e) { message.error('删除失败') }
}

onMounted(() => { fetchApps() })
</script>

<style scoped>
.traffic-config { padding: 0; }
</style>
