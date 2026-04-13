<template>
  <div class="instance-page">
    <a-card :bordered="false">
      <template #title>
        <div class="page-header">
          <a-button @click="goBack">
            <template #icon><ArrowLeftOutlined /></template>
            返回
          </a-button>
          <span class="title">审批实例详情</span>
          <a-button v-if="instance?.status === 'pending'" danger @click="handleCancel">取消审批</a-button>
        </div>
      </template>

      <a-spin :spinning="loading">
        <ApprovalInstanceDetail
          v-if="instance"
          :instance="instance"
          @refresh="loadInstance"
        />
        <a-empty v-else-if="!loading" description="实例不存在" />
      </a-spin>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message, Modal } from 'ant-design-vue'
import { ArrowLeftOutlined } from '@ant-design/icons-vue'
import ApprovalInstanceDetail from './ApprovalInstanceDetail.vue'
import { getInstance, cancelInstance, type ApprovalInstance } from '@/services/approvalChain'

const route = useRoute()
const router = useRouter()

const loading = ref(false)
const instance = ref<ApprovalInstance | null>(null)

const loadInstance = async () => {
  const id = Number(route.params.id)
  if (!id) return

  loading.value = true
  try {
    const res = await getInstance(id)
    instance.value = res.data
  } catch (error: any) {
    message.error(error.message || '加载失败')
  } finally {
    loading.value = false
  }
}

const goBack = () => {
  router.push('/approval/instances')
}

const handleCancel = () => {
  Modal.confirm({
    title: '取消审批',
    content: '确定要取消此审批实例吗？',
    okType: 'danger',
    onOk: async () => {
      const reason = window.prompt('请输入取消原因（可选）') || ''
      try {
        await cancelInstance(instance.value!.id, reason)
        message.success('已取消')
        loadInstance()
      } catch (error: any) {
        message.error(error.message || '取消失败')
      }
    }
  })
}

onMounted(() => {
  loadInstance()
})
</script>

<style scoped>
.instance-page {
  padding: 16px;
}

.page-header {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-header .title {
  flex: 1;
  font-size: 16px;
  font-weight: 500;
}
</style>
