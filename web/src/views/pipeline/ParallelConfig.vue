<template>
  <div class="parallel-config">
    <!-- 面包屑导航 -->
    <a-breadcrumb style="margin-bottom: 16px">
      <a-breadcrumb-item>
        <router-link to="/pipeline/list">流水线列表</router-link>
      </a-breadcrumb-item>
      <a-breadcrumb-item>{{ pipelineName }}</a-breadcrumb-item>
      <a-breadcrumb-item>并行构建配置</a-breadcrumb-item>
    </a-breadcrumb>

    <!-- 配置卡片 -->
    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-form
        ref="formRef"
        :model="formData"
        :label-col="{ span: 4 }"
        :wrapper-col="{ span: 20 }"
      >
        <a-form-item label="启用并行构建">
          <a-switch
            v-model:checked="formData.enabled"
            checked-children="开"
            un-checked-children="关"
          />
          <span class="form-tip">启用后，可配置的阶段将并行执行</span>
        </a-form-item>

        <template v-if="formData.enabled">
          <a-form-item label="最大并行数">
            <a-slider
              v-model:value="formData.max_parallel"
              :min="1"
              :max="10"
              :marks="{ 1: '1', 5: '5', 10: '10' }"
            />
            <span class="form-tip">同时执行的最大阶段数: {{ formData.max_parallel }}</span>
          </a-form-item>

          <a-form-item label="快速失败">
            <a-switch
              v-model:checked="formData.fail_fast"
              checked-children="开"
              un-checked-children="关"
            />
            <span class="form-tip">任一并行任务失败时立即停止其他任务</span>
          </a-form-item>

          <a-form-item label="可并行阶段">
            <a-checkbox-group v-model:value="formData.parallel_stages">
              <a-row>
                <a-col :span="8" v-for="stage in availableStages" :key="stage.name">
                  <a-checkbox :value="stage.name">
                    {{ stage.name }}
                  </a-checkbox>
                </a-col>
              </a-row>
            </a-checkbox-group>
            <div class="form-tip">选择可以并行执行的阶段</div>
          </a-form-item>
        </template>
      </a-form>
    </a-card>

    <!-- 依赖图编辑器 -->
    <a-card title="阶段依赖关系" :bordered="false" v-if="formData.enabled">
      <template #extra>
        <a-space>
          <a-button @click="autoLayout">
            <template #icon><BranchesOutlined /></template>
            自动布局
          </a-button>
          <a-button type="primary" @click="saveConfig" :loading="saving">
            <template #icon><SaveOutlined /></template>
            保存配置
          </a-button>
        </a-space>
      </template>

      <div class="dependency-editor">
        <div class="editor-toolbar">
          <a-alert
            message="提示"
            description="拖拽连线表示依赖关系。例如：A → B 表示 B 依赖于 A，A 完成后才能执行 B。"
            type="info"
            show-icon
            closable
            style="margin-bottom: 16px"
          />
        </div>

        <div ref="graphContainer" class="graph-container"></div>

        <!-- 依赖关系列表 -->
        <a-divider />
        <h4>当前依赖关系</h4>
        <a-table
          :columns="dependencyColumns"
          :data-source="dependencyList"
          :pagination="false"
          size="small"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'action'">
              <a-button
                type="link"
                size="small"
                danger
                @click="removeDependency(record)"
              >
                删除
              </a-button>
            </template>
          </template>
        </a-table>
      </div>
    </a-card>

    <!-- 配置预览 -->
    <a-card title="配置预览" :bordered="false" style="margin-top: 16px" v-if="formData.enabled">
      <pre class="config-preview">{{ JSON.stringify(getConfigPreview(), null, 2) }}</pre>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  BranchesOutlined,
  SaveOutlined,
} from '@ant-design/icons-vue'
import request from '@/services/api'
import { Graph } from '@antv/x6'

const route = useRoute()
const pipelineId = computed(() => Number(route.params.pipelineId))

const formRef = ref()
const graphContainer = ref<HTMLElement>()
const saving = ref(false)
const pipelineName = ref('')

