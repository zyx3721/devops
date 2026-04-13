<template>
  <div class="traffic-config">
    <a-page-header title="流量路由" sub-title="灰度发布、A/B测试、按条件路由">
      <template #extra>
        <a-select v-model:value="selectedAppId" placeholder="选择应用" style="width: 200px" @change="onAppChange" show-search option-filter-prop="label">
          <a-select-option v-for="app in apps" :key="app.id" :value="app.id" :label="app.display_name || app.name">
            {{ app.display_name || app.name }}
          </a-select-option>
        </a-select>
        <a-button type="primary" @click="showModal()" :disabled="!selectedAppId"><PlusOutlined /> 添加路由</a-button>
      </template>
    </a-page-header>

    <a-alert v-if="!selectedAppId" type="info" show-icon style="margin-bottom: 16px">
      <template #message>请先选择一个应用来管理其流量路由规则</template>
    </a-alert>

    <!-- 路由类型说明 -->
    <a-row :gutter="16" style="margin-bottom: 16px" v-if="selectedAppId">
      <a-col :span="6">
        <a-card size="small" hoverable @click="filterByType('weight')">
          <a-statistic title="权重路由" :value="stats.weight" suffix="条">
            <template #prefix><PieChartOutlined style="color: #1890ff" /></template>
          </a-statistic>
          <div style="color: #999; font-size: 12px">按比例分配流量</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small" hoverable @click="filterByType('header')">
          <a-statistic title="Header 路由" :value="stats.header" suffix="条">
            <template #prefix><TagOutlined style="color: #52c41a" /></template>
          </a-statistic>
          <div style="color: #999; font-size: 12px">按请求头匹配</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small" hoverable @click="filterByType('cookie')">
          <a-statistic title="Cookie 路由" :value="stats.cookie" suffix="条">
            <template #prefix><CoffeeOutlined style="color: #fa8c16" /></template>
          </a-statistic>
          <div style="color: #999; font-size: 12px">按 Cookie 匹配</div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card size="small" hoverable @click="filterByType('param')">
          <a-statistic title="参数路由" :value="stats.param" suffix="条">
            <template #prefix><FilterOutlined style="color: #722ed1" /></template>
          </a-statistic>
          <div style="color: #999; font-size: 12px">按请求参数匹配</div>
        </a-card>
      </a-col>
    </a-row>

    <a-card :bordered="false" v-if="selectedAppId">
      <a-table :columns="columns" :data-source="filteredRules" :loading="loading" row-key="id" size="middle">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <div>
              <span>{{ record.name }}</span>
              <div style="color: #999; font-size: 12px">{{ record.description }}</div>
            </div>
          </template>
          <template v-if="column.key === 'type'">
            <a-tag :color="getTypeColor(record.route_type)">{{ getTypeText(record.route_type) }}</a-tag>
          </template>
          <template v-if="column.key === 'condition'">
            <div v-if="record.route_type === 'weight'">
              <div v-for="(dest, idx) in record.destinations" :key="idx" style="margin-bottom: 4px">
                <a-tag>{{ dest.subset || dest.version }}</a-tag>
                <a-progress :percent="dest.weight" size="small" style="width: 80px; display: inline-block" />
              </div>
            </div>
            <div v-else>
              <code>{{ record.match_key }} {{ record.match_operator }} {{ record.match_value }}</code>
              <span style="margin-left: 8px">→</span>
              <a-tag color="blue">{{ record.target_subset || record.target_version }}</a-tag>
            </div>
          </template>
          <template v-if="column.key === 'priority'">
            <a-tag>{{ record.priority }}</a-tag>
          </template>
          <template v-if="column.key === 'enabled'">
            <a-switch v-model:checked="record.enabled" size="small" @change="toggleRule(record)" />
          </template>
          <template v-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showModal(record)">编辑</a-button>
              <a-popconfirm title="确定删除？" @confirm="deleteRule(record.id)">
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 编辑弹窗 -->
    <a-modal v-model:open="modalVisible" :title="editingRule ? '编辑路由规则' : '添加路由规则'" @ok="saveRule" :confirm-loading="saving" width="700px">
      <a-form :model="form" :label-col="{ span: 5 }" :wrapper-col="{ span: 17 }">
        <a-form-item label="规则名称" required>
          <a-input v-model:value="form.name" placeholder="如：灰度发布-v2" />
        </a-form-item>
        <a-form-item label="规则描述">
          <a-input v-model:value="form.description" placeholder="规则用途说明" />
        </a-form-item>
        <a-form-item label="优先级">
          <a-input-number v-model:value="form.priority" :min="1" :max="1000" style="width: 100%" />
          <div style="color: #999; font-size: 12px">数字越小优先级越高</div>
        </a-form-item>

        <a-divider>路由类型</a-divider>
        <a-form-item label="路由类型" required>
          <a-radio-group v-model:value="form.route_type" @change="onTypeChange">
            <a-radio-button value="weight">权重路由</a-radio-button>
            <a-radio-button value="header">Header 匹配</a-radio-button>
            <a-radio-button value="cookie">Cookie 匹配</a-radio-button>
            <a-radio-button value="param">参数匹配</a-radio-button>
          </a-radio-group>
        </a-form-item>

        <!-- 权重路由 -->
        <template v-if="form.route_type === 'weight'">
          <a-form-item label="流量分配">
            <div v-for="(dest, idx) in form.destinations" :key="idx" style="margin-bottom: 8px; display: flex; align-items: center">
              <a-input v-model:value="dest.subset" placeholder="版本/子集" style="width: 120px" />
              <a-slider v-model:value="dest.weight" :min="0" :max="100" style="flex: 1; margin: 0 12px" />
              <span style="width: 40px">{{ dest.weight }}%</span>
              <a-button type="link" danger size="small" @click="removeDestination(idx)" v-if="form.destinations.length > 1">
                <DeleteOutlined />
              </a-button>
            </div>
            <a-button type="dashed" block @click="addDestination"><PlusOutlined /> 添加版本</a-button>
            <div v-if="totalWeight !== 100" style="color: #ff4d4f; margin-top: 8px">
              权重总和必须为 100%，当前: {{ totalWeight }}%
            </div>
          </a-form-item>
        </template>

        <!-- 条件路由 -->
        <template v-else>
          <a-form-item label="匹配键" required>
            <a-input v-model:value="form.match_key" :placeholder="getMatchKeyPlaceholder()" />
          </a-form-item>
          <a-form-item label="匹配方式" required>
            <a-select v-model:value="form.match_operator">
              <a-select-option value="exact">精确匹配</a-select-option>
              <a-select-option value="prefix">前缀匹配</a-select-option>
              <a-select-option value="regex">正则匹配</a-select-option>
              <a-select-option value="present">存在即匹配</a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item label="匹配值" required v-if="form.match_operator !== 'present'">
            <a-input v-model:value="form.match_value" placeholder="匹配的值" />
          </a-form-item>
          <a-form-item label="目标版本" required>
            <a-input v-model:value="form.target_subset" placeholder="如: v2, canary" />
          </a-form-item>
        </template>

        <a-form-item label="启用">
          <a-switch v-model:checked="form.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, DeleteOutlined, PieChartOutlined, TagOutlined, CoffeeOutlined, FilterOutlined } from '@ant-design/icons-vue'
