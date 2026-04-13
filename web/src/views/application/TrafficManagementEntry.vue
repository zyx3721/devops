<template>
  <div class="traffic-entry">
    <a-page-header title="流量治理" sub-title="选择应用进行流量管理配置">
      <template #extra>
        <a-button type="primary" @click="fetchApps"><ReloadOutlined /> 刷新</a-button>
      </template>
    </a-page-header>

    <a-alert type="info" show-icon style="margin-bottom: 16px">
      <template #message>
        流量治理功能支持对 K8s 应用进行限流、熔断、超时重试、流量镜像和故障注入等配置。
        请选择一个已配置 K8s 部署信息的应用。
      </template>
    </a-alert>

    <!-- 筛选 -->
    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-form layout="inline">
        <a-form-item label="应用名">
          <a-input v-model:value="filter.name" placeholder="搜索应用" allow-clear style="width: 200px" @pressEnter="fetchApps" />
        </a-form-item>
        <a-form-item label="团队">
          <a-select v-model:value="filter.team" placeholder="全部" allow-clear style="width: 150px">
            <a-select-option v-for="team in teams" :key="team" :value="team">{{ team }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" @click="fetchApps">查询</a-button>
          <a-button style="margin-left: 8px" @click="resetFilter">重置</a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 应用卡片列表 -->
    <a-spin :spinning="loading">
      <a-row :gutter="[16, 16]">
        <a-col v-for="app in apps" :key="app.id" :xs="24" :sm="12" :md="8" :lg="6">
          <a-card hoverable @click="goTraffic(app)" :class="{ 'disabled-card': !app.k8s_deployment }">
            <template #cover>
              <div class="app-cover" :style="{ background: getAppColor(app.language) }">
                <AppstoreOutlined style="font-size: 48px; color: #fff" />
              </div>
            </template>
            <a-card-meta :title="app.display_name || app.name">
              <template #description>
                <div class="app-info">
                  <div v-if="app.team"><TeamOutlined /> {{ app.team }}</div>
                  <div v-if="app.k8s_deployment">
                    <CloudOutlined /> {{ app.k8s_namespace }}/{{ app.k8s_deployment }}
                  </div>
                  <div v-else class="no-k8s">
                    <WarningOutlined /> 未配置 K8s 部署
                  </div>
                </div>
              </template>
            </a-card-meta>
            <template #actions>
              <a-tooltip :title="app.k8s_deployment ? '进入流量治理' : '请先配置 K8s 部署信息'">
                <SettingOutlined :style="{ color: app.k8s_deployment ? '#1890ff' : '#ccc' }" />
              </a-tooltip>
            </template>
          </a-card>
        </a-col>
      </a-row>

      <a-empty v-if="!loading && apps.length === 0" description="暂无应用" />
    </a-spin>

    <!-- 分页 -->
    <div style="margin-top: 16px; text-align: right">
      <a-pagination
        v-model:current="pagination.current"
        v-model:pageSize="pagination.pageSize"
        :total="pagination.total"
        show-size-changer
        @change="fetchApps"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { ReloadOutlined, AppstoreOutlined, TeamOutlined, CloudOutlined, WarningOutlined, SettingOutlined } from '@ant-design/icons-vue'
import { applicationApi, type Application } from '@/services/application'

const router = useRouter()
const loading = ref(false)
const apps = ref<Application[]>([])
const teams = ref<string[]>([])

const filter = reactive({ name: '', team: '' })
const pagination = reactive({ current: 1, pageSize: 12, total: 0 })

const langColors: Record<string, string> = {
  go: 'linear-gradient(135deg, #00ADD8 0%, #00758D 100%)',
  java: 'linear-gradient(135deg, #ED8B00 0%, #B07219 100%)',
  python: 'linear-gradient(135deg, #3776AB 0%, #FFD43B 100%)',
  nodejs: 'linear-gradient(135deg, #339933 0%, #68A063 100%)',
  php: 'linear-gradient(135deg, #777BB4 0%, #4F5B93 100%)',
  default: 'linear-gradient(135deg, #1890ff 0%, #096dd9 100%)'
}

const getAppColor = (lang?: string) => langColors[lang || ''] || langColors.default

const fetchApps = async () => {
  loading.value = true
  try {
    const response = await applicationApi.list({
      page: pagination.current,
      page_size: pagination.pageSize,
      ...filter
    })
    if (response.code === 0 && response.data) {
      apps.value = response.data.list || []
      pagination.total = response.data.total
    }
  } catch (error) {
    console.error('获取应用列表失败', error)
  } finally {
    loading.value = false
  }
}

const fetchTeams = async () => {
  try {
    const response = await applicationApi.getTeams()
    if (response.code === 0 && response.data) {
      teams.value = response.data
    }
  } catch (error) {
    console.error('获取团队列表失败', error)
  }
}

const resetFilter = () => {
  filter.name = ''
  filter.team = ''
  pagination.current = 1
  fetchApps()
}

const goTraffic = (app: Application) => {
  if (!app.k8s_deployment) {
    message.warning('该应用未配置 K8s 部署信息，请先在应用管理中配置')
    return
  }
  router.push(`/applications/${app.id}/traffic`)
}

onMounted(() => {
  fetchApps()
  fetchTeams()
})
</script>

<style scoped>
.traffic-entry {
  padding: 0;
}

.app-cover {
  height: 100px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.app-info {
  font-size: 12px;
  color: #666;
}

.app-info > div {
  margin-bottom: 4px;
}

.no-k8s {
  color: #faad14;
}

.disabled-card {
  opacity: 0.6;
  cursor: not-allowed;
}

.disabled-card:hover {
  box-shadow: none;
}
</style>
