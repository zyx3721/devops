<template>
  <div class="deploy-request-detail">
    <a-page-header :title="`发布请求 #${requestId}`" @back="goBack">
      <template #extra>
        <a-space v-if="request?.status === 'pending' && canApprove">
          <a-button type="primary" @click="handleApprove"><CheckOutlined /> 通过</a-button>
          <a-button danger @click="showRejectModal"><CloseOutlined /> 拒绝</a-button>
        </a-space>
      </template>
    </a-page-header>

    <a-spin :spinning="loading">
      <a-row :gutter="16">
        <a-col :span="16">
          <a-card title="请求信息" :bordered="false">
            <a-descriptions :column="2" bordered size="small">
              <a-descriptions-item label="应用名称">{{ request?.app_name }}</a-descriptions-item>
              <a-descriptions-item label="环境">
                <a-tag :color="envColors[request?.env_name]">{{ request?.env_name }}</a-tag>
              </a-descriptions-item>
              <a-descriptions-item label="版本">{{ request?.version || request?.image_tag || '-' }}</a-descriptions-item>
              <a-descriptions-item label="分支">{{ request?.branch || '-' }}</a-descriptions-item>
              <a-descriptions-item label="Commit">{{ request?.commit_id?.substring(0, 8) || '-' }}</a-descriptions-item>
              <a-descriptions-item label="申请人">{{ request?.operator }}</a-descriptions-item>
              <a-descriptions-item label="申请时间">{{ fmtTime(request?.created_at) }}</a-descriptions-item>
              <a-descriptions-item label="状态">
                <a-badge :status="statusType[request?.status]" :text="statusText[request?.status]" />
              </a-descriptions-item>
              <a-descriptions-item label="发布说明" :span="2">{{ request?.description || '-' }}</a-descriptions-item>
              <a-descriptions-item v-if="request?.is_emergency" label="紧急发布" :span="2">
                <a-tag color="red">是</a-tag>
                <span style="margin-left: 8px">原因: {{ request?.emergency_reason }}</span>
              </a-descriptions-item>
            </a-descriptions>
          </a-card>

          <!-- 审批流程时间线 -->
          <a-card title="审批流程" :bordered="false" style="margin-top: 16px">
            <a-timeline>
              <a-timeline-item color="blue">
                <p><strong>{{ request?.operator }}</strong> 提交发布请求</p>
                <p class="timeline-time">{{ fmtTime(request?.created_at) }}</p>
              </a-timeline-item>
              <a-timeline-item 
                v-for="record in approvalRecords" 
                :key="record.id"
                :color="record.action === 'approve' ? 'green' : 'red'"
              >
                <p>
                  <strong>{{ record.approver_name }}</strong> 
                  {{ record.action === 'approve' ? '通过' : '拒绝' }}了审批
                </p>
                <p class="timeline-time">{{ fmtTime(record.created_at) }}</p>
                <p v-if="record.comment" class="timeline-comment">{{ record.comment }}</p>
              </a-timeline-item>
              <a-timeline-item v-if="request?.status === 'pending'" color="gray">
                <p>等待审批中...</p>
              </a-timeline-item>
              <a-timeline-item v-if="request?.status === 'approved' && request?.deploy_started_at" color="blue">
                <p>开始执行部署</p>
                <p class="timeline-time">{{ fmtTime(request?.deploy_started_at) }}</p>
              </a-timeline-item>
              <a-timeline-item v-if="request?.status === 'success'" color="green">
                <p>部署成功</p>
                <p class="timeline-time">{{ fmtTime(request?.finished_at) }}</p>
              </a-timeline-item>
              <a-timeline-item v-if="request?.status === 'failed'" color="red">
                <p>部署失败</p>
                <p class="timeline-time">{{ fmtTime(request?.finished_at) }}</p>
                <p v-if="request?.error_msg" class="timeline-error">{{ request?.error_msg }}</p>
              </a-timeline-item>
            </a-timeline>
          </a-card>
        </a-col>

        <a-col :span="8">
          <!-- 审批人信息 -->
          <a-card title="审批人" :bordered="false">
            <a-list :data-source="approvers" size="small">
              <template #renderItem="{ item }">
                <a-list-item>
                  <a-list-item-meta :title="item.name" :description="item.role">
                    <template #avatar>
                      <a-avatar :style="{ backgroundColor: item.approved ? '#52c41a' : '#1890ff' }">
                        {{ item.name?.charAt(0) }}
                      </a-avatar>
                    </template>
                  </a-list-item-meta>
                  <template #actions>
                    <a-tag v-if="item.approved" color="green">已审批</a-tag>
                    <a-tag v-else-if="item.rejected" color="red">已拒绝</a-tag>
                    <a-tag v-else>待审批</a-tag>
                  </template>
                </a-list-item>
              </template>
            </a-list>
          </a-card>

          <!-- 关联信息 -->
          <a-card title="关联信息" :bordered="false" style="margin-top: 16px">
            <a-descriptions :column="1" size="small">
              <a-descriptions-item label="审批规则">{{ request?.approval_rule_name || '-' }}</a-descriptions-item>
              <a-descriptions-item label="发布窗口">{{ request?.deploy_window_name || '-' }}</a-descriptions-item>
              <a-descriptions-item label="Jenkins Job">{{ request?.jenkins_job || '-' }}</a-descriptions-item>
              <a-descriptions-item label="K8s Deployment">{{ request?.k8s_deployment || '-' }}</a-descriptions-item>
            </a-descriptions>
          </a-card>
        </a-col>
      </a-row>
    </a-spin>

    <!-- 拒绝原因弹窗 -->
    <a-modal v-model:open="rejectModalVisible" title="拒绝原因" @ok="handleReject" :confirmLoading="rejecting">
      <a-form layout="vertical">
        <a-form-item label="拒绝原因" required>
          <a-textarea v-model:value="rejectReason" :rows="3" placeholder="请输入拒绝原因" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import { CheckOutlined, CloseOutlined } from '@ant-design/icons-vue'
