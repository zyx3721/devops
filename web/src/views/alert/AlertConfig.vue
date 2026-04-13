<template>
  <div class="alert-config">
    <a-card :bordered="false">
      <template #extra>
        <a-button type="primary" @click="showModal()">
          <template #icon><PlusOutlined /></template>
          添加告警配置
        </a-button>
      </template>
      <a-table :columns="columns" :data-source="list" :loading="loading" row-key="id" :pagination="pagination" @change="onTableChange">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <a-tag :color="getTypeColor(record.type)">{{ getTypeLabel(record.type) }}</a-tag>
          </template>
          <template v-if="column.key === 'template'">
            <span v-if="record.template_id">{{ getTemplateName(record.template_id) }}</span>
            <span v-else class="text-gray">默认</span>
          </template>
          <template v-if="column.key === 'channels'">
             <a-tag v-for="ch in parseChannels(record.channels)" :key="ch.url" :color="getChannelColor(ch.type)">
               {{ getChannelLabel(ch.type) }}
             </a-tag>
          </template>
          <template v-if="column.key === 'conditions'">
            <span v-if="record.conditions">{{ formatConditions(record.conditions) }}</span>
            <span v-else style="color: #999">全部</span>
          </template>
          <template v-if="column.key === 'enabled'">
            <a-switch :checked="record.enabled" @change="toggleEnabled(record)" size="small" />
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showModal(record)">编辑</a-button>
              <a-popconfirm title="确定删除？" @confirm="deleteItem(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:open="modalVisible" :title="editingId ? '编辑告警配置' : '添加告警配置'" @ok="save" :confirm-loading="saving" width="700px">
      <a-form :model="form" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="配置名称" required>
              <a-input v-model:value="form.name" placeholder="如：Jenkins构建失败告警" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
             <a-form-item label="消息模板">
              <a-select v-model:value="form.template_id" placeholder="使用默认模板" allowClear>
                <a-select-option v-for="t in templates" :key="t.id" :value="t.id">{{ t.name }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="告警类型" required>
              <a-select v-model:value="form.type" style="width: 100%" @change="onTypeChange">
                <a-select-option value="jenkins_build">Jenkins 构建</a-select-option>
                <a-select-option value="k8s_pod">K8s Pod 异常</a-select-option>
                <a-select-option value="health_check">健康检查</a-select-option>
                <a-select-option value="test">测试告警</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="告警级别">
              <a-select v-model:value="conditionForm.level" mode="multiple" style="width: 100%" placeholder="全部级别" allowClear>
                <a-select-option value="info">信息</a-select-option>
                <a-select-option value="warning">警告</a-select-option>
                <a-select-option value="error">错误</a-select-option>
                <a-select-option value="critical">严重</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>

        <!-- Jenkins 构建条件 -->
        <template v-if="form.type === 'jenkins_build'">
          <a-form-item label="构建结果">
            <a-checkbox-group v-model:value="conditionForm.result">
              <a-checkbox value="FAILURE">失败 (FAILURE)</a-checkbox>
              <a-checkbox value="ABORTED">中止 (ABORTED)</a-checkbox>
              <a-checkbox value="UNSTABLE">不稳定 (UNSTABLE)</a-checkbox>
              <a-checkbox value="SUCCESS">成功 (SUCCESS)</a-checkbox>
            </a-checkbox-group>
          </a-form-item>
          <a-form-item label="Job 名称匹配">
            <a-input v-model:value="conditionForm.job_pattern" placeholder="支持通配符，如：deploy-* 或留空匹配全部" />
          </a-form-item>
        </template>

        <!-- K8s Pod 条件 -->
        <template v-if="form.type === 'k8s_pod'">
          <a-form-item label="Pod 状态">
            <a-checkbox-group v-model:value="conditionForm.pod_status">
              <a-checkbox value="CrashLoopBackOff">CrashLoopBackOff</a-checkbox>
              <a-checkbox value="ImagePullBackOff">ImagePullBackOff</a-checkbox>
              <a-checkbox value="OOMKilled">OOMKilled</a-checkbox>
              <a-checkbox value="Error">Error</a-checkbox>
              <a-checkbox value="Pending">Pending</a-checkbox>
            </a-checkbox-group>
          </a-form-item>
          <a-form-item label="命名空间">
            <a-input v-model:value="conditionForm.namespace" placeholder="指定命名空间，留空匹配全部" />
          </a-form-item>
        </template>

        <!-- 健康检查条件 -->
        <template v-if="form.type === 'health_check'">
          <a-form-item label="检查结果">
            <a-checkbox-group v-model:value="conditionForm.check_status">
              <a-checkbox value="failed">检查失败</a-checkbox>
              <a-checkbox value="timeout">超时</a-checkbox>
              <a-checkbox value="recovered">恢复正常</a-checkbox>
            </a-checkbox-group>
          </a-form-item>
          <a-form-item label="连续失败次数">
            <a-input-number v-model:value="conditionForm.fail_count" :min="1" :max="10" placeholder="达到此次数才告警" style="width: 150px" />
            <span style="margin-left: 8px; color: #999">次后触发告警</span>
          </a-form-item>
        </template>

        <a-divider orientation="left">通知渠道</a-divider>
        
        <div v-for="(channel, index) in channels" :key="index" class="channel-item">
          <a-space align="start" style="width: 100%">
            <a-select v-model:value="channel.type" style="width: 120px">
              <a-select-option value="feishu">飞书</a-select-option>
              <a-select-option value="dingtalk">钉钉</a-select-option>
              <a-select-option value="email">邮件</a-select-option>
              <a-select-option value="webhook">Webhook</a-select-option>
            </a-select>
            
            <div style="flex: 1">
              <a-input v-model:value="channel.url" :placeholder="getChannelPlaceholder(channel.type)" style="margin-bottom: 8px" />
              <a-input v-if="channel.type === 'dingtalk'" v-model:value="channel.secret" placeholder="钉钉加签密钥 (SEC...)" />
            </div>

            <MinusCircleOutlined @click="removeChannel(index)" style="color: #ff4d4f; cursor: pointer; margin-top: 8px" />
          </a-space>
        </div>
        
        <a-button type="dashed" block @click="addChannel" style="margin-top: 8px">
          <PlusOutlined /> 添加通知渠道
        </a-button>

        <a-divider />

        <a-form-item label="描述">
          <a-textarea v-model:value="form.description" :rows="2" placeholder="可选，描述此告警配置的用途" />
        </a-form-item>
        <a-form-item><a-checkbox v-model:checked="form.enabled">启用</a-checkbox></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, MinusCircleOutlined } from '@ant-design/icons-vue'
