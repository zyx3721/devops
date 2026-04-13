<template>
  <div class="alert-silence">
    <a-card :bordered="false">
      <template #extra>
        <a-button type="primary" @click="showModal()">
          <template #icon><PlusOutlined /></template>
          添加静默
        </a-button>
      </template>
      <a-table :columns="columns" :data-source="list" :loading="loading" row-key="id" :pagination="pagination" @change="onTableChange">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <a-tag :color="record.type === 'all' ? 'default' : getTypeColor(record.type)">{{ record.type === 'all' ? '全部' : getTypeLabel(record.type) }}</a-tag>
          </template>
          <template v-if="column.key === 'time_range'">
            {{ formatTime(record.start_time) }} ~ {{ formatTime(record.end_time) }}
          </template>
          <template v-if="column.key === 'status'">
            <a-badge :status="getStatusBadge(record.status)" :text="getStatusLabel(record.status)" />
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button v-if="record.status === 'active'" type="link" size="small" @click="showModal(record)">编辑</a-button>
              <a-popconfirm v-if="record.status === 'active'" title="确定取消静默？" @confirm="cancelSilence(record.id)">
                <a-button type="link" size="small" danger>取消</a-button>
              </a-popconfirm>
              <a-popconfirm v-else title="确定删除？" @confirm="deleteSilence(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:open="modalVisible" :title="editingId ? '编辑静默规则' : '添加静默规则'" @ok="save" :confirm-loading="saving" width="600px">
      <a-form :model="form" layout="vertical">
        <a-form-item label="规则名称" required>
          <a-input v-model:value="form.name" placeholder="如：发布窗口静默" />
        </a-form-item>
        <a-form-item label="告警类型" required>
          <a-select v-model:value="form.type" style="width: 100%">
            <a-select-option value="all">全部类型</a-select-option>
            <a-select-option value="jenkins_build">Jenkins 构建</a-select-option>
            <a-select-option value="k8s_pod">K8s Pod 异常</a-select-option>
            <a-select-option value="health_check">健康检查</a-select-option>
          </a-select>
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="开始时间" required>
              <a-date-picker v-model:value="form.start_time" show-time style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="结束时间" required>
              <a-date-picker v-model:value="form.end_time" show-time style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="匹配条件">
          <a-textarea v-model:value="form.matchers" :rows="2" placeholder='{"source_id": "job-name", "level": ["warning"]}' />
          <div style="color: #999; font-size: 12px">JSON 格式，可选。留空表示匹配所有该类型告警</div>
        </a-form-item>
        <a-form-item label="静默原因">
          <a-textarea v-model:value="form.reason" :rows="2" placeholder="如：计划维护窗口" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import dayjs from 'dayjs'
import { alertApi, type AlertSilence } from '@/services/alert'

const loading = ref(false)
const saving = ref(false)
const modalVisible = ref(false)
const editingId = ref<number>()
const list = ref<AlertSilence[]>([])
const pagination = reactive({ current: 1, pageSize: 10, total: 0 })
const form = reactive<any>({ name: '', type: 'all', matchers: '', start_time: null, end_time: null, reason: '' })

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '类型', dataIndex: 'type', key: 'type', width: 120 },
  { title: '时间范围', key: 'time_range', width: 300 },
  { title: '原因', dataIndex: 'reason', key: 'reason', ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const typeLabels: Record<string, string> = { jenkins_build: 'Jenkins构建', k8s_pod: 'K8s Pod', health_check: '健康检查' }
const typeColors: Record<string, string> = { jenkins_build: 'red', k8s_pod: 'purple', health_check: 'blue' }
const statusLabels: Record<string, string> = { active: '生效中', expired: '已过期', cancelled: '已取消' }

const getTypeLabel = (type: string) => typeLabels[type] || type
const getTypeColor = (type: string) => typeColors[type] || 'default'
const getStatusLabel = (status: string) => statusLabels[status] || status
const getStatusBadge = (status: string) => status === 'active' ? 'processing' : status === 'expired' ? 'default' : 'error'
const formatTime = (time: string) => time ? time.replace('T', ' ').substring(0, 19) : '-'

const fetchData = async () => {
  loading.value = true
  try {
    const res = await alertApi.listSilences({ page: pagination.current, page_size: pagination.pageSize })
    if (res.code === 0 && res.data) { list.value = res.data.list || []; pagination.total = res.data.total }
  } finally { loading.value = false }
}

const onTableChange = (pag: any) => { pagination.current = pag.current; fetchData() }

const showModal = (record?: AlertSilence) => {
  if (record) {
    editingId.value = record.id
    Object.assign(form, { ...record, start_time: dayjs(record.start_time), end_time: dayjs(record.end_time) })
  } else {
    editingId.value = undefined
    Object.assign(form, { name: '', type: 'all', matchers: '', start_time: dayjs(), end_time: dayjs().add(2, 'hour'), reason: '' })
  }
  modalVisible.value = true
}

const save = async () => {
  if (!form.name || !form.start_time || !form.end_time) { message.error('请填写必填项'); return }
  saving.value = true
  try {
    const data = { ...form, start_time: form.start_time.format('YYYY-MM-DDTHH:mm:ss') + 'Z', end_time: form.end_time.format('YYYY-MM-DDTHH:mm:ss') + 'Z' }
    const res = editingId.value ? await alertApi.updateSilence(editingId.value, data) : await alertApi.createSilence(data)
    if (res.code === 0) { message.success(editingId.value ? '更新成功' : '添加成功'); modalVisible.value = false; fetchData() }
    else message.error(res.message || '保存失败')
  } catch (e: any) { message.error(e.message || '保存失败') }
  finally { saving.value = false }
}

const cancelSilence = async (id: number) => {
  try {
    const res = await alertApi.cancelSilence(id)
    if (res.code === 0) { message.success('已取消'); fetchData() }
  } catch (e: any) { message.error(e.message || '操作失败') }
}

const deleteSilence = async (id: number) => {
  try {
    const res = await alertApi.deleteSilence(id)
    if (res.code === 0) { message.success('已删除'); fetchData() }
  } catch (e: any) { message.error(e.message || '操作失败') }
}

onMounted(fetchData)
</script>
