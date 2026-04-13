<template>
  <div class="notify-config">
    <a-card title="通知配置">
      <template #extra>
        <a-space>
          <a-button @click="showTemplateModal">
            <template #icon><FileTextOutlined /></template>
            模板管理
          </a-button>
          <a-button type="primary" @click="showAddModal">
            <template #icon><PlusOutlined /></template>
            添加通知
          </a-button>
        </a-space>
      </template>

      <a-table :dataSource="configs" :loading="loading" rowKey="id" :pagination="false">
        <a-table-column title="类型" dataIndex="type" :width="120">
          <template #default="{ record }">
            <a-tag :color="getTypeColor(record.type)">{{ getTypeLabel(record.type) }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="名称" dataIndex="name" :width="200" />
        <a-table-column title="Webhook URL" dataIndex="webhook_url" ellipsis />
        <a-table-column title="触发事件" dataIndex="events" :width="200">
          <template #default="{ record }">
            <a-tag v-for="e in record.events" :key="e" size="small">{{ getEventLabel(e) }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="模板" dataIndex="template_name" :width="120">
          <template #default="{ record }">
            <span v-if="record.template_id">{{ record.template_name || '自定义' }}</span>
            <span v-else class="text-gray">默认</span>
          </template>
        </a-table-column>
        <a-table-column title="状态" dataIndex="enabled" :width="80">
          <template #default="{ record }">
            <a-switch v-model:checked="record.enabled" size="small" @change="toggleEnabled(record)" />
          </template>
        </a-table-column>
        <a-table-column title="操作" :width="150">
          <template #default="{ record }">
            <a-space>
              <a-button type="link" size="small" @click="previewNotify(record)">预览</a-button>
              <a-button type="link" size="small" @click="editConfig(record)">编辑</a-button>
              <a-popconfirm title="确定删除?" @confirm="deleteConfig(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </a-table-column>
      </a-table>
    </a-card>

    <!-- 添加/编辑弹窗 -->
    <a-modal v-model:open="modalVisible" :title="isEdit ? '编辑通知' : '添加通知'" width="700px" @ok="handleSave">
      <a-form :model="form" :label-col="{ span: 5 }" :wrapper-col="{ span: 19 }">
        <a-form-item label="通知类型" required>
          <a-select v-model:value="form.type" @change="onTypeChange">
            <a-select-option value="feishu">飞书</a-select-option>
            <a-select-option value="dingtalk">钉钉</a-select-option>
            <a-select-option value="wechat">企业微信</a-select-option>
            <a-select-option value="webhook">自定义 Webhook</a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="名称" required>
          <a-input v-model:value="form.name" placeholder="如: 研发群通知" />
        </a-form-item>

        <a-form-item label="Webhook URL" required>
          <a-input v-model:value="form.webhook_url" placeholder="https://..." />
          <div class="hint" v-if="form.type === 'feishu'">飞书机器人 Webhook 地址</div>
          <div class="hint" v-if="form.type === 'dingtalk'">钉钉机器人 Webhook 地址</div>
          <div class="hint" v-if="form.type === 'wechat'">企业微信机器人 Webhook 地址</div>
        </a-form-item>

        <a-form-item label="签名密钥" v-if="form.type === 'dingtalk'">
          <a-input-password v-model:value="form.secret" placeholder="钉钉加签密钥 (可选)" />
        </a-form-item>

        <a-form-item label="触发事件" required>
          <a-checkbox-group v-model:value="form.events">
            <a-checkbox value="success">构建成功</a-checkbox>
            <a-checkbox value="failed">构建失败</a-checkbox>
            <a-checkbox value="cancelled">构建取消</a-checkbox>
            <a-checkbox value="started">构建开始</a-checkbox>
          </a-checkbox-group>
        </a-form-item>

        <a-form-item label="@用户" v-if="form.type !== 'webhook'">
          <a-select v-model:value="form.at_users" mode="tags" placeholder="输入手机号或用户ID">
          </a-select>
        </a-form-item>

        <a-form-item label="@所有人" v-if="form.type !== 'webhook'">
          <a-switch v-model:checked="form.at_all" />
        </a-form-item>

        <a-form-item label="消息模板">
          <a-radio-group v-model:value="form.template_type" @change="onTemplateTypeChange">
            <a-radio value="default">使用默认模板</a-radio>
            <a-radio value="system">选择系统模板</a-radio>
            <a-radio value="custom">自定义模板</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item label="选择模板" v-if="form.template_type === 'system'">
          <a-select v-model:value="form.template_id" placeholder="选择模板" @change="onTemplateSelect">
            <a-select-option v-for="t in templates" :key="t.id" :value="t.id">{{ t.name }}</a-select-option>
          </a-select>
        </a-form-item>

        <a-form-item label="模板内容" v-if="form.template_type === 'custom'">
          <a-textarea v-model:value="form.template" :rows="6" placeholder="支持变量: {{.PipelineName}}, {{.Status}}, {{.TriggerBy}}, {{.Duration}}" />
          <div class="template-vars">
            <span class="label">可用变量:</span>
            <a-tag v-for="v in availableVars" :key="v" size="small" @click="insertVar(v)">{{ v }}</a-tag>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 模板管理弹窗 -->
    <a-modal v-model:open="templateModalVisible" title="通知模板管理" width="900px" :footer="null">
      <a-tabs v-model:activeKey="templateTab">
        <a-tab-pane key="list" tab="模板列表">
          <a-table :dataSource="allTemplates" :loading="templateLoading" rowKey="id" size="small">
            <a-table-column title="名称" dataIndex="name" :width="150" />
            <a-table-column title="类型" dataIndex="type" :width="100">
              <template #default="{ record }">
                <a-tag :color="getTypeColor(record.type)">{{ getTypeLabel(record.type) }}</a-tag>
              </template>
            </a-table-column>
            <a-table-column title="分类" dataIndex="category" :width="100" />
            <a-table-column title="描述" dataIndex="description" ellipsis />
            <a-table-column title="默认" dataIndex="is_default" :width="60">
              <template #default="{ record }">
                <a-tag v-if="record.is_default" color="green">是</a-tag>
              </template>
            </a-table-column>
            <a-table-column title="操作" :width="150">
              <template #default="{ record }">
                <a-space>
                  <a-button type="link" size="small" @click="previewTemplate(record)">预览</a-button>
                  <a-button type="link" size="small" @click="editTemplate(record)" :disabled="record.is_system">编辑</a-button>
                  <a-popconfirm title="确定删除?" @confirm="deleteTemplate(record.id)" :disabled="record.is_system">
                    <a-button type="link" size="small" danger :disabled="record.is_system">删除</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </a-table-column>
          </a-table>
        </a-tab-pane>
        <a-tab-pane key="create" tab="创建模板">
          <a-form :model="templateForm" :label-col="{ span: 4 }" :wrapper-col="{ span: 20 }">
            <a-form-item label="模板名称" required>
              <a-input v-model:value="templateForm.name" placeholder="输入模板名称" />
            </a-form-item>
            <a-form-item label="通知类型" required>
              <a-select v-model:value="templateForm.type">
                <a-select-option value="feishu">飞书</a-select-option>
                <a-select-option value="dingtalk">钉钉</a-select-option>
                <a-select-option value="wechat">企业微信</a-select-option>
                <a-select-option value="webhook">Webhook</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item label="分类">
              <a-select v-model:value="templateForm.category">
                <a-select-option value="pipeline">流水线</a-select-option>
                <a-select-option value="deploy">部署</a-select-option>
                <a-select-option value="alert">告警</a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item label="模板内容" required>
              <a-textarea v-model:value="templateForm.content" :rows="8" />
            </a-form-item>
            <a-form-item label="描述">
              <a-input v-model:value="templateForm.description" />
            </a-form-item>
            <a-form-item label="设为默认">
              <a-switch v-model:checked="templateForm.is_default" />
            </a-form-item>
            <a-form-item :wrapper-col="{ offset: 4 }">
              <a-space>
                <a-button type="primary" @click="saveTemplate">保存模板</a-button>
                <a-button @click="previewTemplateContent">预览效果</a-button>
              </a-space>
            </a-form-item>
          </a-form>
        </a-tab-pane>
      </a-tabs>
    </a-modal>

    <!-- 预览弹窗 -->
    <a-modal v-model:open="previewVisible" title="通知预览" width="600px" :footer="null">
      <div class="preview-content" v-html="previewContent"></div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, FileTextOutlined } from '@ant-design/icons-vue'

interface NotifyConfig {
  id: number
  type: string
  name: string
  webhook_url: string
  secret: string
  events: string[]
  at_users: string[]
  at_all: boolean
  template: string
  template_id: number | null
  template_name: string
  template_type: string
  enabled: boolean
}

interface NotifyTemplate {
  id: number
  name: string
  type: string
  category: string
  content: string
  description: string
  is_default: boolean
  is_system: boolean
}

const props = defineProps<{ pipelineId?: number }>()

const loading = ref(false)
const configs = ref<NotifyConfig[]>([])
const modalVisible = ref(false)
const isEdit = ref(false)
const templates = ref<NotifyTemplate[]>([])
const allTemplates = ref<NotifyTemplate[]>([])
const templateModalVisible = ref(false)
const templateLoading = ref(false)
const templateTab = ref('list')
const previewVisible = ref(false)
const previewContent = ref('')

const availableVars = [
  '{{.PipelineName}}', '{{.RunID}}', '{{.Status}}', '{{.TriggerBy}}',
  '{{.GitBranch}}', '{{.GitCommit}}', '{{.Duration}}', '{{.URL}}'
]

const defaultForm = (): Partial<NotifyConfig> => ({
  type: 'feishu',
  name: '',
  webhook_url: '',
  secret: '',
  events: ['success', 'failed'],
  at_users: [],
  at_all: false,
  template: '',
  template_id: null,
  template_type: 'default',
  enabled: true
})

const form = ref<Partial<NotifyConfig>>(defaultForm())

const defaultTemplateForm = () => ({
  name: '',
  type: 'feishu',
  category: 'pipeline',
  content: '',
  description: '',
  is_default: false
})

const templateForm = ref(defaultTemplateForm())

const loadConfigs = async () => {
  loading.value = true
  try {
    // TODO: 调用 API 获取通知配置
    configs.value = []
  } finally {
    loading.value = false
  }
}

const loadTemplates = async () => {
  templateLoading.value = true
  try {
    // TODO: 调用 API 获取模板列表
    allTemplates.value = []
    templates.value = allTemplates.value.filter(t => t.type === form.value.type)
  } finally {
    templateLoading.value = false
  }
}

const showAddModal = () => {
  isEdit.value = false
  form.value = defaultForm()
  modalVisible.value = true
}

const showTemplateModal = () => {
  templateModalVisible.value = true
  loadTemplates()
}

const editConfig = (record: NotifyConfig) => {
  isEdit.value = true
  form.value = { ...record }
  modalVisible.value = true
}

const handleSave = async () => {
  if (!form.value.name || !form.value.webhook_url || !form.value.events?.length) {
    message.warning('请填写完整信息')
    return
  }
  try {
    // TODO: 调用 API 保存
    message.success(isEdit.value ? '更新成功' : '添加成功')
    modalVisible.value = false
    loadConfigs()
  } catch {
    message.error('保存失败')
  }
}

const deleteConfig = async (id: number) => {
  try {
    // TODO: 调用 API 删除
    message.success('删除成功')
    loadConfigs()
  } catch {
    message.error('删除失败')
  }
}

const toggleEnabled = async (record: NotifyConfig) => {
  try {
    // TODO: 调用 API 更新状态
    message.success(record.enabled ? '已启用' : '已禁用')
  } catch {
    message.error('操作失败')
  }
}

const onTypeChange = () => {
  form.value.secret = ''
  templates.value = allTemplates.value.filter(t => t.type === form.value.type)
}

const onTemplateTypeChange = () => {
  if (form.value.template_type === 'default') {
    form.value.template_id = null
    form.value.template = ''
  }
}

const onTemplateSelect = (id: number) => {
  const t = templates.value.find(t => t.id === id)
  if (t) {
    form.value.template = t.content
  }
}

const insertVar = (v: string) => {
  form.value.template = (form.value.template || '') + v
}

const previewNotify = (record: NotifyConfig) => {
  previewContent.value = renderPreview(record.template || getDefaultTemplate(record.type))
  previewVisible.value = true
}

const previewTemplate = (record: NotifyTemplate) => {
  previewContent.value = renderPreview(record.content)
  previewVisible.value = true
}

const previewTemplateContent = () => {
  previewContent.value = renderPreview(templateForm.value.content)
  previewVisible.value = true
}

const renderPreview = (template: string) => {
  const sampleData: Record<string, string> = {
    PipelineName: 'frontend-build',
    RunID: '123',
    Status: 'success',
    TriggerBy: 'admin',
    GitBranch: 'main',
    GitCommit: 'abc123def',
    Duration: '120',
    URL: 'https://devops.example.com/pipeline/123'
  }
  let result = template
  for (const [key, value] of Object.entries(sampleData)) {
    result = result.replace(new RegExp(`\\{\\{\\.${key}\\}\\}`, 'g'), value)
  }
  return result.replace(/\n/g, '<br>')
}

const getDefaultTemplate = (type: string) => {
  const templates: Record<string, string> = {
    feishu: '**流水线**: {{.PipelineName}}\n**状态**: {{.Status}}\n**触发人**: {{.TriggerBy}}',
    dingtalk: '### 流水线 {{.PipelineName}}\n- 状态: {{.Status}}\n- 触发人: {{.TriggerBy}}',
    wechat: '## 流水线 {{.PipelineName}}\n> 状态: {{.Status}}\n> 触发人: {{.TriggerBy}}',
    webhook: '{"pipeline": "{{.PipelineName}}", "status": "{{.Status}}"}'
  }
  return templates[type] || ''
}

const editTemplate = (record: NotifyTemplate) => {
  templateForm.value = { ...record }
  templateTab.value = 'create'
}

const saveTemplate = async () => {
  if (!templateForm.value.name || !templateForm.value.content) {
    message.warning('请填写模板名称和内容')
    return
  }
  try {
    // TODO: 调用 API 保存模板
    message.success('模板保存成功')
    templateForm.value = defaultTemplateForm()
    templateTab.value = 'list'
    loadTemplates()
  } catch {
    message.error('保存失败')
  }
}

const deleteTemplate = async (id: number) => {
  try {
    // TODO: 调用 API 删除模板
    message.success('删除成功')
    loadTemplates()
  } catch {
    message.error('删除失败')
  }
}

const getTypeColor = (type: string) => ({ feishu: 'blue', dingtalk: 'orange', wechat: 'green', webhook: 'purple' }[type] || 'default')
const getTypeLabel = (type: string) => ({ feishu: '飞书', dingtalk: '钉钉', wechat: '企业微信', webhook: 'Webhook' }[type] || type)
const getEventLabel = (event: string) => ({ success: '成功', failed: '失败', cancelled: '取消', started: '开始' }[event] || event)

onMounted(() => {
  loadConfigs()
  loadTemplates()
})
</script>

<style scoped>
.hint { color: #999; font-size: 12px; margin-top: 4px; }
.text-gray { color: #999; }
.template-vars { margin-top: 8px; }
.template-vars .label { color: #666; margin-right: 8px; }
.template-vars .ant-tag { cursor: pointer; margin: 2px; }
.template-vars .ant-tag:hover { background: #e6f7ff; }
.preview-content { 
  background: #f5f5f5; 
  padding: 16px; 
  border-radius: 4px; 
  font-family: monospace;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
