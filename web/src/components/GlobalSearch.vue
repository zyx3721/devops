<template>
  <div>
    <!-- 搜索触发按钮 -->
    <a-tooltip :title="t('common.search') + ' (Ctrl+K)'">
      <a-button type="text" @click="showModal = true">
        <SearchOutlined />
      </a-button>
    </a-tooltip>

    <!-- 搜索弹窗 -->
    <a-modal
      v-model:open="showModal"
      :footer="null"
      :closable="false"
      :mask-closable="true"
      width="600px"
      class="global-search-modal"
      :body-style="{ padding: 0 }"
    >
      <div class="search-container">
        <!-- 搜索输入 -->
        <div class="search-input-wrapper">
          <SearchOutlined class="search-icon" />
          <input
            ref="searchInput"
            v-model="keyword"
            type="text"
            :placeholder="t('search.placeholder')"
            class="search-input"
            @input="handleSearch"
            @keydown="handleKeydown"
          />
          <span class="search-shortcut">ESC</span>
        </div>

        <!-- 搜索结果 -->
        <div class="search-results" v-if="keyword">
          <div v-if="loading" class="search-loading">
            <a-spin size="small" />
            <span>{{ t('search.searching') }}</span>
          </div>
          <template v-else>
            <!-- 分组结果 -->
            <div v-for="group in groupedResults" :key="group.type" class="result-group">
              <div class="group-title">
                <component :is="group.icon" />
                <span>{{ group.label }}</span>
                <span class="group-count">{{ group.items.length }}</span>
              </div>
              <div
                v-for="(item, index) in group.items"
                :key="item.id"
                class="result-item"
                :class="{ active: selectedIndex === getGlobalIndex(group.type, index) }"
                @click="goToResult(item)"
                @mouseenter="selectedIndex = getGlobalIndex(group.type, index)"
              >
                <div class="item-icon">
                  <component :is="group.icon" />
                </div>
                <div class="item-content">
                  <div class="item-title" v-html="highlightKeyword(item.name || item.title)"></div>
                  <div class="item-desc" v-if="item.description">{{ item.description }}</div>
                </div>
                <div class="item-meta">
                  <a-tag v-if="item.status" :color="getStatusColor(item.status)" size="small">
                    {{ item.status }}
                  </a-tag>
                </div>
              </div>
            </div>

            <!-- 无结果 -->
            <div v-if="groupedResults.length === 0 && !loading" class="no-results">
              <SearchOutlined style="font-size: 32px; color: #d9d9d9" />
              <p>{{ t('search.noResults') }}</p>
            </div>
          </template>
        </div>

        <!-- 快捷导航 -->
        <div class="quick-nav" v-else>
          <div class="nav-title">{{ t('search.quickNav') }}</div>
          <div class="nav-items">
            <div
              v-for="(item, index) in quickNavItems"
              :key="item.path"
              class="nav-item"
              :class="{ active: selectedIndex === index }"
              @click="goToPath(item.path)"
              @mouseenter="selectedIndex = index"
            >
              <component :is="item.icon" class="nav-icon" />
              <span>{{ t(item.labelKey) }}</span>
              <span class="nav-shortcut" v-if="item.shortcut">{{ item.shortcut }}</span>
            </div>
          </div>
        </div>

        <!-- 底部提示 -->
        <div class="search-footer">
          <span><kbd>↑</kbd><kbd>↓</kbd> {{ t('common.select') }}</span>
          <span><kbd>Enter</kbd> {{ t('common.open') }}</span>
          <span><kbd>ESC</kbd> {{ t('common.close') }}</span>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import {
  SearchOutlined, RocketOutlined, CloudOutlined, AppstoreOutlined,
  UserOutlined, SettingOutlined, AlertOutlined, SafetyCertificateOutlined,
  DollarOutlined, FileSearchOutlined, AuditOutlined, ThunderboltOutlined,
  LayoutOutlined, ShopOutlined, DatabaseOutlined, BarChartOutlined,
  FundOutlined, KeyOutlined, ControlOutlined,
  HeartOutlined
} from '@ant-design/icons-vue'
import request from '@/services/api'

const router = useRouter()
const { t } = useI18n()
const showModal = ref(false)
const keyword = ref('')
const loading = ref(false)
const selectedIndex = ref(0)
const searchInput = ref<HTMLInputElement | null>(null)

interface SearchResult {
  id: string | number
  type: string
  name?: string
  title?: string
  description?: string
  status?: string
  path: string
}

