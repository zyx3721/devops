<template>
  <div class="oa-data">
    <div class="page-header">
      <h1>OA 管理</h1>
      <a-space>
        <a-tag :color="syncStatus.running ? 'green' : 'default'">
          {{ syncStatus.running ? '同步运行中' : '同步已停止' }}
        </a-tag>
        <a-tag color="blue">已同步: {{ syncStatus.synced_count }} 条</a-tag>
        <a-button @click="triggerSync" :loading="syncing" size="small">
          <template #icon><SyncOutlined /></template>
          立即同步
        </a-button>
        <a-button @click="triggerForceSync" :loading="forceSyncing" size="small" type="dashed">
          强制同步
        </a-button>
        <a-divider type="vertical" />
        <a-input-search
          v-if="activeTab === 'data'"
          v-model:value="sourceSearch"
          placeholder="按来源搜索"
          style="width: 200px"
          allow-clear
          @search="fetchData"
        />
        <a-button v-if="activeTab === 'address'" type="primary" @click="showAddressModal()">
          <template #icon><PlusOutlined /></template>
          添加地址
        </a-button>
        <a-button v-if="activeTab === 'notify'" type="primary" @click="showNotifyModal()">
          <template #icon><PlusOutlined /></template>
          添加配置
        </a-button>
      </a-space>
    </div>

    <a-tabs v-model:activeKey="activeTab" @change="onTabChange">
      <!-- 数据列表 Tab -->
      <a-tab-pane key="data" tab="数据列表">
        <a-card :bordered="false">
          <a-table :columns="dataColumns" :data-source="dataList" :loading="loadingData" row-key="id" :pagination="dataPagination" @change="onDataTableChange">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'unique_id'">
                <a-typography-text copyable :content="record.unique_id">{{ truncateId(record.unique_id) }}</a-typography-text>
              </template>
              <template v-if="column.key === 'source'">
                <a-tag v-if="record.source" color="blue">{{ record.source }}</a-tag>
                <span v-else>-</span>
              </template>
              <template v-if="column.key === 'created_at'">{{ formatTime(record.created_at) }}</template>
              <template v-if="column.key === 'preview'">
                <a-typography-text type="secondary" ellipsis style="max-width: 250px">{{ getPreviewFromString(record.original_data) }}</a-typography-text>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="viewDetail(record)">查看</a-button>
                  <a-button type="link" size="small" @click="sendCard(record)" :loading="sendingCardId === record.id">发卡片</a-button>
                  <a-popconfirm title="确定删除？" @confirm="() => deleteData(record.id)">
                    <a-button type="link" size="small" danger>删除</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- OA 地址管理 Tab -->
      <a-tab-pane key="address" tab="地址管理">
        <a-card :bordered="false">
          <a-table :columns="addressColumns" :data-source="addressList" :loading="loadingAddress" row-key="id" :pagination="{ pageSize: 10 }">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'url'">
                <a-typography-text copyable :content="record.url" ellipsis style="max-width: 300px">{{ record.url }}</a-typography-text>
              </template>
              <template v-if="column.key === 'type'">
                <a-tag :color="getTypeColor(record.type)">{{ record.type }}</a-tag>
              </template>
              <template v-if="column.key === 'status'">
                <a-badge :status="record.status === 'active' ? 'success' : 'default'" :text="record.status === 'active' ? '启用' : '禁用'" />
              </template>
              <template v-if="column.key === 'is_default'">
                <a-tag v-if="record.is_default" color="blue">默认</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="testAddressConnection(record)" :loading="testingConnectionId === record.id">测试连接</a-button>
                  <a-button type="link" size="small" @click="showAddressModal(record)">编辑</a-button>
                  <a-popconfirm title="确定删除？" @confirm="() => deleteAddress(record.id)" :disabled="!record.id">
                    <a-button type="link" size="small" danger :disabled="!record.id">删除</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 通知配置 Tab -->
      <a-tab-pane key="notify" tab="通知配置">
        <a-alert message="配置多个通知接收者，OA 数据同步时会自动向所有启用的配置发送飞书卡片" type="info" show-icon style="margin-bottom: 16px" />
        <a-card :bordered="false">
          <a-table :columns="notifyColumns" :data-source="notifyList" :loading="loadingNotify" row-key="id" :pagination="{ pageSize: 10 }">
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'app'">
                <span>{{ getAppName(record.app_id) }}</span>
              </template>
              <template v-if="column.key === 'receive_id_type'">
                <a-tag>{{ getReceiveTypeLabel(record.receive_id_type) }}</a-tag>
              </template>
              <template v-if="column.key === 'status'">
                <a-badge :status="record.status === 'active' ? 'success' : 'default'" :text="record.status === 'active' ? '启用' : '禁用'" />
              </template>
              <template v-if="column.key === 'is_default'">
                <a-tag v-if="record.is_default" color="blue">默认</a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space>
                  <a-button type="link" size="small" @click="showNotifyModal(record)">编辑</a-button>
                  <a-popconfirm title="确定删除？" @confirm="() => deleteNotify(record.id)">
                    <a-button type="link" size="small" danger>删除</a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <!-- 流程说明 Tab -->
      <a-tab-pane key="flow" tab="流程说明">
        <a-card title="OA-Jenkins 流程" :bordered="false">
          <a-timeline>
            <a-timeline-item color="blue">OA 系统推送审批数据</a-timeline-item>
            <a-timeline-item color="blue">同步服务拉取并解析数据</a-timeline-item>
            <a-timeline-item color="green">自动发送飞书发布卡片（向所有启用的通知配置）</a-timeline-item>
            <a-timeline-item color="orange">用户点击卡片按钮操作</a-timeline-item>
            <a-timeline-item color="purple">触发 Jenkins 构建任务</a-timeline-item>
            <a-timeline-item color="green">飞书推送构建结果</a-timeline-item>
          </a-timeline>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <!-- 数据详情抽屉 -->
    <a-drawer v-model:open="detailDrawerVisible" title="数据详情" placement="right" :width="700">
      <a-descriptions :column="1" bordered size="small">
        <a-descriptions-item label="ID"><a-typography-text copyable>{{ currentDetail?.unique_id }}</a-typography-text></a-descriptions-item>
        <a-descriptions-item label="来源"><a-tag v-if="currentDetail?.source" color="blue">{{ currentDetail?.source }}</a-tag><span v-else>-</span></a-descriptions-item>
        <a-descriptions-item label="同步时间">{{ formatTime(currentDetail?.created_at) }}</a-descriptions-item>
        <a-descriptions-item label="IP 地址">{{ currentDetail?.ip_address || '-' }}</a-descriptions-item>
        <a-descriptions-item label="User Agent">{{ currentDetail?.user_agent || '-' }}</a-descriptions-item>
      </a-descriptions>
      <a-divider orientation="left">原始数据</a-divider>
      <div class="json-toolbar"><a-button size="small" @click="copyCurrentJson"><CopyOutlined /> 复制</a-button></div>
      <pre class="json-preview">{{ formatJsonString(currentDetail?.original_data) }}</pre>
    </a-drawer>

    <!-- 地址编辑弹窗 -->
    <a-modal v-model:open="addressModalVisible" :title="editingAddressId ? '编辑地址' : '添加地址'" @ok="saveAddress" :confirm-loading="savingAddress">
      <a-form :model="editingAddress" layout="vertical">
        <a-form-item label="名称" required><a-input v-model:value="editingAddress.name" placeholder="OA 回调地址" /></a-form-item>
        <a-form-item label="URL" required><a-input v-model:value="editingAddress.url" placeholder="https://example.com/callback" /></a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="类型">
              <a-select v-model:value="editingAddress.type" style="width: 100%">
                <a-select-option value="webhook">Webhook</a-select-option>
                <a-select-option value="callback">Callback</a-select-option>
                <a-select-option value="api">API</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="状态">
              <a-select v-model:value="editingAddress.status" style="width: 100%">
                <a-select-option value="active">启用</a-select-option>
                <a-select-option value="inactive">禁用</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="描述"><a-textarea v-model:value="editingAddress.description" :rows="2" /></a-form-item>
        <a-form-item><a-checkbox v-model:checked="editingAddress.is_default">设为默认</a-checkbox></a-form-item>
      </a-form>
    </a-modal>

    <!-- 通知配置编辑弹窗 -->
    <a-modal v-model:open="notifyModalVisible" :title="editingNotifyId ? '编辑通知配置' : '添加通知配置'" @ok="saveNotify" :confirm-loading="savingNotify">
      <a-form :model="editingNotify" layout="vertical">
        <a-form-item label="配置名称" required><a-input v-model:value="editingNotify.name" placeholder="如：研发群通知" /></a-form-item>
        <a-form-item label="飞书应用" required>
          <a-select v-model:value="editingNotify.app_id" placeholder="选择飞书应用" style="width: 100%">
            <a-select-option :value="0">使用默认应用</a-select-option>
            <a-select-option v-for="app in feishuAppList" :key="app.id" :value="app.id">{{ app.name }} ({{ app.app_id }})</a-select-option>
          </a-select>
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="接收者ID" required><a-input v-model:value="editingNotify.receive_id" placeholder="飞书群组或用户ID" /></a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="ID类型" required>
              <a-select v-model:value="editingNotify.receive_id_type" style="width: 100%">
                <a-select-option value="chat_id">Chat ID (群组)</a-select-option>
                <a-select-option value="open_id">Open ID</a-select-option>
                <a-select-option value="user_id">User ID</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="状态">
          <a-select v-model:value="editingNotify.status" style="width: 100%">
            <a-select-option value="active">启用</a-select-option>
            <a-select-option value="inactive">禁用</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="描述"><a-textarea v-model:value="editingNotify.description" :rows="2" /></a-form-item>
        <a-form-item><a-checkbox v-model:checked="editingNotify.is_default">设为默认</a-checkbox></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined, CopyOutlined, SyncOutlined } from '@ant-design/icons-vue'
