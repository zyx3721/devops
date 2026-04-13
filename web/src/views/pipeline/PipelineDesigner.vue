<template>
  <div class="pipeline-designer">
    <div class="designer-layout">
      <!-- 左侧：组件面板 -->
      <div class="component-panel">
        <a-card title="组件库" :bordered="false" size="small">
          <a-collapse v-model:activeKey="activePanels" ghost>
            <a-collapse-panel key="stages" header="阶段模板">
              <div
                v-for="stage in stageTemplates"
                :key="stage.type"
                class="component-item"
                draggable="true"
                @dragstart="handleDragStart($event, stage)"
              >
                <component :is="stage.icon" />
                <span>{{ stage.label }}</span>
              </div>
            </a-collapse-panel>
            
            <a-collapse-panel key="steps" header="步骤模板">
              <div
                v-for="step in stepTemplates"
                :key="step.type"
                class="component-item"
                draggable="true"
                @dragstart="handleDragStart($event, step)"
              >
                <component :is="step.icon" />
                <span>{{ step.label }}</span>
              </div>
            </a-collapse-panel>
          </a-collapse>
        </a-card>
      </div>

      <!-- 中间：画布区域 -->
      <div class="canvas-area">
        <div class="canvas-toolbar">
          <a-space>
            <a-button @click="zoomIn" size="small">
              <template #icon><ZoomInOutlined /></template>
            </a-button>
            <a-button @click="zoomOut" size="small">
              <template #icon><ZoomOutOutlined /></template>
            </a-button>
            <a-button @click="resetZoom" size="small">
              <template #icon><FullscreenOutlined /></template>
            </a-button>
            <a-divider type="vertical" />
            <a-button @click="clearCanvas" size="small" danger>
              <template #icon><ClearOutlined /></template>
              清空
            </a-button>
          </a-space>
        </div>

        <div
          ref="canvasContainer"
          class="canvas-container"
          @drop="handleDrop"
          @dragover="handleDragOver"
        >
          <!-- 画布内容由 @antv/x6 渲染 -->
        </div>

        <div class="canvas-footer">
          <a-space>
            <a-button type="primary" @click="savePipeline" :loading="saving">
              <template #icon><SaveOutlined /></template>
              保存
            </a-button>
            <a-button @click="previewPipeline">
              <template #icon><EyeOutlined /></template>
              预览
            </a-button>
            <a-button @click="exportJSON">
              <template #icon><DownloadOutlined /></template>
              导出 JSON
            </a-button>
            <a-button @click="openImportModal">
              <template #icon><UploadOutlined /></template>
              导入 JSON
            </a-button>
          </a-space>
        </div>
      </div>

      <!-- 右侧：属性面板 -->
      <div class="property-panel">
        <a-card title="属性配置" :bordered="false" size="small">
          <a-empty v-if="!selectedNode" description="请选择一个节点" :image="simpleImage" />
          
          <a-form
            v-else
            :model="nodeConfig"
            :label-col="{ span: 24 }"
            :wrapper-col="{ span: 24 }"
            layout="vertical"
          >
            <a-form-item label="节点名称">
              <a-input v-model:value="nodeConfig.name" @change="updateNodeConfig" />
            </a-form-item>

            <a-form-item label="节点类型">
              <a-tag>{{ nodeConfig.type }}</a-tag>
            </a-form-item>

            <a-divider />

            <!-- 根据节点类型显示不同的配置项 -->
            <template v-if="nodeConfig.type === 'build'">
              <a-form-item label="构建命令">
                <a-textarea
                  v-model:value="nodeConfig.config.command"
                  :rows="3"
                  placeholder="npm run build"
                  @change="updateNodeConfig"
                />
              </a-form-item>
              <a-form-item label="工作目录">
                <a-input
                  v-model:value="nodeConfig.config.workdir"
                  placeholder="./"
                  @change="updateNodeConfig"
                />
              </a-form-item>
            </template>

            <template v-else-if="nodeConfig.type === 'test'">
              <a-form-item label="测试命令">
                <a-textarea
                  v-model:value="nodeConfig.config.command"
                  :rows="3"
                  placeholder="npm test"
                  @change="updateNodeConfig"
                />
              </a-form-item>
              <a-form-item label="覆盖率要求">
                <a-input-number
                  v-model:value="nodeConfig.config.coverage"
                  :min="0"
                  :max="100"
                  addon-after="%"
                  style="width: 100%"
                  @change="updateNodeConfig"
                />
              </a-form-item>
            </template>

            <template v-else-if="nodeConfig.type === 'deploy'">
              <a-form-item label="部署环境">
                <a-select
                  v-model:value="nodeConfig.config.environment"
                  style="width: 100%"
                  @change="updateNodeConfig"
                >
                  <a-select-option value="dev">开发环境</a-select-option>
                  <a-select-option value="test">测试环境</a-select-option>
                  <a-select-option value="staging">预发布环境</a-select-option>
                  <a-select-option value="prod">生产环境</a-select-option>
                </a-select>
              </a-form-item>
              <a-form-item label="部署策略">
                <a-select
                  v-model:value="nodeConfig.config.strategy"
                  style="width: 100%"
                  @change="updateNodeConfig"
                >
                  <a-select-option value="rolling">滚动更新</a-select-option>
                  <a-select-option value="bluegreen">蓝绿部署</a-select-option>
                  <a-select-option value="canary">金丝雀发布</a-select-option>
                </a-select>
              </a-form-item>
            </template>

            <a-form-item label="超时时间（秒）">
              <a-input-number
                v-model:value="nodeConfig.config.timeout"
                :min="0"
                style="width: 100%"
                @change="updateNodeConfig"
              />
            </a-form-item>

            <a-form-item label="失败时继续">
              <a-switch
                v-model:checked="nodeConfig.config.continueOnError"
                @change="updateNodeConfig"
              />
            </a-form-item>

            <a-divider />

            <a-button type="primary" danger block @click="deleteNode">
              <template #icon><DeleteOutlined /></template>
              删除节点
            </a-button>
          </a-form>
        </a-card>
      </div>
    </div>

    <!-- 预览弹窗 -->
    <a-modal
      v-model:open="previewVisible"
      title="流水线预览"
      width="800px"
      :footer="null"
    >
      <div class="preview-content">
        <a-descriptions title="流水线信息" :column="2" bordered>
          <a-descriptions-item label="节点数">
            {{ pipelineGraph.nodes.length }}
          </a-descriptions-item>
          <a-descriptions-item label="连接数">
            {{ pipelineGraph.edges.length }}
          </a-descriptions-item>
        </a-descriptions>

        <a-divider />

        <h4>执行流程</h4>
        <a-timeline>
          <a-timeline-item v-for="node in pipelineGraph.nodes" :key="node.id" color="blue">
            <strong>{{ node.name }}</strong>
            <div class="text-gray">类型: {{ node.type }}</div>
          </a-timeline-item>
        </a-timeline>
      </div>
    </a-modal>

    <!-- JSON 导入弹窗 -->
    <a-modal
      v-model:open="importVisible"
      title="导入 JSON 配置"
      @ok="handleImport"
      width="600px"
    >
      <a-textarea
        v-model:value="importJSON"
        :rows="15"
        placeholder="粘贴 JSON 配置..."
      />
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import { message, Empty, Modal } from 'ant-design-vue'
import {
  ZoomInOutlined,
  ZoomOutOutlined,
  FullscreenOutlined,
  ClearOutlined,
  SaveOutlined,
  EyeOutlined,
  DownloadOutlined,
  UploadOutlined,
  DeleteOutlined,
  BuildOutlined,
  CodeOutlined,
  RocketOutlined,
  CheckCircleOutlined,
  BugOutlined,
  SafetyOutlined,
} from '@ant-design/icons-vue'
import { Graph, Node } from '@antv/x6'

