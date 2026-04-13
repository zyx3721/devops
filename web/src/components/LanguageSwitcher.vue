<template>
  <a-dropdown>
    <a-button type="text">
      <GlobalOutlined />
      <span style="margin-left: 4px">{{ currentLocaleName }}</span>
    </a-button>
    <template #overlay>
      <a-menu @click="handleLocaleChange">
        <a-menu-item 
          v-for="locale in SUPPORT_LOCALES" 
          :key="locale"
          :class="{ 'ant-menu-item-selected': locale === currentLocale }"
        >
          <CheckOutlined v-if="locale === currentLocale" style="margin-right: 8px" />
          <span :style="{ marginLeft: locale === currentLocale ? '0' : '24px' }">
            {{ LOCALE_NAMES[locale] }}
          </span>
        </a-menu-item>
      </a-menu>
    </template>
  </a-dropdown>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { GlobalOutlined, CheckOutlined } from '@ant-design/icons-vue'
import { SUPPORT_LOCALES, LOCALE_NAMES, setLocale, getLocale, type SupportLocale } from '@/locales'

const currentLocale = computed(() => getLocale())
const currentLocaleName = computed(() => LOCALE_NAMES[currentLocale.value])

const handleLocaleChange = ({ key }: { key: string }) => {
  setLocale(key as SupportLocale)
}
</script>

<style scoped>
:deep(.ant-menu-item-selected) {
  background-color: #e6f7ff;
}
</style>
