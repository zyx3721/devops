<template>
  <div class="pending-approvals">
    <!-- 切换标签 -->
    <a-tabs v-model:activeKey="activeTab" @change="handleTabChange">
      <a-tab-pane key="chain" tab="审批链待审批">
        <!-- 审批链待审批列表 -->
        <a-card :bordered="false">
          <a-table
            :columns="chainColumns"
            :data-source="chainRecords"
            :loading="chainLoading"
            row-key="id"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'chain_name'">
                <span>{{ record.chain_name }}</span>
              </template>
              <template v-else-if="column.key === 'current_node'">
                <a-tag color="processing">
                  第 {{ record.current_node_order }} 节点
                </a-tag>
                <span v-if="getCurrentNodeName(record)" style="margin-left: 8px; color: #666;">
                  {{ getCurrentNodeName(record) }}
                </span>
              </template>
              <template v-else-if="column.key === 'status'">
                <a-tag :color="getChainStatusColor(record.status)">
                  {{ getChainStatusText(record.status) }}
                </a-tag>
              </template>
              <template v-else-if="column.key === 'started_at'">
                {{ formatTime(record.started_at) }}
              </template>
              <template v-else-if="column.key === 'action'">
                <a-space>
                  <a-button type="primary" size="small" @click="showChainApproveModal(record)">
                    审批
                  </a-button>
                  <a-button type="link" size="small" @click="showChainDetail(record)">
                    详情
                  </a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="legacy" tab="旧版待审批">
        <!-- 旧版待审批列表 -->
        <a-card :bordered="false">
          <a-table
            :columns="legacyColumns"
            :data-source="legacyRecords"
            :loading="legacyLoading"
            row-key="id"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'app_name'">
                <span>{{ record.app_name }}</span>
              </template>
              <template v-else-if="column.key === 'env_name'">
                <a-tag :color="getEnvColor(record.env_name)">{{ record.env_name }}</a-tag>
              </template>
              <template v-else-if="column.key === 'status'">
                <a-tag :color="getStatusColor(record.status)">{{ getStatusText(record.status) }}</a-tag>
              </template>
              <template v-else-if="column.key === 'operator'">
                {{ record.operator }}
              </template>
              <template v-else-if="column.key === 'created_at'">
                {{ formatTime(record.created_at) }}
              </template>
              <template v-else-if="column.key === 'action'">
                <a-space>
                  <a-button type="primary" size="small" @click="showApproveModal(record)">通过</a-button>
                  <a-button type="primary" size="small" danger @click="showRejectModal(record)">拒绝</a-button>
                  <a-button type="link" size="small" @click="showDetail(record)">详情</a-button>
                </a-space>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-tab-pane>
    </a-tabs>

    <!-- 审批链审批弹窗 -->
    <a-modal
      v-model:open="chainApproveModalVisible"
      title="审批操作"
      width="600px"
      :footer="null"
    >
      <div v-if="currentChainRecord">
        <a-descriptions :column="2" bordered size="small" style="margin-bottom: 16px;">
          <a-descriptions-item label="审批链">{{ currentChainRecord.chain_name }}</a-descriptions-item>
          <a-descriptions-item label="部署记录">{{ currentChainRecord.record_id }}</a-descriptions-item>
          <a-descriptions-item label="当前节点">
            第 {{ currentChainRecord.current_node_order }} 节点
          </a-descriptions-item>
          <a-descriptions-item label="开始时间">{{ formatTime(currentChainRecord.started_at) }}</a-descriptions-item>
        </a-descriptions>

        <!-- 当前节点信息 -->
        <div v-if="currentNodeInstance" class="current-node-info">
          <h4>当前审批节点：{{ currentNodeInstance.node_name }}</h4>
          <p>审批模式：{{ getModeLabel(currentNodeInstance.approve_mode, currentNodeInstance.approve_count) }}</p>
          <p>审批进度：{{ currentNodeInstance.approved_count }} 通过 / {{ currentNodeInstance.rejected_count }} 拒绝</p>
        </div>

        <a-divider />

        <a-form :label-col="{ span: 4 }">
          <a-form-item label="审批意见">
            <a-textarea v-model:value="chainApproveComment" placeholder="可选，填写审批意见" :rows="3" />
          </a-form-item>
        </a-form>

        <div style="text-align: right; margin-top: 16px;">
          <a-space>
            <a-button @click="chainApproveModalVisible = false">取消</a-button>
            <a-button type="primary" danger @click="handleChainReject" :loading="submitting">
              拒绝
            </a-button>
            <a-button type="primary" @click="handleChainApprove" :loading="submitting">
              通过
            </a-button>
          </a-space>
        </div>
      </div>
    </a-modal>

    <!-- 审批链详情弹窗 -->
    <a-modal
      v-model:open="chainDetailModalVisible"
      title="审批详情"
      width="800px"
      :footer="null"
    >
      <ApprovalInstanceDetail
        v-if="currentChainRecord"
        :instance="currentChainRecord"
        @refresh="loadChainDetail"
      />
    </a-modal>

    <!-- 旧版审批通过弹窗 -->
    <a-modal v-model:open="approveModalVisible" title="审批通过" @ok="handleApprove" :confirm-loading="submitting">
      <a-form :label-col="{ span: 4 }">
        <a-form-item label="应用">{{ currentRecord?.app_name }}</a-form-item>
        <a-form-item label="环境">{{ currentRecord?.env_name }}</a-form-item>
        <a-form-item label="版本">{{ currentRecord?.version || currentRecord?.image_tag }}</a-form-item>
        <a-form-item label="申请人">{{ currentRecord?.operator }}</a-form-item>
        <a-form-item label="审批意见">
          <a-textarea v-model:value="approveComment" placeholder="可选，填写审批意见" :rows="3" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 旧版审批拒绝弹窗 -->
    <a-modal v-model:open="rejectModalVisible" title="审批拒绝" @ok="handleReject" :confirm-loading="submitting">
      <a-form :label-col="{ span: 4 }">
        <a-form-item label="应用">{{ currentRecord?.app_name }}</a-form-item>
        <a-form-item label="环境">{{ currentRecord?.env_name }}</a-form-item>
        <a-form-item label="版本">{{ currentRecord?.version || currentRecord?.image_tag }}</a-form-item>
        <a-form-item label="申请人">{{ currentRecord?.operator }}</a-form-item>
        <a-form-item label="拒绝原因" required>
          <a-textarea v-model:value="rejectReason" placeholder="请填写拒绝原因" :rows="3" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 旧版详情弹窗 -->
    <a-modal v-model:open="detailModalVisible" title="发布详情" :footer="null" width="700px">
      <a-descriptions :column="2" bordered size="small">
        <a-descriptions-item label="应用名称">{{ currentRecord?.app_name }}</a-descriptions-item>
        <a-descriptions-item label="环境">{{ currentRecord?.env_name }}</a-descriptions-item>
        <a-descriptions-item label="版本">{{ currentRecord?.version }}</a-descriptions-item>
        <a-descriptions-item label="镜像标签">{{ currentRecord?.image_tag }}</a-descriptions-item>
        <a-descriptions-item label="分支">{{ currentRecord?.branch }}</a-descriptions-item>
        <a-descriptions-item label="Commit">{{ currentRecord?.commit_id?.substring(0, 8) }}</a-descriptions-item>
        <a-descriptions-item label="申请人">{{ currentRecord?.operator }}</a-descriptions-item>
        <a-descriptions-item label="申请时间">{{ formatTime(currentRecord?.created_at) }}</a-descriptions-item>
        <a-descriptions-item label="发布说明" :span="2">{{ currentRecord?.description || '-' }}</a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </div>
