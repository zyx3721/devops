<template>
  <a-row :gutter="24">
    <a-col :xs="24" :lg="10">
      <a-card title="基本配置" :bordered="false">
        <a-form :model="form" layout="vertical">
          <a-row :gutter="16">
            <a-col :span="12"><a-form-item label="接收者ID" required><a-input v-model:value="form.receive_id" placeholder="群组或用户ID" /></a-form-item></a-col>
            <a-col :span="12">
              <a-form-item label="ID类型" required>
                <a-select v-model:value="form.receive_id_type" style="width: 100%">
                  <a-select-option value="chat_id">Chat ID (群组)</a-select-option>
                  <a-select-option value="open_id">Open ID</a-select-option>
                </a-select>
              </a-form-item>
            </a-col>
          </a-row>
          <a-form-item label="卡片标题"><a-input v-model:value="form.title" placeholder="应用发布申请" /></a-form-item>
        </a-form>
      </a-card>
    </a-col>
    <a-col :xs="24" :lg="14">
      <a-card :bordered="false">
        <template #title><span>服务配置</span><a-button type="link" size="small" @click="addService" style="float: right"><template #icon><PlusOutlined /></template>添加服务</a-button></template>
        <a-empty v-if="form.services.length === 0" description="暂无服务配置"><a-button type="primary" @click="addService">添加第一个服务</a-button></a-empty>
        <div v-else class="service-list">
          <div v-for="(service, index) in form.services" :key="index" class="service-item">
            <div class="service-header">
              <span class="service-title">服务 {{ index + 1 }}</span>
              <a-button type="text" danger size="small" @click="removeService(index)" v-if="form.services.length > 1"><template #icon><DeleteOutlined /></template></a-button>
            </div>
            <a-row :gutter="12">
              <a-col :span="8"><a-input v-model:value="service.name" placeholder="服务名称" /></a-col>
              <a-col :span="8"><a-input v-model:value="service.object_id" placeholder="分支" /></a-col>
              <a-col :span="8">
                <a-select v-model:value="service.actions" mode="multiple" placeholder="操作" style="width: 100%">
                  <a-select-option value="gray">灰度</a-select-option>
                  <a-select-option value="official">正式</a-select-option>
                  <a-select-option value="rollback">回滚</a-select-option>
                  <a-select-option value="restart">重启</a-select-option>
                </a-select>
              </a-col>
            </a-row>
          </div>
        </div>
        <a-divider />
        <a-button type="primary" @click="handleSend" :loading="sending" block :disabled="form.services.length === 0">
          <template #icon><SendOutlined /></template>发送发布卡片
        </a-button>
      </a-card>
    </a-col>
  </a-row>
</template>

<script setup lang="ts">
import { reactive } from 'vue'
import { PlusOutlined, DeleteOutlined, SendOutlined } from '@ant-design/icons-vue'

const props = defineProps<{
  sending: boolean
}>()

const emit = defineEmits<{
  (e: 'send', form: typeof form): void
}>()

const form = reactive({
  receive_id: '',
  receive_id_type: 'chat_id' as const,
  title: '应用发布申请',
  services: [{ name: '', object_id: '', actions: [] as string[] }]
})

const addService = () => {
  form.services.push({ name: '', object_id: '', actions: [] })
}

const removeService = (index: number) => {
  form.services.splice(index, 1)
}

const handleSend = () => {
  emit('send', form)
}
</script>

<style scoped>
.service-list { max-height: 400px; overflow-y: auto; }
.service-item { background: #fafafa; padding: 12px; margin-bottom: 12px; border-radius: 6px; border: 1px solid #f0f0f0; }
.service-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; }
.service-title { font-weight: 500; color: #666; }
</style>
