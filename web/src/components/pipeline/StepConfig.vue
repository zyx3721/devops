<template>
  <a-modal v-model:open="visible" :title="isEdit ? '编辑步骤' : '添加步骤'" width="700px" @ok="handleOk" @cancel="handleCancel">
    <a-form :model="form" :label-col="{ span: 4 }" :wrapper-col="{ span: 20 }">
      <a-form-item label="步骤名称" required>
        <a-input v-model:value="form.name" placeholder="如: 构建镜像" />
      </a-form-item>

      <a-form-item label="容器镜像" required>
        <a-auto-complete v-model:value="form.image" :options="imageOptions" placeholder="如: node:18-alpine">
          <template #option="{ value }">
            <div>{{ value }}</div>
          </template>
        </a-auto-complete>
        <div class="hint">支持变量: ${IMAGE_TAG}, ${CI_COMMIT_SHA}</div>
      </a-form-item>

      <a-form-item label="执行命令" required>
        <a-textarea v-model:value="commandsText" :rows="6" placeholder="每行一条命令&#10;npm install&#10;npm run build" />
      </a-form-item>

      <a-form-item label="工作目录">
        <a-input v-model:value="form.work_dir" placeholder="/workspace (默认)" />
      </a-form-item>

      <a-collapse v-model:activeKey="activeKeys" ghost>
        <a-collapse-panel key="env" header="环境变量">
          <div v-for="(_, index) in envList" :key="index" class="env-row">
            <a-input v-model:value="envList[index].key" placeholder="变量名" style="width: 40%" />
            <span style="margin: 0 8px">=</span>
            <a-input v-model:value="envList[index].value" placeholder="变量值" style="width: 40%" />
            <a-button type="text" danger @click="removeEnv(index)">
              <template #icon><DeleteOutlined /></template>
            </a-button>
          </div>
          <a-button type="dashed" block @click="addEnv">
            <template #icon><PlusOutlined /></template>
            添加环境变量
          </a-button>
        </a-collapse-panel>

        <a-collapse-panel key="secrets" header="密钥引用">
          <a-select v-model:value="form.secrets" mode="multiple" placeholder="选择要注入的密钥" style="width: 100%">
            <a-select-option v-for="s in secretOptions" :key="s.name" :value="s.name">{{ s.name }}</a-select-option>
          </a-select>
          <div class="hint">密钥将作为环境变量注入容器</div>
        </a-collapse-panel>

        <a-collapse-panel key="resources" header="资源配置">
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="CPU 请求">
                <a-input v-model:value="form.resources.cpu_request" placeholder="100m" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="CPU 限制">
                <a-input v-model:value="form.resources.cpu_limit" placeholder="1000m" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="内存请求">
                <a-input v-model:value="form.resources.memory_request" placeholder="256Mi" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="内存限制">
                <a-input v-model:value="form.resources.memory_limit" placeholder="1Gi" />
              </a-form-item>
            </a-col>
          </a-row>
        </a-collapse-panel>

        <a-collapse-panel key="advanced" header="高级选项">
          <a-form-item label="超时时间">
            <a-input-number v-model:value="form.timeout" :min="0" :max="7200" style="width: 150px" />
            <span style="margin-left: 8px">秒 (0 表示不限制)</span>
          </a-form-item>
          <a-form-item label="重试次数">
            <a-input-number v-model:value="form.retry_count" :min="0" :max="5" style="width: 150px" />
          </a-form-item>
          <a-form-item label="执行条件">
            <a-select v-model:value="form.condition" style="width: 200px">
              <a-select-option value="">始终执行</a-select-option>
              <a-select-option value="on_success">仅成功时</a-select-option>
              <a-select-option value="on_failure">仅失败时</a-select-option>
              <a-select-option value="always">总是执行</a-select-option>
            </a-select>
          </a-form-item>
        </a-collapse-panel>
      </a-collapse>
    </a-form>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons-vue'

interface StepForm {
  id: string
  name: string
  image: string
  commands: string[]
  work_dir: string
  env: Record<string, string>
  secrets: string[]
  resources: {
    cpu_request: string
    cpu_limit: string
    memory_request: string
    memory_limit: string
  }
  timeout: number
  retry_count: number
  condition: string
}

const props = defineProps<{
  open: boolean
  step?: StepForm | null
  secrets?: { name: string }[]
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'save', step: StepForm): void
}>()

const visible = computed({
  get: () => props.open,
  set: (val) => emit('update:open', val)
})

const isEdit = computed(() => !!props.step?.id)

const defaultForm = (): StepForm => ({
  id: '',
  name: '',
  image: '',
  commands: [],
  work_dir: '/workspace',
  env: {},
  secrets: [],
  resources: { cpu_request: '100m', cpu_limit: '1000m', memory_request: '256Mi', memory_limit: '1Gi' },
  timeout: 600,
  retry_count: 0,
  condition: ''
})

const form = ref<StepForm>(defaultForm())
const commandsText = ref('')
const envList = ref<{ key: string; value: string }[]>([])
const activeKeys = ref<string[]>([])

const imageOptions = ref([
  { value: 'node:18-alpine' },
  { value: 'node:20-alpine' },
  { value: 'golang:1.21-alpine' },
  { value: 'python:3.11-slim' },
  { value: 'maven:3.9-eclipse-temurin-17' },
  { value: 'gradle:8-jdk17' },
  { value: 'docker:24-dind' },
  { value: 'alpine:3.18' },
  { value: 'ubuntu:22.04' }
])

const secretOptions = computed(() => props.secrets || [])

watch(() => props.open, (val) => {
  if (val) {
    if (props.step) {
      form.value = { ...defaultForm(), ...props.step }
      commandsText.value = (props.step.commands || []).join('\n')
      envList.value = Object.entries(props.step.env || {}).map(([key, value]) => ({ key, value }))
    } else {
      form.value = defaultForm()
      form.value.id = `step-${Date.now()}`
      commandsText.value = ''
      envList.value = []
    }
  }
})

const addEnv = () => envList.value.push({ key: '', value: '' })
const removeEnv = (index: number) => envList.value.splice(index, 1)

const handleOk = () => {
  form.value.commands = commandsText.value.split('\n').filter(c => c.trim())
  form.value.env = envList.value.reduce((acc, { key, value }) => {
    if (key.trim()) acc[key.trim()] = value
    return acc
  }, {} as Record<string, string>)
  emit('save', { ...form.value })
  visible.value = false
}

const handleCancel = () => {
  visible.value = false
}
</script>

<style scoped>
.env-row { display: flex; align-items: center; margin-bottom: 8px; }
.hint { color: #999; font-size: 12px; margin-top: 4px; }
</style>