const results = ref<SearchResult[]>([])

// 菜单搜索项 - 用于快速访问功能页面
interface MenuSearchItem {
  key: string
  titleKey: string  // i18n key
  path: string
  categoryKey: string  // i18n key
  keywords: string[]
  icon?: any
}

const getMenuSearchItems = (): MenuSearchItem[] => [
  // 弹性工程（流量治理子菜单）
  {
    key: 'resilience',
    titleKey: 'search.resilience',
    path: '/resilience',
    categoryKey: 'search.categoryTraffic',
    keywords: ['弹性', '容错', '降级', 'resilience', 'fault', 'tolerance', '流量'],
    icon: ThunderboltOutlined
  },
  
  // 流水线功能
  {
    key: 'pipeline-designer',
    titleKey: 'search.pipelineDesigner',
    path: '/pipeline/designer',
    categoryKey: 'search.categoryCICD',
    keywords: ['流水线', '设计', '设计器', 'pipeline', 'designer', '可视化'],
    icon: LayoutOutlined
  },
  {
    key: 'template-market',
    titleKey: 'search.templates',
    path: '/pipeline/templates',
    categoryKey: 'search.categoryCICD',
    keywords: ['模板', '市场', 'template', 'market', '流水线模板'],
    icon: ShopOutlined
  },
  {
    key: 'build-cache',
    titleKey: 'search.buildCache',
    path: '/pipeline/cache',
    categoryKey: 'search.categoryCICD',
    keywords: ['构建', '缓存', 'cache', 'build', '加速'],
    icon: DatabaseOutlined
  },
  {
    key: 'build-stats',
    titleKey: 'search.buildStats',
    path: '/pipeline/stats/usage',
    categoryKey: 'search.categoryCICD',
    keywords: ['构建', '统计', 'stats', 'usage', '分析', '报表'],
    icon: BarChartOutlined
  },
  {
    key: 'resource-quota',
    titleKey: 'search.quota',
    path: '/pipeline/quota',
    categoryKey: 'search.categoryCICD',
    keywords: ['资源', '配额', 'quota', 'resource', '限制'],
    icon: FundOutlined
  },
  {
    key: 'credentials',
    titleKey: 'search.credentials',
    path: '/pipeline/credentials',
    categoryKey: 'search.categoryCICD',
    keywords: ['凭证', '密钥', 'credentials', 'secret', '认证'],
    icon: KeyOutlined
  },
  {
    key: 'variables',
    titleKey: 'search.variables',
    path: '/pipeline/variables',
    categoryKey: 'search.categoryCICD',
    keywords: ['变量', 'variables', 'env', '环境变量', '配置'],
    icon: ControlOutlined
  },
  
  // 健康检查
  {
    key: 'service-health',
    titleKey: 'search.serviceHealth',
    path: '/healthcheck',
    categoryKey: 'search.categoryMonitor',
    keywords: ['健康', '检查', 'health', 'check', '服务', '监控'],
    icon: HeartOutlined
  },
  {
    key: 'ssl-cert-check',
    titleKey: 'search.sslCert',
    path: '/healthcheck/ssl-cert',
    categoryKey: 'search.categoryMonitor',
    keywords: ['ssl', '证书', 'cert', 'certificate', '健康检查', 'https'],
    icon: SafetyCertificateOutlined
  }
]

// 快捷导航项
const getQuickNavItems = () => [
  { labelKey: 'search.quickPipeline', path: '/pipeline/list', icon: RocketOutlined },
  { labelKey: 'search.quickK8s', path: '/k8s/clusters', icon: CloudOutlined },
  { labelKey: 'search.quickTraffic', path: '/traffic/ratelimit', icon: ThunderboltOutlined },
  { labelKey: 'search.quickLogs', path: '/logs/center', icon: FileSearchOutlined },
  { labelKey: 'search.quickCost', path: '/cost/overview', icon: DollarOutlined },
  { labelKey: 'search.quickSecurity', path: '/security/overview', icon: SafetyCertificateOutlined },
  { labelKey: 'search.quickAlert', path: '/alert/overview', icon: AlertOutlined },
  { labelKey: 'search.quickUsers', path: '/users', icon: UserOutlined },
  { labelKey: 'search.quickApproval', path: '/approval/pending', icon: AuditOutlined },
]

const menuSearchItems = computed(() => getMenuSearchItems())
const quickNavItems = computed(() => getQuickNavItems())

