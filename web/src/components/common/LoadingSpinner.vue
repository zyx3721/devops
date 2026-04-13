<template>
  <div class="loading-spinner" :class="{ fullscreen, overlay }">
    <div class="spinner-container">
      <a-spin :size="size" :tip="tip">
        <template #indicator v-if="customIcon">
          <LoadingOutlined style="font-size: 48px" spin />
        </template>
      </a-spin>
      <div v-if="message" class="loading-message">{{ message }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { LoadingOutlined } from '@ant-design/icons-vue'

interface Props {
  size?: 'small' | 'default' | 'large'
  tip?: string
  message?: string
  fullscreen?: boolean
  overlay?: boolean
  customIcon?: boolean
}

withDefaults(defineProps<Props>(), {
  size: 'large',
  fullscreen: false,
  overlay: true,
  customIcon: false,
})
</script>

<style scoped>
.loading-spinner {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
}

.loading-spinner.fullscreen {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 9999;
  min-height: 100vh;
}

.loading-spinner.overlay {
  background: rgba(255, 255, 255, 0.9);
}

.spinner-container {
  text-align: center;
}

.loading-message {
  margin-top: 16px;
  color: #666;
  font-size: 14px;
}
</style>