import { oaAddressApi, oaSyncApi, oaDataApi, oaNotifyApi, type OAAddress, type OASyncData, type OANotifyConfig } from '@/services/oa'
import { feishuAppApi, type FeishuApp } from '@/services/feishu'

const activeTab = ref('data')
const loadingData = ref(false)
const loadingAddress = ref(false)
const loadingNotify = ref(false)
const savingAddress = ref(false)
const savingNotify = ref(false)
const syncing = ref(false)
const forceSyncing = ref(false)
const sendingCardId = ref<number | null>(null)
const testingConnectionId = ref<number | null>(null)
const detailDrawerVisible = ref(false)
const addressModalVisible = ref(false)
const notifyModalVisible = ref(false)
const sourceSearch = ref('')
const dataList = ref<OASyncData[]>([])
const addressList = ref<OAAddress[]>([])
const notifyList = ref<OANotifyConfig[]>([])
const currentDetail = ref<OASyncData | null>(null)
const editingAddressId = ref<number | undefined>(undefined)
const editingAddress = reactive<Omit<OAAddress, 'id'>>({ name: '', url: '', type: 'webhook', status: 'active', is_default: false })
const editingNotifyId = ref<number | undefined>(undefined)
const editingNotify = reactive<Omit<OANotifyConfig, 'id'>>({ name: '', app_id: 0, receive_id: '', receive_id_type: 'chat_id', status: 'active', is_default: false })
const syncStatus = reactive({ running: false, synced_count: 0 })
const dataPagination = reactive({ current: 1, pageSize: 20, total: 0, showSizeChanger: true, showTotal: (total: number) => `共 ${total} 条` })
const feishuAppList = ref<FeishuApp[]>([])

