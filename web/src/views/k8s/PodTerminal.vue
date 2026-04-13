<template>
  <div class="pod-terminal">
    <a-card title="Pod 终端" :bordered="false">
      <template #extra>
        <a-space>
          <a-select v-model:value="shell" style="width: 120px" @change="reconnect">
            <a-select-option value="/bin/sh">sh</a-select-option>
            <a-select-option value="/bin/bash">bash</a-select-option>
            <a-select-option value="/bin/zsh">zsh</a-select-option>
          </a-select>
          <a-button @click="reconnect" :loading="connecting">
            <ReloadOutlined /> 重连
          </a-button>
          <a-button @click="goBack">
            <RollbackOutlined /> 返回
          </a-button>
        </a-space>
      </template>

      <!-- 连接信息 -->
      <a-descriptions :column="4" size="small" style="margin-bottom: 12px">
        <a-descriptions-item label="集群">{{ clusterName }}</a-descriptions-item>
        <a-descriptions-item label="命名空间">{{ namespace }}</a-descriptions-item>
        <a-descriptions-item label="Pod">{{ podName }}</a-descriptions-item>
        <a-descriptions-item label="容器">{{ container || '默认' }}</a-descriptions-item>
      </a-descriptions>

      <!-- 状态提示 -->
      <a-alert 
        v-if="error" 
        :message="error" 
        type="error" 
        show-icon 
        closable 
        style="margin-bottom: 12px"
        @close="error = ''"
      />

      <!-- 终端 -->
      <div class="terminal-container" ref="terminalContainer">
        <div v-if="connecting" class="terminal-loading">
          <a-spin tip="连接中..." />
        </div>
      </div>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { ReloadOutlined, RollbackOutlined } from '@ant-design/icons-vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'

const route = useRoute()
const router = useRouter()

const terminalContainer = ref<HTMLElement | null>(null)
const clusterID = ref(Number(route.query.cluster_id))
const namespace = ref(route.query.namespace as string)
const podName = ref(route.query.pod as string)
const container = ref(route.query.container as string || '')
const clusterName = ref(route.query.cluster_name as string || `集群 #${clusterID.value}`)
const shell = ref('/bin/sh')

const connecting = ref(false)
const error = ref('')

let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let ws: WebSocket | null = null

// 初始化终端
const initTerminal = () => {
  if (!terminalContainer.value) return

  terminal = new Terminal({
    cursorBlink: true,
    cursorStyle: 'block',
    fontSize: 14,
    fontFamily: 'Monaco, Menlo, Consolas, monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#d4d4d4',
      selectionBackground: '#264f78',
      black: '#000000',
      red: '#cd3131',
      green: '#0dbc79',
      yellow: '#e5e510',
      blue: '#2472c8',
      magenta: '#bc3fbc',
      cyan: '#11a8cd',
      white: '#e5e5e5',
      brightBlack: '#666666',
      brightRed: '#f14c4c',
      brightGreen: '#23d18b',
      brightYellow: '#f5f543',
      brightBlue: '#3b8eea',
      brightMagenta: '#d670d6',
      brightCyan: '#29b8db',
      brightWhite: '#e5e5e5'
    }
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(new WebLinksAddon())

  terminal.open(terminalContainer.value)
  fitAddon.fit()

  // 监听终端输入
  terminal.onData((data) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })

  // 监听终端大小变化
  terminal.onResize(({ cols, rows }) => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      // 发送终端大小调整消息
      const msg = new Uint8Array(5)
      msg[0] = 1 // 类型标识
      msg[1] = (rows >> 8) & 0xff
      msg[2] = rows & 0xff
      msg[3] = (cols >> 8) & 0xff
      msg[4] = cols & 0xff
      ws.send(msg)
    }
  })

  // 监听窗口大小变化
  window.addEventListener('resize', handleResize)
}

// 连接 WebSocket
const connect = () => {
  if (!clusterID.value || !namespace.value || !podName.value) {
    error.value = '缺少必要参数'
    return
  }

  connecting.value = true
  error.value = ''

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const params = new URLSearchParams({
    cluster_id: String(clusterID.value),
    namespace: namespace.value,
    pod: podName.value,
    container: container.value,
    shell: shell.value
  })

  const wsUrl = `${protocol}//${window.location.host}/api/v1/k8s/exec/shell?${params.toString()}`

  ws = new WebSocket(wsUrl)

  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    connecting.value = false
    terminal?.focus()
    message.success('终端已连接')

    // 发送初始终端大小
    if (terminal) {
      const { cols, rows } = terminal
      const msg = new Uint8Array(5)
      msg[0] = 1
      msg[1] = (rows >> 8) & 0xff
      msg[2] = rows & 0xff
      msg[3] = (cols >> 8) & 0xff
      msg[4] = cols & 0xff
      ws?.send(msg)
    }
  }

  ws.onmessage = (event) => {
    if (event.data instanceof ArrayBuffer) {
      const text = new TextDecoder().decode(event.data)
      terminal?.write(text)
    } else {
      terminal?.write(event.data)
    }
  }

  ws.onerror = () => {
    error.value = 'WebSocket 连接错误'
    connecting.value = false
  }

  ws.onclose = (event) => {
    connecting.value = false
    if (event.code !== 1000) {
      terminal?.write('\r\n\x1b[31m连接已断开\x1b[0m\r\n')
    }
  }
}

// 重新连接
const reconnect = () => {
  disconnect()
  terminal?.clear()
  connect()
}

// 断开连接
const disconnect = () => {
  if (ws) {
    ws.close()
    ws = null
  }
}

// 处理窗口大小变化
const handleResize = () => {
  fitAddon?.fit()
}

// 返回
const goBack = () => {
  router.back()
}

onMounted(() => {
  nextTick(() => {
    initTerminal()
    connect()
  })
})

onUnmounted(() => {
  disconnect()
  terminal?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped>
.pod-terminal {
  height: 100%;
}

.terminal-container {
  background: #1e1e1e;
  border-radius: 4px;
  padding: 8px;
  min-height: 500px;
  height: calc(100vh - 280px);
}

.terminal-loading {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  color: #999;
}

:deep(.xterm) {
  height: 100%;
}

:deep(.xterm-viewport) {
  overflow-y: auto !important;
}
</style>
