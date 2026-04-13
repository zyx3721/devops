<template>
  <div class="loading-skeleton">
    <!-- 卡片骨架屏 -->
    <template v-if="type === 'card'">
      <a-card v-for="i in count" :key="i" :loading="true">
        <a-skeleton active :paragraph="{ rows: 4 }" />
      </a-card>
    </template>

    <!-- 列表骨架屏 -->
    <template v-else-if="type === 'list'">
      <a-list :data-source="Array(count).fill(null)">
        <template #renderItem>
          <a-list-item>
            <a-skeleton active :avatar="avatar" :paragraph="{ rows: 2 }" />
          </a-list-item>
        </template>
      </a-list>
    </template>

    <!-- 表格骨架屏 -->
    <template v-else-if="type === 'table'">
      <a-table :columns="columns" :data-source="tableData" :pagination="false">
        <template #bodyCell>
          <a-skeleton-button active :size="size" style="width: 100%" />
        </template>
      </a-table>
    </template>

    <!-- 图表骨架屏 -->
    <template v-else-if="type === 'chart'">
      <div class="chart-skeleton">
        <a-skeleton active :paragraph="{ rows: 8 }" />
      </div>
    </template>

    <!-- 表单骨架屏 -->
    <template v-else-if="type === 'form'">
      <a-skeleton active :paragraph="{ rows: count || 6 }" />
    </template>

    <!-- 默认骨架屏 -->
    <template v-else>
      <a-skeleton active :paragraph="{ rows: count || 4 }" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  type?: 'card' | 'list' | 'table' | 'chart' | 'form' | 'default'
  count?: number
  avatar?: boolean
  columns?: any[]
  size?: 'small' | 'default' | 'large'
}

const props = withDefaults(defineProps<Props>(), {
  type: 'default',
  count: 3,
  avatar: false,
  size: 'default',
})

const tableData = computed(() => Array(props.count || 5).fill({}))
</script>

<style scoped>
.loading-skeleton {
  padding: 16px;
}

.loading-skeleton .ant-card {
  margin-bottom: 16px;
}

.chart-skeleton {
  height: 400px;
  padding: 20px;
  background: #f5f5f5;
  border-radius: 4px;
}
</style>
