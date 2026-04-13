<template>
  <a-popconfirm
    v-if="needConfirm"
    :title="confirmTitle"
    :description="confirmDescription"
    ok-text="确认执行"
    cancel-text="取消"
    @confirm="handleExecute"
  >
    <a-button
      :type="buttonType"
      :size="size"
      :loading="loading"
      :danger="danger"
    >
      <template #icon>
        <component :is="iconComponent" v-if="iconComponent" />
      </template>
      {{ label }}
    </a-button>
  </a-popconfirm>
  <a-button
    v-else
    :type="buttonType"
    :size="size"
    :loading="loading"
    :danger="danger"
    @click="handleExecute"
  >
    <template #icon>
      <component :is="iconComponent" v-if="iconComponent" />
    </template>
    {{ label }}
  </a-button>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { message } from 'ant-design-vue'
import {
  ReloadOutlined,
  ScissorOutlined,
  RollbackOutlined,
  BellOutlined,
  PlayCircleOutlined,
} from '@ant-design/icons-vue'
import { useAIStore } from '../../stores/ai'

const props = defineProps<{
  action: string
  params: Record<string, any>
  label?: string
  size?: 'small' | 'middle' | 'large'
  danger?: boolean
  needConfirm?: boolean
  confirmTitle?: string
  confirmDescription?: string
}>()

const emit = defineEmits<{
  (e: 'success', result: any): void
  (e: 'error', error: any): void
}>()

const aiStore = useAIStore()
const loading = ref(false)

// 操作配置
const actionConfig: Record<string, { icon: any; label: string; danger: boolean; confirm: boolean }> = {
  restart_app: { icon: ReloadOutlined, label: '重启应用', danger: true, confirm: true },
  scale_pod: { icon: ScissorOutlined, label: '扩缩容', danger: false, confirm: true },
  rollback: { icon: RollbackOutlined, label: '回滚', danger: true, confirm: true },
  silence_alert: { icon: BellOutlined, label: '静默告警', danger: false, confirm: true },
  execute: { icon: PlayCircleOutlined, label: '执行', danger: false, confirm: false },
}

const config = computed(() => actionConfig[props.action] || actionConfig.execute)

const iconComponent = computed(() => config.value.icon)
const label = computed(() => props.label || config.value.label)
const buttonType = computed(() => props.danger || config.value.danger ? 'primary' : 'default')
const needConfirm = computed(() => props.needConfirm ?? config.value.confirm)
const confirmTitle = computed(() => props.confirmTitle || `确认${label.value}？`)
const confirmDescription = computed(() => props.confirmDescription || '此操作可能会影响服务，请确认后执行。')

const handleExecute = async () => {
  loading.value = true
  try {
    const result = await aiStore.executeAction(props.action, props.params)
    if (result?.success) {
      message.success(result.message || '操作成功')
      emit('success', result)
    } else if (result?.need_confirm) {
      message.warning(result.confirm_msg || '需要确认')
    } else {
      message.error(result?.message || '操作失败')
      emit('error', result)
    }
  } catch (e: any) {
    message.error(e.message || '操作失败')
    emit('error', e)
  } finally {
    loading.value = false
  }
}
</script>
