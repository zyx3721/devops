<template>
  <div class="traffic-config">
    <a-page-header title="超时重试" sub-title="管理应用超时和重试策略">
      <template #extra>
        <a-select v-model:value="selectedAppId" placeholder="选择应用" style="width: 200px" @change="onAppChange" show-search option-filter-prop="label">
          <a-select-option v-for="app in apps" :key="app.id" :value="app.id" :label="app.display_name || app.name">
            {{ app.display_name || app.name }}
          </a-select-option>
        </a-select>
        <a-button type="primary" @click="showModal()" :disabled="!selectedAppId"><SettingOutlined /> 配置</a-button>
      </template>
    </a-page-header>

    <a-alert v-if="!selectedAppId" type="info" show-icon style="margin-bottom: 16px">
      <template #message>请先选择一个应用来管理其超时重试配置</template>
    </a-alert>

    <a-card title="超时配置 (VirtualService)" :bordered="false" :loading="loading" v-if="selectedAppId">
      <a-descriptions v-if="config" :column="2" bordered size="small">
        <a-descriptions-item label="请求超时">{{ config.timeout || '30s' }}</a-descriptions-item>
        <a-descriptions-item label="重试次数">{{ config.retries || 3 }}</a-descriptions-item>
        <a-descriptions-item label="重试超时">{{ config.per_try_timeout || '10s' }}</a-descriptions-item>
        <a-descriptions-item label="重试条件">
          <a-tag v-for="c in (config.retry_on || ['5xx'])" :key="c" style="margin: 2px">{{ c }}</a-tag>
        </a-descriptions-item>
      </a-descriptions>
      <a-empty v-else description="使用默认超时配置" />
    </a-card>

    <!-- 配置弹窗 -->
    <a-modal v-model:open="modalVisible" title="超时重试配置" @ok="saveConfig" :confirm-loading="saving">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-form-item label="请求超时">
          <a-input v-model:value="form.timeout" placeholder="30s" />
          <div style="color: #999; font-size: 12px">请求的最大等待时间，如 30s、1m</div>
        </a-form-item>
        <a-form-item label="重试次数">
          <a-input-number v-model:value="form.retries" :min="0" :max="10" style="width: 100%" />
        </a-form-item>
        <a-form-item label="单次重试超时">
          <a-input v-model:value="form.per_try_timeout" placeholder="10s" />
        </a-form-item>
        <a-form-item label="重试条件">
          <a-select v-model:value="form.retry_on" mode="multiple" placeholder="选择重试条件">
            <a-select-option value="5xx">5xx 错误</a-select-option>
            <a-select-option value="gateway-error">网关错误</a-select-option>
            <a-select-option value="connect-failure">连接失败</a-select-option>
            <a-select-option value="retriable-4xx">可重试 4xx</a-select-option>
            <a-select-option value="reset">连接重置</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { SettingOutlined } from '@ant-design/icons-vue'
import request from '@/utils/request'
import { applicationApi, type Application } from '@/services/application'

const loading = ref(false)
const saving = ref(false)
const modalVisible = ref(false)
const selectedAppId = ref<number | undefined>()

const apps = ref<Application[]>([])
const config = ref<any>(null)

const form = reactive({
  timeout: '30s',
  retries: 3,
  per_try_timeout: '10s',
  retry_on: ['5xx']
})

const fetchApps = async () => {
  try {
    const response = await applicationApi.list({ page: 1, page_size: 1000 })
    if (response.code === 0 && response.data) {
      apps.value = (response.data.list || []).filter((a: Application) => a.k8s_deployment)
    }
  } catch (e) { console.error('获取应用列表失败', e) }
}

const fetchConfig = async () => {
  if (!selectedAppId.value) return
  loading.value = true
  try {
    const res = await request.get(`/applications/${selectedAppId.value}/traffic/timeout`)
    config.value = res.data || null
  } catch (e) { console.error('获取超时配置失败', e) }
  finally { loading.value = false }
}

const onAppChange = () => { fetchConfig() }

const showModal = () => {
  if (config.value) {
    Object.assign(form, config.value)
  }
  modalVisible.value = true
}

const saveConfig = async () => {
  saving.value = true
  try {
    await request.put(`/applications/${selectedAppId.value}/traffic/timeout`, form)
    message.success('保存成功')
    modalVisible.value = false
    fetchConfig()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

onMounted(() => { fetchApps() })
</script>

<style scoped>
.traffic-config { padding: 0; }
</style>