const dataColumns = [
  { title: 'ID', dataIndex: 'unique_id', key: 'unique_id', width: 180 },
  { title: '来源', dataIndex: 'source', key: 'source', width: 120 },
  { title: '同步时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: 'IP 地址', dataIndex: 'ip_address', key: 'ip_address', width: 140 },
  { title: '数据预览', key: 'preview' },
  { title: '操作', key: 'action', width: 120 }
]

const addressColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 150 },
  { title: 'URL', dataIndex: 'url', key: 'url' },
  { title: '类型', dataIndex: 'type', key: 'type', width: 100 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '默认', dataIndex: 'is_default', key: 'is_default', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const notifyColumns = [
  { title: '名称', dataIndex: 'name', key: 'name', width: 150 },
  { title: '飞书应用', key: 'app', width: 180 },
  { title: '接收者ID', dataIndex: 'receive_id', key: 'receive_id', width: 180 },
  { title: 'ID类型', dataIndex: 'receive_id_type', key: 'receive_id_type', width: 120 },
  { title: '状态', dataIndex: 'status', key: 'status', width: 80 },
  { title: '默认', dataIndex: 'is_default', key: 'is_default', width: 80 },
  { title: '操作', key: 'action', width: 120 }
]

const getTypeColor = (type: string) => ({ webhook: 'blue', callback: 'green', api: 'orange' }[type] || 'default')
const getReceiveTypeLabel = (type: string) => ({ chat_id: '群组', open_id: 'Open ID', user_id: 'User ID' }[type] || type)
const getAppName = (appId: number) => {
  if (!appId) return '默认应用'
  const app = feishuAppList.value.find(a => a.id === appId)
  return app ? app.name : `应用ID: ${appId}`
}

const onTabChange = (key: string) => {
  if (key === 'data') fetchData()
  else if (key === 'address') fetchAddresses()
  else if (key === 'notify') fetchNotifyConfigs()
}

const fetchData = async () => {
  loadingData.value = true
  try {
    const response = await oaDataApi.list(dataPagination.current, dataPagination.pageSize, sourceSearch.value)
    if (response.code === 0 && response.data) {
      dataList.value = (response.data.list || []).map((item: any) => ({ ...item, id: item.id || item.ID }))
      dataPagination.total = response.data.total
    }
  } catch (error: any) { console.error('获取数据失败', error) }
  finally { loadingData.value = false }
}

const onDataTableChange = (pagination: any) => {
  dataPagination.current = pagination.current
  dataPagination.pageSize = pagination.pageSize
  fetchData()
}

const viewDetail = (record: OASyncData) => { currentDetail.value = record; detailDrawerVisible.value = true }

const sendCard = async (record: OASyncData) => {
  sendingCardId.value = record.id
  try {
    const response = await oaSyncApi.testSendCard(record.unique_id)
    if (response.code === 0) {
      message.success('卡片发送成功')
    } else {
      message.error(response.message || '发送失败')
    }
  } catch (error: any) {
    message.error(error.message || '发送失败')
  } finally {
    sendingCardId.value = null
  }
}

const deleteData = async (id: number) => {
  try {
    const response = await oaDataApi.delete(id)
    if (response.code === 0) { message.success('删除成功'); fetchData(); fetchSyncStatus() }
    else { message.error(response.message || '删除失败') }
  } catch (error: any) { message.error(error.message || '删除失败') }
}

const getPreviewFromString = (data: string) => (!data ? '-' : (data.length > 50 ? data.substring(0, 50) + '...' : data))

const fetchAddresses = async () => {
  loadingAddress.value = true
  try {
    const response = await oaAddressApi.list()
    if (response.code === 0 && response.data) {
      addressList.value = (response.data.list || []).map((item: any) => ({ ...item, id: item.id || item.ID }))
    }
  } catch (error: any) { console.error('获取地址失败', error) }
  finally { loadingAddress.value = false }
}

const fetchNotifyConfigs = async () => {
  loadingNotify.value = true
  try {
    const response = await oaNotifyApi.list()
    if (response.code === 0 && response.data) {
      notifyList.value = (response.data.list || []).map((item: any) => ({ ...item, id: item.id || item.ID }))
    }
  } catch (error: any) { console.error('获取通知配置失败', error) }
  finally { loadingNotify.value = false }
}

const fetchSyncStatus = async () => {
  try {
    const response = await oaSyncApi.getStatus()
    if (response.code === 0 && response.data) {
      syncStatus.running = response.data.running
      syncStatus.synced_count = response.data.synced_count
    }
  } catch (error: any) { console.error('获取同步状态失败', error) }
}

const triggerSync = async () => {
  syncing.value = true
  try {
    const response = await oaSyncApi.syncNow()
    if (response.code === 0) { message.success('同步已触发'); setTimeout(() => { fetchSyncStatus(); fetchData() }, 2000) }
    else { message.error(response.message || '触发失败') }
  } catch (error: any) { message.error(error.message || '触发失败') }
  finally { syncing.value = false }
}

const triggerForceSync = async () => {
  forceSyncing.value = true
  try {
    const response = await oaSyncApi.syncForce()
    if (response.code === 0) { message.success('强制同步已触发，缓存已清除'); setTimeout(() => { fetchSyncStatus(); fetchData() }, 3000) }
    else { message.error(response.message || '触发失败') }
  } catch (error: any) { message.error(error.message || '触发失败') }
  finally { forceSyncing.value = false }
}

const fetchFeishuApps = async () => {
  try {
    const response = await feishuAppApi.list()
    if (response.code === 0 && response.data) {
      feishuAppList.value = (response.data.list || []).filter((app: any) => app.status === 'active')
    }
  } catch (error: any) { console.error('获取飞书应用列表失败', error) }
}

const truncateId = (id: string) => (!id || id.length <= 20) ? id : id.substring(0, 8) + '...' + id.substring(id.length - 8)
const formatTime = (time: string | undefined) => time ? time.replace('T', ' ').substring(0, 19) : '-'
const formatJsonString = (data: string | undefined) => {
  if (!data) return ''
  try { return JSON.stringify(JSON.parse(data), null, 2) }
  catch { return data }
}
const copyCurrentJson = () => {
  if (currentDetail.value) { navigator.clipboard.writeText(formatJsonString(currentDetail.value.original_data)); message.success('已复制') }
}

const showAddressModal = (addr?: OAAddress) => {
  if (addr) {
    editingAddressId.value = addr.id
    editingAddress.name = addr.name; editingAddress.url = addr.url; editingAddress.type = addr.type
    editingAddress.description = addr.description; editingAddress.status = addr.status; editingAddress.is_default = addr.is_default
  } else {
    editingAddressId.value = undefined
    editingAddress.name = ''; editingAddress.url = ''; editingAddress.type = 'webhook'
    editingAddress.description = ''; editingAddress.status = 'active'; editingAddress.is_default = false
  }
  addressModalVisible.value = true
}

const saveAddress = async () => {
  if (!editingAddress.name || !editingAddress.url) { message.error('请填写名称和URL'); return }
  savingAddress.value = true
  try {
    const data = { ...editingAddress }
    const response = editingAddressId.value ? await oaAddressApi.update(editingAddressId.value, data) : await oaAddressApi.create(data)
    if (response.code === 0) { message.success(editingAddressId.value ? '更新成功' : '添加成功'); addressModalVisible.value = false; fetchAddresses() }
    else { message.error(response.message || '保存失败') }
  } catch (error: any) { message.error(error.message || '保存失败') }
  finally { savingAddress.value = false }
}

const deleteAddress = async (id: number | undefined) => {
  if (!id) { message.error('无效的地址ID'); return }
  try {
    const response = await oaAddressApi.delete(id)
    if (response.code === 0) { message.success('删除成功'); fetchAddresses() }
    else { message.error(response.message || '删除失败') }
  } catch (error: any) { message.error(error.message || '删除失败') }
}

const testAddressConnection = async (record: OAAddress) => {
  if (!record.id) { message.error('无效的地址ID'); return }
  testingConnectionId.value = record.id
  try {
    const response = await oaAddressApi.testConnection(record.id)
    if (response.code === 0 && response.data) {
      if (response.data.connected) {
        message.success(`连接成功！响应时间: ${response.data.response_time_ms}ms${response.data.server_version ? `, 服务器: ${response.data.server_version}` : ''}`)
      } else {
        message.error(`连接失败: ${response.data.error || '未知错误'}`)
      }
    } else {
      message.error(response.message || '测试失败')
    }
  } catch (error: any) {
    message.error(error.message || '测试失败')
  } finally {
    testingConnectionId.value = null
  }
}

const showNotifyModal = (config?: OANotifyConfig) => {
  if (config) {
    editingNotifyId.value = config.id
    editingNotify.name = config.name; editingNotify.app_id = config.app_id; editingNotify.receive_id = config.receive_id
    editingNotify.receive_id_type = config.receive_id_type; editingNotify.description = config.description || ''
    editingNotify.status = config.status; editingNotify.is_default = config.is_default
  } else {
    editingNotifyId.value = undefined
    editingNotify.name = ''; editingNotify.app_id = 0; editingNotify.receive_id = ''
    editingNotify.receive_id_type = 'chat_id'; editingNotify.description = ''; editingNotify.status = 'active'; editingNotify.is_default = false
  }
  notifyModalVisible.value = true
}

const saveNotify = async () => {
  if (!editingNotify.name || !editingNotify.receive_id) { message.error('请填写名称和接收者ID'); return }
  savingNotify.value = true
  try {
    const data = { ...editingNotify }
    const response = editingNotifyId.value ? await oaNotifyApi.update(editingNotifyId.value, data) : await oaNotifyApi.create(data)
    if (response.code === 0) { message.success(editingNotifyId.value ? '更新成功' : '添加成功'); notifyModalVisible.value = false; fetchNotifyConfigs() }
    else { message.error(response.message || '保存失败') }
  } catch (error: any) { message.error(error.message || '保存失败') }
  finally { savingNotify.value = false }
}

const deleteNotify = async (id: number | undefined) => {
  if (!id) { message.error('无效的配置ID'); return }
  try {
    const response = await oaNotifyApi.delete(id)
    if (response.code === 0) { message.success('删除成功'); fetchNotifyConfigs() }
    else { message.error(response.message || '删除失败') }
  } catch (error: any) { message.error(error.message || '删除失败') }
}

onMounted(() => { fetchData(); fetchAddresses(); fetchNotifyConfigs(); fetchSyncStatus(); fetchFeishuApps() })
</script>

<style scoped>
.oa-data { padding: 0; }
.page-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 16px; }
.page-header h1 { font-size: 20px; font-weight: 500; margin: 0; }
.json-toolbar { margin-bottom: 12px; }
.json-preview { background: #1e1e1e; color: #d4d4d4; padding: 16px; border-radius: 6px; max-height: 500px; overflow: auto; font-size: 13px; font-family: monospace; }
</style>