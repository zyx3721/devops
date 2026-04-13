<template>
  <div class="llm-config">
    <a-card title="LLM 配置管理">
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <template #icon><PlusOutlined /></template>
          添加配置
        </a-button>
      </template>

      <!-- 配置列表 -->
      <a-table
        :columns="columns"
        :data-source="configList"
        :loading="loading"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <span>{{ record.name }}</span>
            <a-tag v-if="record.is_default" color="green" style="margin-left: 8px">默认</a-tag>
          </template>
          <template v-else-if="column.key === 'provider'">
            <a-tag :color="getProviderColor(record.provider)">
              {{ getProviderLabel(record.provider) }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-badge :status="record.is_active ? 'success' : 'default'" :text="record.is_active ? '启用' : '禁用'" />
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showEditModal(record)">编辑</a-button>
              <a-button
                type="link"
                size="small"
                :disabled="record.is_default"
                @click="handleSetDefault(record.id)"
              >
                设为默认
              </a-button>
              <a-popconfirm
                title="确定删除此配置？"
                @confirm="handleDelete(record.id)"
              >
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 创建/编辑弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      :title="isEdit ? '编辑配置' : '添加配置'"
      width="600px"
      @ok="handleSubmit"
      :confirmLoading="submitting"
    >
      <a-form :model="formData" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
        <a-form-item label="配置名称" required>
          <a-input v-model:value="formData.name" placeholder="请输入配置名称" />
        </a-form-item>
        <a-form-item label="提供商" required>
          <a-select v-model:value="formData.provider" placeholder="请选择提供商">
            <a-select-option v-for="p in providers" :key="p.value" :value="p.value">
              {{ p.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="API 地址" required>
          <a-input v-model:value="formData.api_url" placeholder="https://api.openai.com/v1" />
        </a-form-item>
        <a-form-item label="API Key" required>
          <a-input-password v-model:value="formData.api_key" placeholder="请输入 API Key" />
        </a-form-item>
        <a-form-item label="模型名称" required>
          <a-input v-model:value="formData.model_name" placeholder="gpt-4" />
        </a-form-item>
        <a-form-item label="最大 Token">
          <a-input-number v-model:value="formData.max_tokens" :min="100" :max="128000" style="width: 100%" />
        </a-form-item>
        <a-form-item label="温度">
          <a-slider v-model:value="formData.temperature" :min="0" :max="2" :step="0.1" />
        </a-form-item>
        <a-form-item label="超时时间(秒)">
          <a-input-number v-model:value="formData.timeout_seconds" :min="10" :max="300" style="width: 100%" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="formData.description" :rows="2" placeholder="配置描述" />
        </a-form-item>
        <a-form-item label="状态">
          <a-switch v-model:checked="formData.is_active" checked-children="启用" un-checked-children="禁用" />
        </a-form-item>
        <a-form-item label="设为默认">
          <a-switch v-model:checked="formData.is_default" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { llmConfigApi } from '../../services/ai'
import type { AILLMConfig } from '../../types/ai'

const loading = ref(false)
const submitting = ref(false)
const configList = ref<AILLMConfig[]>([])
const providers = ref<{ value: string; label: string }[]>([])
const modalVisible = ref(false)
const isEdit = ref(false)
const editId = ref<number | null>(null)

const formData = reactive({
  name: '',
  provider: '',
  api_url: '',
  api_key: '',
  model_name: '',
  max_tokens: 4096,
  temperature: 0.7,
  timeout_seconds: 60,
  description: '',
  is_active: true,
  is_default: false,
})

const columns = [
  { title: '配置名称', dataIndex: 'name', key: 'name' },
  { title: '提供商', dataIndex: 'provider', key: 'provider', width: 120 },
  { title: '模型', dataIndex: 'model_name', key: 'model_name', width: 150 },
  { title: 'API 地址', dataIndex: 'api_url', key: 'api_url', ellipsis: true },
  { title: '状态', key: 'status', width: 100 },
  { title: '操作', key: 'action', width: 200 },
]

const providerColors: Record<string, string> = {
  openai: 'green',
  azure: 'blue',
  deepseek: 'geekblue',
  qwen: 'orange',
  zhipu: 'purple',
  ollama: 'cyan',
  custom: 'magenta',
}

const getProviderColor = (provider: string) => providerColors[provider] || 'default'

const getProviderLabel = (value: string) => {
  const p = providers.value.find(p => p.value === value)
  return p?.label || value
}

const loadProviders = async () => {
  try {
    const res = await llmConfigApi.getProviders()
    if (res.data) {
      providers.value = res.data
    }
  } catch (e) {
    console.error('Load providers error:', e)
  }
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await llmConfigApi.list()
    if (res.data) {
      configList.value = res.data
    }
  } catch (e) {
    console.error('Load data error:', e)
  } finally {
    loading.value = false
  }
}

const showCreateModal = () => {
  isEdit.value = false
  editId.value = null
  Object.assign(formData, {
    name: '',
    provider: 'openai',
    api_url: 'https://api.openai.com/v1',
    api_key: '',
    model_name: 'gpt-4',
    max_tokens: 4096,
    temperature: 0.7,
    timeout_seconds: 60,
    description: '',
    is_active: true,
    is_default: false,
  })
  modalVisible.value = true
}

const showEditModal = (record: AILLMConfig) => {
  isEdit.value = true
  editId.value = record.id
  Object.assign(formData, {
    name: record.name,
    provider: record.provider,
    api_url: record.api_url,
    api_key: '', // 不回显密钥
    model_name: record.model_name,
    max_tokens: record.max_tokens,
    temperature: record.temperature,
    timeout_seconds: record.timeout_seconds,
    description: record.description || '',
    is_active: record.is_active,
    is_default: record.is_default,
  })
  modalVisible.value = true
}

const handleSubmit = async () => {
  if (!formData.name || !formData.provider || !formData.api_url || !formData.model_name) {
    message.warning('请填写完整信息')
    return
  }

  if (!isEdit.value && !formData.api_key) {
    message.warning('请输入 API Key')
    return
  }

  submitting.value = true
  try {
    const data = { ...formData }
    if (isEdit.value && !data.api_key) {
      delete (data as any).api_key // 不更新空密钥
    }

    if (isEdit.value && editId.value) {
      await llmConfigApi.update(editId.value, data)
      message.success('更新成功')
    } else {
      await llmConfigApi.create(data)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadData()
  } catch (e: any) {
    message.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleSetDefault = async (id: number) => {
  try {
    await llmConfigApi.setDefault(id)
    message.success('设置成功')
    loadData()
  } catch (e: any) {
    message.error(e.message || '设置失败')
  }
}

const handleDelete = async (id: number) => {
  try {
    await llmConfigApi.delete(id)
    message.success('删除成功')
    loadData()
  } catch (e: any) {
    message.error(e.message || '删除失败')
  }
}

onMounted(() => {
  loadProviders()
  loadData()
})
</script>

<style scoped>
.llm-config {
  padding: 0;
}
</style>
