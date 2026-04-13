<template>
  <div class="pipeline-editor">
    <a-page-header
      :title="isEdit ? '编辑流水线' : '创建流水线'"
      @back="goBack"
    >
      <template #extra>
        <a-space>
          <a-button v-if="!isEdit" @click="showTemplateSelector"><FileAddOutlined /> 从模板创建</a-button>
          <a-button @click="goBack">取消</a-button>
          <a-button type="primary" :loading="saving" @click="savePipeline">
            保存
          </a-button>
        </a-space>
      </template>
    </a-page-header>

    <!-- 模板选择弹窗 -->
    <a-modal v-model:open="templateModalVisible" title="选择模板" width="900px" :footer="null">
      <template-selector @select="applyTemplate" @cancel="templateModalVisible = false" />
    </a-modal>

    <a-row :gutter="16">
      <!-- 基本信息 -->
      <a-col :span="24">
        <a-card title="基本信息" size="small" :bordered="false">
          <a-form :model="form" :label-col="{ span: 3 }" :wrapper-col="{ span: 8 }">
            <a-form-item label="流水线名称" required>
              <a-input v-model:value="form.name" placeholder="请输入流水线名称" />
            </a-form-item>
            <a-form-item label="描述">
              <a-textarea v-model:value="form.description" placeholder="流水线描述" :rows="2" />
            </a-form-item>
            <a-form-item label="Git 仓库">
              <a-select v-model:value="form.git_repo_id" placeholder="选择 Git 仓库" allowClear style="width: 300px">
                <a-select-option v-for="repo in gitRepos" :key="repo.id" :value="repo.id">
                  {{ repo.name }} ({{ repo.url }})
                </a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item label="Git 分支" v-if="form.git_repo_id">
              <a-input v-model:value="form.git_branch" placeholder="main" style="width: 200px" />
            </a-form-item>
            <a-form-item label="构建集群">
              <a-select v-model:value="form.build_cluster_id" placeholder="选择 K8s 集群" allowClear style="width: 300px">
                <a-select-option v-for="cluster in clusters" :key="cluster.id" :value="cluster.id">
                  {{ cluster.name }}
                </a-select-option>
              </a-select>
            </a-form-item>
            <a-form-item label="构建命名空间" v-if="form.build_cluster_id">
              <a-input v-model:value="form.build_namespace" placeholder="devops-build" style="width: 200px" />
            </a-form-item>
          </a-form>
        </a-card>
      </a-col>

      <!-- 触发器配置 -->
      <a-col :span="24" style="margin-top: 16px">
        <a-card title="触发器配置" size="small" :bordered="false">
          <a-form :model="form.trigger_config" :label-col="{ span: 3 }" :wrapper-col="{ span: 18 }">
            <a-form-item label="手动触发">
              <a-switch v-model:checked="form.trigger_config.manual" />
              <span class="trigger-hint">允许手动触发流水线执行</span>
            </a-form-item>
            <a-form-item label="定时触发">
              <a-space direction="vertical" style="width: 100%">
                <a-switch v-model:checked="scheduledEnabled" />
                <template v-if="scheduledEnabled">
                  <a-input
                    v-model:value="form.trigger_config.scheduled.cron"
                    placeholder="Cron 表达式，如: 0 0 2 * * * (每天凌晨2点)"
                    style="width: 400px"
                  />
                  <div class="cron-presets">
                    <span>快捷设置：</span>
                    <a-button size="small" type="link" @click="setCron('0 0 2 * * *')">每天凌晨2点</a-button>
                    <a-button size="small" type="link" @click="setCron('0 0 */6 * * *')">每6小时</a-button>
                    <a-button size="small" type="link" @click="setCron('0 30 8 * * 1-5')">工作日8:30</a-button>
                  </div>
                </template>
              </a-space>
            </a-form-item>
            <a-form-item label="Webhook 触发">
              <a-space direction="vertical" style="width: 100%">
                <a-switch v-model:checked="webhookEnabled" />
                <template v-if="webhookEnabled">
                  <a-input
                    v-model:value="form.trigger_config.webhook.url"
                    placeholder="Webhook URL (保存后自动生成)"
                    disabled
                    style="width: 400px"
                  >
                    <template #addonAfter>
                      <a-button type="link" size="small" @click="copyWebhookUrl" :disabled="!form.trigger_config.webhook.url">
                        复制
                      </a-button>
                    </template>
                  </a-input>
                  <a-input
                    v-model:value="form.trigger_config.webhook.secret"
                    placeholder="Webhook Secret (可选，用于签名验证)"
                    style="width: 400px"
                  />
                  <a-select
                    v-model:value="form.trigger_config.webhook.branch_filter"
                    mode="tags"
                    placeholder="分支过滤 (留空匹配所有分支)"
                    style="width: 400px"
                  />
                </template>
              </a-space>
            </a-form-item>
          </a-form>
        </a-card>
      </a-col>

      <!-- 编辑模式切换 -->
      <a-col :span="24" style="margin-top: 16px">
        <a-card :bordered="false">
          <template #title>
            <a-radio-group v-model:value="editMode" button-style="solid">
              <a-radio-button value="visual">可视化编排</a-radio-button>
              <a-radio-button value="yaml">YAML 配置</a-radio-button>
            </a-radio-group>
          </template>

          <!-- 可视化编排 -->
          <div v-if="editMode === 'visual'" class="visual-editor">
            <div class="stages-container">
              <div v-for="(stage, stageIndex) in form.stages" :key="stageIndex" class="stage-card">
                <div class="stage-header">
                  <a-input v-model:value="stage.name" placeholder="阶段名称" style="width: 150px" />
                  <a-space>
                    <a-button type="text" size="small" @click="addStep(stageIndex)">
                      <PlusOutlined /> 添加步骤
                    </a-button>
                    <a-button type="text" size="small" danger @click="removeStage(stageIndex)">
                      <DeleteOutlined />
                    </a-button>
                  </a-space>
                </div>

                <div class="steps-container">
                  <div v-for="(step, stepIndex) in stage.steps" :key="stepIndex" class="step-card">
                    <div class="step-header">
                      <a-input v-model:value="step.name" placeholder="步骤名称" size="small" style="width: 120px" />
                      <a-button type="text" size="small" danger @click="removeStep(stageIndex, stepIndex)">
                        <DeleteOutlined />
                      </a-button>
                    </div>
                    <div class="step-content">
                      <a-form-item label="镜像" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
                        <a-auto-complete
                          v-model:value="step.image"
                          :options="imageOptions"
                          placeholder="node:18-alpine"
                          size="small"
                        />
                      </a-form-item>
                      <a-form-item label="命令" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
                        <a-textarea
                          v-model:value="step.commandsText"
                          placeholder="npm install&#10;npm run build"
                          :rows="3"
                          size="small"
                          @change="parseCommands(step)"
                        />
                      </a-form-item>
                      <a-collapse ghost size="small">
                        <a-collapse-panel key="advanced" header="高级配置">
                          <a-form-item label="工作目录" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
                            <a-input v-model:value="step.work_dir" placeholder="/workspace" size="small" />
                          </a-form-item>
                          <a-form-item label="超时(秒)" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
                            <a-input-number v-model:value="step.timeout" :min="0" size="small" />
                          </a-form-item>
                        </a-collapse-panel>
                      </a-collapse>
                    </div>
                  </div>

                  <div v-if="stage.steps.length === 0" class="empty-steps">
                    <a-button type="dashed" block @click="addStep(stageIndex)">
                      <PlusOutlined /> 添加步骤
                    </a-button>
                  </div>
                </div>
              </div>

              <div class="add-stage">
                <a-button type="dashed" block @click="addStage">
                  <PlusOutlined /> 添加阶段
                </a-button>
              </div>
            </div>
          </div>

          <!-- YAML 编辑 -->
          <div v-else class="yaml-editor">
            <a-textarea
              v-model:value="yamlContent"
              :rows="25"
              class="yaml-textarea"
              @change="parseYaml"
            />
            <div v-if="yamlError" class="yaml-error">
              <a-alert :message="yamlError" type="error" show-icon />
            </div>
          </div>
        </a-card>
      </a-col>

      <!-- 环境变量 -->
      <a-col :span="24" style="margin-top: 16px">
        <a-card title="环境变量" size="small" :bordered="false">
          <template #extra>
            <a-button type="link" size="small" @click="addVariable">
              <PlusOutlined /> 添加变量
            </a-button>
          </template>
          <a-table :columns="varColumns" :data-source="form.variables" :pagination="false" size="small">
            <template #bodyCell="{ column, record, index }">
              <template v-if="column.key === 'name'">
                <a-input v-model:value="record.name" placeholder="变量名" size="small" />
              </template>
              <template v-else-if="column.key === 'value'">
                <a-input-password v-if="record.is_secret" v-model:value="record.value" placeholder="变量值" size="small" />
                <a-input v-else v-model:value="record.value" placeholder="变量值" size="small" />
              </template>
              <template v-else-if="column.key === 'is_secret'">
                <a-switch v-model:checked="record.is_secret" size="small" />
              </template>
              <template v-else-if="column.key === 'action'">
                <a-button type="link" size="small" danger @click="removeVariable(index)">删除</a-button>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { PlusOutlined, DeleteOutlined, FileAddOutlined } from '@ant-design/icons-vue'
