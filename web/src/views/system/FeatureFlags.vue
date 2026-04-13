<template>
  <div class="feature-flags">
    <!-- 统计卡片 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="总开关数" :value="stats.total">
            <template #prefix><AppstoreOutlined style="color: #1890ff" /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="已启用" :value="stats.enabled" value-style="color: #52c41a">
            <template #prefix><CheckCircleOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="灰度中" :value="stats.rollout" value-style="color: #faad14">
            <template #prefix><ExperimentOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small">
          <a-statistic title="已禁用" :value="stats.disabled" value-style="color: #999">
            <template #prefix><StopOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <a-card title="功能开关管理" :bordered="false">
      <template #extra>
        <a-button type="primary" @click="showModal()">
          <PlusOutlined /> 新建开关
        </a-button>
      </template>

      <a-table :columns="columns" :data-source="flags" :loading="loading" row-key="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <a-space>
              <span style="font-weight: 500">{{ record.name }}</span>
              <a-tag v-if="record.display_name" size="small">{{ record.display_name }}</a-tag>
            </a-space>
          </template>
          <template v-if="column.key === 'is_enabled'">
            <a-switch v-model:checked="record.is_enabled" size="small" @change="toggleEnabled(record)" />
          </template>
          <template v-if="column.key === 'rollout'">
            <template v-if="record.is_enabled && record.rollout_percentage > 0 && record.rollout_percentage < 100">
              <a-progress :percent="record.rollout_percentage" size="small" style="width: 100px" status="active" />
            </template>
            <template v-else-if="record.is_enabled">
              <a-tag color="green">全量</a-tag>
            </template>
            <template v-else>
              <span style="color: #999">-</span>
            </template>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showModal(record)">编辑</a-button>
              <a-button type="link" size="small" @click="showTestModal(record)">测试</a-button>
              <a-popconfirm title="确定删除该开关？" @confirm="deleteFlag(record.name)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 新建/编辑弹窗 -->
    <a-modal v-model:open="modalVisible" :title="editingFlag ? '编辑开关' : '新建开关'" @ok="saveFlag" :confirm-loading="saving" width="600px">
      <a-form :model="form" :label-col="{ span: 5 }" :wrapper-col="{ span: 17 }">
        <a-form-item label="开关标识" required>
          <a-input v-model:value="form.name" placeholder="如：new_feature" :disabled="!!editingFlag" />
          <div style="color: #999; font-size: 12px">唯一标识，创建后不可修改</div>
        </a-form-item>
        <a-form-item label="显示名称">
          <a-input v-model:value="form.display_name" placeholder="如：新功能" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="form.description" :rows="2" placeholder="功能描述" />
        </a-form-item>
        <a-form-item label="启用状态">
          <a-switch v-model:checked="form.is_enabled" />
        </a-form-item>
        <a-form-item label="灰度比例" v-if="form.is_enabled">
          <a-slider v-model:value="form.rollout_percentage" :min="0" :max="100" :marks="{ 0: '0%', 50: '50%', 100: '100%' }" />
          <div style="color: #999; font-size: 12px">0% 表示仅白名单可用，100% 表示全量开放</div>
        </a-form-item>
        <a-form-item label="租户白名单" v-if="form.is_enabled">
          <a-select v-model:value="form.tenant_whitelist" mode="tags" placeholder="输入租户ID，回车添加" style="width: 100%" />
        </a-form-item>
        <a-form-item label="租户黑名单" v-if="form.is_enabled">
          <a-select v-model:value="form.tenant_blacklist" mode="tags" placeholder="输入租户ID，回车添加" style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 测试弹窗 -->
    <a-modal v-model:open="testModalVisible" title="测试开关" :footer="null" width="500px">
      <template v-if="testingFlag">
        <a-descriptions :column="1" size="small" style="margin-bottom: 16px">
          <a-descriptions-item label="开关名称">{{ testingFlag.name }}</a-descriptions-item>
          <a-descriptions-item label="当前状态">
            <a-tag :color="testingFlag.is_enabled ? 'green' : 'default'">
              {{ testingFlag.is_enabled ? '启用' : '禁用' }}
            </a-tag>
          </a-descriptions-item>
        </a-descriptions>

        <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
          <a-form-item label="租户ID">
            <a-input-number v-model:value="testParams.tenant_id" :min="1" style="width: 150px" />
          </a-form-item>
          <a-form-item label="用户ID">
            <a-input-number v-model:value="testParams.user_id" :min="1" style="width: 150px" />
          </a-form-item>
          <a-form-item :wrapper-col="{ offset: 6 }">
            <a-button type="primary" @click="testFlag" :loading="testing">检测</a-button>
          </a-form-item>
        </a-form>

        <a-alert v-if="testResult !== null" :type="testResult ? 'success' : 'warning'" style="margin-top: 16px">
          <template #message>
            <span v-if="testResult">✅ 该用户/租户可以使用此功能</span>
            <span v-else>❌ 该用户/租户无法使用此功能</span>
          </template>
        </a-alert>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, AppstoreOutlined, CheckCircleOutlined, ExperimentOutlined, StopOutlined } from '@ant-design/icons-vue'