let graph: Graph | null = null

interface ParallelConfig {
  id?: number
  pipeline_id: number
  enabled: boolean
  max_parallel: number
  fail_fast: boolean
  parallel_stages: string[]
  dependency_graph: Record<string, string[]>
}

const formData = reactive<ParallelConfig>({
  pipeline_id: pipelineId.value,
  enabled: false,
  max_parallel: 3,
  fail_fast: true,
  parallel_stages: [],
  dependency_graph: {},
})

const availableStages = ref<any[]>([
  { name: 'build', label: '构建' },
  { name: 'test', label: '测试' },
  { name: 'lint', label: '代码检查' },
  { name: 'security-scan', label: '安全扫描' },
  { name: 'package', label: '打包' },
  { name: 'deploy', label: '部署' },
])

const dependencyColumns = [
  { title: '源阶段', dataIndex: 'source', key: 'source' },
  { title: '目标阶段', dataIndex: 'target', key: 'target' },
  { title: '说明', dataIndex: 'description', key: 'description' },
  { title: '操作', key: 'action', width: 100 },
]

const dependencyList = computed(() => {
  const list: any[] = []
  Object.entries(formData.dependency_graph).forEach(([target, sources]) => {
    sources.forEach(source => {
      list.push({
        source,
        target,
        description: `${target} 依赖于 ${source}`,
      })
    })
  })
  return list
})

const initGraph = () => {
  if (!graphContainer.value) return

  graph = new Graph({
    container: graphContainer.value,
    width: graphContainer.value.clientWidth,
    height: 400,
    grid: {
      size: 10,
      visible: true,
    },
    panning: {
      enabled: true,
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
      validateConnection({ targetMagnet }) {
        return !!targetMagnet
      },
    },
  })

  // 监听连线事件
  graph.on('edge:connected', ({ edge }) => {
    const source = edge.getSourceNode()
    const target = edge.getTargetNode()
    
    if (source && target) {
      const sourceId = source.id
      const targetId = target.id
      
      // 更新依赖图
      if (!formData.dependency_graph[targetId]) {
        formData.dependency_graph[targetId] = []
      }
      if (!formData.dependency_graph[targetId].includes(sourceId)) {
        formData.dependency_graph[targetId].push(sourceId)
      }
    }
  })

  // 监听删除连线事件
  graph.on('edge:removed', ({ edge }) => {
    const source = edge.getSourceCell()
    const target = edge.getTargetCell()
    
    if (source && target) {
      const sourceId = source.id
      const targetId = target.id
      
      if (formData.dependency_graph[targetId]) {
        const index = formData.dependency_graph[targetId].indexOf(sourceId)
        if (index > -1) {
          formData.dependency_graph[targetId].splice(index, 1)
        }
      }
    }
  })

  // 添加节点
  renderStages()
}

