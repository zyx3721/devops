<template>
  <div class="artifact-scan">
    <!-- 面包屑导航 -->
    <a-breadcrumb style="margin-bottom: 16px">
      <a-breadcrumb-item>
        <router-link to="/pipeline/artifacts">制品管理</router-link>
      </a-breadcrumb-item>
      <a-breadcrumb-item>{{ artifactName }}</a-breadcrumb-item>
      <a-breadcrumb-item>扫描结果</a-breadcrumb-item>
    </a-breadcrumb>

    <!-- 扫描概览卡片 -->
    <a-row :gutter="16" style="margin-bottom: 16px">
      <a-col :xs="24" :sm="8">
        <ScanResultCard
          title="漏洞扫描"
          scan-type="vulnerability"
          :result="scanResults.vulnerability"
        />
      </a-col>
      <a-col :xs="24" :sm="8">
        <ScanResultCard
          title="许可证扫描"
          scan-type="license"
          :result="scanResults.license"
        />
      </a-col>
      <a-col :xs="24" :sm="8">
        <ScanResultCard
          title="质量扫描"
          scan-type="quality"
          :result="scanResults.quality"
        />
      </a-col>
    </a-row>

    <!-- 操作栏 -->
    <a-card :bordered="false" style="margin-bottom: 16px">
      <a-space>
        <a-button
          type="primary"
          @click="triggerScan"
          :loading="scanning"
          :disabled="isScanning"
        >
          <template #icon><ScanOutlined /></template>
          {{ isScanning ? '扫描中...' : '触发扫描' }}
        </a-button>
        <a-button @click="loadData" :loading="loading">
          <template #icon><ReloadOutlined /></template>
          刷新
        </a-button>
        <a-select
          v-model:value="selectedScanTypes"
          mode="multiple"
          placeholder="选择扫描类型"
          style="width: 300px"
          :disabled="isScanning"
        >
          <a-select-option value="vulnerability">漏洞扫描</a-select-option>
          <a-select-option value="license">许可证扫描</a-select-option>
          <a-select-option value="quality">质量扫描</a-select-option>
        </a-select>
      </a-space>
    </a-card>

    <!-- Tab 切换 -->
    <a-card :bordered="false">
      <a-tabs v-model:activeKey="activeTab">
        <!-- 漏洞扫描 Tab -->
        <a-tab-pane key="vulnerability" tab="漏洞扫描">
          <VulnerabilityList
            :vulnerabilities="vulnerabilities"
            :loading="loading"
          />
        </a-tab-pane>

        <!-- 许可证扫描 Tab -->
        <a-tab-pane key="license" tab="许可证扫描">
          <a-table
            :columns="licenseColumns"
            :data-source="licenses"
            :loading="loading"
            :pagination="{ pageSize: 10 }"
            row-key="package"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'license'">
                <a-tag>{{ record.license }}</a-tag>
              </template>
              <template v-else-if="column.key === 'risk'">
                <a-tag :color="getRiskColor(record.risk)">
                  {{ getRiskText(record.risk) }}
                </a-tag>
              </template>
            </template>
          </a-table>
        </a-tab-pane>

        <!-- 质量扫描 Tab -->
        <a-tab-pane key="quality" tab="质量扫描">
          <a-row :gutter="16">
            <a-col :xs="24" :sm="12" :md="6">
              <a-card :bordered="false">
                <a-statistic
                  title="代码异味"
                  :value="qualityMetrics.code_smells"
                  :value-style="{ color: '#faad14' }"
                >
                  <template #prefix>
                    <WarningOutlined />
                  </template>
                </a-statistic>
              </a-card>
            </a-col>
            <a-col :xs="24" :sm="12" :md="6">
              <a-card :bordered="false">
                <a-statistic
                  title="Bug"
                  :value="qualityMetrics.bugs"
                  :value-style="{ color: '#ff4d4f' }"
                >
                  <template #prefix>
                    <BugOutlined />
                  </template>
                </a-statistic>
              </a-card>
            </a-col>
            <a-col :xs="24" :sm="12" :md="6">
              <a-card :bordered="false">
                <a-statistic
                  title="覆盖率"
                  :value="qualityMetrics.coverage"
                  suffix="%"
                  :precision="1"
                  :value-style="{ color: '#52c41a' }"
                >
                  <template #prefix>
                    <CheckCircleOutlined />
                  </template>
                </a-statistic>
              </a-card>
            </a-col>
            <a-col :xs="24" :sm="12" :md="6">
              <a-card :bordered="false">
                <a-statistic
                  title="技术债"
                  :value="qualityMetrics.tech_debt"
                  suffix="天"
                  :value-style="{ color: '#1890ff' }"
                >
                  <template #prefix>
                    <ClockCircleOutlined />
                  </template>
                </a-statistic>
              </a-card>
            </a-col>
          </a-row>

          <a-divider />

          <a-table
            :columns="qualityColumns"
            :data-source="qualityIssues"
            :loading="loading"
            :pagination="{ pageSize: 10 }"
            row-key="id"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'type'">
                <a-tag :color="getIssueTypeColor(record.type)">
                  {{ record.type }}
                </a-tag>
              </template>
              <template v-else-if="column.key === 'severity'">
                <a-tag :color="getSeverityColor(record.severity)">
                  {{ getSeverityText(record.severity) }}
                </a-tag>
              </template>
            </template>
          </a-table>
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  ScanOutlined,
  ReloadOutlined,
  WarningOutlined,
  BugOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
} from '@ant-design/icons-vue'
import request from '@/services/api'
import ScanResultCard from '@/components/pipeline/ScanResultCard.vue'
import VulnerabilityList from '@/components/pipeline/VulnerabilityList.vue'

