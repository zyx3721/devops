<template>
  <div class="traffic-config">
    <a-page-header title="负载均衡" sub-title="配置服务负载均衡策略和健康检查">
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
      <template #message>请先选择一个应用来管理其负载均衡配置</template>
    </a-alert>

    <a-row :gutter="16" v-if="selectedAppId">
      <a-col :span="12">
        <a-card title="负载均衡策略" :bordered="false" :loading="loading">
          <a-descriptions :column="1" bordered size="small" v-if="config">
            <a-descriptions-item label="负载均衡算法">
              <a-tag :color="getLbColor(config.lb_policy)">{{ getLbText(config.lb_policy) }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="会话保持" v-if="config.lb_policy === 'consistent_hash'">
              <a-tag>{{ getHashKeyText(config.hash_key) }}</a-tag>
            </a-descriptions-item>
            <a-descriptions-item label="一致性哈希环大小" v-if="config.lb_policy === 'consistent_hash'">
              {{ config.ring_size || 1024 }}
            </a-descriptions-item>
            <a-descriptions-item label="最小请求数" v-if="config.lb_policy === 'least_request'">
              {{ config.choice_count || 2 }}
            </a-descriptions-item>
            <a-descriptions-item label="预热时间">
              {{ config.warmup_duration || '0s' }}
            </a-descriptions-item>
          </a-descriptions>
          <a-empty v-else description="使用默认负载均衡策略 (Round Robin)" />
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card title="健康检查" :bordered="false" :loading="loading">
          <a-descriptions :column="1" bordered size="small" v-if="config?.health_check_enabled">
            <a-descriptions-item label="状态">
              <a-badge status="success" text="已启用" />
            </a-descriptions-item>
            <a-descriptions-item label="检查路径">
              {{ config.health_check_path || '/health' }}
            </a-descriptions-item>
            <a-descriptions-item label="检查间隔">
              {{ config.health_check_interval || '10s' }}
            </a-descriptions-item>
            <a-descriptions-item label="超时时间">
              {{ config.health_check_timeout || '5s' }}
            </a-descriptions-item>
            <a-descriptions-item label="健康阈值">
              {{ config.healthy_threshold || 2 }} 次成功
            </a-descriptions-item>
            <a-descriptions-item label="不健康阈值">
              {{ config.unhealthy_threshold || 3 }} 次失败
            </a-descriptions-item>
          </a-descriptions>
          <a-empty v-else description="未启用健康检查" />
        </a-card>
      </a-col>
    </a-row>

    <a-card title="连接池配置" :bordered="false" style="margin-top: 16px" :loading="loading" v-if="selectedAppId">
      <a-row :gutter="16">
        <a-col :span="12">
          <a-descriptions title="HTTP 连接池" :column="1" bordered size="small" v-if="config">
            <a-descriptions-item label="最大连接数">{{ config.http_max_connections || 1024 }}</a-descriptions-item>
            <a-descriptions-item label="每连接最大请求">{{ config.http_max_requests_per_conn || 0 }}</a-descriptions-item>
            <a-descriptions-item label="最大等待请求">{{ config.http_max_pending_requests || 1024 }}</a-descriptions-item>
            <a-descriptions-item label="最大重试次数">{{ config.http_max_retries || 3 }}</a-descriptions-item>
            <a-descriptions-item label="空闲超时">{{ config.http_idle_timeout || '1h' }}</a-descriptions-item>
          </a-descriptions>
        </a-col>
        <a-col :span="12">
          <a-descriptions title="TCP 连接池" :column="1" bordered size="small" v-if="config">
            <a-descriptions-item label="最大连接数">{{ config.tcp_max_connections || 1024 }}</a-descriptions-item>
            <a-descriptions-item label="连接超时">{{ config.tcp_connect_timeout || '10s' }}</a-descriptions-item>
            <a-descriptions-item label="TCP Keepalive">{{ config.tcp_keepalive_enabled ? '启用' : '禁用' }}</a-descriptions-item>
            <a-descriptions-item label="Keepalive 间隔" v-if="config.tcp_keepalive_enabled">{{ config.tcp_keepalive_interval || '60s' }}</a-descriptions-item>
          </a-descriptions>
        </a-col>
      </a-row>
    </a-card>

    <!-- 配置弹窗 -->
    <a-modal v-model:open="modalVisible" title="负载均衡配置" @ok="saveConfig" :confirm-loading="saving" width="700px">
      <a-form :model="form" :label-col="{ span: 6 }" :wrapper-col="{ span: 16 }">
        <a-divider>负载均衡策略</a-divider>
        <a-form-item label="负载均衡算法">
          <a-select v-model:value="form.lb_policy" @change="onLbPolicyChange">
            <a-select-option value="round_robin">轮询 (Round Robin)</a-select-option>
            <a-select-option value="random">随机 (Random)</a-select-option>
            <a-select-option value="least_request">最少请求 (Least Request)</a-select-option>
            <a-select-option value="consistent_hash">一致性哈希 (Consistent Hash)</a-select-option>
            <a-select-option value="passthrough">直通 (Passthrough)</a-select-option>
          </a-select>
        </a-form-item>
        <template v-if="form.lb_policy === 'consistent_hash'">
          <a-form-item label="哈希键">
            <a-select v-model:value="form.hash_key">
              <a-select-option value="header">请求头</a-select-option>
              <a-select-option value="cookie">Cookie</a-select-option>
              <a-select-option value="source_ip">源 IP</a-select-option>
              <a-select-option value="query_param">查询参数</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="哈希键名" v-if="form.hash_key !== 'source_ip'">
            <a-input v-model:value="form.hash_key_name" placeholder="如: X-User-Id, session_id" />
          </a-form-item>
          <a-form-item label="哈希环大小">
            <a-input-number v-model:value="form.ring_size" :min="64" :max="8192" style="width: 100%" />
          </a-form-item>
        </template>
        <template v-if="form.lb_policy === 'least_request'">
          <a-form-item label="选择数量">
            <a-input-number v-model:value="form.choice_count" :min="2" :max="10" style="width: 100%" />
            <div style="color: #999; font-size: 12px">随机选择 N 个实例，选择请求数最少的</div>
          </a-form-item>
        </template>
        <a-form-item label="预热时间">
          <a-input v-model:value="form.warmup_duration" placeholder="60s" />
          <div style="color: #999; font-size: 12px">新实例启动后逐渐增加流量的时间</div>
        </a-form-item>

        <a-divider>健康检查</a-divider>
        <a-form-item label="启用健康检查">
          <a-switch v-model:checked="form.health_check_enabled" />
        </a-form-item>
        <template v-if="form.health_check_enabled">
          <a-form-item label="检查路径">
            <a-input v-model:value="form.health_check_path" placeholder="/health" />
          </a-form-item>
          <a-form-item label="检查间隔">
            <a-input v-model:value="form.health_check_interval" placeholder="10s" />
          </a-form-item>
          <a-form-item label="超时时间">
            <a-input v-model:value="form.health_check_timeout" placeholder="5s" />
          </a-form-item>
          <a-form-item label="健康阈值">
            <a-input-number v-model:value="form.healthy_threshold" :min="1" :max="10" style="width: 100%" />
          </a-form-item>
          <a-form-item label="不健康阈值">
            <a-input-number v-model:value="form.unhealthy_threshold" :min="1" :max="10" style="width: 100%" />
          </a-form-item>
        </template>

        <a-divider>连接池</a-divider>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="HTTP 最大连接" :label-col="{ span: 12 }" :wrapper-col="{ span: 12 }">
              <a-input-number v-model:value="form.http_max_connections" :min="1" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="TCP 最大连接" :label-col="{ span: 12 }" :wrapper-col="{ span: 12 }">
              <a-input-number v-model:value="form.tcp_max_connections" :min="1" style="width: 100%" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="每连接最大请求" :label-col="{ span: 12 }" :wrapper-col="{ span: 12 }">
              <a-input-number v-model:value="form.http_max_requests_per_conn" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="连接超时" :label-col="{ span: 12 }" :wrapper-col="{ span: 12 }">
              <a-input v-model:value="form.tcp_connect_timeout" placeholder="10s" />
            </a-form-item>
          </a-col>
        </a-row>
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
  lb_policy: 'round_robin',
  hash_key: 'header',
  hash_key_name: '',
  ring_size: 1024,
  choice_count: 2,
  warmup_duration: '60s',
  health_check_enabled: false,
  health_check_path: '/health',
  health_check_interval: '10s',
  health_check_timeout: '5s',
  healthy_threshold: 2,
  unhealthy_threshold: 3,
  http_max_connections: 1024,
  http_max_requests_per_conn: 0,
  http_max_pending_requests: 1024,
  http_max_retries: 3,
  http_idle_timeout: '1h',
  tcp_max_connections: 1024,
  tcp_connect_timeout: '10s',
  tcp_keepalive_enabled: true,
  tcp_keepalive_interval: '60s'
})

