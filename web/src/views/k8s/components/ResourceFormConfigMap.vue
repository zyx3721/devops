<template>
  <a-form :label-col="{ span: 5 }" :wrapper-col="{ span: 18 }">
    <a-divider orientation="left">基本信息</a-divider>
    <a-form-item label="名称" required>
      <a-input v-model:value="form.name" placeholder="资源名称" />
    </a-form-item>
    <a-form-item label="标签">
      <div v-for="(item, index) in form.labelItems" :key="'label'+index" style="display: flex; gap: 8px; margin-bottom: 8px">
        <a-input v-model:value="item.key" placeholder="Key" style="width: 150px" />
        <a-input v-model:value="item.value" placeholder="Value" style="width: 200px" />
        <a-button @click="removeLabelItem(index)" danger size="small"><MinusOutlined /></a-button>
      </div>
      <a-button @click="addLabelItem" type="dashed" size="small"><PlusOutlined /> 添加标签</a-button>
    </a-form-item>

    <a-divider orientation="left">数据</a-divider>
    <div v-for="(item, index) in form.dataItems" :key="index" 
         style="margin-bottom: 12px; border: 1px solid #f0f0f0; padding: 12px; border-radius: 4px">
      <div style="display: flex; justify-content: space-between; margin-bottom: 8px">
        <a-input v-model:value="item.key" placeholder="Key" style="width: 300px" />
        <a-button @click="removeDataItem(index)" danger size="small"><MinusOutlined /></a-button>
      </div>
      <a-textarea v-model:value="item.value" placeholder="Value (支持多行)" :rows="4" />
    </div>
    <a-button @click="addDataItem" type="dashed" block><PlusOutlined /> 添加数据</a-button>
  </a-form>
</template>

<script setup lang="ts">
import { PlusOutlined, MinusOutlined } from '@ant-design/icons-vue'
import type { ConfigMapFormData } from './types'

const props = defineProps<{ form: ConfigMapFormData }>()

const addLabelItem = () => props.form.labelItems.push({ key: '', value: '' })
const removeLabelItem = (index: number) => {
  props.form.labelItems.splice(index, 1)
  if (props.form.labelItems.length === 0) props.form.labelItems.push({ key: '', value: '' })
}

const addDataItem = () => props.form.dataItems.push({ key: '', value: '' })
const removeDataItem = (index: number) => {
  props.form.dataItems.splice(index, 1)
  if (props.form.dataItems.length === 0) props.form.dataItems.push({ key: '', value: '' })
}
</script>