const renderStages = () => {
  if (!graph) return

  graph.clearCells()

  const stages = formData.parallel_stages.length > 0
    ? availableStages.value.filter(s => formData.parallel_stages.includes(s.name))
    : availableStages.value

  const spacing = 150
  const startX = 50
  const startY = 100

  stages.forEach((stage, index) => {
    const x = startX + (index % 3) * spacing
    const y = startY + Math.floor(index / 3) * 100

    graph!.addNode({
      id: stage.name,
      x,
      y,
      width: 120,
      height: 60,
      label: stage.label || stage.name,
      attrs: {
        body: {
          fill: '#1890ff',
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

  // 渲染现有的依赖关系
  Object.entries(formData.dependency_graph).forEach(([target, sources]) => {
    sources.forEach(source => {
      const sourceNode = graph!.getCellById(source)
      const targetNode = graph!.getCellById(target)
      
      if (sourceNode && targetNode) {
        graph!.addEdge({
          source: sourceNode,
          target: targetNode,
        })
      }
    })
  })
}

const autoLayout = () => {
  if (!graph) return
  
  // 简单的自动布局
  const nodes = graph.getNodes()
  const spacing = 150
  const startX = 50
  const startY = 100

  nodes.forEach((node, index) => {
    const x = startX + (index % 3) * spacing
    const y = startY + Math.floor(index / 3) * 100
    node.setPosition({ x, y })
  })

  message.success('布局已更新')
}

const removeDependency = (record: any) => {
  const { source, target } = record
  
  if (formData.dependency_graph[target]) {
    const index = formData.dependency_graph[target].indexOf(source)
    if (index > -1) {
      formData.dependency_graph[target].splice(index, 1)
    }
  }

  // 从图中删除对应的边
  if (graph) {
    const edges = graph.getEdges()
    edges.forEach(edge => {
      const edgeSource = edge.getSourceNode()
      const edgeTarget = edge.getTargetNode()
      
      if (edgeSource?.id === source && edgeTarget?.id === target) {
        graph!.removeEdge(edge)
      }
    })
  }

  message.success('依赖关系已删除')
}

const getConfigPreview = () => {
  return {
    enabled: formData.enabled,
    max_parallel: formData.max_parallel,
    fail_fast: formData.fail_fast,
    parallel_stages: formData.parallel_stages,
    dependency_graph: formData.dependency_graph,
  }
}

const loadConfig = async () => {
  try {
    const res = await request.get(`/build/parallel/${pipelineId.value}`)
    if (res?.data) {
      Object.assign(formData, res.data)
      
      // 重新渲染图
      if (graph && formData.enabled) {
        renderStages()
      }
    }
  } catch (error) {
    console.error('加载配置失败:', error)
  }
}

const loadPipelineInfo = async () => {
  try {
    const res = await request.get(`/pipelines/${pipelineId.value}`)
    pipelineName.value = res?.data?.name || ''
  } catch (error) {
    console.error('加载流水线信息失败:', error)
  }
}

const saveConfig = async () => {
  // 验证配置
  if (formData.enabled && formData.parallel_stages.length === 0) {
    message.warning('请至少选择一个可并行阶段')
    return
  }

  // 检查循环依赖
  if (hasCircularDependency()) {
    message.error('检测到循环依赖，请修改依赖关系')
    return
  }

  saving.value = true
  try {
    await request.put(`/build/parallel/${pipelineId.value}`, formData)
    message.success('配置保存成功')
  } catch (error: any) {
    message.error(error?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

const hasCircularDependency = (): boolean => {
  const visited = new Set<string>()
  const recursionStack = new Set<string>()

  const dfs = (node: string): boolean => {
    visited.add(node)
    recursionStack.add(node)

    const dependencies = formData.dependency_graph[node] || []
    for (const dep of dependencies) {
      if (!visited.has(dep)) {
        if (dfs(dep)) return true
      } else if (recursionStack.has(dep)) {
        return true
      }
    }

    recursionStack.delete(node)
    return false
  }

  for (const node of formData.parallel_stages) {
    if (!visited.has(node)) {
      if (dfs(node)) return true
    }
  }

  return false
}

const handleResize = () => {
  if (graph && graphContainer.value) {
    graph.resize(graphContainer.value.clientWidth, 400)
  }
}

onMounted(() => {
  loadPipelineInfo()
  loadConfig()
  
  // 延迟初始化图，确保 DOM 已渲染
  setTimeout(() => {
    initGraph()
  }, 100)

  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  graph?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.parallel-config {
  padding: 0;
}

.form-tip {
  margin-left: 12px;
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
}

.dependency-editor {
  width: 100%;
}

.graph-container {
  width: 100%;
  height: 400px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fafafa;
}

.config-preview {
  background: #f5f5f5;
  padding: 16px;
  border-radius: 4px;
  font-size: 12px;
  max-height: 400px;
  overflow: auto;
  margin: 0;
}

:deep(.ant-slider-mark-text) {
  font-size: 12px;
}

@media (max-width: 768px) {
  :deep(.ant-form-item-label) {
    text-align: left;
  }
}
</style>
