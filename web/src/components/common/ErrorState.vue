<template>
  <div class="error-state" :class="{ 'error-state--small': size === 'small' }">
    <div class="error-state__icon">
      <component :is="iconComponent" :style="{ fontSize: iconSize, color: iconColor }" />
    </div>
    <div class="error-state__content">
      <h3 v-if="title" class="error-state__title">{{ title }}</h3>
      <p v-if="description" class="error-state__description">{{ description }}</p>
      <div v-if="showDetails && errorDetails" class="error-state__details">
        <a-collapse ghost>
          <a-collapse-panel key="1" header="查看详细信息">
            <pre>{{ errorDetails }}</pre>
          </a-collapse-panel>
        </a-collapse>
      </div>
      <slot name="extra"></slot>
    </div>
    <div class="error-state__action">
      <a-space>
        <a-button v-if="showRetry" type="primary" @click="handleRetry">
          <template #icon><ReloadOutlined /></template>
          重试
        </a-button>
        <a-button v-if="showBack" @click="handleBack">
          <template #icon><RollbackOutlined /></template>
          返回
        </a-button>
        <slot name="action"></slot>
      </a-space>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  ExclamationCircleOutlined,
  CloseCircleOutlined,
  WarningOutlined,
  DisconnectOutlined,
  ReloadOutlined,
  RollbackOutlined,
} from '@ant-design/icons-vue'
import { useRouter } from 'vue-router'

interface Props {
  type?: 'error' | 'warning' | 'network' | 'server' | 'custom'
  title?: string
  description?: string
  errorDetails?: string | object
  showDetails?: boolean
  showRetry?: boolean
  showBack?: boolean
  size?: 'default' | 'small'
}

const props = withDefaults(defineProps<Props>(), {
  type: 'error',
  title: '加载失败',
  description: '抱歉，数据加载失败，请稍后重试',
  showDetails: false,
  showRetry: true,
  showBack: false,
  size: 'default',
})

const emit = defineEmits<{
  retry: []
  back: []
}>()

const router = useRouter()

const iconComponent = computed(() => {
  const iconMap = {
    error: CloseCircleOutlined,
    warning: WarningOutlined,
    network: DisconnectOutlined,
    server: ExclamationCircleOutlined,
    custom: ExclamationCircleOutlined,
  }
  return iconMap[props.type] || CloseCircleOutlined
})

const iconColor = computed(() => {
  const colorMap = {
    error: '#ff4d4f',
    warning: '#faad14',
    network: '#ff7a45',
    server: '#ff4d4f',
    custom: '#d9d9d9',
  }
  return colorMap[props.type] || '#ff4d4f'
})

const iconSize = computed(() => {
  return props.size === 'small' ? '48px' : '64px'
})

const handleRetry = () => {
  emit('retry')
}

const handleBack = () => {
  if (emit) {
    emit('back')
  } else {
    router.back()
  }
}
</script>

<style scoped lang="scss">
.error-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
  text-align: center;

  &--small {
    padding: 24px 16px;
  }

  &__icon {
    margin-bottom: 16px;
  }

  &__content {
    margin-bottom: 24px;
    max-width: 600px;
  }

  &__title {
    font-size: 16px;
    font-weight: 500;
    color: rgba(0, 0, 0, 0.85);
    margin: 0 0 8px;
  }

  &__description {
    font-size: 14px;
    color: rgba(0, 0, 0, 0.65);
    margin: 0;
    line-height: 1.5;
  }

  &__details {
    margin-top: 16px;
    text-align: left;

    pre {
      background: #f5f5f5;
      padding: 12px;
      border-radius: 4px;
      font-size: 12px;
      overflow: auto;
      max-height: 200px;
      margin: 0;
    }
  }

  &__action {
    margin-top: 8px;
  }
}

.error-state--small {
  .error-state__title {
    font-size: 14px;
  }

  .error-state__description {
    font-size: 12px;
  }
}
</style>
