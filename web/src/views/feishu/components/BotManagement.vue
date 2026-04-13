<template>
  <a-card :bordered="false">
    <a-table :columns="botColumns" :data-source="botList" :loading="loadingBots" row-key="id" :pagination="{ pageSize: 10 }">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'webhook_url'">
          <a-typography-text copyable :content="record.webhook_url" ellipsis style="max-width: 300px">{{ record.webhook_url }}</a-typography-text>
        </template>
        <template v-if="column.key === 'status'">
          <a-badge :status="record.status === 'active' ? 'success' : 'default'" :text="record.status === 'active' ? '启用' : '禁用'" />
        </template>
        <template v-if="column.key === 'action'">
          <a-space>
            <a-button type="link" size="small" @click="$emit('edit', record)">编辑</a-button>
            <a-popconfirm title="确定删除？" @confirm="() => $emit('delete', record.id)" :disabled="!record.id">
              <a-button type="link" size="small" danger>删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>
  </a-card>
</template>

<script setup lang="ts">
import type { FeishuBot } from '@/services/feishu'

defineProps<{
  botList: FeishuBot[]
  loadingBots: boolean
}>()

defineEmits<{
  (e: 'edit', bot?: FeishuBot): void
  (e: 'delete', id: number): void
}>()

const botColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 150 },
  { title: 'Webhook URL', dataIndex: 'webhook_url', key: 'webhook_url', ellipsis: true },
  { title: '关联项目', dataIndex: 'project', key: 'project', width: 120 },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]
</script>