const simpleImage = Empty.PRESENTED_IMAGE_SIMPLE

interface PipelineNode {
  id: string
  type: string
  name: string
  config: Record<string, any>
  position: { x: number; y: number }
}

interface PipelineEdge {
  id: string
  source: string
  target: string
  type: 'default' | 'conditional'
}

interface PipelineGraph {
  nodes: PipelineNode[]
  edges: PipelineEdge[]
}

const canvasContainer = ref<HTMLElement>()
const saving = ref(false)
const previewVisible = ref(false)
const importVisible = ref(false)
const importJSON = ref('')
const activePanels = ref(['stages', 'steps'])
const selectedNode = ref<Node | null>(null)

let graph: Graph | null = null
let draggedComponent: any = null

const nodeConfig = reactive({
  id: '',
  name: '',
  type: '',
  config: {} as Record<string, any>,
})

const pipelineGraph = reactive<PipelineGraph>({
  nodes: [],
  edges: [],
})

const stageTemplates = [
  { type: 'build', label: '构建', icon: BuildOutlined },
  { type: 'test', label: '测试', icon: CheckCircleOutlined },
  { type: 'lint', label: '代码检查', icon: CodeOutlined },
  { type: 'security', label: '安全扫描', icon: SafetyOutlined },
  { type: 'deploy', label: '部署', icon: RocketOutlined },
]

