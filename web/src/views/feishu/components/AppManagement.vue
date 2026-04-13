<template>
  <a-card :bordered="false">
    <template #extra>
      <a-space>
        <a-tag color="green">回调运行: {{ runningCallbacks.length }} 个</a-tag>
        <a-button size="small" @click="$emit('refreshCallbacks')" :loading="refreshingCallbacks">
          <template #icon><ReloadOutlined /></template>
          刷新回调
        </a-button>
      </a-space>
    </template>
    <a-table :columns="appColumns" :data-source="appList" :loading="loadingApps" row-key="id" :pagination="{ pageSize: 10 }">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'app_id'">
          <a-typography-text copyable :content="record.app_id">{{ record.app_id }}</a-typography-text>
        </template>
        <template v-if="column.key === 'callback'">
          <a-tag v-if="isCallbackRunning(record.app_id)" color="green">
            <template #icon><CheckCircleOutlined /></template>
            已连接
          </a-tag>
          <a-tag v-else color="default">
            <template #icon><CloseCircleOutlined /></template>
            未连接
          </a-tag>
        </template>
        <template v-if="column.key === 'webhook'">
          <a-typography-text v-if="record.webhook" copyable :content="record.webhook" ellipsis style="max-width: 200px">{{ record.webhook }}</a-typography-text>
          <span v-else>-</span>
        </template>
        <template v-if="column.key === 'status'">
          <a-badge :status="record.status === 'active' ? 'success' : 'default'" :text="record.status === 'active' ? '启用' : '禁用'" />
        </template>
        <template v-if="column.key === 'is_default'">
          <a-tag v-if="record.is_default" color="blue">默认</a-tag>
          <a-button v-else type="link" size="small" @click="$emit('setDefault', record.id)">设为默认</a-button>
        </template>
        <template v-if="column.key === 'action'">
          <a-space>
            <a-button type="link" size="small" @click="$emit('edit', record)">编辑</a-button>
            <a-popconfirm title="确定删除？" @confirm="() => $emit('delete', record.id)" :disabled="!record.id">
              <a-button type="link" size="small" danger :disabled="record.is_default">删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>
  </a-card>
</template>

<script setup lang="ts">
import { ReloadOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons-vue'
import type { FeishuApp } from '@/services/feishu'

defineProps<{
  appList: FeishuApp[]
  loadingApps: boolean
  runningCallbacks: string[]
  refreshingCallbacks: boolean
}>()

defineEmits<{
  (e: 'edit', app?: FeishuApp): void
  (e: 'delete', id: number): void
  (e: 'setDefault', id: number): void
  (e: 'refreshCallbacks'): void
}>()

const isCallbackRunning = (appId: string) => {
  const props = defineProps<{ runningCallbacks: string[] }>()
  return props.runningCallbacks.includes(appId)
}

const appColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 120 },
  { title: 'App ID', dataIndex: 'app_id', key: 'app_id', width: 180 },
  { title: '回调状态', key: 'callback', width: 100 },
  { title: '项目', dataIndex: 'project', key: 'project', width: 100 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '默认', dataIndex: 'is_default', key: 'is_default', width: 100 },
  { title: '操作', key: 'action', width: 120 }
]
</script>
