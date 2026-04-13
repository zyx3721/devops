<template>
  <div class="pipeline-detail">
    <a-page-header :title="pipeline?.name || '流水线详情'" @back="goBack">
      <template #subTitle>
        <a-space>
          <a-tag :color="pipeline?.status === 'active' ? 'green' : 'default'">
            {{ pipeline?.status === 'active' ? '启用' : '禁用' }}
          </a-tag>
          <span v-if="pipeline?.description" style="color: #999">{{ pipeline.description }}</span>
        </a-space>
      </template>
      <template #extra>
        <a-space>
          <a-button @click="runPipeline" type="primary" :disabled="pipeline?.status !== 'active'">
            <PlayCircleOutlined /> 执行
          </a-button>
          <a-button @click="editPipeline">
            <EditOutlined /> 编辑
          </a-button>
          <a-dropdown>
            <a-button><MoreOutlined /></a-button>
            <template #overlay>
              <a-menu>
                <a-menu-item @click="toggleStatus">
                  {{ pipeline?.status === 'active' ? '禁用' : '启用' }}
                </a-menu-item>
                <a-menu-divider />
                <a-menu-item danger @click="confirmDelete">删除</a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </a-space>
      </template>
    </a-page-header>

    <a-row :gutter="16">
      <!-- 左侧：流水线信息 -->
      <a-col :span="8">
        <a-card title="基本信息" size="small" :loading="loading">
          <a-descriptions :column="1" size="small">
            <a-descriptions-item label="流水线ID">{{ pipeline?.id }}</a-descriptions-item>
            <a-descriptions-item label="Git 仓库">
              <template v-if="pipeline?.git_repo_url">
                <a :href="pipeline.git_repo_url" target="_blank">{{ pipeline.git_repo_name || getRepoName(pipeline.git_repo_url) }}</a>
              </template>
              <span v-else style="color: #999">-</span>
            </a-descriptions-item>
            <a-descriptions-item label="默认分支">{{ pipeline?.git_branch || 'main' }}</a-descriptions-item>
            <a-descriptions-item label="构建集群">
              <router-link v-if="pipeline?.build_cluster_id" :to="`/k8s/clusters/${pipeline.build_cluster_id}/resources`">
                {{ pipeline.build_cluster_name }}
              </router-link>
              <span v-else style="color: #999">-</span>
            </a-descriptions-item>
            <a-descriptions-item label="构建命名空间">
              <router-link 
                v-if="pipeline?.build_cluster_id && pipeline?.build_namespace" 
                :to="`/k8s/clusters/${pipeline.build_cluster_id}/resources?namespace=${pipeline.build_namespace}`"
              >
                {{ pipeline.build_namespace }}
              </router-link>
              <span v-else style="color: #999">{{ pipeline?.build_namespace || '-' }}</span>
            </a-descriptions-item>
            <a-descriptions-item label="创建时间">{{ formatTime(pipeline?.created_at) }}</a-descriptions-item>
            <a-descriptions-item label="更新时间">{{ formatTime(pipeline?.updated_at) }}</a-descriptions-item>
          </a-descriptions>
        </a-card>

        <a-card title="阶段配置" size="small" style="margin-top: 16px" :loading="loading">
          <a-timeline v-if="pipeline?.stages?.length">
            <a-timeline-item v-for="stage in pipeline.stages" :key="stage.id" color="blue">
              <div style="font-weight: 500">{{ stage.name }}</div>
              <div style="color: #999; font-size: 12px">{{ stage.steps?.length || 0 }} 个步骤</div>
            </a-timeline-item>
          </a-timeline>
          <a-empty v-else description="暂无阶段配置" :image="Empty.PRESENTED_IMAGE_SIMPLE" />
        </a-card>
      </a-col>

      <!-- 右侧：执行历史 -->
      <a-col :span="16">
        <a-card title="执行历史" size="small">
          <template #extra>
            <a-button type="link" size="small" @click="loadRuns">
              <ReloadOutlined /> 刷新
            </a-button>
          </template>

          <a-table
            :columns="runColumns"
            :data-source="runs"
            :loading="runsLoading"
            :pagination="runsPagination"
            @change="handleRunsTableChange"
            row-key="id"
            size="small"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'id'">
                <a @click="showRunDetail(record)">#{{ record.id }}</a>
              </template>
              <template v-if="column.key === 'status'">
                <a-tag :color="getStatusColor(record.status)">{{ getStatusText(record.status) }}</a-tag>
              </template>
              <template v-if="column.key === 'trigger'">
                <span>{{ record.trigger_type }} / {{ record.trigger_by }}</span>
              </template>
              <template v-if="column.key === 'duration'">
                {{ formatDuration(record.duration) }}
              </template>
              <template v-if="column.key === 'started_at'">
                {{ record.started_at || '-' }}
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="showRunDetail(record)">日志</a-button>
                  <a-button type="link" size="small" @click="cancelRun(record)" v-if="record.status === 'running'" danger>取消</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>

    <!-- 执行流水线弹窗 -->
    <a-modal v-model:open="showRunModal" title="执行流水线" @ok="handleRun" :confirmLoading="running">
      <a-form :label-col="{ span: 6 }">
        <a-form-item label="Git Ref" v-if="pipeline?.git_repo_id">
          <a-radio-group v-model:value="runForm.ref_type" style="margin-bottom: 8px">
            <a-radio-button value="branch">分支</a-radio-button>
            <a-radio-button value="tag">Tag</a-radio-button>
          </a-radio-group>
          <a-auto-complete
            v-model:value="runForm.ref"
            :options="refOptions"
            :placeholder="runForm.ref_type === 'branch' ? '选择或输入分支' : '选择或输入 Tag'"
            style="width: 100%"
          />
        </a-form-item>
        <a-form-item label="参数 (JSON)">
          <a-textarea v-model:value="runForm.parameters_json" placeholder='{"key": "value"}' :rows="4" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 执行详情抽屉 -->
    <a-drawer v-model:open="runDetailVisible" title="执行详情" width="700" :footer="null">
      <template v-if="currentRun">
        <a-descriptions :column="2" size="small" style="margin-bottom: 16px">
          <a-descriptions-item label="执行ID">#{{ currentRun.id }}</a-descriptions-item>
          <a-descriptions-item label="状态">
            <a-tag :color="getStatusColor(currentRun.status)">{{ getStatusText(currentRun.status) }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="触发方式">{{ currentRun.trigger_type }}</a-descriptions-item>
          <a-descriptions-item label="触发者">{{ currentRun.trigger_by }}</a-descriptions-item>
          <a-descriptions-item label="开始时间">{{ currentRun.started_at || '-' }}</a-descriptions-item>
          <a-descriptions-item label="耗时">{{ formatDuration(currentRun.duration) }}</a-descriptions-item>
        </a-descriptions>

        <a-collapse v-model:activeKey="logsActiveKey" accordion>
          <a-collapse-panel v-for="stage in currentRun.stage_runs" :key="stage.id" :header="stage.stage_name">
            <template #extra>
              <a-tag :color="getStatusColor(stage.status)" size="small">{{ getStatusText(stage.status) }}</a-tag>
            </template>
            <div v-for="step in stage.step_runs" :key="step.id" class="step-log-item">
              <div class="step-header">
                <span>{{ step.step_name }}</span>
                <a-tag :color="getStatusColor(step.status)" size="small">{{ getStatusText(step.status) }}</a-tag>
              </div>
              <pre class="step-logs">{{ step.logs || '暂无日志' }}</pre>
            </div>
          </a-collapse-panel>
        </a-collapse>
      </template>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal, Empty } from 'ant-design-vue'
