<template>
  <div class="deploy-locks">
    <a-card title="发布锁管理">
      <template #extra>
        <a-button @click="loadLocks">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
      </template>

      <a-alert v-if="locks.length === 0 && !loading" message="当前没有活跃的发布锁" type="info" show-icon style="margin-bottom: 16px" />

      <a-table :columns="columns" :data-source="locks" :loading="loading" row-key="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'env_name'">
            <a-tag :color="getEnvColor(record.env_name)">{{ record.env_name }}</a-tag>
          </template>
          <template v-else-if="column.key === 'status'">
            <a-tag :color="record.status === 'active' ? 'green' : 'default'">
              {{ record.status === 'active' ? '活跃' : record.status }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'expires_at'">
            <span :style="{ color: isExpiringSoon(record.expires_at) ? 'orange' : '' }">
              {{ formatTime(record.expires_at) }}
            </span>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-popconfirm title="确定强制释放此锁？" @confirm="showReleaseModal(record)">
              <a-button type="link" size="small" danger>强制释放</a-button>
            </a-popconfirm>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 释放原因弹窗 -->
    <a-modal v-model:open="releaseModalVisible" title="强制释放锁" @ok="handleRelease" :confirm-loading="submitting">
      <a-form :label-col="{ span: 6 }">
        <a-form-item label="应用ID">{{ currentLock?.application_id }}</a-form-item>
        <a-form-item label="环境">{{ currentLock?.env_name }}</a-form-item>
        <a-form-item label="锁定人">{{ currentLock?.locked_by_name }}</a-form-item>
        <a-form-item label="释放原因" required>
          <a-textarea v-model:value="releaseReason" placeholder="请填写释放原因" :rows="3" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { ReloadOutlined } from '@ant-design/icons-vue'
import { deployLockApi, type DeployLock } from '@/services/approval'
import dayjs from 'dayjs'

const loading = ref(false)
const submitting = ref(false)
const locks = ref<DeployLock[]>([])
const currentLock = ref<DeployLock | null>(null)
const releaseModalVisible = ref(false)
const releaseReason = ref('')

const columns = [
  { title: '应用ID', dataIndex: 'application_id' },
  { title: '环境', key: 'env_name', dataIndex: 'env_name' },
  { title: '关联记录', dataIndex: 'record_id' },
  { title: '锁定人', dataIndex: 'locked_by_name' },
  { title: '状态', key: 'status', dataIndex: 'status' },
  { title: '过期时间', key: 'expires_at', dataIndex: 'expires_at' },
  { title: '创建时间', key: 'created_at', dataIndex: 'created_at' },
  { title: '操作', key: 'action', width: 120 }
]

const getEnvColor = (env: string) => {
  const colors: Record<string, string> = {
    dev: 'blue', test: 'cyan', staging: 'orange', prod: 'red', production: 'red'
  }
  return colors[env] || 'default'
}

const formatTime = (time: string) => {
  return time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'
}

const isExpiringSoon = (time: string) => {
  if (!time) return false
  return dayjs(time).diff(dayjs(), 'minute') < 10
}

const loadLocks = async () => {
  loading.value = true
  try {
    const res = await deployLockApi.list()
    locks.value = res.data || []
  } finally {
    loading.value = false
  }
}

const showReleaseModal = (lock: DeployLock) => {
  currentLock.value = lock
  releaseReason.value = ''
  releaseModalVisible.value = true
}

const handleRelease = async () => {
  if (!releaseReason.value) {
    message.error('请填写释放原因')
    return
  }
  if (!currentLock.value) return

  submitting.value = true
  try {
    await deployLockApi.forceRelease(currentLock.value.application_id, currentLock.value.env_name, releaseReason.value)
    message.success('释放成功')
    releaseModalVisible.value = false
    loadLocks()
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadLocks()
})
</script>

<style scoped>
.deploy-locks {
  padding: 16px;
}
</style>
