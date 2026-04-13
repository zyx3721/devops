<template>
  <div class="parse-config">
    <div class="toolbar">
      <a-button type="primary" @click="showCreateDialog">
        <template #icon><PlusOutlined /></template>
        新建模板
      </a-button>
    </div>

    <!-- 预设模板 -->
    <div class="section">
      <h4>预设模板</h4>
      <div class="template-grid">
        <div 
          v-for="template in presetTemplates" 
          :key="template.id" 
          :class="['template-card', { active: selectedTemplate?.id === template.id }]"
          @click="selectTemplate(template)"
        >
          <div class="template-name">{{ template.name }}</div>
          <div class="template-type">
            <a-tag size="small">{{ typeLabels[template.type] }}</a-tag>
          </div>
          <div class="template-desc">{{ template.description || '无描述' }}</div>
        </div>
      </div>
    </div>

    <!-- 自定义模板 -->
    <div class="section" v-if="customTemplates.length > 0">
      <h4>自定义模板</h4>
      <div class="template-grid">
        <div 
          v-for="template in customTemplates" 
          :key="template.id" 
          :class="['template-card', { active: selectedTemplate?.id === template.id }]"
          @click="selectTemplate(template)"
        >
          <div class="template-name">{{ template.name }}</div>
          <div class="template-type">
            <a-tag size="small">{{ typeLabels[template.type] }}</a-tag>
          </div>
          <div class="template-desc">{{ template.description || '无描述' }}</div>
          <div class="template-actions">
            <a-button type="link" size="small" @click.stop="editTemplate(template)">编辑</a-button>
            <a-button type="link" danger size="small" @click.stop="deleteTemplate(template)">删除</a-button>
          </div>
        </div>
      </div>
    </div>

    <!-- 测试区域 -->
    <div class="section" v-if="selectedTemplate">
      <h4>模板测试</h4>
      <a-textarea
        v-model:value="testContent"
        :rows="4"
        placeholder="输入日志内容进行测试"
      />
      <a-button type="primary" @click="testParse" :loading="testing" style="margin-top: 10px">
        测试解析
      </a-button>
      
      <div v-if="testResult" class="test-result">
        <a-alert v-if="testResult.success" type="success" :show-icon="false">
          <template #message>
            <div class="alert-title">解析成功</div>
            <pre>{{ JSON.stringify(testResult.parsed, null, 2) }}</pre>
          </template>
        </a-alert>
        <a-alert v-else type="error" :show-icon="false">
          <template #message>
            <div class="alert-title">解析失败</div>
            {{ testResult.error }}
          </template>
        </a-alert>
      </div>
    </div>

    <!-- 创建/编辑对话框 -->
    <a-modal v-model:open="showDialog" :title="editingTemplate ? '编辑模板' : '新建模板'" width="600px" @ok="saveTemplate" :confirmLoading="saving">
      <a-form :model="templateForm" :label-col="{ span: 4 }" :wrapper-col="{ span: 20 }" :rules="formRules" ref="formRef">
        <a-form-item label="模板名称" name="name">
          <a-input v-model:value="templateForm.name" placeholder="输入模板名称" />
        </a-form-item>
        <a-form-item label="描述">
          <a-textarea v-model:value="templateForm.description" placeholder="输入描述（可选）" :rows="3" />
        </a-form-item>
        <a-form-item label="解析类型" name="type">
          <a-radio-group v-model:value="templateForm.type">
            <a-radio value="json">JSON</a-radio>
            <a-radio value="regex">正则表达式</a-radio>
            <a-radio value="grok">Grok</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item v-if="templateForm.type !== 'json'" label="解析模式" name="pattern">
          <a-textarea 
            v-model:value="templateForm.pattern" 
            :rows="3"
            :placeholder="patternPlaceholder"
          />
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="templateForm.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { message, Modal } from 'ant-design-vue'
import type { FormInstance } from 'ant-design-vue'
import type { Rule } from 'ant-design-vue/es/form'
import { logApi } from '@/services/logs'

interface ParseTemplate {
  id: number
  name: string
  description: string
  type: string
  pattern: string
  is_preset: boolean
  enabled: boolean
}

const props = defineProps<{
  modelValue?: ParseTemplate | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: ParseTemplate | null): void
}>()

const presetTemplates = ref<ParseTemplate[]>([])
const customTemplates = ref<ParseTemplate[]>([])
const selectedTemplate = ref<ParseTemplate | null>(null)
const testContent = ref('')
const testResult = ref<{ success: boolean; parsed?: any; error?: string } | null>(null)
const testing = ref(false)

const showDialog = ref(false)
const editingTemplate = ref<ParseTemplate | null>(null)
const saving = ref(false)
const formRef = ref<FormInstance>()

