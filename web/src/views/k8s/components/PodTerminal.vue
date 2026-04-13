<template>
  <div class="pod-terminal">
    <div class="terminal-toolbar">
      <a-space>
        <a-select v-model:value="selectedContainer" placeholder="选择容器" style="width: 200px">
          <a-select-option v-for="c in containers" :key="c.name" :value="c.name">
            {{ c.name }}
          </a-select-option>
        </a-select>
        <a-select v-model:value="selectedShell" style="width: 120px">
          <a-select-option value="/bin/sh">sh</a-select-option>
          <a-select-option value="/bin/bash">bash</a-select-option>
        </a-select>
        <a-button type="primary" @click="connect" :loading="connecting" :disabled="connected">
          {{ connected ? '已连接' : '连接' }}
        </a-button>
        <a-button @click="disconnect" :disabled="!connected" danger>断开</a-button>
      </a-space>
      <span v-if="connected" style="margin-left: 16px; color: #52c41a">
        <CheckCircleOutlined /> 已连接
      </span>
      <span v-if="connectionError" style="margin-left: 16px; color: #ff4d4f">
        {{ connectionError }}
      </span>
    </div>
    <div class="terminal-container" ref="terminalContainer"></div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import { CheckCircleOutlined } from '@ant-design/icons-vue'
import { k8sPodApi, type K8sContainer } from '@/services/k8s'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

const props = defineProps<{
  clusterId: number
  namespace: string
  podName: string
}>()

const containers = ref<K8sContainer[]>([])
const selectedContainer = ref('')
const selectedShell = ref('/bin/sh')
const connecting = ref(false)
const connected = ref(false)
const connectionError = ref('')
const terminalContainer = ref<HTMLElement | null>(null)

let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null
let isDisposed = false

const loadContainers = async () => {
  if (!props.namespace || !props.podName) {
    console.warn('namespace 或 podName 为空', props)
    connectionError.value = 'namespace 或 podName 为空'
    return
  }
  try {
    const res = await k8sPodApi.getContainers(props.clusterId, props.namespace, props.podName)
    containers.value = res.data || []
    if (containers.value.length > 0 && !selectedContainer.value) {
      selectedContainer.value = containers.value[0].name
    }
    console.log('加载容器列表成功:', containers.value)
  } catch (e: any) {
    console.error('加载容器列表失败:', e)
    connectionError.value = '加载容器列表失败: ' + (e.message || '未知错误')
    // 不再使用默认容器名，让用户知道出错了
    containers.value = []
  }
}

const initTerminal = async () => {
  // 等待 DOM 更新
  await nextTick()
  
  if (!terminalContainer.value || terminal || isDisposed) return

  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Consolas, Monaco, monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#d4d4d4'
    }
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon())

  terminal.open(terminalContainer.value)
  
  // 延迟 fit 以确保容器尺寸正确
  setTimeout(() => {
    if (fitAddon && !isDisposed) {
      fitAddon.fit()
    }
  }, 100)

  terminal.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'input', data }))
    }
  })

  terminal.onResize(({ cols, rows }) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: 'resize', cols, rows }))
    }
  })

  window.addEventListener('resize', handleResize)
}

const handleResize = () => {
  if (fitAddon && !isDisposed) {
    fitAddon.fit()
  }
}

const connect = () => {
  if (!selectedContainer.value) {
    message.warning('请先选择容器')
    return
  }

  if (!props.namespace) {
    message.warning('命名空间不能为空')
    return
  }

  if (!props.podName) {
    message.warning('Pod 名称不能为空')
    return
  }

  connectionError.value = ''
  connecting.value = true

  const url = k8sPodApi.getTerminalUrl(
    props.clusterId, props.namespace, props.podName,
    selectedContainer.value, selectedShell.value
  )
  
  console.log('终端连接参数:', {
    clusterId: props.clusterId,
    namespace: props.namespace,
    podName: props.podName,
    container: selectedContainer.value,
    shell: selectedShell.value
  })
  console.log('终端连接 URL:', url)

  try {
    ws = new WebSocket(url)
  } catch (e) {
    console.error('创建 WebSocket 失败:', e)
    connectionError.value = '创建连接失败'
    connecting.value = false
    return
  }

  ws.onopen = () => {
    console.log('WebSocket onopen 触发')
    if (isDisposed) {
      ws?.close()
      return
    }
    connecting.value = false
    connected.value = true
    connectionError.value = ''
    message.success('终端已连接')
    
    if (terminal && ws) {
      const resizeMsg = {
        type: 'resize',
        cols: terminal.cols,
        rows: terminal.rows
      }
      console.log('发送 resize 消息:', resizeMsg)
      ws.send(JSON.stringify(resizeMsg))
    }
  }

  ws.onmessage = (event) => {
    console.log('WebSocket onmessage:', event.data)
    if (isDisposed) return
    try {
      const data = JSON.parse(event.data)
      if (data.type === 'output' && data.data && terminal) {
        terminal.write(data.data)
      } else if (data.type === 'pong') {
        // 心跳响应
      }
    } catch {
      if (terminal && event.data) {
        terminal.write(event.data)
      }
    }
  }

  ws.onerror = (e) => {
    console.error('WebSocket 错误:', e)
    if (!isDisposed) {
      connectionError.value = '连接错误'
      message.error('终端连接错误')
      connecting.value = false
      connected.value = false
    }
  }

  ws.onclose = (e) => {
    console.log('WebSocket 关闭:', e.code, e.reason)
    if (!isDisposed) {
      connected.value = false
      connecting.value = false
      if (terminal) {
        terminal.write('\r\n\x1b[31m连接已断开\x1b[0m\r\n')
      }
    }
  }
}

const disconnect = () => {
  if (ws) {
    ws.close()
    ws = null
  }
  connected.value = false
}

const cleanup = () => {
  isDisposed = true
  disconnect()
  window.removeEventListener('resize', handleResize)
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  fitAddon = null
}

watch(() => props.podName, (newVal, oldVal) => {
  if (newVal !== oldVal) {
    disconnect()
    if (terminal) {
      terminal.clear()
    }
    selectedContainer.value = ''
    loadContainers()
  }
})

onMounted(() => {
  isDisposed = false
  loadContainers()
  initTerminal()
})

onUnmounted(() => {
  cleanup()
})
</script>

<style scoped>
.pod-terminal {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 400px;
}

.terminal-toolbar {
  padding: 12px;
  border-bottom: 1px solid #f0f0f0;
  background: #fafafa;
}

.terminal-container {
  flex: 1;
  background: #1e1e1e;
  padding: 8px;
  min-height: 350px;
}
</style>
