<template>
  <div class="log-alert-config">
    <a-tabs v-model:activeKey="activeTab">
      <a-tab-pane key="rules" tab="告警规则">
        <div class="toolbar">
          <a-button type="primary" @click="showCreateDialog">
            <template #icon><PlusOutlined /></template>
            新建规则
          </a-button>
          <a-select v-model:value="filterCluster" placeholder="筛选集群" allow-clear style="width: 200px; margin-left: 10px" @change="loadRules">
            <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">{{ cluster.name }}</a-select-option>
          </a-select>
        </div>

        <a-table :dataSource="rules" :columns="rulesColumns" :loading="loading" rowKey="id" style="margin-top: 15px">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'match_type'">
              <a-tag>{{ matchTypeLabels[record.match_type] }}</a-tag>
            </template>
            <template v-else-if="column.key === 'level'">
              <a-tag v-if="record.level" :color="levelTypes[record.level]">{{ record.level }}</a-tag>
              <span v-else>-</span>
            </template>
            <template v-else-if="column.key === 'channels'">
              <a-tag v-for="ch in record.channels" :key="ch" style="margin-right: 4px">{{ ch }}</a-tag>
            </template>
            <template v-else-if="column.key === 'enabled'">
              <a-switch v-model:checked="record.enabled" @change="toggleRule(record)" />
            </template>
            <template v-else-if="column.key === 'action'">
              <a-button type="link" size="small" @click="editRule(record)">编辑</a-button>
              <a-button type="link" danger size="small" @click="deleteRule(record)">删除</a-button>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="history" tab="告警历史">
        <div class="toolbar">
          <a-select v-model:value="historyRuleId" placeholder="筛选规则" allow-clear style="width: 200px" @change="loadHistory">
            <a-select-option v-for="rule in rules" :key="rule.id" :value="rule.id">{{ rule.name }}</a-select-option>
          </a-select>
        </div>

        <a-table :dataSource="histories" :columns="historiesColumns" :loading="historyLoading" :pagination="false" rowKey="id" style="margin-top: 15px">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <a-tag :color="statusTypes[record.status]">{{ statusLabels[record.status] }}</a-tag>
            </template>
            <template v-else-if="column.key === 'created_at'">
              {{ formatTime(record.created_at) }}
            </template>
          </template>
        </a-table>

        <a-pagination
          v-if="historyTotal > 0"
          v-model:current="historyPage"
          :total="historyTotal"
          :pageSize="20"
          show-quick-jumper
          @change="loadHistory"
          style="margin-top: 15px; text-align: right"
        />
      </a-tab-pane>
    </a-tabs>

    <!-- 创建/编辑对话框 -->
    <a-modal v-model:open="showDialog" :title="editingRule ? '编辑规则' : '新建规则'" width="600px" @ok="saveRule" :confirmLoading="saving">
      <a-form :model="ruleForm" :label-col="{ span: 4 }" :wrapper-col="{ span: 20 }" :rules="formRules" ref="formRef">
        <a-form-item label="规则名称" name="name">
          <a-input v-model:value="ruleForm.name" placeholder="输入规则名称" />
        </a-form-item>
        <a-form-item label="集群" name="cluster_id">
          <a-select v-model:value="ruleForm.cluster_id" placeholder="选择集群" style="width: 100%">
            <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">{{ cluster.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="命名空间">
          <a-input v-model:value="ruleForm.namespace" placeholder="留空表示所有命名空间" />
        </a-form-item>
        <a-form-item label="匹配类型" name="match_type">
          <a-radio-group v-model:value="ruleForm.match_type">
            <a-radio value="keyword">关键词</a-radio>
            <a-radio value="regex">正则表达式</a-radio>
            <a-radio value="level">日志级别</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="匹配值" name="match_value">
          <a-input v-if="ruleForm.match_type !== 'level'" v-model:value="ruleForm.match_value" placeholder="输入匹配值" />
          <a-select v-else v-model:value="ruleForm.match_value" placeholder="选择日志级别" style="width: 100%">
            <a-select-option value="ERROR">ERROR</a-select-option>
            <a-select-option value="FATAL">FATAL</a-select-option>
            <a-select-option value="WARN">WARN</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="告警级别">
          <a-select v-model:value="ruleForm.level" placeholder="选择告警级别" style="width: 100%">
            <a-select-option value="info">信息</a-select-option>
            <a-select-option value="warning">警告</a-select-option>
            <a-select-option value="error">错误</a-select-option>
            <a-select-option value="critical">严重</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="通知渠道">
          <a-checkbox-group v-model:value="ruleForm.channels">
            <a-checkbox value="feishu">飞书</a-checkbox>
            <a-checkbox value="dingtalk">钉钉</a-checkbox>
            <a-checkbox value="wechat">企业微信</a-checkbox>
            <a-checkbox value="email">邮件</a-checkbox>
          </a-checkbox-group>
        </a-form-item>
        <a-form-item label="聚合时间">
          <a-input-number v-model:value="ruleForm.aggregate_min" :min="0" :max="60" />
          <span style="margin-left: 10px; color: rgba(0, 0, 0, 0.45)">分钟（0表示不聚合）</span>
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="ruleForm.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { message, Modal } from 'ant-design-vue'
import type { FormInstance } from 'ant-design-vue'
import type { Rule } from 'ant-design-vue/es/form'
import { k8sApi } from '@/services/k8s'
import { logApi } from '@/services/logs'