import { pipelineApi, gitRepoApi } from '@/services/pipeline'
import request from '@/services/api'
import yaml from 'js-yaml'
import TemplateSelector from '@/components/pipeline/TemplateSelector.vue'

interface Step {
  id: string
  name: string
  image: string
  commands: string[]
  commandsText: string
  work_dir: string
  timeout: number
  env: Record<string, string>
}

interface Stage {
  id: string
  name: string
  steps: Step[]
  needs: string[]
}

interface Variable {
  name: string
  value: string
  is_secret: boolean
}

interface GitRepo {
  id: number
  name: string
  url: string
}

interface Cluster {
  id: number
  name: string
}

interface TriggerConfig {
  manual: boolean
  scheduled: {
    enabled: boolean
    cron: string
    timezone: string
  }
  webhook: {
    enabled: boolean
    secret: string
    branch_filter: string[]
    url: string
  }
}

const route = useRoute()
const router = useRouter()

const isEdit = computed(() => !!route.params.id)
const pipelineId = computed(() => Number(route.params.id) || 0)

const saving = ref(false)
const editMode = ref<'visual' | 'yaml'>('visual')
const yamlContent = ref('')
const yamlError = ref('')
const gitRepos = ref<GitRepo[]>([])
const clusters = ref<Cluster[]>([])
const templateModalVisible = ref(false)