import { approvalApi } from '@/services/approval'

const route = useRoute()
const router = useRouter()
const requestId = computed(() => Number(route.params.id))

const loading = ref(false)
const request = ref<any>(null)
const approvalRecords = ref<any[]>([])
const approvers = ref<any[]>([])
const canApprove = ref(false)

const rejectModalVisible = ref(false)
const rejectReason = ref('')
const rejecting = ref(false)

const envColors: Record<string, string> = { dev: 'blue', test: 'cyan', staging: 'orange', prod: 'red' }
const statusType: Record<string, string> = { pending: 'warning', approved: 'processing', running: 'processing', success: 'success', failed: 'error', rejected: 'error', cancelled: 'default' }
const statusText: Record<string, string> = { pending: '待审批', approved: '已通过', running: '运行中', success: '成功', failed: '失败', rejected: '已拒绝', cancelled: '已取消' }

const fmtTime = (t: string) => t ? t.replace('T', ' ').substring(0, 19) : '-'
const goBack = () => router.back()

const fetchRequest = async () => {
  loading.value = true
  try {
    const res = await approvalApi.getDeployRequest(requestId.value)
    if (res.code === 0 && res.data) {
      request.value = res.data.request
      approvalRecords.value = res.data.records || []
      approvers.value = res.data.approvers || []
      canApprove.value = res.data.can_approve || false
    }
  } catch (e) { console.error(e) }
  finally { loading.value = false }
}

const handleApprove = () => {
  Modal.confirm({
    title: '确认通过',
    content: '确定要通过此发布请求吗？通过后将自动执行部署。',
    onOk: async () => {
      try {
        const res = await approvalApi.approve(requestId.value, { comment: '' })
        if (res.code === 0) {
          message.success('审批通过')
          fetchRequest()
        }
      } catch (e: any) { message.error(e.message || '操作失败') }
    }
  })
}

const showRejectModal = () => {
  rejectReason.value = ''
  rejectModalVisible.value = true
}

const handleReject = async () => {
  if (!rejectReason.value) { message.error('请输入拒绝原因'); return }
  rejecting.value = true
  try {
    const res = await approvalApi.reject(requestId.value, { comment: rejectReason.value })
    if (res.code === 0) {
      message.success('已拒绝')
      rejectModalVisible.value = false
      fetchRequest()
    }
  } catch (e: any) { message.error(e.message || '操作失败') }
  finally { rejecting.value = false }
}

onMounted(() => { fetchRequest() })
</script>

<style scoped>
.deploy-request-detail { padding: 0; }
.timeline-time { color: #999; font-size: 12px; margin: 0; }
.timeline-comment { color: #666; font-style: italic; margin: 4px 0 0; }
.timeline-error { color: #ff4d4f; margin: 4px 0 0; }
</style>