// 分组结果
const groupedResults = computed(() => {
  const groups: { type: string; label: string; icon: any; items: SearchResult[] }[] = []
  
  const typeConfig: Record<string, { labelKey: string; icon: any }> = {
    menu: { labelKey: 'search.menu', icon: SettingOutlined },
    pipeline: { labelKey: 'search.pipeline', icon: RocketOutlined },
    cluster: { labelKey: 'search.cluster', icon: CloudOutlined },
    application: { labelKey: 'search.application', icon: AppstoreOutlined },
    user: { labelKey: 'search.user', icon: UserOutlined },
  }

  const grouped = results.value.reduce((acc, item) => {
    if (!acc[item.type]) acc[item.type] = []
    acc[item.type].push(item)
    return acc
  }, {} as Record<string, SearchResult[]>)

  for (const [type, items] of Object.entries(grouped)) {
    const config = typeConfig[type] || { labelKey: 'search.menu', icon: SettingOutlined }
    groups.push({ type, label: t(config.labelKey), icon: config.icon, items: items.slice(0, 5) })
  }

  return groups
})

// 获取全局索引
const getGlobalIndex = (type: string, index: number) => {
  let offset = 0
  for (const group of groupedResults.value) {
    if (group.type === type) return offset + index
    offset += group.items.length
  }
  return offset + index
}

// 总结果数
const totalResults = computed(() => {
  return groupedResults.value.reduce((sum, g) => sum + g.items.length, 0)
})

// 搜索
let searchTimer: number | null = null
const handleSearch = () => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = window.setTimeout(async () => {
    if (!keyword.value.trim()) {
      results.value = []
      return
    }
    
    loading.value = true
    selectedIndex.value = 0
    
    try {
      const searchResults: SearchResult[] = []
      
      // 首先搜索菜单项（本地搜索，速度快）
      const lowerKeyword = keyword.value.toLowerCase()
      menuSearchItems.value.forEach(item => {
        const title = t(item.titleKey)
        const matchTitle = title.toLowerCase().includes(lowerKeyword)
        const matchKeywords = item.keywords.some(k => k.toLowerCase().includes(lowerKeyword))
        
        if (matchTitle || matchKeywords) {
          searchResults.push({
            id: item.key,
            type: 'menu',
            title: title,
            description: t(item.categoryKey),
            path: item.path
          })
        }
      })
      
      // 并行搜索多个资源
      const [pipelinesRes, clustersRes, usersRes] = await Promise.allSettled([
        request.get('/pipelines', { params: { name: keyword.value, page_size: 5 } }),
        request.get('/k8s-clusters', { params: { name: keyword.value, page_size: 5 } }),
        request.get('/users', { params: { keyword: keyword.value, page_size: 5 } }),
      ])

      // 处理流水线结果
      if (pipelinesRes.status === 'fulfilled') {
        const items = pipelinesRes.value?.data?.items || []
        items.forEach((item: any) => {
          searchResults.push({
            id: item.id,
            type: 'pipeline',
            name: item.name,
            description: item.description,
            status: item.status,
            path: `/pipeline/${item.id}`
          })
        })
      }

      // 处理集群结果
      if (clustersRes.status === 'fulfilled') {
        const items = clustersRes.value?.data?.items || clustersRes.value?.data || []
        items.forEach((item: any) => {
          searchResults.push({
            id: item.id,
            type: 'cluster',
            name: item.name,
            description: item.description,
            status: item.status,
            path: `/k8s/clusters/${item.id}/resources`
          })
        })
      }

      // 处理用户结果
      if (usersRes.status === 'fulfilled') {
        const items = usersRes.value?.data?.items || usersRes.value?.data || []
        items.forEach((item: any) => {
          searchResults.push({
            id: item.id,
            type: 'user',
            name: item.username,
            description: item.email,
            status: item.status === 1 ? 'active' : 'disabled',
            path: '/users'
          })
        })
      }

      results.value = searchResults
    } catch (error) {
      console.error('搜索失败', error)
    } finally {
      loading.value = false
    }
  }, 300)
}

// 键盘导航
const handleKeydown = (e: KeyboardEvent) => {
  const maxIndex = keyword.value ? totalResults.value - 1 : quickNavItems.length - 1
  
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    selectedIndex.value = Math.min(selectedIndex.value + 1, maxIndex)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    if (keyword.value) {
      // 找到选中的结果
      let currentIndex = 0
      for (const group of groupedResults.value) {
        for (const item of group.items) {
          if (currentIndex === selectedIndex.value) {
            goToResult(item)
            return
          }
          currentIndex++
        }
      }
    } else {
      goToPath(quickNavItems[selectedIndex.value].path)
    }
  } else if (e.key === 'Escape') {
    showModal.value = false
  }
}

