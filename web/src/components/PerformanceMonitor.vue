<template>
  <div v-if="showMonitor" class="performance-monitor">
    <div class="monitor-header">
      <span>性能监控</span>
      <a-button size="small" type="text" @click="toggleMonitor">
        {{ collapsed ? '展开' : '收起' }}
      </a-button>
    </div>
    <div v-show="!collapsed" class="monitor-content">
      <div class="metric-item">
        <span class="metric-label">FPS:</span>
        <span class="metric-value" :class="{ warning: fps < 30, danger: fps < 20 }">
          {{ fps }}
        </span>
      </div>
      <div class="metric-item">
        <span class="metric-label">首屏加载:</span>
        <span class="metric-value" :class="{ warning: loadTime > 2000 }">
          {{ loadTime }}ms
        </span>
      </div>
      <div class="metric-item">
        <span class="metric-label">路由切换:</span>
        <span class="metric-value" :class="{ warning: routeTime > 100 }">
          {{ routeTime }}ms
        </span>
      </div>
      <div class="metric-item">
        <span class="metric-label">内存使用:</span>
        <span class="metric-value">
          {{ memoryUsage }}MB
        </span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { monitorFPS, measureFirstScreenLoad, measureRouteChange } from '@/utils/performance'

const route = useRoute()

const showMonitor = ref(import.meta.env.DEV) // 仅在开发环境显示
const collapsed = ref(false)
const fps = ref(60)
const loadTime = ref(0)
const routeTime = ref(0)
const memoryUsage = ref(0)

let stopFPSMonitor: (() => void) | null = null
let routeChangeEnd: (() => void) | null = null

const toggleMonitor = () => {
  collapsed.value = !collapsed.value
}

const updateMemoryUsage = () => {
  if ('memory' in performance) {
    const memory = (performance as any).memory
    memoryUsage.value = Math.round(memory.usedJSHeapSize / 1024 / 1024)
  }
}

onMounted(() => {
  // 测量首屏加载时间
  setTimeout(() => {
    loadTime.value = measureFirstScreenLoad()
  }, 0)

  // 监控 FPS
  stopFPSMonitor = monitorFPS((currentFps) => {
    fps.value = currentFps
  })

  // 更新内存使用
  const memoryInterval = setInterval(updateMemoryUsage, 1000)
  
  onUnmounted(() => {
    clearInterval(memoryInterval)
  })
})

onUnmounted(() => {
  if (stopFPSMonitor) {
    stopFPSMonitor()
  }
})

// 监听路由变化，测量路由切换性能
watch(() => route.path, (newPath) => {
  routeChangeEnd = measureRouteChange(newPath)
  
  // 在下一帧结束测量
  requestAnimationFrame(() => {
    if (routeChangeEnd) {
      routeTime.value = Math.round(routeChangeEnd())
    }
  })
})
</script>

<style scoped>
.performance-monitor {
  position: fixed;
  bottom: 20px;
  right: 20px;
  background: rgba(0, 0, 0, 0.8);
  color: #fff;
  padding: 12px;
  border-radius: 8px;
  font-size: 12px;
  z-index: 9999;
  min-width: 200px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.monitor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
  font-weight: bold;
}

.monitor-content {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.metric-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.metric-label {
  color: #999;
}

.metric-value {
  color: #52c41a;
  font-weight: bold;
}

.metric-value.warning {
  color: #faad14;
}

.metric-value.danger {
  color: #ff4d4f;
}
</style>