const stepTemplates = [
  { type: 'script', label: '脚本执行', icon: CodeOutlined },
  { type: 'docker', label: 'Docker 构建', icon: BuildOutlined },
  { type: 'notification', label: '通知', icon: BugOutlined },
]

const initGraph = () => {
  if (!canvasContainer.value) return

  graph = new Graph({
    container: canvasContainer.value,
    width: canvasContainer.value.clientWidth,
    height: canvasContainer.value.clientHeight,
    grid: {
      size: 10,
      visible: true,
    },
    panning: {
      enabled: true,
      modifiers: 'shift',
    },
    mousewheel: {
      enabled: true,
      modifiers: ['ctrl', 'meta'],
    },
    connecting: {
      router: 'manhattan',
      connector: {
        name: 'rounded',
        args: {
          radius: 8,
        },
      },
      anchor: 'center',
      connectionPoint: 'anchor',
      allowBlank: false,
      snap: {
        radius: 20,
      },
      createEdge() {
        return graph!.createEdge({
          attrs: {
            line: {
              stroke: '#1890ff',
              strokeWidth: 2,
              targetMarker: {
                name: 'block',
                width: 12,
                height: 8,
              },
            },
          },
          zIndex: 0,
        })
      },
      validateConnection({ targetMagnet }: any) {
        return !!targetMagnet
      },
    },
  })

  // 监听节点选择
  graph.on('node:click', ({ node }: any) => {
    selectedNode.value = node
    const data = node.getData()
    Object.assign(nodeConfig, {
      id: node.id,
      name: node.label || '',
      type: data.type || '',
      config: data.config || {},
    })
  })

  // 监听画布点击（取消选择）
  graph.on('blank:click', () => {
    selectedNode.value = null
  })
}

const handleDragStart = (event: DragEvent, component: any) => {
  draggedComponent = component
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'copy'
  }
}

const handleDragOver = (event: DragEvent) => {
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'copy'
  }
}

const handleDrop = (event: DragEvent) => {
  event.preventDefault()
  
  if (!graph || !draggedComponent || !canvasContainer.value) return

  const rect = canvasContainer.value.getBoundingClientRect()
  const x = event.clientX - rect.left
  const y = event.clientY - rect.top

  // 转换为画布坐标
  const point = graph.clientToLocal({ x, y })

  // 创建节点
  const node = graph.addNode({
    x: point.x - 60,
    y: point.y - 30,
    width: 120,
    height: 60,
    label: draggedComponent.label,
    data: {
      type: draggedComponent.type,
      config: {
        command: '',
        timeout: 300,
        continueOnError: false,
      },
    },
    attrs: {
      body: {
        fill: getNodeColor(draggedComponent.type),
        stroke: '#096dd9',
        strokeWidth: 2,
        rx: 6,
        ry: 6,
      },
      label: {
        fill: '#fff',
        fontSize: 14,
        fontWeight: 'bold',
      },
    },
  })

  // 自动选中新创建的节点
  selectedNode.value = node
  const data = node.getData()
  Object.assign(nodeConfig, {
    id: node.id,
    name: node.label || '',
    type: data.type || '',
    config: data.config || {},
  })

  draggedComponent = null
}

const getNodeColor = (type: string): string => {
  const colors: Record<string, string> = {
    build: '#1890ff',
    test: '#52c41a',
    lint: '#faad14',
    security: '#eb2f96',
    deploy: '#722ed1',
    script: '#13c2c2',
    docker: '#2f54eb',
    notification: '#fa8c16',
  }
  return colors[type] || '#1890ff'
}

const updateNodeConfig = () => {
  if (!selectedNode.value) return

  selectedNode.value.setData({
    type: nodeConfig.type,
    config: nodeConfig.config,
  })

  if (nodeConfig.name) {
    selectedNode.value.setAttrs({
      label: {
        text: nodeConfig.name,
      },
    })
  }
}

const deleteNode = () => {
  if (!graph || !selectedNode.value) return

  graph.removeNode(selectedNode.value)
  selectedNode.value = null
}

const zoomIn = () => {
  graph?.zoom(0.1)
}

const zoomOut = () => {
  graph?.zoom(-0.1)
}

