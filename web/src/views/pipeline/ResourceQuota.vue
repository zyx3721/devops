<template>
  <div class="resource-quota">
    <!-- 操作栏 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="24" style="text-align: right">
        <a-button type="primary" @click="showCreateModal">
          <template #icon><PlusOutlined /></template>
          新建配额
        </a-button>
      </a-col>
    </a-row>

    <!-- 配额卡片列表 -->
    <a-row :gutter="16">
      <a-col
        v-for="quota in quotas"
        :key="quota.id"
        :xs="24"
        :sm="12"
        :lg="8"
        :xl="6"
        style="margin-bottom: 16px"
      >
        <a-card
          :bordered="false"
          :class="{ 'default-quota': quota.is_default, 'disabled-quota': !quota.enabled }"
        >
          <template #title>
            <div class="quota-title">
              <span>{{ quota.name }}</span>
              <a-tag v-if="quota.is_default" color="blue">默认</a-tag>
              <a-tag v-if="!quota.enabled" color="red">已禁用</a-tag>
            </div>
          </template>
          <template #extra>
            <a-dropdown>
              <a-button type="text" size="small">
                <MoreOutlined />
              </a-button>
              <template #overlay>
                <a-menu>
                  <a-menu-item @click="editQuota(quota)">
                    <EditOutlined /> 编辑
                  </a-menu-item>
                  <a-menu-item
                    v-if="!quota.is_default"
                    @click="setDefault(quota.id)"
                  >
                    <CheckOutlined /> 设为默认
                  </a-menu-item>
                  <a-menu-item @click="toggleEnabled(quota)">
                    <PoweroffOutlined />
                    {{ quota.enabled ? '禁用' : '启用' }}
                  </a-menu-item>
                  <a-menu-divider />
                  <a-menu-item danger @click="confirmDelete(quota.id)">
                    <DeleteOutlined /> 删除
                  </a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </template>

          <p class="quota-description">{{ quota.description || '暂无描述' }}</p>

          <!-- 资源限制 -->
          <a-divider style="margin: 12px 0" />
          <div class="quota-resources">
            <div class="resource-item">
              <span class="resource-label">CPU:</span>
              <span class="resource-value">{{ quota.max_cpu }}</span>
            </div>
            <div class="resource-item">
              <span class="resource-label">内存:</span>
              <span class="resource-value">{{ quota.max_memory }}</span>
            </div>
            <div class="resource-item">
              <span class="resource-label">存储:</span>
              <span class="resource-value">{{ quota.max_storage }}</span>
            </div>
            <div class="resource-item">
              <span class="resource-label">并发数:</span>
              <span class="resource-value">{{ quota.max_concurrent }}</span>
            </div>
            <div class="resource-item">
              <span class="resource-label">时长限制:</span>
              <span class="resource-value">{{ formatDuration(quota.max_duration) }}</span>
            </div>
          </div>

          <!-- 使用情况 -->
          <a-divider style="margin: 12px 0" />
          <QuotaUsageChart
            v-if="usageMap[quota.id]"
            :usage="usageMap[quota.id]"
            :quota="quota"
          />
          <a-button
            type="link"
            size="small"
            @click="loadUsage(quota.id)"
            :loading="loadingUsage[quota.id]"
          >
            <ReloadOutlined /> 刷新使用情况
          </a-button>
        </a-card>
      </a-col>
    </a-row>

    <!-- 空状态 -->
    <a-empty v-if="!loading && quotas.length === 0" description="暂无配额">
      <a-button type="primary" @click="showCreateModal">立即创建</a-button>
    </a-empty>

    <!-- 创建/编辑配额弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      :title="isEdit ? '编辑配额' : '新建配额'"
      :confirm-loading="submitting"
      @ok="handleSubmit"
      width="600px"
    >
      <a-form
        ref="formRef"
        :model="formData"
        :rules="rules"
        :label-col="{ span: 6 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="配额名称" name="name">
          <a-input v-model:value="formData.name" placeholder="请输入配额名称" />
        </a-form-item>

        <a-form-item label="描述" name="description">
          <a-textarea
            v-model:value="formData.description"
            placeholder="请输入配额描述"
            :rows="3"
          />
        </a-form-item>

        <a-form-item label="CPU 限制" name="max_cpu">
          <a-input
            v-model:value="formData.max_cpu"
            placeholder="例如: 2 或 2000m"
            addon-after="cores"
          />
        </a-form-item>

        <a-form-item label="内存限制" name="max_memory">
          <a-input
            v-model:value="formData.max_memory"
            placeholder="例如: 4Gi 或 4096Mi"
            addon-after="Gi/Mi"
          />
        </a-form-item>

        <a-form-item label="存储限制" name="max_storage">
          <a-input
            v-model:value="formData.max_storage"
            placeholder="例如: 10Gi"
            addon-after="Gi"
          />
        </a-form-item>

        <a-form-item label="最大并发数" name="max_concurrent">
          <a-input-number
            v-model:value="formData.max_concurrent"
            :min="1"
            :max="100"
            style="width: 100%"
          />
        </a-form-item>

        <a-form-item label="时长限制" name="max_duration">
          <a-input-number
            v-model:value="formData.max_duration"
            :min="60"
            :max="86400"
            addon-after="秒"
            style="width: 100%"
          />
        </a-form-item>

        <a-form-item label="优先级" name="priority">
          <a-input-number
            v-model:value="formData.priority"
            :min="0"
            :max="100"
            style="width: 100%"
          />
        </a-form-item>

        <a-form-item label="设为默认" name="is_default">
          <a-switch v-model:checked="formData.is_default" />
        </a-form-item>

        <a-form-item label="启用" name="enabled">
          <a-switch v-model:checked="formData.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import {
  PlusOutlined,
  MoreOutlined,
  EditOutlined,
  DeleteOutlined,
  CheckOutlined,
  PoweroffOutlined,
  ReloadOutlined,
} from '@ant-design/icons-vue'
import request from '@/services/api'
import QuotaUsageChart from '@/components/pipeline/QuotaUsageChart.vue'

