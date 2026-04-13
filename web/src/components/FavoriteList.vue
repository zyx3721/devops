<template>
  <a-dropdown :trigger="['click']">
    <a-badge :count="favorites.length" :offset="[-5, 5]">
      <a-button type="text">
        <template #icon><StarOutlined /></template>
      </a-button>
    </a-badge>
    <template #overlay>
      <div class="favorite-dropdown">
        <div class="favorite-header">
          <span>我的收藏</span>
          <a-button v-if="favorites.length > 0" type="link" size="small" danger @click="clearAll">
            清空
          </a-button>
        </div>
        <a-divider style="margin: 8px 0" />
        <div v-if="favorites.length === 0" class="favorite-empty">
          <a-empty description="暂无收藏" :image="Empty.PRESENTED_IMAGE_SIMPLE" />
        </div>
        <div v-else class="favorite-list">
          <div 
            v-for="item in favorites" 
            :key="`${item.type}-${item.id}`" 
            class="favorite-item"
            @click="goToFavorite(item)"
          >
            <div class="favorite-item-icon">
              <RocketOutlined v-if="item.type === 'pipeline'" />
              <CloudOutlined v-else-if="item.type === 'cluster'" />
              <AppstoreOutlined v-else-if="item.type === 'application'" />
              <DeploymentUnitOutlined v-else />
            </div>
            <div class="favorite-item-content">
              <div class="favorite-item-name">{{ item.name }}</div>
              <div class="favorite-item-type">{{ getTypeLabel(item.type) }}</div>
            </div>
            <a-button type="text" size="small" @click.stop="removeFavorite(item)">
              <template #icon><CloseOutlined /></template>
            </a-button>
          </div>
        </div>
      </div>
    </template>
  </a-dropdown>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Empty } from 'ant-design-vue'
import { StarOutlined, CloseOutlined, RocketOutlined, CloudOutlined, AppstoreOutlined, DeploymentUnitOutlined } from '@ant-design/icons-vue'
import { useFavoriteStore, type FavoriteItem } from '@/stores/favorite'

const router = useRouter()
const favoriteStore = useFavoriteStore()

const favorites = computed(() => favoriteStore.favorites)

const getTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    pipeline: '流水线',
    cluster: '集群',
    application: '应用',
    deployment: 'Deployment'
  }
  return labels[type] || type
}

const goToFavorite = (item: FavoriteItem) => {
  if (item.path) {
    router.push(item.path)
  } else {
    // 根据类型生成路径
    const paths: Record<string, string> = {
      pipeline: `/pipeline/${item.id}`,
      cluster: `/k8s/clusters/${item.id}/resources`,
      application: `/applications?id=${item.id}`,
      deployment: `/k8s/clusters/${item.id}/deployments`
    }
    router.push(paths[item.type] || '/')
  }
}

const removeFavorite = (item: FavoriteItem) => {
  favoriteStore.removeFavorite(item.type, item.id)
}

const clearAll = () => {
  favoriteStore.clearAll()
}
</script>

<style scoped>
.favorite-dropdown {
  width: 280px;
  padding: 12px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 3px 6px -4px rgba(0,0,0,.12), 0 6px 16px 0 rgba(0,0,0,.08);
}

.favorite-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 500;
}

.favorite-empty {
  padding: 16px 0;
}

.favorite-list {
  max-height: 300px;
  overflow-y: auto;
}

.favorite-item {
  display: flex;
  align-items: center;
  padding: 8px;
  border-radius: 4px;
  cursor: pointer;
  transition: background 0.2s;
}

.favorite-item:hover {
  background: #f5f5f5;
}

.favorite-item-icon {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #e6f7ff;
  border-radius: 4px;
  color: #1890ff;
  margin-right: 12px;
}

.favorite-item-content {
  flex: 1;
  min-width: 0;
}

.favorite-item-name {
  font-size: 14px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.favorite-item-type {
  font-size: 12px;
  color: #999;
}
</style>
