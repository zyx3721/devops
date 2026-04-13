<template>
  <div class="jenkins-jobs">
    <div class="page-header">
      <div class="header-left">
        <a-button @click="goBack" style="margin-right: 12px;">
          <template #icon><ArrowLeftOutlined /></template>
        </a-button>
        <h1>{{ instanceName }}</h1>
      </div>
      <a-button @click="fetchJobs" :loading="loading">
        <template #icon><ReloadOutlined /></template>
        刷新
      </a-button>
    </div>

    <div class="content-wrapper">
      <div class="job-list-panel">
        <div class="search-box">
          <a-input-search v-model:value="searchKeyword" placeholder="搜索 Job" allow-clear />
        </div>
        <div class="job-list">
          <div v-if="loading" class="loading-wrapper">
            <a-spin />
          </div>
          <template v-else>
            <div v-if="filteredJobs.length" class="job-items">
              <div 
                v-for="job in filteredJobs" 
                :key="job.name" 
                :class="['job-item', { active: selectedJob?.name === job.name }]"
                @click="selectJob(job)"
              >
                <a-badge :status="getStatusType(job.color)" />
                <span class="job-name" :title="job.name">{{ job.name }}</span>
              </div>
            </div>
            <a-empty v-else :description="searchKeyword ? '未找到匹配的 Job' : '暂无 Job'" style="margin-top: 60px;" />
          </template>
        </div>
      </div>

      <div class="build-content">
        <template v-if="selectedJob">
          <div class="build-header">
            <h2>{{ selectedJob.name }}</h2>
            <span class="build-count">共 {{ builds.length }} 条构建记录</span>
          </div>
          <a-spin :spinning="buildsLoading">
            <a-table 
              :columns="buildColumns" 
              :data-source="builds" 
              :pagination="{ pageSize: 15, showSizeChanger: false }" 
              row-key="number" 
              size="small"
            >
              <template #bodyCell="{ column, record }">
                <template v-if="column.key === 'number'">
                  <a-tag :color="getBuildResultColor(record.result)">
                    #{{ record.number }}
                  </a-tag>
                </template>
                <template v-if="column.key === 'result'">
                  <span v-if="record.building" style="color: #1890ff;">
                    <LoadingOutlined style="margin-right: 4px;" />构建中
                  </span>
                  <span v-else>{{ record.result || '-' }}</span>
                </template>
                <template v-if="column.key === 'duration'">
                  {{ record.building ? '-' : formatDuration(record.duration) }}
                </template>
              </template>
            </a-table>
            <a-empty v-if="!buildsLoading && !builds.length" description="暂无构建记录" />
          </a-spin>
        </template>
        <div v-else class="empty-placeholder">
          <a-empty description="请从左侧选择一个 Job 查看构建历史" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { ArrowLeftOutlined, ReloadOutlined, LoadingOutlined } from '@ant-design/icons-vue'
import { jenkinsInstanceApi, jenkinsJobApi, type JenkinsJob, type JenkinsBuild } from '@/services/jenkins'

const route = useRoute()
const router = useRouter()
const instanceId = Number(route.params.id)

const loading = ref(false)
const instanceName = ref('')
const jobs = ref<JenkinsJob[]>([])
const searchKeyword = ref('')
const selectedJob = ref<JenkinsJob | null>(null)

const buildsLoading = ref(false)
const builds = ref<JenkinsBuild[]>([])

const filteredJobs = computed(() => {
  if (!searchKeyword.value) return jobs.value
  const keyword = searchKeyword.value.toLowerCase()
  return jobs.value.filter(job => job.name.toLowerCase().includes(keyword))
})

const buildColumns = [
  { title: '构建号', key: 'number', width: 100 },
  { title: '状态', key: 'result', width: 120 },
  { title: '开始时间', dataIndex: 'timestamp', key: 'timestamp' },
  { title: '耗时', key: 'duration', width: 120 }
]

const goBack = () => {
  router.push('/jenkins/instances')
}

const fetchInstance = async () => {
  try {
    const res = await jenkinsInstanceApi.getInstance(instanceId)
    if (res.code === 0 && res.data) {
      instanceName.value = res.data.name
    }
  } catch {}
}

const fetchJobs = async () => {
  loading.value = true
  try {
    const res = await jenkinsJobApi.getJobs(instanceId)
    if (res.code === 0) {
      jobs.value = res.data || []
      if (selectedJob.value) {
        fetchBuilds(selectedJob.value.name)
      }
    }
  } catch (error: any) {
    message.error(error.message || '获取 Job 列表失败')
  } finally {
    loading.value = false
  }
}

const selectJob = (job: JenkinsJob) => {
  selectedJob.value = job
  fetchBuilds(job.name)
}

const fetchBuilds = async (jobName: string) => {
  buildsLoading.value = true
  try {
    const res = await jenkinsJobApi.getJobBuilds(instanceId, jobName)
    if (res.code === 0) {
      builds.value = res.data || []
    }
  } catch (error: any) {
    message.error(error.message || '获取构建历史失败')
  } finally {
    buildsLoading.value = false
  }
}

const getStatusType = (color: string): 'success' | 'processing' | 'error' | 'warning' | 'default' => {
  if (color?.includes('blue')) return 'success'
  if (color?.includes('anime')) return 'processing'
  if (color?.includes('red')) return 'error'
  if (color?.includes('yellow')) return 'warning'
  return 'default'
}

const getBuildResultColor = (result: string): string => {
  switch (result) {
    case 'SUCCESS': return 'green'
    case 'FAILURE': return 'red'
    case 'UNSTABLE': return 'orange'
    case 'ABORTED': return 'gray'
    default: return 'blue'
  }
}

const formatDuration = (seconds: number): string => {
  if (seconds < 60) return `${seconds}秒`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}分${seconds % 60}秒`
  return `${Math.floor(seconds / 3600)}时${Math.floor((seconds % 3600) / 60)}分`
}

onMounted(() => {
  fetchInstance()
  fetchJobs()
})
</script>

<style scoped>
.jenkins-jobs {
  height: calc(100vh - 120px);
  display: flex;
  flex-direction: column;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: center;
}

.page-header h1 {
  font-size: 20px;
  font-weight: 500;
  margin: 0;
}

.content-wrapper {
  display: flex;
  flex: 1;
  gap: 16px;
  min-height: 0;
}

.job-list-panel {
  width: 300px;
  flex-shrink: 0;
  background: #fff;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  border: 1px solid #f0f0f0;
}

.search-box {
  padding: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.job-list {
  flex: 1;
  overflow: auto;
}

.loading-wrapper {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 40px 0;
}

.job-items {
  display: flex;
  flex-direction: column;
}

.job-item {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 0.2s;
  gap: 8px;
}

.job-item:hover {
  background: #f5f5f5;
}

.job-item.active {
  background: #e6f7ff;
  border-right: 3px solid #1890ff;
}

.job-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.build-content {
  flex: 1;
  background: #fff;
  border-radius: 6px;
  padding: 16px;
  overflow: auto;
  border: 1px solid #f0f0f0;
  display: flex;
  flex-direction: column;
}

.build-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.build-header h2 {
  margin: 0;
  font-size: 16px;
  font-weight: 500;
}

.build-count {
  color: #999;
  font-size: 13px;
}

.empty-placeholder {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>