const route = useRoute()
const versionId = computed(() => Number(route.params.versionId))

const loading = ref(false)
const scanning = ref(false)
const activeTab = ref('vulnerability')
const artifactName = ref('')
const selectedScanTypes = ref(['vulnerability', 'license', 'quality'])

const scanResults = reactive({
  vulnerability: {
    scan_type: 'vulnerability',
    scanner: '',
    status: 'pending' as const,
    critical_count: 0,
    high_count: 0,
    medium_count: 0,
    low_count: 0,
    details: '',
    scanned_at: '',
  },
  license: {
    scan_type: 'license',
    scanner: '',
    status: 'pending' as const,
    critical_count: 0,
    high_count: 0,
    medium_count: 0,
    low_count: 0,
    details: '',
    scanned_at: '',
  },
  quality: {
    scan_type: 'quality',
    scanner: '',
    status: 'pending' as const,
    critical_count: 0,
    high_count: 0,
    medium_count: 0,
    low_count: 0,
    details: '',
    scanned_at: '',
  },
})

const vulnerabilities = ref<any[]>([])
const licenses = ref<any[]>([])
const qualityIssues = ref<any[]>([])

const qualityMetrics = reactive({
  code_smells: 0,
  bugs: 0,
  coverage: 0,
  tech_debt: 0,
})

const isScanning = computed(() => {
  return (
    scanResults.vulnerability.status === 'scanning' ||
    scanResults.license.status === 'scanning' ||
    scanResults.quality.status === 'scanning'
  )
})

const licenseColumns = [
  { title: '包名', dataIndex: 'package', key: 'package' },
  { title: '版本', dataIndex: 'version', key: 'version' },
  { title: '许可证', key: 'license' },
  { title: '风险等级', key: 'risk' },
]

const qualityColumns = [
  { title: '类型', key: 'type', width: 100 },
  { title: '严重程度', key: 'severity', width: 100 },
  { title: '文件', dataIndex: 'file', key: 'file' },
  { title: '行号', dataIndex: 'line', key: 'line', width: 80 },
  { title: '描述', dataIndex: 'message', key: 'message', ellipsis: true },
]

