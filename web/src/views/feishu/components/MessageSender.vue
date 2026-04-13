<template>
  <a-row :gutter="24">
    <a-col :xs="24" :lg="12">
      <a-card title="消息配置" :bordered="false">
        <a-form :model="form" layout="vertical">
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="接收者ID" required><a-input v-model:value="form.receive_id" placeholder="请输入接收者ID" /></a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="ID类型" required>
                <a-select v-model:value="form.receive_id_type" style="width: 100%">
                  <a-select-option value="open_id">Open ID</a-select-option>
                  <a-select-option value="user_id">User ID</a-select-option>
                  <a-select-option value="chat_id">Chat ID (群组)</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
          <a-form-item label="消息类型" required>
            <a-radio-group v-model:value="form.msg_type" button-style="solid">
              <a-radio-button value="text">文本</a-radio-button>
              <a-radio-button value="post">富文本</a-radio-button>
              <a-radio-button value="interactive">卡片</a-radio-button>
            </a-radio-group>
          </a-form-item>
          <a-form-item label="消息内容" required><a-textarea v-model:value="form.content" :placeholder="contentPlaceholder" :rows="6" /></a-form-item>
          <a-form-item>
            <a-button type="primary" @click="handleSend" :loading="sending" block>
              <template #icon><SendOutlined /></template>发送消息
            </a-button>
          </a-form-item>
        </a-form>
      </a-card>
    </a-col>
    <a-col :xs="24" :lg="12">
      <a-card title="使用说明" :bordered="false">
        <a-list size="small" :data-source="idTypeHelp" :split="false">
          <template #renderItem="{ item }">
            <a-list-item><a-typography-text code>{{ item.type }}</a-typography-text><span style="margin-left: 8px">{{ item.desc }}</span></a-list-item>
          </template>
        </a-list>
      </a-card>
    </a-col>
  </a-row>
</template>

<script setup lang="ts">
import { reactive, computed } from 'vue'
import { SendOutlined } from '@ant-design/icons-vue'

const props = defineProps<{
  sending: boolean
}>()

const emit = defineEmits<{
  (e: 'send', form: typeof form): void
}>()

const form = reactive({
  receive_id: '',
  receive_id_type: 'chat_id' as string,
  msg_type: 'text' as string,
  content: ''
})

const idTypeHelp = [
  { type: 'open_id', desc: '用户在应用内的唯一标识' },
  { type: 'chat_id', desc: '群组的唯一标识' }
]

const contentPlaceholder = computed(() => {
  switch (form.msg_type) {
    case 'text':
      return '示例：\n你好，这是一条测试消息'
    case 'post':
      return `示例：\n{\n  "zh_cn": {\n    "title": "消息标题",\n    "content": [\n      [\n        {"tag": "text", "text": "这是一段普通文字，"},\n        {"tag": "a", "text": "点击跳转", "href": "https://example.com"}\n      ]\n    ]\n  }\n}`
    case 'interactive':
      return `示例：\n{\n  "schema": "2.0",\n  "header": {\n    "title": {"content": "卡片标题", "tag": "plain_text"},\n    "template": "blue"\n  },\n  "body": {\n    "elements": [\n      {"tag": "markdown", "content": "**加粗文字**\\n普通文字"}\n    ]\n  }\n}`
    default:
      return '请输入消息内容'
  }
})

const handleSend = () => {
  emit('send', form)
}
</script>