const lbMap: Record<string, { text: string; color: string }> = {
  round_robin: { text: '轮询', color: 'blue' },
  random: { text: '随机', color: 'green' },
  least_request: { text: '最少请求', color: 'purple' },
  consistent_hash: { text: '一致性哈希', color: 'orange' },
  passthrough: { text: '直通', color: 'default' }
}
const getLbText = (p: string) => lbMap[p]?.text || p
const getLbColor = (p: string) => lbMap[p]?.color || 'default'

const hashKeyMap: Record<string, string> = {
  header: '请求头',
  cookie: 'Cookie',
  source_ip: '源 IP',
  query_param: '查询参数'
}
const getHashKeyText = (k: string) => hashKeyMap[k] || k

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
    const res = await request.get(`/applications/${selectedAppId.value}/traffic/loadbalance`)
    config.value = res.data || null
  } catch (e) { console.error('获取负载均衡配置失败', e) }
  finally { loading.value = false }
}

const onAppChange = () => { fetchConfig() }
const onLbPolicyChange = () => {
  if (form.lb_policy === 'consistent_hash') { form.hash_key = 'header' }
}

const showModal = () => {
  if (config.value) {
    Object.assign(form, config.value)
  }
  modalVisible.value = true
}

const saveConfig = async () => {
  saving.value = true
  try {
    await request.put(`/applications/${selectedAppId.value}/traffic/loadbalance`, form)
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