import request from '@/utils/request'
import { applicationApi, type Application } from '@/services/application'

const loading = ref(false)
const saving = ref(false)
const modalVisible = ref(false)
const selectedAppId = ref<number | undefined>()
const editingRule = ref<any>(null)
const filterType = ref<string>('')

const apps = ref<Application[]>([])
const rules = ref<any[]>([])

const form = reactive({
  name: '',
  description: '',
  priority: 100,
  route_type: 'weight',
  destinations: [{ subset: 'v1', weight: 90 }, { subset: 'v2', weight: 10 }],
  match_key: '',
  match_operator: 'exact',
  match_value: '',
  target_subset: '',
  enabled: true
})

const totalWeight = computed(() => form.destinations.reduce((sum, d) => sum + (d.weight || 0), 0))

const stats = computed(() => ({
  weight: rules.value.filter(r => r.route_type === 'weight').length,
  header: rules.value.filter(r => r.route_type === 'header').length,
  cookie: rules.value.filter(r => r.route_type === 'cookie').length,
  param: rules.value.filter(r => r.route_type === 'param').length
}))

const filteredRules = computed(() => {
  if (!filterType.value) return rules.value
  return rules.value.filter(r => r.route_type === filterType.value)
})

const columns = [
  { title: '规则名称', key: 'name', width: 200 },
  { title: '类型', key: 'type', width: 120 },
  { title: '路由条件', key: 'condition' },
  { title: '优先级', key: 'priority', width: 80 },
  { title: '启用', key: 'enabled', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const typeMap: Record<string, { text: string; color: string }> = {
  weight: { text: '权重路由', color: 'blue' },
  header: { text: 'Header', color: 'green' },
  cookie: { text: 'Cookie', color: 'orange' },
  param: { text: '参数', color: 'purple' }
}
const getTypeText = (t: string) => typeMap[t]?.text || t
const getTypeColor = (t: string) => typeMap[t]?.color || 'default'

const getMatchKeyPlaceholder = () => {
  if (form.route_type === 'header') return 'X-User-Type'
  if (form.route_type === 'cookie') return 'user_group'
  return 'version'
}

const filterByType = (type: string) => {
  filterType.value = filterType.value === type ? '' : type
}

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
    const res = await request.get(`/applications/${selectedAppId.value}/traffic/routes`)
    rules.value = res.data?.items || []
  } catch (e) { console.error('获取路由规则失败', e) }
  finally { loading.value = false }
}