interface Cluster {
  id: number
  name: string
}

interface AlertRule {
  id: number
  name: string
  cluster_id: number
  namespace: string
  match_type: string
  match_value: string
  level: string
  channels: string[]
  enabled: boolean
  aggregate_min: number
}

interface AlertHistory {
  id: number
  rule_id: number
  rule_name: string
  pod_name: string
  matched_content: string
  alert_count: number
  status: string
  created_at: string
}

const activeTab = ref('rules')
const clusters = ref<Cluster[]>([])
const rules = ref<AlertRule[]>([])
const histories = ref<AlertHistory[]>([])
const loading = ref(false)
const historyLoading = ref(false)
const filterCluster = ref<number | null>(null)
const historyRuleId = ref<number | null>(null)
const historyPage = ref(1)
const historyTotal = ref(0)

const showDialog = ref(false)
const editingRule = ref<AlertRule | null>(null)
const saving = ref(false)
const formRef = ref<FormInstance>()

const ruleForm = reactive({
  name: '',
  cluster_id: null as number | null,
  namespace: '',
  match_type: 'keyword',
  match_value: '',
  level: 'error',
  channels: [] as string[],
  aggregate_min: 0,
  enabled: true
})

const formRules: Record<string, Rule[]> = {
  name: [{ required: true, message: '请输入规则名称', trigger: 'blur' }],
  cluster_id: [{ required: true, message: '请选择集群', trigger: 'change' }],
  match_type: [{ required: true, message: '请选择匹配类型', trigger: 'change' }],
  match_value: [{ required: true, message: '请输入匹配值', trigger: 'blur' }]
}

const matchTypeLabels: Record<string, string> = {
  keyword: '关键词',
  regex: '正则',
  level: '级别'
}

const levelTypes: Record<string, string> = {
  info: 'blue',
  warning: 'orange',
  error: 'red',
  critical: 'red'
}

const statusTypes: Record<string, string> = {
  pending: 'blue',
  sent: 'green',
  failed: 'red'
}

const statusLabels: Record<string, string> = {
  pending: '待发送',
  sent: '已发送',
  failed: '发送失败'
}