import { PlayCircleOutlined, EditOutlined, MoreOutlined, ReloadOutlined } from '@ant-design/icons-vue'
import { pipelineApi, gitRepoApi } from '@/services/pipeline'

const route = useRoute()
const router = useRouter()
const pipelineId = ref(Number(route.params.id))

const loading = ref(false)
const runsLoading = ref(false)
const running = ref(false)
const pipeline = ref<any>(null)
const runs = ref<any[]>([])
const showRunModal = ref(false)
const runDetailVisible = ref(false)
const currentRun = ref<any>(null)
const logsActiveKey = ref<number[]>([])
const branches = ref<string[]>([])
const tags = ref<string[]>([])
const refOptions = ref<{value: string}[]>([])

const runForm = reactive({
  ref_type: 'branch' as 'branch' | 'tag',
  ref: '',
  parameters_json: '{}'
})

const runsPagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0
})

const runColumns = [
  { title: '#', key: 'id', width: 80 },
  { title: '状态', key: 'status', width: 100 },
  { title: '触发', key: 'trigger', width: 150 },
  { title: '耗时', key: 'duration', width: 100 },
  { title: '开始时间', key: 'started_at', width: 180 },
  { title: '操作', key: 'action', width: 120 }
]

const getStatusColor = (status: string) => ({ success: 'green', running: 'blue', failed: 'red', cancelled: 'orange', pending: 'default' }[status] || 'default')
const getStatusText = (status: string) => ({ success: '成功', running: '运行中', failed: '失败', cancelled: '已取消', pending: '等待中' }[status] || status)
const formatDuration = (seconds: number) => {
  if (!seconds) return '-'
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分${seconds % 60}秒`
  return `${Math.floor(seconds / 3600)}时${Math.floor((seconds % 3600) / 60)}分`
}
const formatTime = (time: string) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}
const getRepoName = (url: string) => {
  if (!url) return ''
  const match = url.match(/[:/]([^/:]+\/[^/.]+)(\.git)?$/)
  return match ? match[1] : url
}

const loadPipeline = async () => {
  loading.value = true
  try {
    const res = await pipelineApi.get(pipelineId.value)
    pipeline.value = res?.data || res
  } catch (error) {
    message.error('加载流水线失败')
  } finally {
    loading.value = false
  }
}

const loadRuns = async () => {
  runsLoading.value = true
  try {
    const res = await pipelineApi.listRuns({
      pipeline_id: pipelineId.value,
      page: runsPagination.current,
      page_size: runsPagination.pageSize
    })
    runs.value = res?.data?.items || []
    runsPagination.total = res?.data?.total || 0
  } catch (error) {
    console.error('加载执行历史失败', error)
  } finally {
    runsLoading.value = false
  }
}

const loadBranchesAndTags = async () => {
  if (!pipeline.value?.git_repo_id) return
  try {
    const [branchRes, tagRes] = await Promise.allSettled([
      gitRepoApi.getBranches(pipeline.value.git_repo_id),
      gitRepoApi.getTags(pipeline.value.git_repo_id)
    ])
    if (branchRes.status === 'fulfilled') {
      branches.value = (branchRes.value?.data || []).map((item: any) => typeof item === 'string' ? item : item.name)
    }
    if (tagRes.status === 'fulfilled') {
      tags.value = (tagRes.value?.data || []).map((item: any) => typeof item === 'string' ? item : item.name)
    }
    refOptions.value = branches.value.map(b => ({ value: b }))
  } catch (error) {
    console.error('加载分支/Tag失败', error)
  }
}

watch(() => runForm.ref_type, (type) => {
  refOptions.value = type === 'branch' 
    ? branches.value.map(b => ({ value: b }))
    : tags.value.map(t => ({ value: t }))
  runForm.ref = type === 'branch' ? (pipeline.value?.git_branch || 'main') : (tags.value[0] || '')
})

const runPipeline = () => {
  runForm.ref = pipeline.value?.git_branch || 'main'
  runForm.ref_type = 'branch'
  runForm.parameters_json = '{}'
  showRunModal.value = true
  loadBranchesAndTags()
}

const handleRun = async () => {
  running.value = true
  try {
    let params = {}
    if (runForm.parameters_json) {
      params = JSON.parse(runForm.parameters_json)
    }
    await pipelineApi.run(pipelineId.value, { parameters: params, branch: runForm.ref || undefined })
    message.success('流水线已开始执行')
    showRunModal.value = false
    loadRuns()
  } catch (error: any) {
    message.error(error.message || '执行失败')
  } finally {
    running.value = false
  }
}

const showRunDetail = async (record: any) => {
  try {
    const res = await pipelineApi.getRun(record.id)
    currentRun.value = res?.data || res
    if (currentRun.value?.stage_runs?.length > 0) {
      logsActiveKey.value = [currentRun.value.stage_runs[0].id]
    }
    runDetailVisible.value = true
  } catch (error) {
    message.error('加载执行详情失败')
  }
}

const cancelRun = async (record: any) => {
  Modal.confirm({
    title: '确认取消',
    content: '确定要取消此次执行吗？',
    onOk: async () => {
      try {
        await pipelineApi.cancelRun(record.id)
        message.success('已取消')
        loadRuns()
      } catch (error: any) {
        message.error(error.message || '取消失败')
      }
    }
  })
}

const editPipeline = () => router.push(`/pipeline/edit/${pipelineId.value}`)
const goBack = () => router.push('/pipeline/list')

const toggleStatus = async () => {
  try {
    await pipelineApi.toggle(pipelineId.value)
    message.success('状态已更新')
    loadPipeline()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  }
}

const confirmDelete = () => {
  Modal.confirm({
    title: '确认删除',
    content: `确定要删除流水线 "${pipeline.value?.name}" 吗？`,
    okType: 'danger',
    onOk: async () => {
      try {
        await pipelineApi.delete(pipelineId.value)
        message.success('删除成功')
        router.push('/pipeline/list')
      } catch (error: any) {
        message.error(error.message || '删除失败')
      }
    }
  })
}

const handleRunsTableChange = (pag: any) => {
  runsPagination.current = pag.current
  runsPagination.pageSize = pag.pageSize
  loadRuns()
}

onMounted(() => {
  loadPipeline()
  loadRuns()
})
</script>

<style scoped>
.pipeline-detail {
  padding: 0;
}
.step-log-item {
  margin-bottom: 12px;
  border: 1px solid #f0f0f0;
  border-radius: 4px;
  overflow: hidden;
}
.step-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: #fafafa;
  border-bottom: 1px solid #f0f0f0;
}
.step-logs {
  margin: 0;
  padding: 12px;
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 12px;
  max-height: 300px;
  overflow: auto;
  white-space: pre-wrap;
  word-wrap: break-word;
}
</style>