import { alertApi, type AlertConfig } from '@/services/alert'
import { templateApi, type MessageTemplate } from '@/services/template'

const loading = ref(false)
const saving = ref(false)
const modalVisible = ref(false)
const editingId = ref<number>()
const list = ref<AlertConfig[]>([])
const templates = ref<MessageTemplate[]>([])
const pagination = reactive({ current: 1, pageSize: 10, total: 0 })

// 表单数据
const form = reactive<Partial<AlertConfig>>({ 
  name: '', 
  type: 'jenkins_build', 
  enabled: true, 
  template_id: undefined,
  conditions: '', 
  description: '' 
})

// 渠道配置
interface ChannelConfig {
  type: string
  url: string
  secret?: string
  receive_id?: string
}
const channels = ref<ChannelConfig[]>([])

// 条件表单（可视化）
const conditionForm = reactive({
  level: [] as string[],
  result: [] as string[],
  job_pattern: '',
  pod_status: [] as string[],
  namespace: '',
  check_status: [] as string[],
  fail_count: undefined as number | undefined
})

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '类型', dataIndex: 'type', key: 'type', width: 120 },
  { title: '模板', key: 'template', width: 150 },
  { title: '通知渠道', key: 'channels', width: 250 },
  { title: '触发条件', key: 'conditions', width: 200, ellipsis: true },
  { title: '启用', dataIndex: 'enabled', key: 'enabled', width: 70 },
  { title: '操作', key: 'action', width: 110 }
]

const typeLabels: Record<string, string> = { jenkins_build: 'Jenkins构建', k8s_pod: 'K8s Pod', health_check: '健康检查', test: '测试告警' }
const typeColors: Record<string, string> = { jenkins_build: 'red', k8s_pod: 'purple', health_check: 'blue', test: 'gray' }

const getTypeLabel = (type: string) => typeLabels[type] || type
const getTypeColor = (type: string) => typeColors[type] || 'default'

const getChannelLabel = (type: string) => ({ feishu: '飞书', dingtalk: '钉钉', email: '邮件', webhook: 'Webhook' }[type] || type)
const getChannelColor = (type: string) => ({ feishu: 'blue', dingtalk: 'cyan', email: 'orange', webhook: 'purple' }[type] || 'default')

const getChannelPlaceholder = (type: string) => {
  switch (type) {
    case 'feishu': return '飞书 Webhook URL'
    case 'dingtalk': return '钉钉 Webhook URL'
    case 'email': return '接收邮箱地址'
    default: return 'Webhook URL'
  }
}

const getTemplateName = (id: number) => {
  return templates.value.find(t => t.id === id)?.name || id
}

// 格式化显示条件
const formatConditions = (conditions: string) => {
  try {
    const c = JSON.parse(conditions)
    const parts: string[] = []
    if (c.level?.length) parts.push(`级别: ${c.level.join(',')}`)
    if (c.result?.length) parts.push(`结果: ${c.result.join(',')}`)
    if (c.job_pattern) parts.push(`Job: ${c.job_pattern}`)
    if (c.pod_status?.length) parts.push(`状态: ${c.pod_status.join(',')}`)
    if (c.namespace) parts.push(`NS: ${c.namespace}`)
    if (c.check_status?.length) parts.push(`检查: ${c.check_status.join(',')}`)
    if (c.fail_count) parts.push(`失败${c.fail_count}次`)
    return parts.join('; ') || '全部'
  } catch { return conditions || '全部' }
}