// 触发器配置的计算属性
const scheduledEnabled = computed({
  get: () => form.trigger_config.scheduled?.enabled || false,
  set: (val) => {
    if (!form.trigger_config.scheduled) {
      form.trigger_config.scheduled = { enabled: false, cron: '', timezone: 'Asia/Shanghai' }
    }
    form.trigger_config.scheduled.enabled = val
  }
})

const webhookEnabled = computed({
  get: () => form.trigger_config.webhook?.enabled || false,
  set: (val) => {
    if (!form.trigger_config.webhook) {
      form.trigger_config.webhook = { enabled: false, secret: '', branch_filter: [], url: '' }
    }
    form.trigger_config.webhook.enabled = val
  }
})

const form = reactive({
  name: '',
  description: '',
  git_repo_id: undefined as number | undefined,
  git_branch: 'main',
  build_cluster_id: undefined as number | undefined,
  build_namespace: 'devops-build',
  stages: [] as Stage[],
  variables: [] as Variable[],
  trigger_config: {
    manual: true,
    scheduled: {
      enabled: false,
      cron: '',
      timezone: 'Asia/Shanghai'
    },
    webhook: {
      enabled: false,
      secret: '',
      branch_filter: [],
      url: ''
    }
  } as TriggerConfig
})

const imageOptions = [
  { value: 'node:18-alpine' },
  { value: 'node:20-alpine' },
  { value: 'golang:1.21-alpine' },
  { value: 'golang:1.22-alpine' },
  { value: 'maven:3.9-eclipse-temurin-17' },
  { value: 'python:3.11-alpine' },
  { value: 'alpine/git:latest' },
  { value: 'gcr.io/kaniko-project/executor:latest' },
  { value: 'bitnami/kubectl:latest' },
  { value: 'docker:latest' }
]

const varColumns = [
  { title: '变量名', key: 'name', width: 200 },
  { title: '变量值', key: 'value' },
  { title: '敏感', key: 'is_secret', width: 80 },
  { title: '操作', key: 'action', width: 80 }
]

const generateId = () => Math.random().toString(36).substring(2, 10)

const addStage = () => {
  form.stages.push({
    id: generateId(),
    name: `阶段 ${form.stages.length + 1}`,
    steps: [],
    needs: []
  })
}

const removeStage = (index: number) => {
  form.stages.splice(index, 1)
}

const addStep = (stageIndex: number) => {
  form.stages[stageIndex].steps.push({
    id: generateId(),
    name: `步骤 ${form.stages[stageIndex].steps.length + 1}`,
    image: '',
    commands: [],
    commandsText: '',
    work_dir: '/workspace',
    timeout: 600,
    env: {}
  })
}

const removeStep = (stageIndex: number, stepIndex: number) => {
  form.stages[stageIndex].steps.splice(stepIndex, 1)
}