const onAppChange = () => { fetchRules() }
const onTypeChange = () => {
  if (form.route_type === 'weight') {
    form.destinations = [{ subset: 'v1', weight: 90 }, { subset: 'v2', weight: 10 }]
  }
}

const addDestination = () => { form.destinations.push({ subset: '', weight: 0 }) }
const removeDestination = (idx: number) => { form.destinations.splice(idx, 1) }

const showModal = (record?: any) => {
  editingRule.value = record || null
  if (record) {
    Object.assign(form, record)
    if (!form.destinations) form.destinations = [{ subset: 'v1', weight: 100 }]
  } else {
    Object.assign(form, {
      name: '', description: '', priority: 100, route_type: 'weight',
      destinations: [{ subset: 'v1', weight: 90 }, { subset: 'v2', weight: 10 }],
      match_key: '', match_operator: 'exact', match_value: '', target_subset: '', enabled: true
    })
  }
  modalVisible.value = true
}

const saveRule = async () => {
  if (!form.name) { message.warning('请填写规则名称'); return }
  if (form.route_type === 'weight' && totalWeight.value !== 100) {
    message.warning('权重总和必须为 100%'); return
  }
  saving.value = true
  try {
    if (editingRule.value) {
      await request.put(`/applications/${selectedAppId.value}/traffic/routes/${editingRule.value.id}`, form)
    } else {
      await request.post(`/applications/${selectedAppId.value}/traffic/routes`, form)
    }
    message.success('保存成功')
    modalVisible.value = false
    fetchRules()
  } catch (e) { message.error('保存失败') }
  finally { saving.value = false }
}

const toggleRule = async (record: any) => {
  try {
    await request.put(`/applications/${selectedAppId.value}/traffic/routes/${record.id}`, { enabled: record.enabled })
  } catch (e) { record.enabled = !record.enabled; message.error('操作失败') }
}

const deleteRule = async (id: number) => {
  try {
    await request.delete(`/applications/${selectedAppId.value}/traffic/routes/${id}`)
    message.success('删除成功')
    fetchRules()
  } catch (e) { message.error('删除失败') }
}

onMounted(() => { fetchApps() })
</script>

<style scoped>
.traffic-config { padding: 0; }
</style>
