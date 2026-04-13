<template>
  <div class="traffic-config">
    <a-page-header title="流量镜像" sub-title="将生产流量复制到测试环境">
      <template #extra>
        <a-select v-model:value="selectedAppId" placeholder="选择应用" style="width: 200px" @change="onAppChange" show-search option-filter-prop="label">
          <a-select-option v-for="app in apps" :key="app.id" :value="app.id" :label="app.display_name || app.name">
            {{ app.display_name || app.name }}
          </a-select-option>
        </a-select>
        <a-button type="primary" @click="showModal()" :disabled="!selectedAppId"><CopyOutlined /> 配置镜像</a-button>
      </template>
    </a-page-header>

    <a-alert v-if="!selectedAppId" type="info" show-icon style="margin-bottom: 16px">
      <template #message>请先选择一个应用来管理其流量镜像配置</template>
    </a-alert>

    <a-card :bordered="false" v-if="selectedAppId">
      <a-table :columns="columns" :data-source="rules" :loading="loading" row-key="id" size="middle">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'target'">
            <span>{{ record.target_service }}</span>
            <a-tag v-if="record.target_subset" size="small" style="margin-left: 4px">{{ record.target_subset }}</a-tag>
          </template>
          <template v-if="column.key === 'percentage'">
            <a-progress :percent="record.percentage" size="small" style="width: 120px" />
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
    <a-modal v-model:open="modalVisible" title="流量镜像配置" @ok="saveRule" :confirm-loading="saving">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="目标服务" required>
          <a-select v-model:value="form.target_service" placeholder="选择服务" show-search>
            <a-select-option v-for="svc in services" :key="svc" :value="svc">{{ svc }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="目标子集">
          <a-input v-model:value="form.target_subset" placeholder="如: canary" />
        </a-form-item>
        <a-form-item label="镜像比例">
          <a-slider v-model:value="form.percentage" :min="1" :max="100" />
          <div style="color: #999; font-size: 12px">复制到目标服务的流量百分比</div>
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
import { CopyOutlined } from '@ant-design/icons-vue'
import request from '@/utils/request'
import { applicationApi, type Application } from '@/services/application'

const loading = ref(false)
const saving = ref(false)
const modalVisible = ref(false)
const selectedAppId = ref<number | undefined>()

const apps = ref<Application[]>([])
const rules = ref<any[]>([])
const services = ref<string[]>([])
const selectedApp = ref<Application | null>(null)

const form = reactive({
  target_service: '',
  target_subset: '',
  percentage: 100,
  enabled: true
})

const columns = [
  { title: '目标服务', key: 'target' },
  { title: '镜像比例', key: 'percentage', width: 180 },
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
    const res = await request.get(`/applications/${selectedAppId.value}/traffic/mirrors`)
    rules.value = res.data?.items || []
  } catch (e) { console.error('获取流量镜像失败', e) }
  finally { loading.value = false }
}

const fetchServices = async () => {
  selectedApp.value = apps.value.find(a => a.id === selectedAppId.value) || null
  if (!selectedApp.value?.k8s_cluster_id || !selectedApp.value?.k8s_namespace) return
  try {
    const res = await request.get(`/k8s/clusters/${selectedApp.value.k8s_cluster_id}/namespaces/${selectedApp.value.k8s_namespace}/services`)
    services.value = (res.data?.items || []).map((s: any) => s.metadata?.name || s.name)
  } catch (e) { console.error('获取服务列表失败', e) }
}

const onAppChange = () => {
  fetchRules()
  fetchServices()
}

const showModal = () => {
  Object.assign(form, { target_service: '', target_subset: '', percentage: 100, enabled: true })
  modalVisible.value = true
}

const saveRule = async () => {
  if (!form.target_service) { message.warning('请选择目标服务'); return }
  saving.value = true
  try {
    await request.post(`/applications/${selectedAppId.value}/traffic/mirrors`, form)
    message.success('保存成功')
    modalVisible.value = false
    fetchRules()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

const toggleRule = async (record: any) => {
  try {
    await request.put(`/applications/${selectedAppId.value}/traffic/mirrors/${record.id}`, { enabled: record.enabled })
  } catch (e) { record.enabled = !record.enabled; message.error('操作失败') }
}

const deleteRule = async (id: number) => {
  try {
    await request.delete(`/applications/${selectedAppId.value}/traffic/mirrors/${id}`)
    message.success('删除成功')
    fetchRules()
  } catch (e) { message.error('删除失败') }
}

onMounted(() => { fetchApps() })
</script>

<style scoped>
.traffic-config { padding: 0; }
</style>
