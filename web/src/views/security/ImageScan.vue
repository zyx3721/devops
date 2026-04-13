<template>
  <div class="image-scan">
    <!-- 扫描表单 -->
    <a-card title="镜像漏洞扫描" style="margin-bottom: 16px">
      <a-form layout="inline">
        <a-form-item label="镜像地址">
          <a-input v-model:value="scanForm.image" placeholder="例如: nginx:latest" style="width: 400px" />
        </a-form-item>
        <a-form-item label="镜像仓库">
          <a-select v-model:value="scanForm.registry_id" placeholder="选择仓库（可选）" allowClear style="width: 200px">
            <a-select-option v-for="r in registries" :key="r.id" :value="r.id">{{ r.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item>
          <a-button type="primary" :loading="scanning" @click="handleScan">
            <template #icon><SearchOutlined /></template>
            开始扫描
          </a-button>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 扫描历史 -->
    <a-card title="扫描历史">
      <template #extra>
        <a-input v-model:value="searchImage" placeholder="搜索镜像" style="width: 200px" allowClear @change="loadHistory">
          <template #prefix><SearchOutlined /></template>
        </a-input>
      </template>

      <a-table :dataSource="scanHistory" :loading="loading" :pagination="pagination" @change="handleTableChange" rowKey="id">
        <a-table-column title="镜像" dataIndex="image" :ellipsis="true" />
        <a-table-column title="状态" dataIndex="status" :width="100">
          <template #default="{ record }">
            <a-tag :color="getStatusColor(record.status)">{{ getStatusLabel(record.status) }}</a-tag>
          </template>
        </a-table-column>
        <a-table-column title="风险等级" dataIndex="risk_level" :width="100">
          <template #default="{ record }">
            <a-tag v-if="record.risk_level" :color="getRiskColor(record.risk_level)">{{ getRiskLabel(record.risk_level) }}</a-tag>
            <span v-else>-</span>
          </template>
        </a-table-column>
        <a-table-column title="漏洞统计" :width="200">
          <template #default="{ record }">
            <div class="vuln-stats">
              <span class="critical">{{ record.critical_count }}</span>
              <span class="high">{{ record.high_count }}</span>
              <span class="medium">{{ record.medium_count }}</span>
              <span class="low">{{ record.low_count }}</span>
            </div>
          </template>
        </a-table-column>
        <a-table-column title="扫描时间" dataIndex="scanned_at" :width="180">
          <template #default="{ record }">{{ formatTime(record.scanned_at) }}</template>
        </a-table-column>
        <a-table-column title="操作" :width="100" fixed="right">
          <template #default="{ record }">
            <a-button type="link" @click="viewResult(record)">查看详情</a-button>
          </template>
        </a-table-column>
      </a-table>
    </a-card>

    <!-- 扫描结果详情 -->
    <a-drawer v-model:open="showResult" title="扫描结果详情" width="60%">
      <template v-if="currentResult">
        <a-descriptions :column="2" bordered>
          <a-descriptions-item label="镜像">{{ currentResult.image }}</a-descriptions-item>
          <a-descriptions-item label="风险等级">
            <a-tag :color="getRiskColor(currentResult.risk_level)">{{ getRiskLabel(currentResult.risk_level) }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="扫描时间">{{ formatTime(currentResult.scanned_at) }}</a-descriptions-item>
          <a-descriptions-item label="漏洞总数">{{ currentResult.vuln_summary?.total || 0 }}</a-descriptions-item>
        </a-descriptions>

        <div class="vuln-summary">
          <a-tag color="red" :class="{ active: severityFilter === 'critical' }" @click="filterBySeverity('critical')">
            严重 {{ currentResult.vuln_summary?.critical || 0 }}
          </a-tag>
          <a-tag color="orange" :class="{ active: severityFilter === 'high' }" @click="filterBySeverity('high')">
            高危 {{ currentResult.vuln_summary?.high || 0 }}
          </a-tag>
          <a-tag color="blue" :class="{ active: severityFilter === 'medium' }" @click="filterBySeverity('medium')">
            中危 {{ currentResult.vuln_summary?.medium || 0 }}
          </a-tag>
          <a-tag color="green" :class="{ active: severityFilter === 'low' }" @click="filterBySeverity('low')">
            低危 {{ currentResult.vuln_summary?.low || 0 }}
          </a-tag>
          <a-tag v-if="severityFilter" @click="filterBySeverity('')">清除筛选</a-tag>
        </div>

        <a-table :dataSource="filteredVulnerabilities" style="margin-top: 16px" :scroll="{ y: 400 }" rowKey="vuln_id">
          <a-table-column title="CVE ID" dataIndex="vuln_id" :width="150" />
          <a-table-column title="包名" dataIndex="pkg_name" :width="150" :ellipsis="true" />
          <a-table-column title="严重程度" dataIndex="severity" :width="100">
            <template #default="{ record }">
              <a-tag :color="getRiskColor(record.severity)">{{ getRiskLabel(record.severity) }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="当前版本" dataIndex="installed_ver" :width="120" />
          <a-table-column title="修复版本" dataIndex="fixed_ver" :width="120" />
          <a-table-column title="描述" dataIndex="title" :ellipsis="true" />
        </a-table>
      </template>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { SearchOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { scanImage, getScanHistory, getScanResult, getRegistries } from '@/services/security'
import dayjs from 'dayjs'

const scanForm = ref({ image: '', registry_id: undefined as number | undefined })
const scanning = ref(false)
const loading = ref(false)
const searchImage = ref('')
const scanHistory = ref<any[]>([])
const registries = ref<any[]>([])
const pagination = ref({ current: 1, pageSize: 20, total: 0 })
const showResult = ref(false)
const currentResult = ref<any>(null)
const severityFilter = ref('')

// 根据严重程度筛选漏洞
const filteredVulnerabilities = computed(() => {
  if (!currentResult.value?.vulnerabilities) return []
  if (!severityFilter.value) return currentResult.value.vulnerabilities
  return currentResult.value.vulnerabilities.filter((v: any) => v.severity === severityFilter.value)
})

const filterBySeverity = (severity: string) => {
  severityFilter.value = severityFilter.value === severity ? '' : severity
}

const loadRegistries = async () => {
  try {
    const res = await getRegistries()
    registries.value = res?.data || []
  } catch (error) {
    console.error('加载仓库列表失败', error)
  }
}

const loadHistory = async () => {
  loading.value = true
  try {
    const res = await getScanHistory({
      image: searchImage.value,
      page: pagination.value.current,
      page_size: pagination.value.pageSize
    })
    scanHistory.value = res?.data?.items || []
    pagination.value.total = res?.data?.total || 0
  } catch (error) {
    console.error('加载扫描历史失败', error)
  } finally {
    loading.value = false
  }
}

const handleTableChange = (pag: any) => {
  pagination.value.current = pag.current
  pagination.value.pageSize = pag.pageSize
  loadHistory()
}

const handleScan = async () => {
  if (!scanForm.value.image) {
    message.warning('请输入镜像地址')
    return
  }
  scanning.value = true
  try {
    const res = await scanImage(scanForm.value)
    message.success('扫描完成')
    currentResult.value = res?.data
    severityFilter.value = ''
    showResult.value = true
    loadHistory()
  } catch (error: any) {
    message.error(error.message || '扫描失败')
  } finally {
    scanning.value = false
  }
}

const viewResult = async (row: any) => {
  try {
    const res = await getScanResult(row.id)
    currentResult.value = res?.data
    severityFilter.value = ''
    showResult.value = true
  } catch (error) {
    message.error('获取扫描结果失败')
  }
}

const getStatusColor = (status: string) => ({ completed: 'green', scanning: 'orange', failed: 'red' }[status] || 'default')
const getStatusLabel = (status: string) => ({ completed: '已完成', scanning: '扫描中', failed: '失败' }[status] || status)
const getRiskColor = (level: string) => ({ critical: 'red', high: 'orange', medium: 'blue', low: 'green', none: 'green' }[level] || 'default')
const getRiskLabel = (level: string) => ({ critical: '严重', high: '高危', medium: '中危', low: '低危', none: '安全' }[level] || level)
const formatTime = (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'

onMounted(() => { loadRegistries(); loadHistory() })
</script>

<style scoped>
.vuln-stats span { margin-right: 8px; padding: 2px 8px; border-radius: 4px; font-size: 12px; }
.vuln-stats .critical { background: #fff1f0; color: #cf1322; }
.vuln-stats .high { background: #fff7e6; color: #d46b08; }
.vuln-stats .medium { background: #e6f7ff; color: #096dd9; }
.vuln-stats .low { background: #f6ffed; color: #389e0d; }
.vuln-summary { margin-top: 16px; display: flex; gap: 8px; }
.vuln-summary :deep(.ant-tag) { cursor: pointer; transition: all 0.2s; }
.vuln-summary :deep(.ant-tag:hover) { opacity: 0.8; transform: scale(1.05); }
.vuln-summary :deep(.ant-tag.active) { border-width: 2px; font-weight: bold; box-shadow: 0 0 4px rgba(0,0,0,0.2); }
</style>