const parseCommands = (step: Step) => {
  step.commands = step.commandsText.split('\n').filter(cmd => cmd.trim())
}

const addVariable = () => {
  form.variables.push({ name: '', value: '', is_secret: false })
}

const removeVariable = (index: number) => {
  form.variables.splice(index, 1)
}

// 触发器相关方法
const setCron = (cron: string) => {
  if (!form.trigger_config.scheduled) {
    form.trigger_config.scheduled = { enabled: true, cron: '', timezone: 'Asia/Shanghai' }
  }
  form.trigger_config.scheduled.cron = cron
}

const copyWebhookUrl = () => {
  if (form.trigger_config.webhook?.url) {
    navigator.clipboard.writeText(window.location.origin + form.trigger_config.webhook.url)
    message.success('Webhook URL 已复制')
  }
}

// 同步可视化配置到 YAML
const syncToYaml = () => {
  const config: any = {
    name: form.name,
    variables: {} as Record<string, string>,
    stages: form.stages.map(stage => ({
      name: stage.name,
      needs: stage.needs.length > 0 ? stage.needs : undefined,
      steps: stage.steps.map(step => ({
        name: step.name,
        image: step.image,
        commands: step.commands,
        work_dir: step.work_dir !== '/workspace' ? step.work_dir : undefined,
        timeout: step.timeout !== 600 ? step.timeout : undefined
      }))
    }))
  }

  form.variables.forEach(v => {
    if (v.name) {
      config.variables[v.name] = v.value
    }
  })

  if (Object.keys(config.variables).length === 0) {
    delete config.variables
  }

  yamlContent.value = yaml.dump(config, { indent: 2, lineWidth: -1 })
}

// 解析 YAML 到可视化配置
const parseYaml = () => {
  try {
    const config = yaml.load(yamlContent.value) as any
    if (!config) {
      yamlError.value = ''
      return
    }

    form.name = config.name || form.name
    
    if (config.variables) {
      form.variables = Object.entries(config.variables).map(([name, value]) => ({
        name,
        value: String(value),
        is_secret: false
      }))
    }

    if (config.stages) {
      form.stages = config.stages.map((stage: any) => ({
        id: generateId(),
        name: stage.name,
        needs: stage.needs || [],
        steps: (stage.steps || []).map((step: any) => ({
          id: generateId(),
          name: step.name,
          image: step.image || '',
          commands: step.commands || [],
          commandsText: (step.commands || []).join('\n'),
          work_dir: step.work_dir || '/workspace',
          timeout: step.timeout || 600,
          env: step.env || {}
        }))
      }))
    }

    yamlError.value = ''
  } catch (e: any) {
    yamlError.value = `YAML 解析错误: ${e.message}`
  }
}

// 监听编辑模式切换
watch(editMode, (mode) => {
  if (mode === 'yaml') {
    syncToYaml()
  }
})

const loadGitRepos = async () => {
  try {
    const res = await gitRepoApi.list({ page_size: 100 })
    if (res?.data?.items) {
      gitRepos.value = res.data.items
    }
  } catch (error) {
    console.error('加载 Git 仓库失败:', error)
  }
}

const loadClusters = async () => {
  try {
    const res = await request.get('/k8s/clusters', { params: { page_size: 100 } })
    if (res?.data?.items) {
      clusters.value = res.data.items
    }
  } catch (error) {
    console.error('加载集群列表失败:', error)
  }
}

const loadPipeline = async () => {
  if (!pipelineId.value) return

  try {
    const res = await pipelineApi.get(pipelineId.value)
    const data = res?.data
    if (data) {
      form.name = data.name
      form.description = data.description
      form.git_repo_id = data.git_repo_id
      form.git_branch = data.git_branch || 'main'
      form.build_cluster_id = data.build_cluster_id
      form.build_namespace = data.build_namespace || 'devops-build'

      // 加载触发器配置
      if (data.trigger_config) {
        form.trigger_config = {
          manual: data.trigger_config.manual ?? true,
          scheduled: data.trigger_config.scheduled || { enabled: false, cron: '', timezone: 'Asia/Shanghai' },
          webhook: data.trigger_config.webhook || { enabled: false, secret: '', branch_filter: [], url: '' }
        }
      }

      // 转换阶段和步骤
      if (data.stages) {
        form.stages = data.stages.map((stage: any) => ({
          id: stage.id || generateId(),
          name: stage.name,
          needs: stage.depends_on || [],
          steps: (stage.steps || []).map((step: any) => {
            const commands = step.config?.commands || []
            return {
              id: step.id || generateId(),
              name: step.name,
              image: step.config?.image || '',
              commands: commands,
              commandsText: commands.join('\n'),
              work_dir: step.config?.work_dir || '/workspace',
              timeout: step.timeout || 600,
              env: step.config?.env || {}
            }
          })
        }))
      }

      if (data.variables) {
        form.variables = data.variables
      }
    }
  } catch (error) {
    console.error('加载流水线失败:', error)
    message.error('加载流水线失败')
  }
}

