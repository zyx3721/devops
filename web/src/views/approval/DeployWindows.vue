<template>
  <div class="deploy-windows">
    <a-card title="发布窗口管理">
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <template #icon><PlusOutlined /></template>
          新建窗口
        </a-button>
      </template>

      <a-table :columns="columns" :data-source="windows" :loading="loading" row-key="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'app_id'">
            {{ record.app_id === 0 ? '全局' : `应用 #${record.app_id}` }}
          </template>
          <template v-else-if="column.key === 'env'">
            <a-tag :color="getEnvColor(record.env)">{{ record.env }}</a-tag>
          </template>
          <template v-else-if="column.key === 'weekdays'">
            {{ formatWeekdays(record.weekdays) }}
          </template>
          <template v-else-if="column.key === 'time_range'">
            {{ record.start_time }} - {{ record.end_time }}
          </template>
          <template v-else-if="column.key === 'allow_emergency'">
            <a-tag :color="record.allow_emergency ? 'green' : 'red'">
              {{ record.allow_emergency ? '允许' : '禁止' }}
            </a-tag>
          </template>
          <template v-else-if="column.key === 'enabled'">
            <a-switch :checked="record.enabled" @change="toggleEnabled(record)" />
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showEditModal(record)">编辑</a-button>
              <a-popconfirm title="确定删除此窗口？" @confirm="deleteWindow(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 创建/编辑弹窗 -->
    <a-modal v-model:open="modalVisible" :title="isEdit ? '编辑窗口' : '新建窗口'" @ok="handleSubmit" :confirm-loading="submitting">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="应用">
          <a-select v-model:value="form.app_id" placeholder="选择应用（留空为全局）" allow-clear>
            <a-select-option :value="0">全局规则</a-select-option>
            <a-select-option v-for="app in applications" :key="app.id" :value="app.id">
              {{ app.name }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="环境" required>
          <a-select v-model:value="form.env" placeholder="选择环境">
            <a-select-option value="dev">开发环境</a-select-option>
            <a-select-option value="test">测试环境</a-select-option>
            <a-select-option value="staging">预发环境</a-select-option>
            <a-select-option value="prod">生产环境</a-select-option>
            <a-select-option value="production">生产环境(production)</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="允许发布日">
          <a-checkbox-group v-model:value="form.weekdayList">
            <a-checkbox :value="1">周一</a-checkbox>
            <a-checkbox :value="2">周二</a-checkbox>
            <a-checkbox :value="3">周三</a-checkbox>
            <a-checkbox :value="4">周四</a-checkbox>
            <a-checkbox :value="5">周五</a-checkbox>
            <a-checkbox :value="6">周六</a-checkbox>
            <a-checkbox :value="7">周日</a-checkbox>
          </a-checkbox-group>
        </a-form-item>
        <a-form-item label="时间范围">
          <a-space>
            <a-time-picker v-model:value="form.startTime" format="HH:mm" value-format="HH:mm" placeholder="开始时间" />
            <span>至</span>
            <a-time-picker v-model:value="form.endTime" format="HH:mm" value-format="HH:mm" placeholder="结束时间" />
          </a-space>
        </a-form-item>
        <a-form-item label="允许紧急发布">
          <a-switch v-model:checked="form.allow_emergency" />
          <span style="margin-left: 8px; color: #999">窗口外是否允许紧急发布</span>
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="form.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, reactive } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { deployWindowApi, type DeployWindow } from '@/services/approval'
import { applicationApi } from '@/services/application'

const loading = ref(false)
const submitting = ref(false)
const modalVisible = ref(false)
const isEdit = ref(false)
const windows = ref<DeployWindow[]>([])
const applications = ref<any[]>([])

const form = reactive({
  id: 0,
  app_id: 0,
  env: '',
  weekdayList: [1, 2, 3, 4, 5] as number[],
  startTime: '10:00',
  endTime: '18:00',
  allow_emergency: true,
  enabled: true
})

const columns = [
  { title: '应用', key: 'app_id', dataIndex: 'app_id' },
  { title: '环境', key: 'env', dataIndex: 'env' },
  { title: '允许发布日', key: 'weekdays', dataIndex: 'weekdays' },
  { title: '时间范围', key: 'time_range' },
  { title: '紧急发布', key: 'allow_emergency', dataIndex: 'allow_emergency' },
  { title: '状态', key: 'enabled', dataIndex: 'enabled' },
  { title: '操作', key: 'action', width: 150 }
]

const weekdayNames = ['', '周一', '周二', '周三', '周四', '周五', '周六', '周日']

const getEnvColor = (env: string) => {
  const colors: Record<string, string> = {
    dev: 'blue', test: 'cyan', staging: 'orange', prod: 'red', production: 'red'
  }
  return colors[env] || 'default'
}

const formatWeekdays = (weekdays: string) => {
  if (!weekdays) return '每天'
  const days = weekdays.split(',').map(Number).filter(Boolean)
  if (days.length === 7) return '每天'
  if (days.length === 5 && days.every(d => d >= 1 && d <= 5)) return '工作日'
  return days.map(d => weekdayNames[d]).join('、')
}

const loadWindows = async () => {
  loading.value = true
  try {
    const res = await deployWindowApi.list()
    windows.value = res.data || []
  } finally {
    loading.value = false
  }
}

const loadApplications = async () => {
  try {
    const res = await applicationApi.list({ page: 1, page_size: 1000 })
    applications.value = res.data?.list || []
  } catch {}
}

const showCreateModal = () => {
  isEdit.value = false
  Object.assign(form, {
    id: 0, app_id: 0, env: '', weekdayList: [1, 2, 3, 4, 5],
    startTime: '10:00', endTime: '18:00', allow_emergency: true, enabled: true
  })
  modalVisible.value = true
}

const showEditModal = (record: DeployWindow) => {
  isEdit.value = true
  Object.assign(form, {
    id: record.id,
    app_id: record.app_id,
    env: record.env,
    weekdayList: record.weekdays ? record.weekdays.split(',').map(Number).filter(Boolean) : [1, 2, 3, 4, 5],
    startTime: record.start_time || '10:00',
    endTime: record.end_time || '18:00',
    allow_emergency: record.allow_emergency,
    enabled: record.enabled
  })
  modalVisible.value = true
}

const handleSubmit = async () => {
  if (!form.env) {
    message.error('请选择环境')
    return
  }
  submitting.value = true
  try {
    const data = {
      app_id: form.app_id,
      env: form.env,
      weekdays: form.weekdayList.sort().join(','),
      start_time: form.startTime,
      end_time: form.endTime,
      allow_emergency: form.allow_emergency,
      enabled: form.enabled
    }
    if (isEdit.value) {
      await deployWindowApi.update(form.id, data)
      message.success('更新成功')
    } else {
      await deployWindowApi.create(data)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadWindows()
  } finally {
    submitting.value = false
  }
}

const toggleEnabled = async (record: DeployWindow) => {
  try {
    await deployWindowApi.update(record.id, { ...record, enabled: !record.enabled })
    message.success('更新成功')
    loadWindows()
  } catch {}
}

const deleteWindow = async (id: number) => {
  try {
    await deployWindowApi.delete(id)
    message.success('删除成功')
    loadWindows()
  } catch {}
}

onMounted(() => {
  loadWindows()
  loadApplications()
})
</script>

<style scoped>
.deploy-windows {
  padding: 16px;
}
</style>