</template>


<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { approvalApi } from '@/services/approval'
import {
  getPendingApprovals,
  getInstance,
  approveNode,
  rejectNode,
  type ApprovalInstance,
  type ApprovalNodeInstance
} from '@/services/approvalChain'
import ApprovalInstanceDetail from './ApprovalInstanceDetail.vue'
import dayjs from 'dayjs'

const activeTab = ref('chain')

// 审批链相关
const chainLoading = ref(false)
const chainRecords = ref<ApprovalInstance[]>([])
const currentChainRecord = ref<ApprovalInstance | null>(null)
const chainApproveModalVisible = ref(false)
const chainDetailModalVisible = ref(false)
const chainApproveComment = ref('')

// 旧版相关
const legacyLoading = ref(false)
const legacyRecords = ref<any[]>([])
const currentRecord = ref<any>(null)
const approveModalVisible = ref(false)
const rejectModalVisible = ref(false)
const detailModalVisible = ref(false)
const approveComment = ref('')
const rejectReason = ref('')

const submitting = ref(false)

// 审批链列表列
const chainColumns = [
  { title: '审批链', key: 'chain_name', dataIndex: 'chain_name' },
  { title: '部署记录', dataIndex: 'record_id', width: 100 },
  { title: '当前节点', key: 'current_node', width: 200 },
  { title: '状态', key: 'status', width: 100 },
  { title: '开始时间', key: 'started_at', width: 180 },
  { title: '操作', key: 'action', width: 150 }
]

