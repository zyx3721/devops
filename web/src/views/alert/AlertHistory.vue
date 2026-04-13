<template>
  <div class="alert-history">
    <a-card :bordered="false">
      <div style="margin-bottom: 16px; display: flex; justify-content: space-between; align-items: center">
        <a-space>
          <a-range-picker v-model:value="dateRange" format="YYYY-MM-DD" @change="fetchData" />
          <a-select v-model:value="filter.ack_status" style="width: 120px" placeholder="处理状态" allowClear @change="fetchData">
            <a-select-option value="pending">待处理</a-select-option>
            <a-select-option value="acked">已确认</a-select-option>
            <a-select-option value="resolved">已解决</a-select-option>
          </a-select>
          <a-select v-model:value="filter.type" style="width: 140px" placeholder="告警类型" allowClear @change="fetchData">
            <a-select-option value="jenkins_build">Jenkins构建</a-select-option>
            <a-select-option value="k8s_pod">K8s Pod</a-select-option>
            <a-select-option value="health_check">健康检查</a-select-option>
          </a-select>
          <a-select v-model:value="filter.level" style="width: 100px" placeholder="级别" allowClear @change="fetchData">
            <a-select-option value="info">信息</a-select-option>
            <a-select-option value="warning">警告</a-select-option>
            <a-select-option value="error">错误</a-select-option>
            <a-select-option value="critical">严重</a-select-option>
          </a-select>
          <a-input-search v-model:value="filter.keyword" placeholder="搜索标题/来源" style="width: 200px" @search="fetchData" allowClear />
        </a-space>
        <a-space>
          <a-button :disabled="!selectedRowKeys.length" @click="batchAck">批量确认</a-button>
          <a-button type="primary" @click="fetchData"><ReloadOutlined /> 刷新</a-button>
        </a-space>
      </div>
      <a-table 
        :columns="columns" 
        :data-source="list" 
        :loading="loading" 
        row-key="id" 
        :pagination="pagination" 
        :row-selection="{ selectedRowKeys, onChange: onSelectChange }"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'created_at'">{{ formatTime(record.created_at) }}</template>
          <template v-if="column.key === 'type'">
            <a-tag :color="getTypeColor(record.type)">{{ getTypeLabel(record.type) }}</a-tag>
          </template>
          <template v-if="column.key === 'level'">
            <a-tag :color="getLevelColor(record.level)">{{ getLevelLabel(record.level) }}</a-tag>
          </template>
          <template v-if="column.key === 'title'">
            <a-tooltip :title="record.content">
              <span>{{ record.title }}</span>
            </a-tooltip>
          </template>
          <template v-if="column.key === 'ack_status'">
            <a-badge :status="getAckStatusBadge(record.ack_status)" :text="getAckStatusLabel(record.ack_status)" />
          </template>
          <template v-if="column.key === 'ack_info'">
            <div v-if="record.ack_status === 'acked'" style="font-size: 12px; color: #666">
              <div>{{ formatTime(record.ack_at) }}</div>
              <div v-if="record.ack_by">User ID: {{ record.ack_by }}</div>
            </div>
            <div v-else-if="record.ack_status === 'resolved'" style="font-size: 12px; color: #666">
              <div>{{ formatTime(record.resolved_at) }}</div>
              <div v-if="record.resolved_by">User ID: {{ record.resolved_by }}</div>
            </div>
            <div v-else>-</div>
          </template>
          <template v-if="column.key === 'flags'">
            <a-space>
              <a-tooltip v-if="record.silenced" title="已静默"><StopOutlined style="color: #1890ff" /></a-tooltip>
              <a-tooltip v-if="record.escalated" title="已升级"><ArrowUpOutlined style="color: #722ed1" /></a-tooltip>
            </a-space>
          </template>
          <template v-if="column.key === 'source_url'">
            <a v-if="record.source_url" :href="record.source_url" target="_blank">查看</a>
            <span v-else>-</span>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button v-if="record.ack_status === 'pending'" type="link" size="small" @click="ackAlert(record.id)">确认</a-button>
              <a-button v-if="record.ack_status !== 'resolved'" type="link" size="small" @click="showResolveModal(record)">解决</a-button>
              <a-button type="link" size="small" @click="showDetail(record)">详情</a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 解决弹窗 -->
    <a-modal v-model:open="resolveModalVisible" title="解决告警" @ok="resolveAlert" :confirm-loading="resolving">
      <a-form layout="vertical">
        <a-form-item label="解决备注">
          <a-textarea v-model:value="resolveComment" :rows="3" placeholder="请输入解决方案或备注" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 详情抽屉 -->
    <a-drawer v-model:open="detailVisible" title="告警详情" width="500">
      <a-descriptions :column="1" bordered size="small" v-if="currentAlert">
        <a-descriptions-item label="告警时间">{{ formatTime(currentAlert.created_at) }}</a-descriptions-item>
        <a-descriptions-item label="告警类型"><a-tag :color="getTypeColor(currentAlert.type)">{{ getTypeLabel(currentAlert.type) }}</a-tag></a-descriptions-item>
        <a-descriptions-item label="告警级别"><a-tag :color="getLevelColor(currentAlert.level)">{{ getLevelLabel(currentAlert.level) }}</a-tag></a-descriptions-item>
        <a-descriptions-item label="处理状态"><a-badge :status="getAckStatusBadge(currentAlert.ack_status)" :text="getAckStatusLabel(currentAlert.ack_status)" /></a-descriptions-item>
        <a-descriptions-item label="告警标题">{{ currentAlert.title }}</a-descriptions-item>
        <a-descriptions-item label="告警内容"><pre style="white-space: pre-wrap; margin: 0">{{ currentAlert.content }}</pre></a-descriptions-item>
        <a-descriptions-item label="来源ID">{{ currentAlert.source_id }}</a-descriptions-item>
        <a-descriptions-item label="来源链接"><a v-if="currentAlert.source_url" :href="currentAlert.source_url" target="_blank">{{ currentAlert.source_url }}</a><span v-else>-</span></a-descriptions-item>
        <a-descriptions-item label="静默状态">{{ currentAlert.silenced ? '已静默' : '未静默' }}</a-descriptions-item>
        <a-descriptions-item label="升级状态">{{ currentAlert.escalated ? '已升级' : '未升级' }}</a-descriptions-item>
        <a-descriptions-item v-if="currentAlert.ack_at" label="确认时间">{{ formatTime(currentAlert.ack_at) }}</a-descriptions-item>
        <a-descriptions-item v-if="currentAlert.resolved_at" label="解决时间">{{ formatTime(currentAlert.resolved_at) }}</a-descriptions-item>
        <a-descriptions-item v-if="currentAlert.resolve_comment" label="解决备注">{{ currentAlert.resolve_comment }}</a-descriptions-item>
      </a-descriptions>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import { StopOutlined, ArrowUpOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import dayjs, { Dayjs } from 'dayjs'
import { alertApi, type AlertHistory } from '@/services/alert'

const route = useRoute()
const loading = ref(false)
const resolving = ref(false)
const resolveModalVisible = ref(false)
const detailVisible = ref(false)
const resolvingAlertId = ref<number>()
const resolveComment = ref('')
const currentAlert = ref<AlertHistory | null>(null)
const list = ref<AlertHistory[]>([])
const selectedRowKeys = ref<number[]>([])
const dateRange = ref<[Dayjs, Dayjs]>([dayjs().subtract(7, 'day'), dayjs()])
const filter = reactive({ 
  type: undefined as string | undefined, 
  ack_status: (route.query.ack_status as string) || undefined,
  level: undefined as string | undefined,
  keyword: ''
})
const pagination = reactive({ current: 1, pageSize: 20, total: 0 })

const columns = [
  { title: '时间', key: 'created_at', width: 150 },
  { title: '类型', key: 'type', width: 110 },
  { title: '级别', key: 'level', width: 70 },
  { title: '标题', key: 'title', ellipsis: true },
  { title: '来源', dataIndex: 'source_id', width: 130, ellipsis: true },
  { title: '状态', key: 'ack_status', width: 80 },
  { title: '认领/解决', key: 'ack_info', width: 140 },
  { title: '标记', key: 'flags', width: 60 },
  { title: '链接', key: 'source_url', width: 50 },
  { title: '操作', key: 'action', width: 130 }
]

const typeLabels: Record<string, string> = { jenkins_build: 'Jenkins构建', k8s_pod: 'K8s Pod', health_check: '健康检查' }
const typeColors: Record<string, string> = { jenkins_build: 'red', k8s_pod: 'purple', health_check: 'blue' }
const levelLabels: Record<string, string> = { info: '信息', warning: '警告', error: '错误', critical: '严重' }
const levelColors: Record<string, string> = { info: 'blue', warning: 'orange', error: 'red', critical: 'magenta' }
const ackStatusLabels: Record<string, string> = { pending: '待处理', acked: '已确认', resolved: '已解决' }

const getTypeLabel = (type: string) => typeLabels[type] || type
const getTypeColor = (type: string) => typeColors[type] || 'default'
const getLevelLabel = (level: string) => levelLabels[level] || level
const getLevelColor = (level: string) => levelColors[level] || 'default'
const getAckStatusLabel = (status: string) => ackStatusLabels[status] || status
const getAckStatusBadge = (status: string) => status === 'resolved' ? 'success' : status === 'acked' ? 'processing' : 'warning'
const formatTime = (time: string) => time ? time.replace('T', ' ').substring(0, 19) : '-'

const fetchData = async () => {
  loading.value = true
  selectedRowKeys.value = []
  try {
    const res = await alertApi.listHistories({ 
      page: pagination.current, 
      page_size: pagination.pageSize, 
      type: filter.type, 
      ack_status: filter.ack_status,
      level: filter.level,
      keyword: filter.keyword || undefined,
      start_time: dateRange.value?.[0]?.format('YYYY-MM-DD'),
      end_time: dateRange.value?.[1]?.format('YYYY-MM-DD')
    })
    if (res.code === 0 && res.data) { list.value = res.data.list || []; pagination.total = res.data.total }
  } finally { loading.value = false }
}

const onTableChange = (pag: any) => { pagination.current = pag.current; fetchData() }
const onSelectChange = (keys: number[]) => { selectedRowKeys.value = keys }

const ackAlert = async (id: number) => {
  try {
    const res = await alertApi.ackHistory(id)
    if (res.code === 0) { message.success('已确认'); fetchData() }
  } catch (e: any) { message.error(e.message || '操作失败') }
}

const batchAck = async () => {
  if (!selectedRowKeys.value.length) return
  try {
    await Promise.all(selectedRowKeys.value.map(id => alertApi.ackHistory(id)))
    message.success(`已确认 ${selectedRowKeys.value.length} 条告警`)
    fetchData()
  } catch (e: any) { message.error(e.message || '操作失败') }
}

const showResolveModal = (record: AlertHistory) => {
  resolvingAlertId.value = record.id
  resolveComment.value = ''
  resolveModalVisible.value = true
}

const showDetail = (record: AlertHistory) => {
  currentAlert.value = record
  detailVisible.value = true
}

const resolveAlert = async () => {
  if (!resolvingAlertId.value) return
  resolving.value = true
  try {
    const res = await alertApi.resolveHistory(resolvingAlertId.value, resolveComment.value)
    if (res.code === 0) { message.success('已解决'); resolveModalVisible.value = false; fetchData() }
  } catch (e: any) { message.error(e.message || '操作失败') }
  finally { resolving.value = false }
}

onMounted(fetchData)
</script>
