<template>
  <el-select
    v-model="selectedNamespace"
    placeholder="选择命名空间"
    filterable
    clearable
    @change="handleChange"
    style="width: 200px"
  >
    <el-option label="全部命名空间" value="" />
    <el-option
      v-for="ns in namespaces"
      :key="ns.name"
      :label="ns.name"
      :value="ns.name"
    >
      <span>{{ ns.name }}</span>
      <span style="float: right; color: var(--el-text-color-secondary); font-size: 12px">
        {{ ns.status }}
      </span>
    </el-option>
  </el-select>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

interface NamespaceInfo {
  name: string
  status: string
  age: string
  labels: Record<string, string>
  created_at: string
}

interface Props {
  namespaces: NamespaceInfo[]
  modelValue?: string
}

interface Emits {
  (e: 'update:modelValue', value: string): void
  (e: 'change', value: string): void
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: ''
})

const emit = defineEmits<Emits>()

const selectedNamespace = ref(props.modelValue)

watch(() => props.modelValue, (newVal) => {
  selectedNamespace.value = newVal
})

const handleChange = (value: string) => {
  emit('update:modelValue', value)
  emit('change', value)
}
</script>

<style scoped>
</style>
