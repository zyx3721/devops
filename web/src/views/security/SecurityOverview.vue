<template>
  <div class="security-overview">
    <!-- 安全评分 -->
    <a-row :gutter="16">
      <a-col :span="6">
        <a-card class="score-card">
          <div style="text-align: center">
            <a-progress type="circle" :percent="overview.security_score" :width="120" :strokeColor="getScoreColor(overview.security_score)">
              <template #format><span style="font-size: 28px; font-weight: bold">{{ overview.security_score || 0 }}</span></template>
            </a-progress>
            <div style="margin-top: 12px; font-size: 16px">安全评分</div>
            <a-tag :color="getRiskColor(overview.risk_level)" style="margin-top: 8px">{{ getRiskLabel(overview.risk_level) }}</a-tag>
          </div>
        </a-card>
      </a-col>
      <a-col :span="18">
        <a-card title="安全概况">
          <a-row :gutter="16">
            <a-col :span="6">
              <div class="stat-item clickable" @click="goToImageScan()">
                <a-statistic title="漏洞总数" :value="overview.vuln_summary?.total || 0" :valueStyle="{ color: '#ff4d4f' }" />
              </div>
            </a-col>
            <a-col :span="6">
              <div class="stat-item clickable" @click="goToConfigCheck()">
                <a-statistic title="配置问题" :value="overview.config_summary?.total || 0" :valueStyle="{ color: '#faad14' }" />
              </div>
            </a-col>
            <a-col :span="6">
              <div class="stat-item clickable" @click="goToImageScan()">
                <a-statistic title="严重漏洞" :value="overview.vuln_summary?.critical || 0" :valueStyle="{ color: '#cf1322' }" />
              </div>
            </a-col>
            <a-col :span="6">
              <div class="stat-item clickable" @click="goToConfigCheck()">
                <a-statistic title="高危问题" :value="(overview.vuln_summary?.high || 0) + (overview.config_summary?.high || 0)" :valueStyle="{ color: '#d46b08' }" />
              </div>
            </a-col>
          </a-row>
        </a-card>
      </a-col>
    </a-row>

    <!-- 漏洞和配置问题分布 -->
    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="12">
        <a-card title="漏洞严重程度分布" class="clickable-card" @click="goToImageScan()">
          <a-row :gutter="16">
            <a-col :span="6">
              <div class="stat-item clickable">
                <a-statistic title="严重" :value="overview.vuln_summary?.critical || 0" :valueStyle="{ color: '#cf1322' }" />
              </div>
            </a-col>
            <a-col :span="6">
              <div class="stat-item clickable">
                <a-statistic title="高危" :value="overview.vuln_summary?.high || 0" :valueStyle="{ color: '#d46b08' }" />
              </div>
            </a-col>
            <a-col :span="6">
              <div class="stat-item clickable">
                <a-statistic title="中危" :value="overview.vuln_summary?.medium || 0" :valueStyle="{ color: '#096dd9' }" />
              </div>
            </a-col>
            <a-col :span="6">
              <div class="stat-item clickable">
                <a-statistic title="低危" :value="overview.vuln_summary?.low || 0" :valueStyle="{ color: '#389e0d' }" />
              </div>
            </a-col>
          </a-row>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card title="配置问题分布" class="clickable-card" @click="goToConfigCheck()">
          <a-row :gutter="16">
            <a-col :span="6">
              <div class="stat-item clickable">
                <a-statistic title="严重" :value="overview.config_summary?.critical || 0" :valueStyle="{ color: '#cf1322' }" />
              </div>
            </a-col>
            <a-col :span="6">
              <div class="stat-item clickable">
                <a-statistic title="高危" :value="overview.config_summary?.high || 0" :valueStyle="{ color: '#d46b08' }" />
              </div>
            </a-col>
            <a-col :span="6">
              <div class="stat-item clickable">
                <a-statistic title="中危" :value="overview.config_summary?.medium || 0" :valueStyle="{ color: '#096dd9' }" />
              </div>
            </a-col>
            <a-col :span="6">
              <div class="stat-item clickable">
                <a-statistic title="通过" :value="overview.config_summary?.passed || 0" :valueStyle="{ color: '#389e0d' }" />
              </div>
            </a-col>
          </a-row>
        </a-card>
      </a-col>
    </a-row>

    <!-- 快捷操作 -->
    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="24">
        <a-card title="快捷操作">
          <a-space size="middle">
            <a-button type="primary" @click="goToImageScan()">
              <template #icon><ScanOutlined /></template>
              镜像扫描
            </a-button>
            <a-button @click="goToConfigCheck()">
              <template #icon><SafetyCertificateOutlined /></template>
              配置检查
            </a-button>
            <a-button @click="goToAuditLog()">
              <template #icon><FileSearchOutlined /></template>
              审计日志
            </a-button>
          </a-space>
        </a-card>
      </a-col>
    </a-row>

    <!-- 最近扫描和检查 -->
    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="12">
        <a-card title="最近镜像扫描">
          <template #extra><a-button type="link" @click="goToImageScan()">查看全部</a-button></template>
          <a-table :dataSource="overview.recent_scans || []" :pagination="false" size="small" rowKey="id">
            <a-table-column title="镜像" dataIndex="image" :ellipsis="true">
              <template #default="{ record }">
                <a @click="viewScanDetail(record)">{{ record.image }}</a>
              </template>
            </a-table-column>
            <a-table-column title="风险" dataIndex="risk_level" :width="80">
              <template #default="{ record }">
                <a-tag :color="getRiskColor(record.risk_level)" class="clickable-tag" @click="viewScanDetail(record)">
                  {{ getRiskLabel(record.risk_level) }}
                </a-tag>
              </template>
            </a-table-column>
            <a-table-column title="漏洞" :width="100">
              <template #default="{ record }">
                <span class="vuln-count" @click="viewScanDetail(record)">
                  <span class="critical">{{ record.critical_count }}</span>/
                  <span class="high">{{ record.high_count }}</span>/
                  <span class="medium">{{ record.medium_count }}</span>/
                  <span class="low">{{ record.low_count }}</span>
                </span>
              </template>
            </a-table-column>
            <a-table-column title="时间" dataIndex="scanned_at" :width="100">
              <template #default="{ record }">{{ formatTime(record.scanned_at) }}</template>
            </a-table-column>
          </a-table>
        </a-card>
      </a-col>
      <a-col :span="12">
        <a-card title="最近配置检查">
          <template #extra><a-button type="link" @click="goToConfigCheck()">查看全部</a-button></template>
          <a-table :dataSource="overview.recent_checks || []" :pagination="false" size="small" rowKey="id">
            <a-table-column title="命名空间" dataIndex="namespace" :width="100">
              <template #default="{ record }">
                <a @click="viewCheckDetail(record)">{{ record.namespace || '全部' }}</a>
              </template>
            </a-table-column>
            <a-table-column title="问题分布" :width="120">
              <template #default="{ record }">
                <span class="vuln-count" @click="viewCheckDetail(record)">
                  <span class="critical">{{ record.critical_count || 0 }}</span>/
                  <span class="high">{{ record.high_count || 0 }}</span>/
                  <span class="medium">{{ record.medium_count || 0 }}</span>/
                  <span class="low">{{ record.passed_count || 0 }}</span>
                </span>
              </template>
            </a-table-column>
            <a-table-column title="时间" dataIndex="checked_at" :width="100">
              <template #default="{ record }">{{ formatTime(record.checked_at) }}</template>
            </a-table-column>
          </a-table>
        </a-card>
      </a-col>
    </a-row>

    <!-- 扫描详情弹窗 -->
    <a-modal v-model:open="showScanDetail" title="扫描详情" width="800px" :footer="null">
      <template v-if="currentScan">
        <a-descriptions :column="2" bordered size="small">
          <a-descriptions-item label="镜像">{{ currentScan.image }}</a-descriptions-item>
          <a-descriptions-item label="风险等级">
            <a-tag :color="getRiskColor(currentScan.risk_level)">{{ getRiskLabel(currentScan.risk_level) }}</a-tag>
          </a-descriptions-item>
          <a-descriptions-item label="扫描时间">{{ formatTimeFull(currentScan.scanned_at) }}</a-descriptions-item>
          <a-descriptions-item label="漏洞统计">
            <span class="critical">严重 {{ currentScan.critical_count }}</span> /
            <span class="high">高危 {{ currentScan.high_count }}</span> /
            <span class="medium">中危 {{ currentScan.medium_count }}</span> /
            <span class="low">低危 {{ currentScan.low_count }}</span>
          </a-descriptions-item>
        </a-descriptions>
        <div style="margin-top: 16px; text-align: right">
          <a-button type="primary" @click="goToImageScanWithId(currentScan.id)">查看完整报告</a-button>
        </div>
      </template>
    </a-modal>

    <!-- 检查详情弹窗 -->
    <a-modal v-model:open="showCheckDetail" title="检查详情" width="800px" :footer="null">
      <template v-if="currentCheck">
        <a-descriptions :column="2" bordered size="small">
          <a-descriptions-item label="集群">{{ currentCheck.cluster_name || '-' }}</a-descriptions-item>
          <a-descriptions-item label="命名空间">{{ currentCheck.namespace || '全部' }}</a-descriptions-item>
          <a-descriptions-item label="检查时间">{{ formatTimeFull(currentCheck.checked_at) }}</a-descriptions-item>
          <a-descriptions-item label="问题统计">
            <span class="critical">严重 {{ currentCheck.critical_count || 0 }}</span> /
            <span class="high">高危 {{ currentCheck.high_count || 0 }}</span> /
            <span class="medium">中危 {{ currentCheck.medium_count || 0 }}</span> /
            <span class="low">通过 {{ currentCheck.passed_count || 0 }}</span>
          </a-descriptions-item>
        </a-descriptions>
        <div style="margin-top: 16px; text-align: right">
          <a-button type="primary" @click="goToConfigCheckWithId(currentCheck.id)">查看完整报告</a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { ScanOutlined, SafetyCertificateOutlined, FileSearchOutlined } from '@ant-design/icons-vue'
