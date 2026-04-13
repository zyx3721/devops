<template>
  <a-tooltip :title="isFavorite ? '取消收藏' : '添加收藏'">
    <a-button 
      :type="isFavorite ? 'primary' : 'default'" 
      shape="circle" 
      size="small"
      @click.stop="toggleFavorite"
    >
      <template #icon>
        <StarFilled v-if="isFavorite" />
        <StarOutlined v-else />
      </template>
    </a-button>
  </a-tooltip>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { StarOutlined, StarFilled } from '@ant-design/icons-vue'
import { useFavoriteStore } from '@/stores/favorite'

const props = defineProps<{
  type: 'pipeline' | 'cluster' | 'application' | 'deployment'
  id: number | string
  name: string
  path?: string
}>()

const favoriteStore = useFavoriteStore()

const isFavorite = computed(() => {
  return favoriteStore.isFavorite(props.type, props.id)
})

const toggleFavorite = () => {
  if (isFavorite.value) {
    favoriteStore.removeFavorite(props.type, props.id)
  } else {
    favoriteStore.addFavorite({
      type: props.type,
      id: props.id,
      name: props.name,
      path: props.path || ''
    })
  }
}
</script>
