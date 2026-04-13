import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export interface FavoriteItem {
  type: 'pipeline' | 'cluster' | 'application' | 'deployment'
  id: number | string
  name: string
  path: string
  addedAt?: string
}

const STORAGE_KEY = 'devops_favorites'

export const useFavoriteStore = defineStore('favorite', () => {
  const favorites = ref<FavoriteItem[]>([])

  // 从 localStorage 加载
  const loadFromStorage = () => {
    if (typeof window === 'undefined') return
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored) {
        favorites.value = JSON.parse(stored)
      }
    } catch (e) {
      console.error('Failed to load favorites', e)
    }
  }

  // 保存到 localStorage
  const saveToStorage = () => {
    if (typeof window === 'undefined') return
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(favorites.value))
    } catch (e) {
      console.error('Failed to save favorites', e)
    }
  }

  // 监听变化自动保存
  watch(favorites, saveToStorage, { deep: true })

  // 初始化加载
  if (typeof window !== 'undefined') {
    loadFromStorage()
  }

  const isFavorite = (type: string, id: number | string) => {
    return favorites.value.some(f => f.type === type && f.id === id)
  }

  const addFavorite = (item: FavoriteItem) => {
    if (!isFavorite(item.type, item.id)) {
      favorites.value.push({
        ...item,
        addedAt: new Date().toISOString()
      })
    }
  }

  const removeFavorite = (type: string, id: number | string) => {
    const index = favorites.value.findIndex(f => f.type === type && f.id === id)
    if (index > -1) {
      favorites.value.splice(index, 1)
    }
  }

  const getFavoritesByType = (type: string) => {
    return favorites.value.filter(f => f.type === type)
  }

  const clearAll = () => {
    favorites.value = []
  }

  return {
    favorites,
    isFavorite,
    addFavorite,
    removeFavorite,
    getFavoritesByType,
    clearAll
  }
})