const savePipeline = async () => {
  if (!form.name) {
    message.error('请输入流水线名称')
    return
  }

  if (form.stages.length === 0) {
    message.error('请至少添加一个阶段')
    return
  }

  saving.value = true
  try {
    // 转换为 API 格式
    const stages = form.stages.map(stage => ({
      id: stage.id,
      name: stage.name,
      depends_on: stage.needs,
      steps: stage.steps.map(step => ({
        id: step.id,
        name: step.name,
        type: 'container',
        timeout: step.timeout,
        config: {
          image: step.image,
          commands: step.commands,
          work_dir: step.work_dir,
          env: step.env
        }
      }))
    }))

    // 构建触发器配置
    const triggerConfig = {
      manual: form.trigger_config.manual,
      scheduled: scheduledEnabled.value ? form.trigger_config.scheduled : null,
      webhook: webhookEnabled.value ? form.trigger_config.webhook : null
    }

    const data = {
      name: form.name,
      description: form.description,
      git_repo_id: form.git_repo_id,
      git_branch: form.git_branch,
      build_cluster_id: form.build_cluster_id,
      build_namespace: form.build_namespace,
      stages,
      variables: form.variables.filter(v => v.name),
      trigger_config: triggerConfig
    }

    if (isEdit.value) {
      await pipelineApi.update(pipelineId.value, data)
      message.success('更新成功')
    } else {
      await pipelineApi.create(data)
      message.success('创建成功')
    }

    router.push('/pipeline/list')
  } catch (error: any) {
    message.error(error?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const goBack = () => {
  router.push('/pipeline/list')
}

const showTemplateSelector = () => {
  templateModalVisible.value = true
}

const applyTemplate = (template: any) => {
  templateModalVisible.value = false
  form.name = template.name || form.name
  form.description = template.description || ''
  
  if (template.stages) {
    form.stages = template.stages.map((stage: any) => ({
      id: generateId(),
      name: stage.name,
      needs: stage.depends_on || [],
      steps: (stage.steps || []).map((step: any) => {
        const commands = step.commands || []
        return {
          id: generateId(),
          name: step.name,
          image: step.image || '',
          commands: commands,
          commandsText: commands.join('\n'),
          work_dir: step.work_dir || '/workspace',
          timeout: step.timeout || 600,
          env: step.env || {}
        }
      })
    }))
  }
  
  if (template.variables) {
    form.variables = Object.entries(template.variables).map(([name, value]) => ({
      name,
      value: String(value),
      is_secret: false
    }))
  }
  
  message.success('模板已应用')
}

onMounted(() => {
  loadGitRepos()
  loadClusters()
  if (isEdit.value) {
    loadPipeline()
  } else {
    // 默认添加一个阶段
    addStage()
  }
})
</script>

<style scoped>
.pipeline-editor {
  padding: 0;
}

.visual-editor {
  min-height: 400px;
}

.stages-container {
  display: flex;
  gap: 16px;
  overflow-x: auto;
  padding: 16px 0;
}

.stage-card {
  min-width: 320px;
  max-width: 320px;
  background: #fafafa;
  border-radius: 8px;
  padding: 12px;
}

.stage-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #e8e8e8;
}

.steps-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.step-card {
  background: #fff;
  border: 1px solid #e8e8e8;
  border-radius: 6px;
  padding: 12px;
}

.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.step-content :deep(.ant-form-item) {
  margin-bottom: 8px;
}

.empty-steps {
  padding: 20px;
}

.add-stage {
  min-width: 200px;
  display: flex;
  align-items: center;
}

.yaml-editor {
  position: relative;
}

.yaml-textarea {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 13px;
}

.yaml-error {
  margin-top: 8px;
}

.trigger-hint {
  margin-left: 12px;
  color: #999;
  font-size: 12px;
}

.cron-presets {
  margin-top: 8px;
  color: #666;
  font-size: 12px;
}

.cron-presets span {
  margin-right: 8px;
}
</style>
