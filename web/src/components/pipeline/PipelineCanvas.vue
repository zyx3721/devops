<template>
  <div class="pipeline-canvas" ref="canvasRef">
    <div class="canvas-toolbar">
      <a-space>
        <a-button size="small" @click="addStage"><PlusOutlined /> 添加阶段</a-button>
        <a-button size="small" @click="zoomIn"><ZoomInOutlined /></a-button>
        <a-button size="small" @click="zoomOut"><ZoomOutOutlined /></a-button>
        <a-button size="small" @click="resetZoom"><ExpandOutlined /></a-button>
        <span class="zoom-level">{{ Math.round(zoom * 100) }}%</span>
      </a-space>
    </div>

    <div 
      class="canvas-content" 
      :style="{ transform: `scale(${zoom})`, transformOrigin: 'top left' }"
      @dragover.prevent
      @drop="handleDrop"
    >
      <!-- 连接线 SVG -->
      <svg class="connections-layer" :width="canvasWidth" :height="canvasHeight">
        <defs>
          <marker id="arrowhead" markerWidth="10" markerHeight="7" refX="9" refY="3.5" orient="auto">
            <polygon points="0 0, 10 3.5, 0 7" fill="#1890ff" />
          </marker>
        </defs>
        <path
          v-for="(conn, index) in connections"
          :key="index"
          :d="conn.path"
          fill="none"
          stroke="#1890ff"
          stroke-width="2"
          marker-end="url(#arrowhead)"
        />
      </svg>

      <!-- 阶段卡片 -->
      <div class="stages-row">
        <div
          v-for="(stage, stageIndex) in stages"
          :key="stage.id"
          class="stage-node"
          :class="{ 'stage-parallel': stage.parallel }"
          :style="{ left: `${stageIndex * 320 + 20}px` }"
          draggable="true"
          @dragstart="handleStageDragStart($event, stageIndex)"
          @dragend="handleDragEnd"
        >
          <div class="stage-header">
            <a-input 
              v-model:value="stage.name" 
              size="small" 
              placeholder="阶段名称"
              class="stage-name-input"
            />
            <a-dropdown>
              <a-button type="text" size="small"><MoreOutlined /></a-button>
              <template #overlay>
                <a-menu @click="({ key }) => handleStageMenu(key, stageIndex)">
                  <a-menu-item key="addStep"><PlusOutlined /> 添加步骤</a-menu-item>
                  <a-menu-item key="parallel"><BranchesOutlined /> 设为并行</a-menu-item>
                  <a-menu-item key="depends"><LinkOutlined /> 设置依赖</a-menu-item>
                  <a-menu-divider />
                  <a-menu-item key="delete" danger><DeleteOutlined /> 删除阶段</a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </div>

          <!-- 步骤列表 -->
          <div class="steps-container">
            <div
              v-for="(step, stepIndex) in stage.steps"
              :key="step.id"
              class="step-node"
              :class="{ 'step-selected': selectedStep?.id === step.id }"
              draggable="true"
              @dragstart="handleStepDragStart($event, stageIndex, stepIndex)"
              @click="selectStep(step, stageIndex, stepIndex)"
            >
              <div class="step-icon">
                <component :is="getStepIcon(step.type)" />
              </div>
              <div class="step-info">
                <span class="step-name">{{ step.name }}</span>
                <span class="step-type">{{ step.type || 'container' }}</span>
              </div>
              <a-button 
                type="text" 
                size="small" 
                class="step-delete"
                @click.stop="removeStep(stageIndex, stepIndex)"
              >
                <CloseOutlined />
              </a-button>
            </div>

            <div 
              class="add-step-btn"
              @click="addStep(stageIndex)"
              @dragover.prevent
              @drop="handleStepDrop($event, stageIndex)"
            >
              <PlusOutlined /> 添加步骤
            </div>
          </div>

          <!-- 依赖指示器 -->
          <div v-if="stage.needs?.length" class="stage-depends">
            依赖: {{ stage.needs.join(', ') }}
          </div>
        </div>

        <!-- 添加阶段按钮 -->
        <div class="add-stage-node" @click="addStage">
          <PlusOutlined />
          <span>添加阶段</span>
        </div>
      </div>
    </div>

    <!-- 步骤配置面板 -->
    <a-drawer
      v-model:open="configDrawerVisible"
      title="步骤配置"
      :width="400"
      placement="right"
    >
      <template v-if="selectedStep">
        <a-form layout="vertical">
          <a-form-item label="步骤名称">
            <a-input v-model:value="selectedStep.name" />
          </a-form-item>
          <a-form-item label="步骤类型">
            <a-select v-model:value="selectedStep.type">
              <a-select-option value="container">容器执行</a-select-option>
              <a-select-option value="shell">Shell 脚本</a-select-option>
              <a-select-option value="k8s_deploy">K8s 部署</a-select-option>
              <a-select-option value="notify">通知</a-select-option>
              <a-select-option value="approval">人工审批</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="镜像" v-if="selectedStep.type === 'container'">
            <a-auto-complete
              v-model:value="selectedStep.image"
              :options="imageOptions"
              placeholder="node:18-alpine"
            />
          </a-form-item>
          <a-form-item label="命令" v-if="['container', 'shell'].includes(selectedStep.type)">
            <a-textarea
              v-model:value="selectedStep.commandsText"
              :rows="5"
              placeholder="npm install&#10;npm run build"
            />
          </a-form-item>
          <a-form-item label="工作目录">
            <a-input v-model:value="selectedStep.work_dir" placeholder="/workspace" />
          </a-form-item>
          <a-form-item label="超时(秒)">
            <a-input-number v-model:value="selectedStep.timeout" :min="0" style="width: 100%" />
          </a-form-item>
          <a-form-item label="环境变量">
            <div v-for="(val, key) in selectedStep.env" :key="key" class="env-item">
              <a-input :value="key" disabled style="width: 40%" />
              <a-input v-model:value="selectedStep.env[key]" style="width: 50%" />
              <a-button type="text" danger @click="delete selectedStep.env[key]">
                <DeleteOutlined />
              </a-button>
            </div>
            <a-button type="dashed" block @click="addEnvVar">
              <PlusOutlined /> 添加环境变量
            </a-button>
          </a-form-item>
        </a-form>
      </template>
    </a-drawer>

    <!-- 依赖设置弹窗 -->
    <a-modal v-model:open="dependsModalVisible" title="设置阶段依赖" @ok="saveDependencies">
      <a-form layout="vertical">
        <a-form-item label="依赖阶段">
          <a-select
            v-model:value="editingStage.needs"
            mode="multiple"
            placeholder="选择依赖的阶段"
          >
            <a-select-option
              v-for="s in availableDependencies"
              :key="s.id"
              :value="s.name"
            >
              {{ s.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  PlusOutlined, DeleteOutlined, CloseOutlined, MoreOutlined,
  ZoomInOutlined, ZoomOutOutlined, ExpandOutlined,
  BranchesOutlined, LinkOutlined,
  CodeOutlined, CloudServerOutlined, BellOutlined, CheckCircleOutlined
} from '@ant-design/icons-vue'

interface Step {
  id: string
  name: string
  type: string
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
  parallel?: boolean
}

const props = defineProps<{
  modelValue: Stage[]
}>()

const emit = defineEmits(['update:modelValue'])

const canvasRef = ref<HTMLElement | null>(null)
const zoom = ref(1)
const canvasWidth = ref(1200)
const canvasHeight = ref(600)

const selectedStep = ref<Step | null>(null)
const selectedStageIndex = ref(-1)
const selectedStepIndex = ref(-1)
const configDrawerVisible = ref(false)

const dependsModalVisible = ref(false)
const editingStage = ref<Stage>({ id: '', name: '', steps: [], needs: [] })
const editingStageIndex = ref(-1)

const stages = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const connections = computed(() => {
  const conns: { path: string }[] = []
  stages.value.forEach((stage, index) => {
    if (stage.needs?.length) {
      stage.needs.forEach(depName => {
        const depIndex = stages.value.findIndex(s => s.name === depName)
        if (depIndex >= 0) {
          const startX = depIndex * 320 + 280
          const endX = index * 320 + 20
          const y = 100
          conns.push({
            path: `M ${startX} ${y} C ${startX + 40} ${y}, ${endX - 40} ${y}, ${endX} ${y}`
          })
        }
      })
    } else if (index > 0) {
      // 默认连接前一个阶段
      const startX = (index - 1) * 320 + 280
      const endX = index * 320 + 20
      const y = 100
      conns.push({
        path: `M ${startX} ${y} C ${startX + 40} ${y}, ${endX - 40} ${y}, ${endX} ${y}`
      })
    }
  })
  return conns
})

const availableDependencies = computed(() => {
  return stages.value.filter((_, i) => i !== editingStageIndex.value)
})

const imageOptions = [
  { value: 'node:18-alpine' },
  { value: 'node:20-alpine' },
  { value: 'golang:1.21-alpine' },
  { value: 'maven:3.9-eclipse-temurin-17' },
  { value: 'python:3.11-alpine' }
]

const generateId = () => Math.random().toString(36).substring(2, 10)

const zoomIn = () => { zoom.value = Math.min(zoom.value + 0.1, 2) }
const zoomOut = () => { zoom.value = Math.max(zoom.value - 0.1, 0.5) }
const resetZoom = () => { zoom.value = 1 }

const addStage = () => {
  const newStages = [...stages.value]
  newStages.push({
    id: generateId(),
    name: `阶段 ${newStages.length + 1}`,
    steps: [],
    needs: []
  })
  stages.value = newStages
}

const removeStage = (index: number) => {
  const newStages = [...stages.value]
  newStages.splice(index, 1)
  stages.value = newStages
}

const addStep = (stageIndex: number) => {
  const newStages = [...stages.value]
  newStages[stageIndex].steps.push({
    id: generateId(),
    name: `步骤 ${newStages[stageIndex].steps.length + 1}`,
    type: 'container',
    image: '',
    commands: [],
    commandsText: '',
    work_dir: '/workspace',
    timeout: 600,
    env: {}
  })
  stages.value = newStages
}

const removeStep = (stageIndex: number, stepIndex: number) => {
  const newStages = [...stages.value]
  newStages[stageIndex].steps.splice(stepIndex, 1)
  stages.value = newStages
  if (selectedStep.value?.id === stages.value[stageIndex]?.steps[stepIndex]?.id) {
    selectedStep.value = null
    configDrawerVisible.value = false
  }
}

const selectStep = (step: Step, stageIndex: number, stepIndex: number) => {
  selectedStep.value = step
  selectedStageIndex.value = stageIndex
  selectedStepIndex.value = stepIndex
  configDrawerVisible.value = true
}

const handleStageMenu = (key: string, stageIndex: number) => {
  switch (key) {
    case 'addStep':
      addStep(stageIndex)
      break
    case 'parallel':
      const newStages = [...stages.value]
      newStages[stageIndex].parallel = !newStages[stageIndex].parallel
      stages.value = newStages
      break
    case 'depends':
      editingStage.value = { ...stages.value[stageIndex] }
      editingStageIndex.value = stageIndex
      dependsModalVisible.value = true
      break
    case 'delete':
      removeStage(stageIndex)
      break
  }
}

const saveDependencies = () => {
  const newStages = [...stages.value]
  newStages[editingStageIndex.value].needs = editingStage.value.needs
  stages.value = newStages
  dependsModalVisible.value = false
}

const addEnvVar = () => {
  if (selectedStep.value) {
    const key = `VAR_${Object.keys(selectedStep.value.env).length + 1}`
    selectedStep.value.env[key] = ''
  }
}

const getStepIcon = (type: string) => {
  const icons: Record<string, any> = {
    container: CodeOutlined,
    shell: CodeOutlined,
    k8s_deploy: CloudServerOutlined,
    notify: BellOutlined,
    approval: CheckCircleOutlined
  }
  return icons[type] || CodeOutlined
}

// 拖拽处理
let dragData: { type: string; stageIndex?: number; stepIndex?: number } | null = null

const handleStageDragStart = (e: DragEvent, stageIndex: number) => {
  dragData = { type: 'stage', stageIndex }
  e.dataTransfer?.setData('text/plain', '')
}

const handleStepDragStart = (e: DragEvent, stageIndex: number, stepIndex: number) => {
  dragData = { type: 'step', stageIndex, stepIndex }
  e.dataTransfer?.setData('text/plain', '')
}

const handleDragEnd = () => {
  dragData = null
}

const handleDrop = (e: DragEvent) => {
  // 处理阶段拖拽排序
}

const handleStepDrop = (e: DragEvent, targetStageIndex: number) => {
  if (dragData?.type === 'step' && dragData.stageIndex !== undefined && dragData.stepIndex !== undefined) {
    const newStages = [...stages.value]
    const [movedStep] = newStages[dragData.stageIndex].steps.splice(dragData.stepIndex, 1)
    newStages[targetStageIndex].steps.push(movedStep)
    stages.value = newStages
  }
  dragData = null
}

// 监听步骤变化，同步命令
watch(() => selectedStep.value?.commandsText, (val) => {
  if (selectedStep.value && val !== undefined) {
    selectedStep.value.commands = val.split('\n').filter(cmd => cmd.trim())
  }
})
</script>

<style scoped>
.pipeline-canvas {
  position: relative;
  height: 100%;
  min-height: 500px;
  background: #f5f5f5;
  border-radius: 8px;
  overflow: hidden;
}

.canvas-toolbar {
  position: absolute;
  top: 12px;
  left: 12px;
  z-index: 10;
  background: #fff;
  padding: 8px 12px;
  border-radius: 6px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.zoom-level {
  color: #666;
  font-size: 12px;
}

.canvas-content {
  position: relative;
  width: 100%;
  height: 100%;
  padding: 60px 20px 20px;
  overflow: auto;
}

.connections-layer {
  position: absolute;
  top: 0;
  left: 0;
  pointer-events: none;
}

.stages-row {
  display: flex;
  gap: 20px;
  padding: 20px;
}

.stage-node {
  position: relative;
  width: 280px;
  min-height: 200px;
  background: #fff;
  border: 2px solid #e8e8e8;
  border-radius: 8px;
  padding: 12px;
  transition: all 0.2s;
}

.stage-node:hover {
  border-color: #1890ff;
  box-shadow: 0 4px 12px rgba(24, 144, 255, 0.15);
}

.stage-parallel {
  border-style: dashed;
}

.stage-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid #f0f0f0;
}

.stage-name-input {
  width: 180px;
  font-weight: 500;
}

.steps-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.step-node {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #fafafa;
  border: 1px solid #e8e8e8;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.step-node:hover {
  background: #e6f7ff;
  border-color: #91d5ff;
}

.step-selected {
  background: #e6f7ff;
  border-color: #1890ff;
}

.step-icon {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #1890ff;
  color: #fff;
  border-radius: 4px;
  font-size: 14px;
}

.step-info {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.step-name {
  font-size: 13px;
  font-weight: 500;
}

.step-type {
  font-size: 11px;
  color: #999;
}

.step-delete {
  opacity: 0;
  transition: opacity 0.2s;
}

.step-node:hover .step-delete {
  opacity: 1;
}

.add-step-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  padding: 8px;
  border: 1px dashed #d9d9d9;
  border-radius: 6px;
  color: #999;
  cursor: pointer;
  transition: all 0.2s;
}

.add-step-btn:hover {
  border-color: #1890ff;
  color: #1890ff;
}

.stage-depends {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid #f0f0f0;
  font-size: 11px;
  color: #999;
}

.add-stage-node {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  width: 120px;
  min-height: 200px;
  border: 2px dashed #d9d9d9;
  border-radius: 8px;
  color: #999;
  cursor: pointer;
  transition: all 0.2s;
}

.add-stage-node:hover {
  border-color: #1890ff;
  color: #1890ff;
}

.env-item {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
</style>
