<template>
  <a-form :label-col="{ span: 5 }" :wrapper-col="{ span: 18 }">
    <a-divider orientation="left">基本信息</a-divider>
    <a-form-item label="名称" required>
      <a-input v-model:value="form.name" placeholder="资源名称" />
    </a-form-item>
    <a-form-item label="类型">
      <a-select v-model:value="form.serviceType" style="width: 200px">
        <a-select-option value="ClusterIP">ClusterIP</a-select-option>
        <a-select-option value="NodePort">NodePort</a-select-option>
        <a-select-option value="LoadBalancer">LoadBalancer</a-select-option>
        <a-select-option value="ExternalName">ExternalName</a-select-option>
      </a-select>
    </a-form-item>
    <a-form-item v-if="form.serviceType === 'ExternalName'" label="外部名称" required>
      <a-input v-model:value="form.externalName" placeholder="如 my.database.example.com" />
    </a-form-item>

    <template v-if="form.serviceType !== 'ExternalName'">
      <a-divider orientation="left">选择器</a-divider>
      <a-form-item label="Pod 选择器">
        <div v-for="(item, index) in form.selectorItems" :key="'sel'+index" style="display: flex; gap: 8px; margin-bottom: 8px">
          <a-input v-model:value="item.key" placeholder="Key" style="width: 150px" />
          <a-input v-model:value="item.value" placeholder="Value" style="width: 200px" />
          <a-button @click="removeSelectorItem(index)" danger size="small"><MinusOutlined /></a-button>
        </div>
        <a-button @click="addSelectorItem" type="dashed" size="small"><PlusOutlined /> 添加选择器</a-button>
      </a-form-item>

      <a-divider orientation="left">端口配置</a-divider>
      <div v-for="(port, pIdx) in form.servicePorts" :key="'sp'+pIdx" 
           style="border: 1px solid #f0f0f0; padding: 12px; margin-bottom: 12px; border-radius: 4px">
        <a-row :gutter="16">
          <a-col :span="8">
            <a-input v-model:value="port.name" placeholder="端口名称" addon-before="名称" />
          </a-col>
          <a-col :span="8">
            <a-select v-model:value="port.protocol" style="width: 100%">
              <a-select-option value="TCP">TCP</a-select-option>
              <a-select-option value="UDP">UDP</a-select-option>
            </a-select>
          </a-col>
          <a-col :span="8">
            <a-button @click="removeServicePort(pIdx)" danger v-if="form.servicePorts.length > 1">
              <MinusOutlined /> 删除
            </a-button>
          </a-col>
        </a-row>
        <a-row :gutter="16" style="margin-top: 8px">
          <a-col :span="8">
            <a-input-number v-model:value="port.port" placeholder="端口" :min="1" :max="65535" style="width: 100%" addon-before="端口" />
          </a-col>
          <a-col :span="8">
            <a-input v-model:value="port.targetPort" placeholder="目标端口" addon-before="目标端口" />
          </a-col>
          <a-col :span="8" v-if="form.serviceType === 'NodePort'">
            <a-input-number v-model:value="port.nodePort" placeholder="30000-32767" :min="30000" :max="32767" style="width: 100%" addon-before="NodePort" />
          </a-col>
        </a-row>
      </div>
      <a-button @click="addServicePort" type="dashed" block><PlusOutlined /> 添加端口</a-button>

      <a-divider orientation="left">高级配置</a-divider>
      <a-form-item label="会话亲和性">
        <a-select v-model:value="form.sessionAffinity" style="width: 200px">
          <a-select-option value="None">None</a-select-option>
          <a-select-option value="ClientIP">ClientIP</a-select-option>
        </a-select>
      </a-form-item>
      <a-form-item v-if="form.serviceType === 'LoadBalancer'" label="负载均衡IP">
        <a-input v-model:value="form.loadBalancerIP" placeholder="指定 LoadBalancer IP" />
      </a-form-item>
    </template>
  </a-form>
</template>

<script setup lang="ts">
import { PlusOutlined, MinusOutlined } from '@ant-design/icons-vue'
import type { ServiceFormData } from './types'

const props = defineProps<{ form: ServiceFormData }>()

const addSelectorItem = () => props.form.selectorItems.push({ key: '', value: '' })
const removeSelectorItem = (index: number) => {
  props.form.selectorItems.splice(index, 1)
  if (props.form.selectorItems.length === 0) props.form.selectorItems.push({ key: '', value: '' })
}

const addServicePort = () => props.form.servicePorts.push({ name: '', port: 80, targetPort: '80', protocol: 'TCP', nodePort: undefined })
const removeServicePort = (index: number) => props.form.servicePorts.splice(index, 1)
</script>