const rulesColumns = [
  { title: '规则名称', dataIndex: 'name', minWidth: 150 },
  { title: '匹配类型', key: 'match_type', width: 100 },
  { title: '匹配值', dataIndex: 'match_value', minWidth: 200, ellipsis: true },
  { title: '级别', key: 'level', width: 80 },
  { title: '通知渠道', key: 'channels', width: 150 },
  { title: '聚合(分钟)', dataIndex: 'aggregate_min', width: 100 },
  { title: '状态', key: 'enabled', width: 80 },
  { title: '操作', key: 'action', width: 150, fixed: 'right' },
]

const historiesColumns = [
  { title: '规则名称', dataIndex: 'rule_name', width: 150 },
  { title: 'Pod', dataIndex: 'pod_name', minWidth: 200, ellipsis: true },
  { title: '匹配内容', dataIndex: 'matched_content', minWidth: 300, ellipsis: true },
  { title: '告警次数', dataIndex: 'alert_count', width: 100 },
  { title: '状态', key: 'status', width: 100 },
  { title: '时间', key: 'created_at', width: 180 },
]

const loadClusters = async () => {
  try {
    const res = await k8sApi.getClusters()
    clusters.value = res.data || []
  } catch (error) {
    message.error('加载集群列表失败')
  }
}

const loadRules = async () => {
  loading.value = true
  try {
    const res = await logApi.getAlertRules(filterCluster.value || undefined)
    rules.value = res.data || []
  } catch (error) {
    message.error('加载告警规则失败')
  } finally {
    loading.value = false
  }
}

const loadHistory = async () => {
  historyLoading.value = true
  try {
    const res = await logApi.getAlertHistory(historyRuleId.value || undefined, historyPage.value)
    histories.value = res.data?.items || []
    historyTotal.value = res.data?.total || 0
  } catch (error) {
    message.error('加载告警历史失败')
  } finally {
    historyLoading.value = false
  }
}

const showCreateDialog = () => {
  editingRule.value = null
  Object.assign(ruleForm, {
    name: '',
    cluster_id: null,
    namespace: '',
    match_type: 'keyword',
    match_value: '',
    level: 'error',
    channels: [],
    aggregate_min: 0,
    enabled: true
  })
  showDialog.value = true
}

const editRule = (rule: AlertRule) => {
  editingRule.value = rule
  Object.assign(ruleForm, {
    name: rule.name,
    cluster_id: rule.cluster_id,
    namespace: rule.namespace,
    match_type: rule.match_type,
    match_value: rule.match_value,
    level: rule.level,
    channels: rule.channels || [],
    aggregate_min: rule.aggregate_min,
    enabled: rule.enabled
  })
  showDialog.value = true
}

const saveRule = async () => {
  if (!formRef.value) return
  
  try {
    await formRef.value.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    if (editingRule.value) {
      await logApi.updateAlertRule(editingRule.value.id, ruleForm)
      message.success('更新成功')
    } else {
      await logApi.createAlertRule(ruleForm)
      message.success('创建成功')
    }
    showDialog.value = false
    loadRules()
  } catch (error) {
    message.error('保存失败')
  } finally {
    saving.value = false
  }
}

const deleteRule = async (rule: AlertRule) => {
  try {
    await Modal.confirm({
      title: '提示',
      content: `确定删除规则 "${rule.name}"？`,
      onOk: async () => {
        await logApi.deleteAlertRule(rule.id)
        message.success('删除成功')
        loadRules()
      }
    })
  } catch (error) {
    // Cancelled
  }
}

const toggleRule = async (rule: AlertRule) => {
  try {
    await logApi.toggleAlertRule(rule.id)
  } catch (error) {
    rule.enabled = !rule.enabled
    message.error('操作失败')
  }
}

const formatTime = (ts: string) => {
  if (!ts) return ''
  try {
    return new Date(ts).toLocaleString('zh-CN')
  } catch {
    return ts
  }
}

onMounted(() => {
  loadClusters()
  loadRules()
})
</script>

<style scoped>
.log-alert-config {
  padding: 20px;
}

.toolbar {
  display: flex;
  align-items: center;
}
</style>
