<template>
  <div class="pipeline-history">
    <div class="page-header">
      <h2>执行历史</h2>
      <div class="header-actions">
        <el-select v-model="filter.status" placeholder="状态" clearable style="width: 120px">
          <el-option label="全部" value="" />
          <el-option label="运行中" value="running" />
          <el-option label="成功" value="success" />
          <el-option label="失败" value="failed" />
          <el-option label="已取消" value="cancelled" />
        </el-select>
        <el-date-picker
          v-model="filter.dateRange"
          type="daterange"
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          style="width: 240px"
        />
        <el-button @click="fetchHistory">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <el-table :data="historyList" v-loading="loading" stripe>
      <el-table-column prop="id" label="运行ID" width="80" />
      <el-table-column prop="pipeline_name" label="流水线" min-width="150" />
      <el-table-column prop="trigger_type" label="触发方式" width="100">
        <template #default="{ row }">
          <el-tag :type="getTriggerType(row.trigger_type)" size="small">
            {{ getTriggerLabel(row.trigger_type) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="status" label="状态" width="100">
        <template #default="{ row }">
          <el-tag :type="getStatusType(row.status)" size="small">
            {{ getStatusLabel(row.status) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="duration" label="耗时" width="100">
        <template #default="{ row }">
          {{ formatDuration(row.duration) }}
        </template>
      </el-table-column>
      <el-table-column prop="triggered_by" label="触发者" width="120" />
      <el-table-column prop="started_at" label="开始时间" width="180">
        <template #default="{ row }">
          {{ formatTime(row.started_at) }}
        </template>
      </el-table-column>
      <el-table-column label="操作" width="150" fixed="right">
        <template #default="{ row }">
          <el-button type="primary" link @click="showDetail(row)">详情</el-button>
          <el-button type="primary" link @click="showLogs(row)">日志</el-button>
          <el-button 
            v-if="row.status === 'running'" 
            type="danger" 
            link 
            @click="cancelRun(row)"
          >
            取消
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      v-model:current-page="pagination.page"
      v-model:page-size="pagination.pageSize"
      :total="pagination.total"
      :page-sizes="[10, 20, 50, 100]"
      layout="total, sizes, prev, pager, next"
      @size-change="fetchHistory"
      @current-change="fetchHistory"
      style="margin-top: 16px; justify-content: flex-end"
    />

    <!-- 执行详情抽屉 -->
    <el-drawer v-model="detailDrawer.visible" title="执行详情" size="60%">
      <div v-if="detailDrawer.data" class="run-detail">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="运行ID">{{ detailDrawer.data.id }}</el-descriptions-item>
          <el-descriptions-item label="流水线">{{ detailDrawer.data.pipeline_name }}</el-descriptions-item>
          <el-descriptions-item label="状态">
            <el-tag :type="getStatusType(detailDrawer.data.status)">
              {{ getStatusLabel(detailDrawer.data.status) }}
            </el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="触发方式">
            {{ getTriggerLabel(detailDrawer.data.trigger_type) }}
          </el-descriptions-item>
          <el-descriptions-item label="触发者">{{ detailDrawer.data.triggered_by }}</el-descriptions-item>
          <el-descriptions-item label="耗时">{{ formatDuration(detailDrawer.data.duration) }}</el-descriptions-item>
          <el-descriptions-item label="开始时间">{{ formatTime(detailDrawer.data.started_at) }}</el-descriptions-item>
          <el-descriptions-item label="结束时间">{{ formatTime(detailDrawer.data.finished_at) }}</el-descriptions-item>
        </el-descriptions>

        <h4 style="margin: 16px 0 8px">阶段执行进度</h4>
        <el-timeline>
          <el-timeline-item
            v-for="stage in detailDrawer.data.stage_runs"
            :key="stage.id"
            :type="getStatusType(stage.status)"
            :timestamp="formatTime(stage.started_at)"
          >
            <div class="stage-item">
              <span class="stage-name">{{ stage.stage_name }}</span>
              <el-tag :type="getStatusType(stage.status)" size="small">
                {{ getStatusLabel(stage.status) }}
              </el-tag>
              <span class="stage-duration">{{ formatDuration(stage.duration) }}</span>
            </div>
            <div v-if="stage.step_runs" class="step-list">
              <div 
                v-for="step in stage.step_runs" 
                :key="step.id" 
                class="step-item"
                @click="showStepLogs(step)"
              >
                <el-icon :class="getStepIconClass(step.status)">
                  <component :is="getStepIcon(step.status)" />
                </el-icon>
                <span>{{ step.step_name }}</span>
                <span class="step-duration">{{ formatDuration(step.duration) }}</span>
              </div>
            </div>
          </el-timeline-item>
        </el-timeline>
      </div>
    </el-drawer>

    <!-- 日志查看器 -->
    <el-dialog v-model="logDialog.visible" :title="logDialog.title" width="80%" top="5vh">
      <log-viewer
        v-if="logDialog.visible"
        :run-id="logDialog.runId"
        :step-run-id="logDialog.stepRunId"
      />
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Check, Close, Loading, Warning } from '@element-plus/icons-vue'
import { pipelineApi } from '@/services/pipeline'
import LogViewer from '@/components/pipeline/LogViewer.vue'

const loading = ref(false)
const historyList = ref([])

const filter = reactive({
  status: '',
  dateRange: null
})

const pagination = reactive({
  page: 1,
  pageSize: 20,
  total: 0
})

const detailDrawer = reactive({
  visible: false,
  data: null
})

const logDialog = reactive({
  visible: false,
  title: '',
  runId: null,
  stepRunId: null
})

const fetchHistory = async () => {
  loading.value = true
  try {
    const params = {
      page: pagination.page,
      page_size: pagination.pageSize,
      status: filter.status
    }
    if (filter.dateRange) {
      params.start_time = filter.dateRange[0]
      params.end_time = filter.dateRange[1]
    }
    const res = await pipelineApi.listRuns(params)
    historyList.value = res.data.list || []
    pagination.total = res.data.total || 0
  } catch (error) {
    ElMessage.error('获取执行历史失败')
  } finally {
    loading.value = false
  }
}

const showDetail = async (row) => {
  try {
    const res = await pipelineApi.getRunDetail(row.id)
    detailDrawer.data = res.data
    detailDrawer.visible = true
  } catch (error) {
    ElMessage.error('获取详情失败')
  }
}

const showLogs = (row) => {
  logDialog.title = `运行日志 - ${row.pipeline_name} #${row.id}`
  logDialog.runId = row.id
  logDialog.stepRunId = null
  logDialog.visible = true
}

const showStepLogs = (step) => {
  logDialog.title = `步骤日志 - ${step.step_name}`
  logDialog.runId = step.pipeline_run_id
  logDialog.stepRunId = step.id
  logDialog.visible = true
}

const cancelRun = async (row) => {
  try {
    await ElMessageBox.confirm('确定要取消此次运行吗？', '确认取消')
    await pipelineApi.cancelRun(row.id)
    ElMessage.success('已取消')
    fetchHistory()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('取消失败')
    }
  }
}

const getStatusType = (status) => {
  const map = {
    pending: 'info',
    running: 'warning',
    success: 'success',
    failed: 'danger',
    cancelled: 'info'
  }
  return map[status] || 'info'
}

const getStatusLabel = (status) => {
  const map = {
    pending: '等待中',
    running: '运行中',
    success: '成功',
    failed: '失败',
    cancelled: '已取消'
  }
  return map[status] || status
}

const getTriggerType = (type) => {
  const map = {
    manual: '',
    webhook: 'success',
    schedule: 'warning'
  }
  return map[type] || ''
}

const getTriggerLabel = (type) => {
  const map = {
    manual: '手动触发',
    webhook: 'Webhook',
    schedule: '定时触发'
  }
  return map[type] || type
}

const getStepIcon = (status) => {
  const map = {
    success: Check,
    failed: Close,
    running: Loading,
    pending: Warning
  }
  return map[status] || Warning
}

const getStepIconClass = (status) => {
  return `step-icon step-icon-${status}`
}

const formatDuration = (seconds) => {
  if (!seconds) return '-'
  if (seconds < 60) return `${seconds}秒`
  const minutes = Math.floor(seconds / 60)
  const secs = seconds % 60
  if (minutes < 60) return `${minutes}分${secs}秒`
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  return `${hours}时${mins}分`
}

const formatTime = (time) => {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  fetchHistory()
})
</script>

<style scoped>
.pipeline-history {
  padding: 20px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 12px;
}

.run-detail {
  padding: 0 20px;
}

.stage-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.stage-name {
  font-weight: 500;
}

.stage-duration {
  color: #909399;
  font-size: 12px;
}

.step-list {
  margin-top: 8px;
  padding-left: 20px;
}

.step-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  cursor: pointer;
  border-radius: 4px;
}

.step-item:hover {
  background: #f5f7fa;
}

.step-duration {
  margin-left: auto;
  color: #909399;
  font-size: 12px;
}

.step-icon-success {
  color: #67c23a;
}

.step-icon-failed {
  color: #f56c6c;
}

.step-icon-running {
  color: #e6a23c;
  animation: spin 1s linear infinite;
}

.step-icon-pending {
  color: #909399;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
