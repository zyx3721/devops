<template>
  <div class="alert-escalation">
    <a-card :bordered="false">
      <template #extra>
        <a-button type="primary" @click="showModal()">
          <template #icon><PlusOutlined /></template>
          添加升级规则
        </a-button>
      </template>
      <a-table :columns="columns" :data-source="list" :loading="loading" row-key="id" :pagination="pagination" @change="onTableChange">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'level'">
            <a-tag :color="getLevelColor(record.level)">{{ getLevelLabel(record.level) }}</a-tag>
          </template>
          <template v-if="column.key === 'delay'">{{ record.delay_minutes }} 分钟</template>
          <template v-if="column.key === 'platform'">
            <a-tag :color="getPlatformColor(record.platform)">{{ getPlatformLabel(record.platform) }}</a-tag>
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

    <a-modal v-model:open="modalVisible" :title="editingId ? '编辑升级规则' : '添加升级规则'" @ok="save" :confirm-loading="saving" width="600px">
      <a-form :model="form" layout="vertical">
        <a-form-item label="规则名称" required>
          <a-input v-model:value="form.name" placeholder="如：严重告警30分钟升级" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="触发级别" required>
              <a-select v-model:value="form.level" style="width: 100%">
                <a-select-option value="warning">警告</a-select-option>
                <a-select-option value="error">错误</a-select-option>
                <a-select-option value="critical">严重</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="延迟时间(分钟)" required>
              <a-input-number v-model:value="form.delay_minutes" :min="1" :max="1440" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="升级通知平台" required>
              <a-select v-model:value="form.platform" style="width: 100%" @change="onPlatformChange">
                <a-select-option value="feishu">飞书</a-select-option>
                <a-select-option value="dingtalk">钉钉</a-select-option>
                <a-select-option value="wechatwork">企业微信</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="升级通知机器人">
              <a-select v-model:value="form.bot_id" style="width: 100%" placeholder="选择机器人" :loading="loadingBots" allowClear>
                <a-select-option v-for="bot in currentBots" :key="bot.id" :value="bot.id">{{ bot.name }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="描述">
          <a-textarea v-model:value="form.description" :rows="2" />
        </a-form-item>
        <a-form-item><a-checkbox v-model:checked="form.enabled">启用</a-checkbox></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { alertApi, type AlertEscalation } from '@/services/alert'
import { feishuBotApi, type FeishuBot } from '@/services/feishu'
import { dingtalkBotApi, type DingtalkBot } from '@/services/dingtalk'
import { wechatworkBotApi, type WechatWorkBot } from '@/services/wechatwork'

const loading = ref(false)
const saving = ref(false)
const loadingBots = ref(false)
const modalVisible = ref(false)
const editingId = ref<number>()
const list = ref<AlertEscalation[]>([])
const pagination = reactive({ current: 1, pageSize: 10, total: 0 })
const form = reactive<Partial<AlertEscalation>>({ name: '', level: 'error', delay_minutes: 30, platform: 'feishu', bot_id: undefined, enabled: true, description: '' })

const feishuBots = ref<FeishuBot[]>([])
const dingtalkBots = ref<DingtalkBot[]>([])
const wechatworkBots = ref<WechatWorkBot[]>([])

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '触发级别', dataIndex: 'level', key: 'level', width: 100 },
  { title: '延迟', key: 'delay', width: 100 },
  { title: '升级平台', dataIndex: 'platform', key: 'platform', width: 100 },
  { title: '启用', dataIndex: 'enabled', key: 'enabled', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const levelLabels: Record<string, string> = { info: '信息', warning: '警告', error: '错误', critical: '严重' }
const levelColors: Record<string, string> = { info: 'blue', warning: 'orange', error: 'red', critical: 'magenta' }
const platformLabels: Record<string, string> = { feishu: '飞书', dingtalk: '钉钉', wechatwork: '企业微信' }
const platformColors: Record<string, string> = { feishu: 'blue', dingtalk: 'cyan', wechatwork: 'green' }

const getLevelLabel = (level: string) => levelLabels[level] || level
const getLevelColor = (level: string) => levelColors[level] || 'default'
const getPlatformLabel = (platform: string) => platformLabels[platform] || platform
const getPlatformColor = (platform: string) => platformColors[platform] || 'default'

const currentBots = computed(() => {
  switch (form.platform) {
    case 'feishu': return feishuBots.value
    case 'dingtalk': return dingtalkBots.value
    case 'wechatwork': return wechatworkBots.value
    default: return []
  }
})

const onPlatformChange = () => { form.bot_id = undefined }

const fetchData = async () => {
  loading.value = true
  try {
    const res = await alertApi.listEscalations({ page: pagination.current, page_size: pagination.pageSize })
    if (res.code === 0 && res.data) { list.value = res.data.list || []; pagination.total = res.data.total }
  } finally { loading.value = false }
}

const fetchBots = async () => {
  loadingBots.value = true
  try {
    const [f, d, w] = await Promise.all([feishuBotApi.list(), dingtalkBotApi.list(), wechatworkBotApi.list()])
    if (f.code === 0 && f.data) feishuBots.value = f.data.list || []
    if (d.code === 0 && d.data) dingtalkBots.value = d.data.list || []
    if (w.code === 0 && w.data) wechatworkBots.value = w.data.list || []
  } finally { loadingBots.value = false }
}

const onTableChange = (pag: any) => { pagination.current = pag.current; fetchData() }

const showModal = (record?: AlertEscalation) => {
  if (record) { editingId.value = record.id; Object.assign(form, record) }
  else { editingId.value = undefined; Object.assign(form, { name: '', level: 'error', delay_minutes: 30, platform: 'feishu', bot_id: undefined, enabled: true, description: '' }) }
  modalVisible.value = true
}

const save = async () => {
  if (!form.name) { message.error('请填写规则名称'); return }
  saving.value = true
  try {
    const res = editingId.value ? await alertApi.updateEscalation(editingId.value, form) : await alertApi.createEscalation(form)
    if (res.code === 0) { message.success(editingId.value ? '更新成功' : '添加成功'); modalVisible.value = false; fetchData() }
    else message.error(res.message || '保存失败')
  } catch (e: any) { message.error(e.message || '保存失败') }
  finally { saving.value = false }
}

const toggleEnabled = async (record: AlertEscalation) => {
  try {
    const res = await alertApi.updateEscalation(record.id, { ...record, enabled: !record.enabled })
    if (res.code === 0) { message.success('已更新'); fetchData() }
  } catch (e: any) { message.error(e.message || '操作失败') }
}

const deleteItem = async (id: number) => {
  try {
    const res = await alertApi.deleteEscalation(id)
    if (res.code === 0) { message.success('已删除'); fetchData() }
  } catch (e: any) { message.error(e.message || '操作失败') }
}

onMounted(() => { fetchData(); fetchBots() })
</script>