// 旧版列表列
const legacyColumns = [
  { title: '应用', key: 'app_name', dataIndex: 'app_name' },
  { title: '环境', key: 'env_name', dataIndex: 'env_name' },
  { title: '版本', dataIndex: 'version' },
  { title: '状态', key: 'status', dataIndex: 'status' },
  { title: '申请人', key: 'operator', dataIndex: 'operator' },
  { title: '申请时间', key: 'created_at', dataIndex: 'created_at' },
  { title: '操作', key: 'action', width: 200 }
]

// 当前节点实例
const currentNodeInstance = computed(() => {
  if (!currentChainRecord.value?.node_instances) return null
  return currentChainRecord.value.node_instances.find(
    n => n.node_order === currentChainRecord.value!.current_node_order && n.status === 'active'
  )
})

// 获取当前节点名称
const getCurrentNodeName = (record: ApprovalInstance) => {
  if (!record.node_instances) return ''
  const node = record.node_instances.find(n => n.node_order === record.current_node_order)
  return node?.node_name || ''
}

// 颜色和文本映射
const getEnvColor = (env: string) => {
  const colors: Record<string, string> = {
    dev: 'blue', test: 'cyan', staging: 'orange', prod: 'red', production: 'red'
  }
  return colors[env] || 'default'
}

const getStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    pending: 'orange', approved: 'green', rejected: 'red', cancelled: 'default'
  }
  return colors[status] || 'default'
}

const getStatusText = (status: string) => {
  const texts: Record<string, string> = {
    pending: '待审批', approved: '已通过', rejected: '已拒绝', cancelled: '已取消'
  }
  return texts[status] || status
}

const getChainStatusColor = (status: string) => {
  const colors: Record<string, string> = {
    pending: 'processing', approved: 'success', rejected: 'error', cancelled: 'default'
  }
  return colors[status] || 'default'
}

const getChainStatusText = (status: string) => {
  const texts: Record<string, string> = {
    pending: '审批中', approved: '已通过', rejected: '已拒绝', cancelled: '已取消'
  }
  return texts[status] || status
}

const getModeLabel = (mode: string, count: number) => {
  const map: Record<string, string> = {
    any: '任一人通过',
    all: '所有人通过',
    count: `${count}人通过`
  }
  return map[mode] || mode
}

const formatTime = (time: string) => {
  return time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'
}

// 加载审批链待审批列表
const loadChainRecords = async () => {
  chainLoading.value = true
  try {
    const res = await getPendingApprovals()
    chainRecords.value = res.data || []
  } catch (error) {
    console.error('加载审批链待审批列表失败:', error)
  } finally {
    chainLoading.value = false
  }
}