interface ResourceQuota {
  id: number
  name: string
  description: string
  project_id?: number
  max_cpu: string
  max_memory: string
  max_storage: string
  max_concurrent: number
  max_duration: number
  priority: number
  is_default: boolean
  enabled: boolean
}

interface QuotaUsage {
  quota_id: number
  cpu_used: string
  memory_used: string
  storage_used: string
  concurrent_used: number
  cpu_percent: number
  memory_percent: number
  storage_percent: number
}

const loading = ref(false)
const submitting = ref(false)
const modalVisible = ref(false)
const isEdit = ref(false)
const quotas = ref<ResourceQuota[]>([])
const usageMap = ref<Record<number, QuotaUsage>>({})
const loadingUsage = ref<Record<number, boolean>>({})
const formRef = ref()

const formData = reactive<Partial<ResourceQuota>>({
  name: '',
  description: '',
  max_cpu: '2',
  max_memory: '4Gi',
  max_storage: '10Gi',
  max_concurrent: 5,
  max_duration: 3600,
  priority: 10,
  is_default: false,
  enabled: true,
})

const rules = {
  name: [{ required: true, message: '请输入配额名称', trigger: 'blur' }],
  max_cpu: [{ required: true, message: '请输入 CPU 限制', trigger: 'blur' }],
  max_memory: [{ required: true, message: '请输入内存限制', trigger: 'blur' }],
  max_storage: [{ required: true, message: '请输入存储限制', trigger: 'blur' }],
  max_concurrent: [{ required: true, message: '请输入最大并发数', trigger: 'blur' }],
  max_duration: [{ required: true, message: '请输入时长限制', trigger: 'blur' }],
}

