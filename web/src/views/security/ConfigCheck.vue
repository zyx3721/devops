<template>
  <div class="config-check">
    <!-- 检查表单 -->
    <a-card title="K8s 配置安全检查" style="margin-bottom: 16px">
      <a-form layout="inline">
        <a-form-item label="集群">
          <a-select v-model:value="checkForm.cluster_id" placeholder="选择集群" style="width: 200px" @change="loadNamespaces">
            <a-select-option v-for="c in clusters" :key="c.id" :value="c.id">{{ c.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="命名空间">
          <a-select v-model:value="checkForm.namespace" placeholder="全部命名空间" style="width: 200px" allowClear>
            <a-select-option value="">全部命名空间</a-select-option>
            <a-select-option v-for="ns in namespaces" :key="ns" :value="ns">{{ ns }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" :loading="checking" @click="handleCheck">
            <template #icon><SafetyCertificateOutlined /></template>
            开始检查
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 检查历史 -->
    <a-card title="检查历史">
      <a-table :dataSource="checkHistory" :loading="loading" :pagination="pagination" @change="handleTableChange" rowKey="id">
        <a-table-column title="集群" dataIndex="cluster_name" :width="150" />
        <a-table-column title="命名空间" dataIndex="namespace" :width="150">
          <template #default="{ record }">{{ record.namespace || '全部' }}</template>
        </a-table-column>
        <a-table-column title="状态" dataIndex="status" :width="100">
          <template #default="{ record }">
            <a-tag :color="getStatusColor(record.status)">{{ getStatusLabel(record.status) }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="问题统计" :width="200">
          <template #default="{ record }">
            <span style="color: #cf1322; margin-right: 8px">高 {{ record.high_count }}</span>
            <span style="color: #d46b08; margin-right: 8px">中 {{ record.medium_count }}</span>
            <span style="color: #389e0d">低 {{ record.low_count }}</span>
          </template>
        </a-table-column>
        <a-table-column title="检查时间" dataIndex="checked_at" :width="180">
          <template #default="{ record }">{{ formatTime(record.checked_at) }}</template>
        </a-table-column>
        <a-table-column title="操作" :width="100" fixed="right">
          <template #default="{ record }">
            <a-button type="link" @click="viewResult(record)">查看详情</a-button>
          </template>
        </a-table-column>
      </a-table>
    </a-card>

    <!-- 检查结果详情 -->
    <a-drawer v-model:open="showResult" title="检查结果详情" width="70%">
      <template v-if="currentResult">
        <a-descriptions :column="3" bordered style="margin-bottom: 16px">
          <a-descriptions-item label="集群">{{ currentResult.cluster_name }}</a-descriptions-item>
          <a-descriptions-item label="命名空间">{{ currentResult.namespace || '全部' }}</a-descriptions-item>
          <a-descriptions-item label="检查时间">{{ formatTime(currentResult.checked_at) }}</a-descriptions-item>
        </a-descriptions>

        <a-table :dataSource="currentResult.issues" :scroll="{ y: 500 }" rowKey="id">
          <a-table-column title="资源" :width="200">
            <template #default="{ record }">
              <div>{{ record.resource_type }}/{{ record.resource_name }}</div>
              <div style="color: #999; font-size: 12px">{{ record.namespace }}</div>
            </template>
          </a-table-column>
          <a-table-column title="规则" dataIndex="rule_name" :width="150" />
          <a-table-column title="严重程度" dataIndex="severity" :width="100">
            <template #default="{ record }">
              <a-tag :color="getSeverityColor(record.severity)">{{ getSeverityLabel(record.severity) }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="问题描述" dataIndex="message" :ellipsis="true" />
          <a-table-column title="修复建议" dataIndex="remediation" :ellipsis="true" />
        </a-table>
      </template>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { SafetyCertificateOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { runConfigCheck, getCheckHistory, getCheckResult } from '@/services/security'
import { k8sClusterApi, k8sResourceApi } from '@/services/k8s'
import dayjs from 'dayjs'

const checkForm = ref({ cluster_id: undefined as number | undefined, namespace: '' })
const checking = ref(false)
const loading = ref(false)
const clusters = ref<any[]>([])
const namespaces = ref<string[]>([])
const checkHistory = ref<any[]>([])
const pagination = ref({ current: 1, pageSize: 20, total: 0 })
const showResult = ref(false)
const currentResult = ref<any>(null)

const loadClusters = async () => {
  try {
    const res: any = await k8sClusterApi.list()
    clusters.value = res?.data?.items || []
    // 如果只有一个集群，自动选中并加载命名空间
    if (clusters.value.length === 1) {
      checkForm.value.cluster_id = clusters.value[0].id
      loadNamespaces()
    }
  } catch (error) {
    console.error('加载集群列表失败', error)
  }
}

const loadNamespaces = async () => {
  if (!checkForm.value.cluster_id) {
    namespaces.value = []
    return
  }
  try {
    const res: any = await k8sResourceApi.getNamespaces(checkForm.value.cluster_id)
    // 提取命名空间名称
    const nsList = res?.data || []
    namespaces.value = nsList.map((ns: any) => ns.name || ns)
  } catch (error) {
    console.error('加载命名空间失败', error)
  }
}

const loadHistory = async () => {
  loading.value = true
  try {
    const res = await getCheckHistory({ page: pagination.value.current, page_size: pagination.value.pageSize })
    checkHistory.value = res?.data?.items || []
    pagination.value.total = res?.data?.total || 0
  } catch (error) {
    console.error('加载检查历史失败', error)
  } finally {
    loading.value = false
  }
}

const handleTableChange = (pag: any) => {
  pagination.value.current = pag.current
  pagination.value.pageSize = pag.pageSize
  loadHistory()
}

const handleCheck = async () => {
  if (!checkForm.value.cluster_id) {
    message.warning('请选择集群')
    return
  }
  checking.value = true
  try {
    // 发送请求，空字符串表示全部命名空间
    const res = await runConfigCheck({
      cluster_id: checkForm.value.cluster_id,
      namespace: checkForm.value.namespace || ''
    })
    message.success('检查完成')
    currentResult.value = res?.data
    showResult.value = true
    loadHistory()
  } catch (error: any) {
    message.error(error.message || '检查失败')
  } finally {
    checking.value = false
  }
}

const viewResult = async (row: any) => {
  try {
    const res = await getCheckResult(row.id)
    currentResult.value = res?.data
    showResult.value = true
  } catch (error) {
    message.error('获取检查结果失败')
  }
}

const getStatusColor = (status: string) => ({ completed: 'green', checking: 'orange', failed: 'red' }[status] || 'default')
const getStatusLabel = (status: string) => ({ completed: '已完成', checking: '检查中', failed: '失败' }[status] || status)
const getSeverityColor = (s: string) => ({ high: 'red', medium: 'orange', low: 'green' }[s] || 'default')
const getSeverityLabel = (s: string) => ({ high: '高', medium: '中', low: '低' }[s] || s)
const formatTime = (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'

onMounted(() => { loadClusters(); loadHistory() })
</script>
