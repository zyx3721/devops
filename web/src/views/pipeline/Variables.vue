<template>
  <div class="variables-page">
    <a-page-header title="变量管理" sub-title="管理全局和流水线级别的环境变量">
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <PlusOutlined /> 新建变量
        </a-button>
      </template>
    </a-page-header>

    <a-card :bordered="false">
      <!-- 筛选 -->
      <a-row :gutter="16" style="margin-bottom: 16px">
        <a-col :span="6">
          <a-select v-model:value="filterScope" placeholder="作用域" allowClear style="width: 100%" @change="loadVariables">
            <a-select-option value="global">全局变量</a-select-option>
            <a-select-option value="pipeline">流水线变量</a-select-option>
          </a-select>
        </a-col>
        <a-col :span="8">
          <a-select
            v-model:value="filterPipelineId"
            placeholder="选择流水线"
            allowClear
            show-search
            :filter-option="filterPipelineOption"
            style="width: 100%"
            @change="loadVariables"
          >
            <a-select-option v-for="p in pipelines" :key="p.id" :value="p.id">
              {{ p.name }}
            </a-select-option>
          </a-select>
        </a-col>
      </a-row>

      <a-table
        :columns="columns"
        :data-source="variables"
        :loading="loading"
        :pagination="false"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'scope'">
            <a-tag :color="record.scope === 'global' ? 'blue' : 'green'">
              {{ record.scope === 'global' ? '全局' : '流水线' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'value'">
            <span v-if="record.is_secret" class="secret-value">******</span>
            <span v-else>{{ record.value }}</span>
          </template>
          <template v-else-if="column.key === 'is_secret'">
            <a-tag v-if="record.is_secret" color="orange">敏感</a-tag>
            <a-tag v-else color="default">普通</a-tag>
          </template>
          <template v-else-if="column.key === 'pipeline'">
            {{ getPipelineName(record.pipeline_id) }}
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatDate(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showEditModal(record)">编辑</a-button>
              <a-popconfirm
                title="确定要删除此变量吗？"
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
      :title="isEdit ? '编辑变量' : '新建变量'"
      :confirm-loading="submitting"
      @ok="handleSubmit"
      width="500px"
    >
      <a-form
        ref="formRef"
        :model="form"
        :rules="rules"
        :label-col="{ span: 5 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="变量名" name="name">
          <a-input v-model:value="form.name" placeholder="如: DATABASE_URL" />
          <div class="form-hint">变量名建议使用大写字母和下划线</div>
        </a-form-item>
        <a-form-item label="变量值" name="value">
          <a-input-password
            v-if="form.is_secret"
            v-model:value="form.value"
            :placeholder="isEdit ? '留空则不修改' : '请输入变量值'"
          />
          <a-input v-else v-model:value="form.value" placeholder="请输入变量值" />
        </a-form-item>
        <a-form-item label="敏感变量" name="is_secret">
          <a-switch v-model:checked="form.is_secret" />
          <span class="form-hint" style="margin-left: 8px">敏感变量在日志中会被脱敏显示</span>
        </a-form-item>
        <a-form-item label="作用域" name="scope">
          <a-radio-group v-model:value="form.scope">
            <a-radio value="global">全局</a-radio>
            <a-radio value="pipeline">流水线</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item v-if="form.scope === 'pipeline'" label="流水线" name="pipeline_id">
          <a-select
            v-model:value="form.pipeline_id"
            placeholder="选择流水线"
            show-search
            :filter-option="filterPipelineOption"
          >
            <a-select-option v-for="p in pipelines" :key="p.id" :value="p.id">
              {{ p.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { pipelineApi } from '@/services/pipeline'
import dayjs from 'dayjs'

interface Variable {
  id: number
  name: string
  value: string
  is_secret: boolean
  scope: string
  pipeline_id?: number
  created_at: string
}

interface Pipeline {
  id: number
  name: string
}

const loading = ref(false)
const submitting = ref(false)
const modalVisible = ref(false)
const isEdit = ref(false)
const variables = ref<Variable[]>([])
const pipelines = ref<Pipeline[]>([])
const filterScope = ref<string>()
const filterPipelineId = ref<number>()
const formRef = ref()

const form = reactive({
  id: 0,
  name: '',
  value: '',
  is_secret: false,
  scope: 'global',
  pipeline_id: undefined as number | undefined
})

const rules = {
  name: [
    { required: true, message: '请输入变量名' },
    { pattern: /^[A-Za-z_][A-Za-z0-9_]*$/, message: '变量名只能包含字母、数字和下划线，且不能以数字开头' }
  ],
  value: [{ required: true, message: '请输入变量值' }]
}

const columns = [
  { title: '变量名', dataIndex: 'name', key: 'name' },
  { title: '变量值', dataIndex: 'value', key: 'value', ellipsis: true },
  { title: '类型', key: 'is_secret', width: 80 },
  { title: '作用域', key: 'scope', width: 100 },
  { title: '流水线', key: 'pipeline', width: 150 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', key: 'action', width: 150 }
]

const formatDate = (date: string) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

const getPipelineName = (pipelineId?: number) => {
  if (!pipelineId) return '-'
  const pipeline = pipelines.value.find(p => p.id === pipelineId)
  return pipeline?.name || '-'
}

const filterPipelineOption = (input: string, option: any) => {
  return option.children[0].children.toLowerCase().includes(input.toLowerCase())
}

const loadVariables = async () => {
  loading.value = true
  try {
    const res = await pipelineApi.getVariables({
      scope: filterScope.value,
      pipeline_id: filterPipelineId.value
    })
    variables.value = res?.data || []
  } catch (error) {
    console.error('加载变量失败:', error)
  } finally {
    loading.value = false
  }
}

const loadPipelines = async () => {
  try {
    const res = await pipelineApi.list({ page_size: 1000 })
    pipelines.value = res?.data?.items || []
  } catch (error) {
    console.error('加载流水线失败:', error)
  }
}

const resetForm = () => {
  form.id = 0
  form.name = ''
  form.value = ''
  form.is_secret = false
  form.scope = 'global'
  form.pipeline_id = undefined
}

const showCreateModal = () => {
  resetForm()
  isEdit.value = false
  modalVisible.value = true
}

const showEditModal = (record: Variable) => {
  resetForm()
  form.id = record.id
  form.name = record.name
  form.value = record.is_secret ? '' : record.value
  form.is_secret = record.is_secret
  form.scope = record.scope
  form.pipeline_id = record.pipeline_id
  isEdit.value = true
  modalVisible.value = true
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  // 编辑敏感变量时，如果值为空则不修改
  if (isEdit.value && form.is_secret && !form.value) {
    // 不传 value 字段
  }

  submitting.value = true
  try {
    const data = {
      name: form.name,
      value: form.value,
      is_secret: form.is_secret,
      scope: form.scope,
      pipeline_id: form.scope === 'pipeline' ? form.pipeline_id : undefined
    }

    if (isEdit.value) {
      await pipelineApi.updateVariable(form.id, data)
      message.success('更新成功')
    } else {
      await pipelineApi.createVariable(data)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadVariables()
  } catch (error: any) {
    message.error(error?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: number) => {
  try {
    await pipelineApi.deleteVariable(id)
    message.success('删除成功')
    loadVariables()
  } catch (error: any) {
    message.error(error?.message || '删除失败')
  }
}

onMounted(() => {
  loadVariables()
  loadPipelines()
})
</script>

<style scoped>
.variables-page {
  padding: 0;
}

.secret-value {
  color: #999;
  font-family: monospace;
}

.form-hint {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}
</style>
