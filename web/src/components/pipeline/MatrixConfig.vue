<template>
  <div class="matrix-config">
    <a-card size="small" title="矩阵构建配置">
      <template #extra>
        <a-switch v-model:checked="enabled" @change="onEnabledChange">
          <template #checkedChildren>启用</template>
          <template #unCheckedChildren>禁用</template>
        </a-switch>
      </template>

      <template v-if="enabled">
        <div class="matrix-hint">
          矩阵构建会根据配置的变量组合，自动生成多个并行任务。
        </div>

        <div v-for="(values, key) in matrix" :key="key" class="matrix-row">
          <div class="matrix-key">
            <a-input v-model:value="matrixKeys[key]" placeholder="变量名" style="width: 120px" @blur="updateKey(key)" />
            <a-button type="text" danger size="small" @click="removeVariable(key)">
              <template #icon><DeleteOutlined /></template>
            </a-button>
          </div>
          <div class="matrix-values">
            <a-tag v-for="(val, idx) in values" :key="idx" closable @close="removeValue(key, idx)">
              {{ val }}
            </a-tag>
            <a-input
              v-model:value="newValues[key]"
              placeholder="添加值"
              size="small"
              style="width: 100px"
              @pressEnter="addValue(key)"
            />
          </div>
        </div>

        <a-button type="dashed" block @click="addVariable" style="margin-top: 12px">
          <template #icon><PlusOutlined /></template>
          添加变量
        </a-button>

        <a-divider />

        <div class="preview">
          <div class="preview-title">预览: 将生成 {{ combinations.length }} 个任务</div>
          <div class="preview-list">
            <a-tag v-for="(combo, idx) in combinations.slice(0, 10)" :key="idx" color="blue">
              {{ formatCombo(combo) }}
            </a-tag>
            <span v-if="combinations.length > 10" class="more">... 还有 {{ combinations.length - 10 }} 个</span>
          </div>
        </div>
      </template>

      <a-empty v-else description="矩阵构建已禁用" />
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons-vue'

interface MatrixData {
  include: Record<string, string[]>
}

const props = defineProps<{
  modelValue?: MatrixData | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: MatrixData | null): void
}>()

const enabled = ref(false)
const matrix = ref<Record<string, string[]>>({})
const matrixKeys = ref<Record<string, string>>({})
const newValues = ref<Record<string, string>>({})

// 初始化
watch(() => props.modelValue, (val) => {
  if (val && val.include && Object.keys(val.include).length > 0) {
    enabled.value = true
    matrix.value = { ...val.include }
    matrixKeys.value = Object.keys(val.include).reduce((acc, k) => ({ ...acc, [k]: k }), {})
  } else {
    enabled.value = false
    matrix.value = {}
    matrixKeys.value = {}
  }
}, { immediate: true })

// 计算所有组合
const combinations = computed(() => {
  const keys = Object.keys(matrix.value)
  if (keys.length === 0) return []

  const result: Record<string, string>[] = []
  const generate = (index: number, current: Record<string, string>) => {
    if (index === keys.length) {
      result.push({ ...current })
      return
    }
    const key = keys[index]
    for (const val of matrix.value[key] || []) {
      current[key] = val
      generate(index + 1, current)
    }
  }
  generate(0, {})
  return result
})

const onEnabledChange = (val: boolean) => {
  if (!val) {
    emit('update:modelValue', null)
  } else {
    // 默认添加一个变量
    if (Object.keys(matrix.value).length === 0) {
      addVariable()
    }
    emitChange()
  }
}

const addVariable = () => {
  const key = `VAR_${Object.keys(matrix.value).length + 1}`
  matrix.value[key] = []
  matrixKeys.value[key] = key
  newValues.value[key] = ''
}

const removeVariable = (key: string) => {
  delete matrix.value[key]
  delete matrixKeys.value[key]
  delete newValues.value[key]
  emitChange()
}

const updateKey = (oldKey: string) => {
  const newKey = matrixKeys.value[oldKey]
  if (newKey && newKey !== oldKey) {
    matrix.value[newKey] = matrix.value[oldKey]
    delete matrix.value[oldKey]
    matrixKeys.value[newKey] = newKey
    delete matrixKeys.value[oldKey]
    newValues.value[newKey] = newValues.value[oldKey] || ''
    delete newValues.value[oldKey]
    emitChange()
  }
}

const addValue = (key: string) => {
  const val = newValues.value[key]?.trim()
  if (val && !matrix.value[key].includes(val)) {
    matrix.value[key].push(val)
    newValues.value[key] = ''
    emitChange()
  }
}

const removeValue = (key: string, index: number) => {
  matrix.value[key].splice(index, 1)
  emitChange()
}

const emitChange = () => {
  if (enabled.value && Object.keys(matrix.value).length > 0) {
    emit('update:modelValue', { include: { ...matrix.value } })
  } else {
    emit('update:modelValue', null)
  }
}

const formatCombo = (combo: Record<string, string>) => {
  return Object.entries(combo).map(([k, v]) => `${k}=${v}`).join(', ')
}
</script>

<style scoped>
.matrix-hint { color: #666; font-size: 13px; margin-bottom: 16px; }
.matrix-row { display: flex; align-items: flex-start; margin-bottom: 12px; padding: 8px; background: #fafafa; border-radius: 4px; }
.matrix-key { display: flex; align-items: center; margin-right: 16px; }
.matrix-values { flex: 1; display: flex; flex-wrap: wrap; gap: 4px; align-items: center; }
.preview { background: #f0f5ff; padding: 12px; border-radius: 4px; }
.preview-title { font-weight: 500; margin-bottom: 8px; }
.preview-list { display: flex; flex-wrap: wrap; gap: 4px; }
.more { color: #999; font-size: 12px; }
</style>
