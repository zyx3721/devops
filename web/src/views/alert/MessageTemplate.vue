<template>
  <div class="message-template">
    <a-card :bordered="false">
      <template #extra>
        <a-button type="primary" @click="showModal()">
          <template #icon><PlusOutlined /></template>
          创建模板
        </a-button>
      </template>
      <a-table :columns="columns" :data-source="list" :loading="loading" row-key="id" :pagination="pagination" @change="onTableChange">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <a-tag :color="getTypeColor(record.template_type)">{{ getTypeLabel(record.template_type) }}</a-tag>
          </template>
          <template v-if="column.key === 'is_active'">
            <a-tag :color="record.is_active ? 'green' : 'red'">{{ record.is_active ? '启用' : '禁用' }}</a-tag>
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showModal(record)">编辑</a-button>
              <a-popconfirm title="确定删除？" @confirm="deleteItem(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <a-modal 
      v-model:open="modalVisible" 
      :title="editingId ? '编辑模板' : '创建模板'" 
      @ok="save" 
      :confirm-loading="saving" 
      width="1000px"
      style="top: 20px"
    >
      <div class="template-editor-layout">
        <!-- 左侧：表单 -->
        <div class="editor-form">
          <a-form :model="form" layout="vertical">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="模板名称" required>
                  <a-input v-model:value="form.name" placeholder="如：COST_BUDGET_WARNING" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="模板类型" required>
                  <a-select v-model:value="form.template_type">
                    <a-select-option value="text">纯文本 (Text)</a-select-option>
                    <a-select-option value="markdown">Markdown</a-select-option>
                    <a-select-option value="card">飞书卡片 (JSON)</a-select-option>
                    <a-select-option value="json">通用 JSON</a-select-option>
                  </a-select>
                </a-form-item>
              </a-col>
            </a-row>

            <a-form-item label="模板标题">
              <a-input v-model:value="form.title" placeholder="可选，用于卡片标题" />
            </a-form-item>

            <a-form-item label="可用变量说明">
              <a-input v-model:value="form.variables" placeholder='如：["Project", "Cost"]，仅作为备注' />
            </a-form-item>

            <a-form-item label="模板内容 (Go Template 语法)" required>
              <div class="editor-container">
                <a-textarea 
                  v-model:value="form.content" 
                  :rows="12" 
                  placeholder="在此输入模板内容..." 
                  style="font-family: monospace; font-size: 13px;"
                />
              </div>
            </a-form-item>
             
            <a-form-item label="描述">
              <a-textarea v-model:value="form.description" :rows="2" />
            </a-form-item>
            
            <a-form-item>
              <a-checkbox v-model:checked="form.is_active">启用</a-checkbox>
            </a-form-item>
          </a-form>
        </div>

        <!-- 右侧：预览与帮助 -->
        <div class="editor-sidebar">
          <a-tabs v-model:activeKey="sidebarTab">
            <a-tab-pane key="preview" tab="实时预览">
              <div class="preview-section">
                <div class="section-title">测试数据 (JSON)</div>
                <a-textarea 
                  v-model:value="testDataJson" 
                  :rows="6" 
                  placeholder='{"Project": "DevOps", "Cost": 100}'
                  style="font-family: monospace; margin-bottom: 10px;"
                />
                <a-button type="dashed" block @click="preview" :loading="previewLoading">刷新预览</a-button>
                
                <a-divider style="margin: 12px 0" />
                
                <div class="section-title">渲染结果</div>
                <div class="preview-result">
                  <pre v-if="previewResult">{{ previewResult }}</pre>
                  <div v-else class="empty-preview">点击刷新预览查看结果</div>
                </div>
                
                <div v-if="jsonError" class="json-error">
                  <CloseCircleOutlined /> JSON 格式错误: {{ jsonError }}
                </div>
                <div v-else-if="isValidJson && previewResult" class="json-success">
                  <CheckCircleOutlined /> JSON 格式有效
                </div>
              </div>
            </a-tab-pane>
            <a-tab-pane key="help" tab="语法指南">
              <div class="help-content">
                <h4>基础语法</h4>
                <ul>
                  <li><code v-pre>{{ .VariableName }}</code> - 输出变量</li>
                  <li><code v-pre>{{ if .Condition }}...{{ end }}</code> - 条件判断</li>
                  <li><code v-pre>{{ range .List }}...{{ end }}</code> - 循环列表</li>
                </ul>
                <h4>内置函数</h4>
                <ul>
                  <li><code>len</code> - 获取长度</li>
                  <li><code>printf</code> - 格式化输出</li>
                  <li><code>date</code> - 日期格式化 (需后端支持)</li>
                </ul>
                <h4>飞书卡片示例</h4>
                <pre class="code-snippet" v-pre>
{
  "header": {
    "title": {
      "content": "{{.Title}}",
      "tag": "plain_text"
    }
  },
  "elements": [
    {
      "tag": "div",
      "text": {
        "content": "**项目**: {{.Project}}",
        "tag": "lark_md"
      }
    }
  ]
}
                </pre>
              </div>
            </a-tab-pane>
          </a-tabs>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons-vue'
import request from '@/services/api' // 临时直接引用 request，建议封装到 api/template.ts

interface MessageTemplate {
  id: number
  name: string
  template_type: string
  title: string
  content: string
  variables: string
  description?: string
  is_active: boolean
}

// 简单的 API 封装 (如果 service/template.ts 还没更新完整)
const api = {
  list: (params: any) => request.get('/notification/templates', { params }),
  create: (data: any) => request.post('/notification/templates', data),
  update: (id: number, data: any) => request.put(`/notification/templates/${id}`, data),
  delete: (id: number) => request.delete(`/notification/templates/${id}`),
  preview: (data: any) => request.post('/notification/templates/preview', data)
}