const parseChannels = (jsonStr?: string): ChannelConfig[] => {
  if (!jsonStr) return []
  try {
    const res = JSON.parse(jsonStr)
    return Array.isArray(res) ? res : []
  } catch { return [] }
}

const onTypeChange = () => { resetConditionForm() }

const resetConditionForm = () => {
  conditionForm.level = []
  conditionForm.result = []
  conditionForm.job_pattern = ''
  conditionForm.pod_status = []
  conditionForm.namespace = ''
  conditionForm.check_status = []
  conditionForm.fail_count = undefined
}

// 解析条件到表单
const parseConditions = (conditions: string) => {
  resetConditionForm()
  if (!conditions) return
  try {
    const c = JSON.parse(conditions)
    if (c.level) conditionForm.level = c.level
    if (c.result) conditionForm.result = c.result
    if (c.job_pattern) conditionForm.job_pattern = c.job_pattern
    if (c.pod_status) conditionForm.pod_status = c.pod_status
    if (c.namespace) conditionForm.namespace = c.namespace
    if (c.check_status) conditionForm.check_status = c.check_status
    if (c.fail_count) conditionForm.fail_count = c.fail_count
  } catch { /* ignore */ }
}

// 构建条件 JSON
const buildConditions = (): string => {
  const c: Record<string, any> = {}
  if (conditionForm.level?.length) c.level = conditionForm.level
  if (form.type === 'jenkins_build') {
    if (conditionForm.result?.length) c.result = conditionForm.result
    if (conditionForm.job_pattern) c.job_pattern = conditionForm.job_pattern
  } else if (form.type === 'k8s_pod') {
    if (conditionForm.pod_status?.length) c.pod_status = conditionForm.pod_status
    if (conditionForm.namespace) c.namespace = conditionForm.namespace
  } else if (form.type === 'health_check') {
    if (conditionForm.check_status?.length) c.check_status = conditionForm.check_status
    if (conditionForm.fail_count) c.fail_count = conditionForm.fail_count
  }
  return Object.keys(c).length ? JSON.stringify(c) : ''
}

const fetchData = async () => {
  loading.value = true
  try {
    const res = await alertApi.listConfigs({ page: pagination.current, page_size: pagination.pageSize })
    if (res.code === 0 && res.data) { list.value = res.data.list || []; pagination.total = res.data.total }
  } finally { loading.value = false }
}

const fetchTemplates = async () => {
  try {
    const res = await templateApi.list()
    if (res.code === 0 && res.data) templates.value = res.data.list || []
  } catch (e) { console.error(e) }
}

const onTableChange = (pag: any) => { pagination.current = pag.current; fetchData() }

const showModal = (record?: AlertConfig) => {
  if (record) {
    editingId.value = record.id
    Object.assign(form, record)
    parseConditions(record.conditions)
    channels.value = parseChannels(record.channels)
  } else {
    editingId.value = undefined
    Object.assign(form, { name: '', type: 'jenkins_build', enabled: true, template_id: undefined, conditions: '', description: '' })
    resetConditionForm()
    channels.value = [{ type: 'feishu', url: '' }]
  }
  modalVisible.value = true
}

const addChannel = () => {
  channels.value.push({ type: 'feishu', url: '' })
}

const removeChannel = (index: number) => {
  channels.value.splice(index, 1)
}

const save = async () => {
  if (!form.name) { message.error('请填写配置名称'); return }
  
  // 验证 Channels
  const validChannels = channels.value.filter(c => c.url)
  if (validChannels.length === 0) {
    message.error('请至少配置一个有效的通知渠道')
    return
  }

  saving.value = true
  try {
    const data = { 
      ...form, 
      conditions: buildConditions(),
      channels: JSON.stringify(validChannels),
      platform: 'multi', // 兼容旧字段
      bot_id: 0         // 兼容旧字段
    }
    const res = editingId.value ? await alertApi.updateConfig(editingId.value, data) : await alertApi.createConfig(data)
    if (res.code === 0) { message.success(editingId.value ? '更新成功' : '添加成功'); modalVisible.value = false; fetchData() }
    else message.error(res.message || '保存失败')
  } catch (e: any) { message.error(e.message || '保存失败') }
  finally { saving.value = false }
}

const toggleEnabled = async (record: AlertConfig) => {
  try {
    const res = await alertApi.updateConfig(record.id!, { ...record, enabled: !record.enabled })
    if (res.code === 0) { message.success('已更新'); fetchData() }
  } catch (e: any) { message.error(e.message || '操作失败') }
}

const deleteItem = async (id: number) => {
  try {
    const res = await alertApi.deleteConfig(id)
    if (res.code === 0) { message.success('已删除'); fetchData() }
  } catch (e: any) { message.error(e.message || '操作失败') }
}

onMounted(() => { fetchData(); fetchTemplates() })
</script>

<style scoped>
.channel-item {
  margin-bottom: 12px;
  padding: 8px;
  background: #f5f5f5;
  border-radius: 4px;
}
.text-gray { color: #999; }
</style>