import { getSecurityOverview } from '@/services/security'
import dayjs from 'dayjs'

const router = useRouter()
const overview = ref<any>({})
const showScanDetail = ref(false)
const showCheckDetail = ref(false)
const currentScan = ref<any>(null)
const currentCheck = ref<any>(null)

const loadData = async () => {
  try {
    const res = await getSecurityOverview()
    overview.value = res?.data || {}
  } catch (error) {
    console.error('加载数据失败', error)
  }
}

// 导航函数
const goToImageScan = () => router.push('/security/image-scan')
const goToConfigCheck = () => router.push('/security/config-check')
const goToAuditLog = () => router.push('/security/audit-log')
const goToImageScanWithId = (id: number) => {
  showScanDetail.value = false
  router.push({ path: '/security/image-scan', query: { id } })
}
const goToConfigCheckWithId = (id: number) => {
  showCheckDetail.value = false
  router.push({ path: '/security/config-check', query: { id } })
}

// 查看详情
const viewScanDetail = (record: any) => {
  currentScan.value = record
  showScanDetail.value = true
}
const viewCheckDetail = (record: any) => {
  currentCheck.value = record
  showCheckDetail.value = true
}

const getScoreColor = (score: number) => score >= 80 ? '#52c41a' : score >= 60 ? '#faad14' : '#ff4d4f'
const getRiskColor = (level: string) => ({ critical: 'red', high: 'orange', medium: 'blue', low: 'green', none: 'green' }[level] || 'default')
const getRiskLabel = (level: string) => ({ critical: '严重', high: '高危', medium: '中危', low: '低危', none: '安全' }[level] || level || '未知')
const formatTime = (time: string) => time ? dayjs(time).format('MM-DD HH:mm') : '-'
const formatTimeFull = (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '-'

onMounted(loadData)
</script>

<style scoped>
.stat-item.clickable { cursor: pointer; padding: 8px; border-radius: 4px; transition: all 0.2s; }
.stat-item.clickable:hover { background: #f5f5f5; }
.clickable-card { cursor: pointer; transition: all 0.2s; }
.clickable-card:hover { box-shadow: 0 2px 8px rgba(0,0,0,0.15); }
.clickable-tag { cursor: pointer; }
.vuln-count { cursor: pointer; font-size: 12px; }
.vuln-count:hover { text-decoration: underline; }
.critical { color: #cf1322; }
.high { color: #d46b08; }
.medium { color: #096dd9; }
.low { color: #389e0d; }
a { color: #1890ff; cursor: pointer; }
a:hover { text-decoration: underline; }
</style>