// 加载旧版待审批列表
const loadLegacyRecords = async () => {
  legacyLoading.value = true
  try {
    const res = await approvalApi.getPendingList()
    legacyRecords.value = res.data || []
  } catch (error) {
    console.error('加载旧版待审批列表失败:', error)
  } finally {
    legacyLoading.value = false
  }
}

// 加载审批链详情
const loadChainDetail = async () => {
  if (!currentChainRecord.value) return
  try {
    const res = await getInstance(currentChainRecord.value.id)
    currentChainRecord.value = res.data
  } catch (error) {
    console.error('加载审批链详情失败:', error)
  }
}

// Tab 切换
const handleTabChange = (key: string) => {
  if (key === 'chain') {
    loadChainRecords()
  } else {
    loadLegacyRecords()
  }
}

// 审批链操作
const showChainApproveModal = async (record: ApprovalInstance) => {
  try {
    const res = await getInstance(record.id)
    currentChainRecord.value = res.data
    chainApproveComment.value = ''
    chainApproveModalVisible.value = true
  } catch (error: any) {
    message.error(error.message || '加载详情失败')
  }
}

const showChainDetail = async (record: ApprovalInstance) => {
  try {
    const res = await getInstance(record.id)
    currentChainRecord.value = res.data
    chainDetailModalVisible.value = true
  } catch (error: any) {
    message.error(error.message || '加载详情失败')
  }
}

const handleChainApprove = async () => {
  if (!currentNodeInstance.value) {
    message.error('未找到当前审批节点')
    return
  }
  submitting.value = true
  try {
    await approveNode(currentNodeInstance.value.id, chainApproveComment.value)
    message.success('审批通过')
    chainApproveModalVisible.value = false
    loadChainRecords()
  } catch (error: any) {
    message.error(error.message || '审批失败')
  } finally {
    submitting.value = false
  }
}

const handleChainReject = async () => {
  if (!currentNodeInstance.value) {
    message.error('未找到当前审批节点')
    return
  }
  if (!chainApproveComment.value) {
    message.error('请填写拒绝原因')
    return
  }
  submitting.value = true
  try {
    await rejectNode(currentNodeInstance.value.id, chainApproveComment.value)
    message.success('已拒绝')
    chainApproveModalVisible.value = false
    loadChainRecords()
  } catch (error: any) {
    message.error(error.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

// 旧版操作
const showApproveModal = (record: any) => {
  currentRecord.value = record
  approveComment.value = ''
  approveModalVisible.value = true
}

const showRejectModal = (record: any) => {
  currentRecord.value = record
  rejectReason.value = ''
  rejectModalVisible.value = true
}

const showDetail = (record: any) => {
  currentRecord.value = record
  detailModalVisible.value = true
}

const handleApprove = async () => {
  submitting.value = true
  try {
    await approvalApi.approve(currentRecord.value.id, approveComment.value)
    message.success('审批通过')
    approveModalVisible.value = false
    loadLegacyRecords()
  } finally {
    submitting.value = false
  }
}

const handleReject = async () => {
  if (!rejectReason.value) {
    message.error('请填写拒绝原因')
    return
  }
  submitting.value = true
  try {
    await approvalApi.reject(currentRecord.value.id, rejectReason.value)
    message.success('已拒绝')
    rejectModalVisible.value = false
    loadLegacyRecords()
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadChainRecords()
})
</script>

<style scoped>
.pending-approvals {
  padding: 16px;
}

.current-node-info {
  background: #f5f5f5;
  padding: 12px 16px;
  border-radius: 4px;
  margin-bottom: 16px;
}

.current-node-info h4 {
  margin: 0 0 8px 0;
  color: #1890ff;
}

.current-node-info p {
  margin: 4px 0;
  color: #666;
}
</style>
