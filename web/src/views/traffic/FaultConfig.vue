<template>
  <div class="traffic-config">
    <a-page-header title="故障注入" sub-title="模拟故障场景进行混沌测试">
      <template #extra>
        <a-select v-model:value="selectedAppId" placeholder="选择应用" style="width: 200px" @change="onAppChange" show-search option-filter-prop="label">
          <a-select-option v-for="app in apps" :key="app.id" :value="app.id" :label="app.display_name || app.name">
            {{ app.display_name || app.name }}
          </a-select-option>
        </a-select>
        <a-button type="primary" @click="showModal()" :disabled="!selectedAppId"><BugOutlined /> 添加故障</a-button>
      </template>
    </a-page-header>

    <a-alert type="warning" show-icon style="margin-bottom: 16px">
      <template #message>故障注入仅用于测试环境，请勿在生产环境使用！</template>
    </a-alert>

    <a-alert v-if="!selectedAppId" type="info" show-icon style="margin-bottom: 16px">
      <template #message>请先选择一个应用来管理其故障注入配置</template>
    </a-alert>

    <a-card :bordered="false" v-if="selectedAppId">
      <a-table :columns="columns" :data-source="rules" :loading="loading" row-key="id" size="middle">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'type'">
            <a-tag :color="record.type === 'delay' ? 'blue' : 'red'">{{ record.type === 'delay' ? '延迟注入' : '请求中断' }}</a-tag>
          </template>
          <template v-if="column.key === 'path'">
            <code>{{ record.path || '/' }}</code>
          </template>
          <template v-if="column.key === 'config'">
            <span v-if="record.type === 'delay'">延迟 {{ record.delay_duration }}</span>
            <span v-else>返回 HTTP {{ record.abort_code }}</span>
          </template>
          <template v-if="column.key === 'percentage'">
            <a-progress :percent="record.percentage" size="small" style="width: 80px" :status="record.enabled ? 'active' : 'normal'" />
          </template>
          <template v-if="column.key === 'enabled'">
            <a-switch v-model:checked="record.enabled" size="small" @change="toggleRule(record)" />
          </template>
          <template v-if="column.key === 'action'">
            <a-popconfirm title="确定删除？" @confirm="deleteRule(record.id)">
              <a-button type="link" size="small" danger>删除</a-button>
            </a-popconfirm>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 配置弹窗 -->
    <a-modal v-model:open="modalVisible" title="故障注入配置" @ok="saveRule" :confirm-loading="saving">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="故障类型" required>
          <a-radio-group v-model:value="form.type">
            <a-radio value="delay">延迟注入</a-radio>
            <a-radio value="abort">请求中断</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="接口路径">
          <a-input v-model:value="form.path" placeholder="/ 表示全部" />
        </a-form-item>
        <template v-if="form.type === 'delay'">
          <a-form-item label="延迟时间" required>
            <a-input v-model:value="form.delay_duration" placeholder="5s" />
            <div style="color: #999; font-size: 12px">如 5s、100ms、1m</div>
          </a-form-item>
        </template>
        <template v-else>
          <a-form-item label="HTTP 状态码" required>
            <a-select v-model:value="form.abort_code">
              <a-select-option :value="500">500 Internal Error</a-select-option>
              <a-select-option :value="502">502 Bad Gateway</a-select-option>
              <a-select-option :value="503">503 Unavailable</a-select-option>
              <a-select-option :value="504">504 Timeout</a-select-option>
              <a-select-option :value="400">400 Bad Request</a-select-option>
              <a-select-option :value="403">403 Forbidden</a-select-option>
              <a-select-option :value="404">404 Not Found</a-select-option>
            </a-select>
          </a-form-item>
        </template>
        <a-form-item label="影响比例">
          <a-slider v-model:value="form.percentage" :min="1" :max="100" />
          <div style="color: #999; font-size: 12px">受故障影响的请求百分比</div>
        </a-form-item>
        <a-form-item label="启用">
          <a-switch v-model:checked="form.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { BugOutlined } from '@ant-design/icons-vue'
import request from '@/utils/request'
import { applicationApi, type Application } from '@/services/application'

const loading = ref(false)
const saving = ref(false)
const modalVisible = ref(false)
const selectedAppId = ref<number | undefined>()

const apps = ref<Application[]>([])
const rules = ref<any[]>([])

const form = reactive({
  type: 'delay',
  path: '/',
  delay_duration: '5s',
  abort_code: 500,
  percentage: 10,
  enabled: false
})

const columns = [
  { title: '类型', key: 'type', width: 120 },
  { title: '接口', key: 'path' },
  { title: '配置', key: 'config', width: 150 },
  { title: '比例', key: 'percentage', width: 120 },
  { title: '启用', key: 'enabled', width: 100 },
  { title: '操作', key: 'action', width: 100 }
]

const fetchApps = async () => {
  try {
    const response = await applicationApi.list({ page: 1, page_size: 1000 })
    if (response.code === 0 && response.data) {
      apps.value = (response.data.list || []).filter((a: Application) => a.k8s_deployment)
    }
  } catch (e) { console.error('获取应用列表失败', e) }
}

const fetchRules = async () => {
  if (!selectedAppId.value) return
  loading.value = true
  try {
    const res = await request.get(`/applications/${selectedAppId.value}/traffic/faults`)
    rules.value = res.data?.items || []
  } catch (e) { console.error('获取故障注入失败', e) }
  finally { loading.value = false }
}

const onAppChange = () => { fetchRules() }

const showModal = () => {
  Object.assign(form, { type: 'delay', path: '/', delay_duration: '5s', abort_code: 500, percentage: 10, enabled: false })
  modalVisible.value = true
}

const saveRule = async () => {
  saving.value = true
  try {
    await request.post(`/applications/${selectedAppId.value}/traffic/faults`, form)
    message.success('保存成功')
    modalVisible.value = false
    fetchRules()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

const toggleRule = async (record: any) => {
  try {
    await request.put(`/applications/${selectedAppId.value}/traffic/faults/${record.id}`, { enabled: record.enabled })
  } catch (e) { record.enabled = !record.enabled; message.error('操作失败') }
}

const deleteRule = async (id: number) => {
  try {
    await request.delete(`/applications/${selectedAppId.value}/traffic/faults/${id}`)
    message.success('删除成功')
    fetchRules()
  } catch (e) { message.error('删除失败') }
}

onMounted(() => { fetchApps() })
</script>

<style scoped>
.traffic-config { padding: 0; }
</style>
