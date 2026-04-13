<template>
  <a-card :bordered="false">
    <template #extra>
      <a-space>
        <a-select v-model:value="filter.source" placeholder="来源" style="width: 120px" allow-clear @change="$emit('filter')">
          <a-select-option value="manual">手动发送</a-select-option>
          <a-select-option value="oa_sync">OA同步</a-select-option>
        </a-select>
        <a-select v-model:value="filter.msg_type" placeholder="类型" style="width: 120px" allow-clear @change="$emit('filter')">
          <a-select-option value="text">文本</a-select-option>
          <a-select-option value="interactive">卡片</a-select-option>
        </a-select>
        <a-button @click="$emit('refresh')"><template #icon><ReloadOutlined /></template></a-button>
      </a-space>
    </template>
    <a-table :columns="logColumns" :data-source="logList" :loading="loading" row-key="id" :pagination="pagination" @change="handleTableChange">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'msg_type'">
          <a-tag :color="record.msg_type === 'interactive' ? 'blue' : 'default'">{{ record.msg_type === 'interactive' ? '卡片' : record.msg_type }}</a-tag>
        </template>
        <template v-if="column.key === 'source'">
          <a-tag :color="record.source === 'oa_sync' ? 'green' : 'default'">{{ record.source === 'oa_sync' ? 'OA同步' : '手动' }}</a-tag>
        </template>
        <template v-if="column.key === 'status'">
          <a-badge :status="record.status === 'success' ? 'success' : 'error'" :text="record.status === 'success' ? '成功' : '失败'" />
        </template>
        <template v-if="column.key === 'created_at'">{{ formatTime(record.created_at) }}</template>
        <template v-if="column.key === 'action'">
          <a-button type="link" size="small" @click="$emit('viewDetail', record)">详情</a-button>
        </template>
      </template>
    </a-table>
  </a-card>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { ReloadOutlined } from '@ant-design/icons-vue'
import type { FeishuMessageLog } from '@/services/feishu'

const props = defineProps<{
  logList: FeishuMessageLog[]
  loading: boolean
  pagination: {
    current: number
    pageSize: number
    total: number
    showSizeChanger: boolean
    showTotal: (total: number) => string
  }
}>()

const emit = defineEmits<{
  (e: 'filter'): void
  (e: 'refresh'): void
  (e: 'viewDetail', log: FeishuMessageLog): void
  (e: 'tableChange', pagination: any): void
}>()

const filter = reactive({ source: '', msg_type: '' })

const logColumns = [
  { title: '时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '类型', dataIndex: 'msg_type', key: 'msg_type', width: 80 },
  { title: '来源', dataIndex: 'source', key: 'source', width: 100 },
  { title: '接收者', dataIndex: 'receive_id', key: 'receive_id', width: 180, ellipsis: true },
  { title: '标题', dataIndex: 'title', key: 'title', ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '操作', key: 'action', width: 80 }
]

const formatTime = (time: string | undefined) => time ? time.replace('T', ' ').substring(0, 19) : '-'

const handleTableChange = (pagination: any) => {
  emit('tableChange', pagination)
}

defineExpose({ filter })
</script>