import request from '@/utils/request'

interface FeatureFlag {
  id: number
  name: string
  display_name: string
  description: string
  is_enabled: boolean
  rollout_percentage: number
  tenant_whitelist: any
  tenant_blacklist: any
  created_at: string
  updated_at: string
}

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const modalVisible = ref(false)
const testModalVisible = ref(false)
const editingFlag = ref<FeatureFlag | null>(null)
const testingFlag = ref<FeatureFlag | null>(null)
const testResult = ref<boolean | null>(null)
const flags = ref<FeatureFlag[]>([])

const stats = reactive({ total: 0, enabled: 0, disabled: 0, rollout: 0 })
const form = reactive({
  name: '',
  display_name: '',
  description: '',
  is_enabled: false,
  rollout_percentage: 100,
  tenant_whitelist: [] as string[],
  tenant_blacklist: [] as string[]
})
const testParams = reactive({ tenant_id: undefined as number | undefined, user_id: undefined as number | undefined })

const columns = [
  { title: '开关标识', key: 'name' },
  { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
  { title: '状态', key: 'is_enabled', width: 80 },
  { title: '灰度比例', key: 'rollout', width: 140 },
  { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', width: 180 },
  { title: '操作', key: 'action', width: 160 }
]

const fetchFlags = async () => {
  loading.value = true
  try {
    const res = await request.get('/feature-flags')
    const data = res.data || res
    flags.value = data?.items || []
  } catch (e) { console.error('获取功能开关失败', e) }
  finally { loading.value = false }
}

const fetchStats = async () => {
  try {
    const res = await request.get('/feature-flags/stats')
    const data = res.data || res
    Object.assign(stats, data || {})
  } catch (e) { console.error('获取统计失败', e) }
}

const showModal = (record?: FeatureFlag) => {
  editingFlag.value = record || null
  if (record) {
    Object.assign(form, {
      name: record.name,
      display_name: record.display_name || '',
      description: record.description || '',
      is_enabled: record.is_enabled,
      rollout_percentage: record.rollout_percentage || 100,
      tenant_whitelist: record.tenant_whitelist?.ids?.map(String) || [],
      tenant_blacklist: record.tenant_blacklist?.ids?.map(String) || []
    })
  } else {
    Object.assign(form, { name: '', display_name: '', description: '', is_enabled: false, rollout_percentage: 100, tenant_whitelist: [], tenant_blacklist: [] })
  }
  modalVisible.value = true
}

const saveFlag = async () => {
  if (!form.name) { message.warning('请输入开关标识'); return }
  saving.value = true
  try {
    const payload = {
      ...form,
      tenant_whitelist: form.tenant_whitelist.map(Number).filter(n => !isNaN(n)),
      tenant_blacklist: form.tenant_blacklist.map(Number).filter(n => !isNaN(n))
    }
    if (editingFlag.value) {
      await request.put(`/feature-flags/${form.name}`, payload)
    } else {
      await request.post('/feature-flags', payload)
    }
    message.success('保存成功')
    modalVisible.value = false
    fetchFlags()
    fetchStats()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

const toggleEnabled = async (record: FeatureFlag) => {
  try {
    await request.put(`/feature-flags/${record.name}`, { is_enabled: record.is_enabled })
    message.success(record.is_enabled ? '已启用' : '已禁用')
    fetchStats()
  } catch (e) {
    record.is_enabled = !record.is_enabled
    message.error('操作失败')
  }
}

const deleteFlag = async (name: string) => {
  try {
    await request.delete(`/feature-flags/${name}`)
    message.success('删除成功')
    fetchFlags()
    fetchStats()
  } catch (e) { message.error('删除失败') }
}

const showTestModal = (record: FeatureFlag) => {
  testingFlag.value = record
  testResult.value = null
  testParams.tenant_id = undefined
  testParams.user_id = undefined
  testModalVisible.value = true
}

const testFlag = async () => {
  if (!testingFlag.value) return
  testing.value = true
  try {
    const params: any = {}
    if (testParams.tenant_id) params.tenant_id = testParams.tenant_id
    if (testParams.user_id) params.user_id = testParams.user_id
    const res = await request.get(`/feature-flags/${testingFlag.value.name}/check`, { params })
    testResult.value = res.data?.enabled || res?.enabled || false
  } catch (e) { message.error('检测失败') }
  finally { testing.value = false }
}

onMounted(() => { fetchFlags(); fetchStats() })
</script>

<style scoped>
.feature-flags :deep(.ant-slider-mark-text) {
  font-size: 12px;
}
</style>
