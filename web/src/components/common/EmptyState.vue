<template>
  <div class="empty-state" :class="{ 'empty-state--small': size === 'small' }">
    <div class="empty-state__icon">
      <component :is="iconComponent" :style="{ fontSize: iconSize }" />
    </div>
    <div class="empty-state__content">
      <h3 v-if="title" class="empty-state__title">{{ title }}</h3>
      <p v-if="description" class="empty-state__description">{{ description }}</p>
      <slot name="extra"></slot>
    </div>
    <div v-if="showAction" class="empty-state__action">
      <a-button v-if="actionText" :type="actionType" @click="handleAction">
        {{ actionText }}
      </a-button>
      <slot name="action"></slot>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  InboxOutlined,
  FileSearchOutlined,
  FolderOpenOutlined,
  DatabaseOutlined,
  CloudServerOutlined,
} from '@ant-design/icons-vue'

interface Props {
  type?: 'default' | 'search' | 'folder' | 'data' | 'server'
  title?: string
  description?: string
  actionText?: string
  actionType?: 'primary' | 'default' | 'dashed' | 'link'
  size?: 'default' | 'small'
}

const props = withDefaults(defineProps<Props>(), {
  type: 'default',
  title: '暂无数据',
  description: '',
  actionText: '',
  actionType: 'primary',
  size: 'default',
})

const emit = defineEmits<{
  action: []
}>()

const iconComponent = computed(() => {
  const iconMap = {
    default: InboxOutlined,
    search: FileSearchOutlined,
    folder: FolderOpenOutlined,
    data: DatabaseOutlined,
    server: CloudServerOutlined,
  }
  return iconMap[props.type] || InboxOutlined
})

const iconSize = computed(() => {
  return props.size === 'small' ? '48px' : '64px'
})

const showAction = computed(() => {
  return props.actionText || !!emit
})

const handleAction = () => {
  emit('action')
}
</script>

<style scoped lang="scss">
.empty-state {
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
    color: #d9d9d9;
    margin-bottom: 16px;
  }

  &__content {
    margin-bottom: 16px;
  }

  &__title {
    font-size: 16px;
    font-weight: 500;
    color: rgba(0, 0, 0, 0.85);
    margin: 0 0 8px;
  }

  &__description {
    font-size: 14px;
    color: rgba(0, 0, 0, 0.45);
    margin: 0;
    line-height: 1.5;
  }

  &__action {
    margin-top: 8px;
  }
}

.empty-state--small {
  .empty-state__title {
    font-size: 14px;
  }

  .empty-state__description {
    font-size: 12px;
  }
}
</style>
