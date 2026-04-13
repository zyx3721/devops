<template>
  <a-dropdown :trigger="['click']">
    <a-button type="text">
      <template #icon>
        <BulbOutlined v-if="!isDark" />
        <BulbFilled v-else />
      </template>
    </a-button>
    <template #overlay>
      <a-menu @click="handleClick">
        <a-menu-item key="light">
          <BulbOutlined /> 浅色模式
          <CheckOutlined v-if="mode === 'light'" style="margin-left: 8px; color: #1890ff" />
        </a-menu-item>
        <a-menu-item key="dark">
          <BulbFilled /> 深色模式
          <CheckOutlined v-if="mode === 'dark'" style="margin-left: 8px; color: #1890ff" />
        </a-menu-item>
        <a-menu-item key="auto">
          <DesktopOutlined /> 跟随系统
          <CheckOutlined v-if="mode === 'auto'" style="margin-left: 8px; color: #1890ff" />
        </a-menu-item>
      </a-menu>
    </template>
  </a-dropdown>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { BulbOutlined, BulbFilled, DesktopOutlined, CheckOutlined } from '@ant-design/icons-vue'
import { useThemeStore, type ThemeMode } from '@/stores/theme'

const themeStore = useThemeStore()

const mode = computed(() => themeStore.mode)
const isDark = computed(() => themeStore.isDark)

const handleClick = ({ key }: { key: string }) => {
  themeStore.setMode(key as ThemeMode)
}
</script>