const loadData = async () => {
  loading.value = true
  try {
    const res = await request.get(`/artifact-versions/${versionId.value}/scan-results`)
    if (res?.data) {
      const results = res.data
      
      // 更新扫描结果
      results.forEach((result: any) => {
        if (result.scan_type === 'vulnerability') {
          Object.assign(scanResults.vulnerability, result)
          // 解析漏洞详情
          if (result.details) {
            try {
              const details = JSON.parse(result.details)
              vulnerabilities.value = details.vulnerabilities || []
            } catch (e) {
              console.error('解析漏洞详情失败:', e)
            }
          }
        } else if (result.scan_type === 'license') {
          Object.assign(scanResults.license, result)
          // 解析许可证详情
          if (result.details) {
            try {
              const details = JSON.parse(result.details)
              licenses.value = details.licenses || []
            } catch (e) {
              console.error('解析许可证详情失败:', e)
            }
          }
        } else if (result.scan_type === 'quality') {
          Object.assign(scanResults.quality, result)
          // 解析质量详情
          if (result.details) {
            try {
              const details = JSON.parse(result.details)
              Object.assign(qualityMetrics, details.metrics || {})
              qualityIssues.value = details.issues || []
            } catch (e) {
              console.error('解析质量详情失败:', e)
            }
          }
        }
      })
    }
  } catch (error) {
    console.error('加载扫描结果失败:', error)
  } finally {
    loading.value = false
  }
}

const triggerScan = async () => {
  if (selectedScanTypes.value.length === 0) {
    message.warning('请选择至少一种扫描类型')
    return
  }

  scanning.value = true
  try {
    await request.post(`/artifact-versions/${versionId.value}/scan`, {
      scan_types: selectedScanTypes.value,
    })
    message.success('扫描任务已启动')
    
    // 更新状态为扫描中
    selectedScanTypes.value.forEach(type => {
      if (type === 'vulnerability') {
        scanResults.vulnerability.status = 'scanning'
      } else if (type === 'license') {
        scanResults.license.status = 'scanning'
      } else if (type === 'quality') {
        scanResults.quality.status = 'scanning'
      }
    })
    
    // 轮询检查扫描状态
    pollScanStatus()
  } catch (error: any) {
    message.error(error?.message || '启动扫描失败')
  } finally {
    scanning.value = false
  }
}

const pollScanStatus = () => {
  const timer = setInterval(async () => {
    await loadData()
    
    // 如果所有扫描都完成，停止轮询
    if (!isScanning.value) {
      clearInterval(timer)
      message.success('扫描完成')
    }
  }, 3000)
  
  // 最多轮询 5 分钟
  setTimeout(() => {
    clearInterval(timer)
  }, 5 * 60 * 1000)
}

const getRiskColor = (risk: string) => {
  switch (risk) {
    case 'high':
      return 'red'
    case 'medium':
      return 'orange'
    case 'low':
      return 'green'
    default:
      return 'default'
  }
}

const getRiskText = (risk: string) => {
  switch (risk) {
    case 'high':
      return '高风险'
    case 'medium':
      return '中风险'
    case 'low':
      return '低风险'
    default:
      return '未知'
  }
}

const getIssueTypeColor = (type: string) => {
  switch (type) {
    case 'BUG':
      return 'red'
    case 'CODE_SMELL':
      return 'orange'
    case 'VULNERABILITY':
      return 'purple'
    default:
      return 'default'
  }
}

const getSeverityColor = (severity: string) => {
  switch (severity) {
    case 'CRITICAL':
      return 'red'
    case 'HIGH':
      return 'orange'
    case 'MEDIUM':
      return 'gold'
    case 'LOW':
      return 'green'
    default:
      return 'default'
  }
}

const getSeverityText = (severity: string) => {
  switch (severity) {
    case 'CRITICAL':
      return '严重'
    case 'HIGH':
      return '高危'
    case 'MEDIUM':
      return '中危'
    case 'LOW':
      return '低危'
    default:
      return severity
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.artifact-scan {
  padding: 0;
}

:deep(.ant-statistic-title) {
  font-size: 14px;
  color: rgba(0, 0, 0, 0.65);
}

:deep(.ant-statistic-content) {
  font-size: 24px;
  font-weight: 600;
}

@media (max-width: 768px) {
  :deep(.ant-col) {
    margin-bottom: 16px;
  }
}
</style>