const templateForm = reactive({
  name: '',
  description: '',
  type: 'json',
  pattern: '',
  enabled: true
})

const formRules: Record<string, Rule[]> = {
  name: [{ required: true, message: '请输入模板名称', trigger: 'blur' }],
  type: [{ required: true, message: '请选择解析类型', trigger: 'change' }],
  pattern: [{ required: true, message: '请输入解析模式', trigger: 'blur' }]
}

const typeLabels: Record<string, string> = {
  json: 'JSON',
  regex: '正则',
  grok: 'Grok'
}

const patternPlaceholder = computed(() => {
  if (templateForm.type === 'regex') {
    return '例如: ^(?P<timestamp>\\S+) (?P<level>\\w+) (?P<message>.*)$'
  }
  if (templateForm.type === 'grok') {
    return '例如: %{TIMESTAMP_ISO8601:timestamp} %{LOGLEVEL:level} %{GREEDYDATA:message}'
  }
  return ''
})

const loadTemplates = async () => {
  try {
    const res = await logApi.getParseTemplates()
    const templates = res.data || []
    presetTemplates.value = templates.filter((t: ParseTemplate) => t.is_preset)
    customTemplates.value = templates.filter((t: ParseTemplate) => !t.is_preset)
  } catch (error) {
    message.error('加载模板失败')
  }
}

const selectTemplate = (template: ParseTemplate) => {
  selectedTemplate.value = template
  testResult.value = null
  emit('update:modelValue', template)
}

const testParse = async () => {
  if (!selectedTemplate.value || !testContent.value) {
    message.warning('请选择模板并输入测试内容')
    return
  }

  testing.value = true
  try {
    const res = await logApi.parseLogs({
      type: selectedTemplate.value.type,
      pattern: selectedTemplate.value.pattern,
      log_content: testContent.value
    })
    testResult.value = res.data
  } catch (error) {
    testResult.value = { success: false, error: '解析请求失败' }
  } finally {
    testing.value = false
  }
}

const showCreateDialog = () => {
  editingTemplate.value = null
  Object.assign(templateForm, {
    name: '',
    description: '',
    type: 'json',
    pattern: '',
    enabled: true
  })
  showDialog.value = true
}

const editTemplate = (template: ParseTemplate) => {
  editingTemplate.value = template
  Object.assign(templateForm, {
    name: template.name,
    description: template.description,
    type: template.type,
    pattern: template.pattern,
    enabled: template.enabled
  })
  showDialog.value = true
}

const saveTemplate = async () => {
  if (!formRef.value) return
  
  // JSON 类型不需要 pattern
  if (templateForm.type === 'json') {
    templateForm.pattern = ''
  }
  
  try {
    await formRef.value.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    if (editingTemplate.value) {
      await logApi.updateParseTemplate(editingTemplate.value.id, templateForm)
      message.success('更新成功')
    } else {
      await logApi.createParseTemplate(templateForm)
      message.success('创建成功')
    }
    showDialog.value = false
    loadTemplates()
  } catch (error) {
    message.error('保存失败')
  } finally {
    saving.value = false
  }
}

const deleteTemplate = async (template: ParseTemplate) => {
  try {
    await Modal.confirm({
      title: '提示',
      content: `确定删除模板 "${template.name}"？`,
      onOk: async () => {
        await logApi.deleteParseTemplate(template.id)
        message.success('删除成功')
        if (selectedTemplate.value?.id === template.id) {
          selectedTemplate.value = null
          emit('update:modelValue', null)
        }
        loadTemplates()
      }
    })
  } catch (error) {
    // Cancelled
  }
}

onMounted(() => {
  loadTemplates()
})
</script>

<style scoped>
.parse-config {
  padding: 15px;
}

.toolbar {
  margin-bottom: 20px;
}

.section {
  margin-bottom: 25px;
}

.section h4 {
  margin-bottom: 15px;
  color: rgba(0, 0, 0, 0.85);
}

.template-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 15px;
}

.template-card {
  padding: 15px;
  border: 1px solid #d9d9d9;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.template-card:hover {
  border-color: #1890ff;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.template-card.active {
  border-color: #1890ff;
  background: #e6f7ff;
}

.template-name {
  font-weight: 500;
  margin-bottom: 8px;
}

.template-type {
  margin-bottom: 8px;
}

.template-desc {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.template-actions {
  margin-top: 10px;
  display: flex;
  gap: 10px;
}

.test-result {
  margin-top: 15px;
}

.alert-title {
  font-weight: 500;
  margin-bottom: 5px;
}

.test-result pre {
  margin: 10px 0 0;
  padding: 10px;
  background: #fafafa;
  border-radius: 4px;
  font-size: 12px;
  overflow: auto;
}
</style>