const loading = ref(false)
const saving = ref(false)
const modalVisible = ref(false)
const editingId = ref<number>()
const list = ref<MessageTemplate[]>([])
const pagination = reactive({ current: 1, pageSize: 10, total: 0 })

const form = reactive<Partial<MessageTemplate>>({
  name: '',
  template_type: 'card',
  title: '',
  content: '',
  variables: '',
  description: '',
  is_active: true
})

// 预览相关
const sidebarTab = ref('preview')
const testDataJson = ref('{\n  "Project": "Demo Project",\n  "CurrentCost": 1200,\n  "Budget": 1000,\n  "UsageRate": 120,\n  "Message": "Cost exceeded budget"\n}')
const previewResult = ref('')
const previewLoading = ref(false)
const isValidJson = ref(false)
const jsonError = ref('')

const columns = [
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '类型', key: 'type', width: 100 },
  { title: '标题', dataIndex: 'title', ellipsis: true },
  { title: '描述', dataIndex: 'description', ellipsis: true },
  { title: '状态', key: 'is_active', width: 80 },
  { title: '操作', key: 'action', width: 150 }
]

const typeLabels: Record<string, string> = { text: '文本', markdown: 'Markdown', card: '飞书卡片', json: 'JSON' }
const typeColors: Record<string, string> = { text: 'default', markdown: 'blue', card: 'orange', json: 'purple' }
const getTypeLabel = (type: string) => typeLabels[type] || type
const getTypeColor = (type: string) => typeColors[type] || 'default'

const fetchData = async () => {
  loading.value = true
  try {
    const res = await api.list({ page: pagination.current, page_size: pagination.pageSize })
    if (res.code === 0 && res.data) {
      list.value = res.data.list || []
      pagination.total = res.data.total
    }
  } finally { loading.value = false }
}

const onTableChange = (pag: any) => { pagination.current = pag.current; fetchData() }

const showModal = (record?: MessageTemplate) => {
  if (record) {
    editingId.value = record.id
    Object.assign(form, record)
  } else {
    editingId.value = undefined
    Object.assign(form, {
      name: '',
      template_type: 'card',
      title: '',
      content: '',
      variables: '',
      description: '',
      is_active: true
    })
  }
  previewResult.value = ''
  jsonError.value = ''
  isValidJson.value = false
  modalVisible.value = true
}

const save = async () => {
  if (!form.name || !form.content) { message.error('请填写必填项'); return }
  saving.value = true
  try {
    const res = editingId.value ? await api.update(editingId.value, form) : await api.create(form)
    if (res.code === 0) {
      message.success('保存成功')
      modalVisible.value = false
      fetchData()
    } else {
      message.error(res.message || '保存失败')
    }
  } catch (e: any) {
    message.error(e.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const deleteItem = async (id: number) => {
  try {
    const res = await api.delete(id)
    if (res.code === 0) { message.success('已删除'); fetchData() }
  } catch (e: any) { message.error(e.message || '操作失败') }
}

const preview = async () => {
  if (!form.content) return
  let dataObj = {}
  try {
    dataObj = JSON.parse(testDataJson.value)
  } catch (e) {
    message.error('测试数据 JSON 格式错误')
    return
  }

  previewLoading.value = true
  try {
    const res = await api.preview({
      content: form.content,
      data: dataObj
    })
    if (res.code === 0 && res.data) {
      previewResult.value = res.data.content
      isValidJson.value = res.data.valid_json
      jsonError.value = res.data.json_error || ''
      if (res.data.json_error) {
        message.warning('渲染结果不是有效的 JSON')
      }
    } else {
      message.error(res.message || '预览失败')
    }
  } catch (e: any) {
    message.error(e.message || '预览请求失败')
  } finally {
    previewLoading.value = false
  }
}

onMounted(() => { fetchData() })
</script>

<style scoped>
.template-editor-layout {
  display: flex;
  height: 600px;
  gap: 16px;
}
.editor-form {
  flex: 3;
  overflow-y: auto;
  padding-right: 8px;
}
.editor-sidebar {
  flex: 2;
  border-left: 1px solid #f0f0f0;
  padding-left: 16px;
  display: flex;
  flex-direction: column;
}

.preview-result {
  background: #f5f5f5;
  padding: 12px;
  border-radius: 4px;
  min-height: 200px;
  max-height: 350px;
  overflow: auto;
  font-family: monospace;
  font-size: 12px;
  white-space: pre-wrap;
  word-break: break-all;
}
.empty-preview {
  color: #999;
  text-align: center;
  margin-top: 80px;
}

.section-title {
  font-weight: bold;
  margin-bottom: 8px;
  color: #333;
}

.json-error {
  margin-top: 8px;
  color: #ff4d4f;
  font-size: 12px;
}
.json-success {
  margin-top: 8px;
  color: #52c41a;
  font-size: 12px;
}

.help-content {
  font-size: 13px;
  color: #666;
  overflow-y: auto;
  max-height: 550px;
}
.help-content h4 {
  margin: 12px 0 8px;
  color: #333;
  font-weight: bold;
}
.help-content ul {
  padding-left: 20px;
  margin-bottom: 12px;
}
.help-content li {
  margin-bottom: 4px;
}
.help-content code {
  background: #f5f5f5;
  padding: 2px 4px;
  border-radius: 2px;
  color: #c41d7f;
}
.code-snippet {
  background: #f5f5f5;
  padding: 8px;
  border-radius: 4px;
  font-size: 12px;
  color: #333;
}
</style>