const resetZoom = () => {
  graph?.zoomTo(1)
  graph?.centerContent()
}

const clearCanvas = () => {
  if (!graph) return

  Modal.confirm({
    title: '确认清空',
    content: '确定要清空画布吗？此操作不可恢复。',
    okText: '确定',
    okType: 'danger',
    cancelText: '取消',
    onOk: () => {
      graph!.clearCells()
      selectedNode.value = null
      message.success('画布已清空')
    },
  })
}

const savePipeline = async () => {
  if (!graph) return

  graphToJSON()
  
  saving.value = true
  try {
    // 这里应该调用后端 API 保存
    await new Promise(resolve => setTimeout(resolve, 1000))
    message.success('保存成功')
  } catch (error: any) {
    message.error(error?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const previewPipeline = () => {
  if (!graph) return

  const graphData = graphToJSON()
  Object.assign(pipelineGraph, graphData)
  previewVisible.value = true
}

const graphToJSON = (): PipelineGraph => {
  if (!graph) return { nodes: [], edges: [] }

  const nodes: PipelineNode[] = graph.getNodes().map((node: any) => ({
    id: node.id,
    type: node.getData()?.type || 'unknown',
    name: node.label || '',
    config: node.getData()?.config || {},
    position: node.getPosition(),
  }))

  const edges: PipelineEdge[] = graph.getEdges().map((edge: any) => ({
    id: edge.id,
    source: edge.getSourceCellId(),
    target: edge.getTargetCellId(),
    type: 'default',
  }))

  return { nodes, edges }
}

const JSONToGraph = (data: PipelineGraph) => {
  if (!graph) return

  graph.clearCells()

  // 添加节点
  data.nodes.forEach(node => {
    graph!.addNode({
      id: node.id,
      x: node.position.x,
      y: node.position.y,
      width: 120,
      height: 60,
      label: node.name,
      data: {
        type: node.type,
        config: node.config,
      },
      attrs: {
        body: {
          fill: getNodeColor(node.type),
          stroke: '#096dd9',
          strokeWidth: 2,
          rx: 6,
          ry: 6,
        },
        label: {
          fill: '#fff',
          fontSize: 14,
          fontWeight: 'bold',
        },
      },
    })
  })

  // 添加连线
  data.edges.forEach(edge => {
    graph!.addEdge({
      id: edge.id,
      source: edge.source,
      target: edge.target,
    })
  })
}

const exportJSON = () => {
  const data = graphToJSON()
  const json = JSON.stringify(data, null, 2)
  
  // 创建下载
  const blob = new Blob([json], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'pipeline-config.json'
  a.click()
  URL.revokeObjectURL(url)
  
  message.success('导出成功')
}

const openImportModal = () => {
  importVisible.value = true
}

const handleImport = () => {
  try {
    const data = JSON.parse(importJSON.value) as PipelineGraph
    JSONToGraph(data)
    importVisible.value = false
    importJSON.value = ''
    message.success('导入成功')
  } catch (error) {
    message.error('JSON 格式错误')
  }
}

const handleResize = () => {
  if (graph && canvasContainer.value) {
    graph.resize(canvasContainer.value.clientWidth, canvasContainer.value.clientHeight)
  }
}

onMounted(() => {
  initGraph()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  graph?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.pipeline-designer {
  height: calc(100vh - 120px);
  padding: 0;
}

.designer-layout {
  display: flex;
  height: 100%;
  gap: 16px;
}

.component-panel {
  width: 250px;
  flex-shrink: 0;
  overflow-y: auto;
}

.component-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  margin-bottom: 8px;
  background: #f5f5f5;
  border-radius: 4px;
  cursor: move;
  transition: all 0.3s;
}

.component-item:hover {
  background: #e6f7ff;
  border-color: #1890ff;
}

.canvas-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #fff;
  border-radius: 4px;
  overflow: hidden;
}

.canvas-toolbar {
  padding: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.canvas-container {
  flex: 1;
  background: #fafafa;
  position: relative;
}

.canvas-footer {
  padding: 12px;
  border-top: 1px solid #f0f0f0;
  text-align: center;
}

.property-panel {
  width: 300px;
  flex-shrink: 0;
  overflow-y: auto;
}

.preview-content {
  max-height: 600px;
  overflow-y: auto;
}

.text-gray {
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
}

:deep(.ant-collapse-ghost > .ant-collapse-item > .ant-collapse-content > .ant-collapse-content-box) {
  padding: 8px 0;
}
</style>