const formatDuration = (seconds: number): string => {
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟`
  return `${Math.floor(seconds / 3600)}小时`
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await request.get('/build/quota')
    quotas.value = res?.data || []
    
    // 加载每个配额的使用情况
    quotas.value.forEach(quota => {
      loadUsage(quota.id)
    })
  } catch (error) {
    console.error('加载配额列表失败:', error)
  } finally {
    loading.value = false
  }
}

const loadUsage = async (quotaId: number) => {
  loadingUsage.value[quotaId] = true
  try {
    // 注意：这里假设有一个获取配额使用情况的 API
    // 如果后端没有提供，可以移除这部分或使用模拟数据
    const res = await request.get(`/build/quota/${quotaId}/usage`)
    if (res?.data) {
      usageMap.value[quotaId] = res.data
    }
  } catch (error) {
    // 如果 API 不存在，使用默认值
    usageMap.value[quotaId] = {
      quota_id: quotaId,
      cpu_used: '0',
      memory_used: '0',
      storage_used: '0',
      concurrent_used: 0,
      cpu_percent: 0,
      memory_percent: 0,
      storage_percent: 0,
    }
  } finally {
    loadingUsage.value[quotaId] = false
  }
}

const showCreateModal = () => {
  isEdit.value = false
  Object.assign(formData, {
    name: '',
    description: '',
    max_cpu: '2',
    max_memory: '4Gi',
    max_storage: '10Gi',
    max_concurrent: 5,
    max_duration: 3600,
    priority: 10,
    is_default: false,
    enabled: true,
  })
  modalVisible.value = true
}

const editQuota = (quota: ResourceQuota) => {
  isEdit.value = true
  Object.assign(formData, quota)
  modalVisible.value = true
}

const handleSubmit = async () => {
  try {
    await formRef.value.validate()
    submitting.value = true
    
    if (isEdit.value && formData.id) {
      await request.put(`/build/quota/${formData.id}`, formData)
      message.success('更新成功')
    } else {
      await request.post('/build/quota', formData)
      message.success('创建成功')
    }
    
    modalVisible.value = false
    loadData()
  } catch (error: any) {
    if (error?.errorFields) return // 表单验证错误
    message.error(error?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const setDefault = async (id: number) => {
  try {
    await request.put(`/build/quota/${id}`, { is_default: true })
    message.success('设置成功')
    loadData()
  } catch (error: any) {
    message.error(error?.message || '设置失败')
  }
}

const toggleEnabled = async (quota: ResourceQuota) => {
  try {
    await request.put(`/build/quota/${quota.id}`, {
      enabled: !quota.enabled,
    })
    message.success(quota.enabled ? '已禁用' : '已启用')
    loadData()
  } catch (error: any) {
    message.error(error?.message || '操作失败')
  }
}

const confirmDelete = (id: number) => {
  Modal.confirm({
    title: '确认删除',
    content: '确定要删除该配额吗？此操作不可恢复。',
    okText: '确定',
    okType: 'danger',
    cancelText: '取消',
    onOk: async () => {
      try {
        await request.delete(`/build/quota/${id}`)
        message.success('删除成功')
        loadData()
      } catch (error: any) {
        message.error(error?.message || '删除失败')
      }
    },
  })
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.resource-quota {
  padding: 0;
}

.quota-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.default-quota {
  border: 2px solid #1890ff;
}

.disabled-quota {
  opacity: 0.6;
}

.quota-description {
  color: rgba(0, 0, 0, 0.45);
  margin-bottom: 0;
  min-height: 22px;
}

.quota-resources {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.resource-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.resource-label {
  color: rgba(0, 0, 0, 0.65);
  font-size: 14px;
}

.resource-value {
  font-weight: 500;
  color: #1890ff;
}

@media (max-width: 768px) {
  :deep(.ant-col) {
    margin-bottom: 16px;
  }
}
</style>