// 跳转到结果
const goToResult = (item: SearchResult) => {
  showModal.value = false
  keyword.value = ''
  router.push(item.path)
}

// 跳转到路径
const goToPath = (path: string) => {
  showModal.value = false
  keyword.value = ''
  router.push(path)
}

// 高亮关键词
const highlightKeyword = (text: string) => {
  if (!keyword.value || !text) return text
  const regex = new RegExp(`(${keyword.value})`, 'gi')
  return text.replace(regex, '<mark>$1</mark>')
}

// 状态颜色
const getStatusColor = (status: string) => {
  const map: Record<string, string> = {
    active: 'green',
    running: 'blue',
    success: 'green',
    failed: 'red',
    disabled: 'default'
  }
  return map[status] || 'default'
}

// 全局快捷键
const handleGlobalKeydown = (e: KeyboardEvent) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
    e.preventDefault()
    showModal.value = true
  }
}

// 打开时聚焦输入框
watch(showModal, (val) => {
  if (val) {
    nextTick(() => {
      searchInput.value?.focus()
    })
    selectedIndex.value = 0
  } else {
    keyword.value = ''
    results.value = []
  }
})

onMounted(() => {
  document.addEventListener('keydown', handleGlobalKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleGlobalKeydown)
})
</script>

<style scoped>
.search-container {
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
}

.search-input-wrapper {
  display: flex;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #f0f0f0;
}

.search-icon {
  font-size: 18px;
  color: #999;
  margin-right: 12px;
}

.search-input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 16px;
  background: transparent;
}

.search-input::placeholder {
  color: #bbb;
}

.search-shortcut {
  padding: 2px 6px;
  background: #f5f5f5;
  border-radius: 4px;
  font-size: 12px;
  color: #999;
}

.search-results {
  max-height: 400px;
  overflow-y: auto;
}

.search-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px;
  color: #999;
}

.result-group {
  padding: 8px 0;
}

.group-title {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  font-size: 12px;
  color: #999;
  text-transform: uppercase;
}

.group-count {
  background: #f0f0f0;
  padding: 0 6px;
  border-radius: 10px;
  font-size: 11px;
}

.result-item {
  display: flex;
  align-items: center;
  padding: 10px 16px;
  cursor: pointer;
  transition: background 0.2s;
}

.result-item:hover,
.result-item.active {
  background: #f5f5f5;
}

.item-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f0f0f0;
  border-radius: 6px;
  margin-right: 12px;
  color: #666;
}

.item-content {
  flex: 1;
  min-width: 0;
}

.item-title {
  font-size: 14px;
  color: #333;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-desc {
  font-size: 12px;
  color: #999;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-meta {
  margin-left: 12px;
}

.no-results {
  text-align: center;
  padding: 40px;
  color: #999;
}

.no-results p {
  margin-top: 12px;
}

/* 快捷导航 */
.quick-nav {
  padding: 16px;
}

.nav-title {
  font-size: 12px;
  color: #999;
  margin-bottom: 12px;
  text-transform: uppercase;
}

.nav-items {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 8px;
}

.nav-item {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s;
}

.nav-item:hover,
.nav-item.active {
  background: #f5f5f5;
}

.nav-icon {
  margin-right: 10px;
  color: #666;
}

.nav-shortcut {
  margin-left: auto;
  font-size: 11px;
  color: #bbb;
}

/* 底部提示 */
.search-footer {
  display: flex;
  gap: 16px;
  padding: 12px 16px;
  border-top: 1px solid #f0f0f0;
  font-size: 12px;
  color: #999;
}

.search-footer kbd {
  display: inline-block;
  padding: 2px 6px;
  background: #f5f5f5;
  border: 1px solid #e8e8e8;
  border-radius: 4px;
  font-family: inherit;
  font-size: 11px;
  margin-right: 4px;
}

:deep(mark) {
  background: #fff3cd;
  color: #856404;
  padding: 0 2px;
  border-radius: 2px;
}
</style>

<style>
.global-search-modal .ant-modal-content {
  border-radius: 12px;
  overflow: hidden;
}

.global-search-modal .ant-modal-body {
  padding: 0 !important;
}
</style>
