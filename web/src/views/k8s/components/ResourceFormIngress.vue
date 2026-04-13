<template>
  <a-form :label-col="{ span: 5 }" :wrapper-col="{ span: 18 }">
    <a-divider orientation="left">基本信息</a-divider>
    <a-form-item label="名称" required>
      <a-input v-model:value="form.name" placeholder="资源名称" />
    </a-form-item>
    <a-form-item label="Ingress Class">
      <a-input v-model:value="form.ingressClass" placeholder="如 nginx" />
    </a-form-item>

    <a-divider orientation="left">TLS 配置</a-divider>
    <div v-for="(tls, tIdx) in form.tlsConfigs" :key="'tls'+tIdx" style="display: flex; gap: 8px; margin-bottom: 8px">
      <a-input v-model:value="tls.hosts" placeholder="域名，多个用逗号分隔" style="flex: 1" />
      <a-input v-model:value="tls.secretName" placeholder="TLS Secret 名称" style="width: 200px" />
      <a-button @click="removeTlsConfig(tIdx)" danger size="small"><MinusOutlined /></a-button>
    </div>
    <a-button @click="addTlsConfig" type="dashed" size="small"><PlusOutlined /> 添加 TLS</a-button>

    <a-divider orientation="left">路由规则</a-divider>
    <div v-for="(rule, rIdx) in form.ingressRules" :key="'rule'+rIdx" 
         style="border: 1px solid #f0f0f0; padding: 12px; margin-bottom: 12px; border-radius: 4px">
      <div style="display: flex; justify-content: space-between; margin-bottom: 8px">
        <a-input v-model:value="rule.host" placeholder="域名 (如 example.com)" style="width: 300px" addon-before="域名" />
        <a-button @click="removeIngressRule(rIdx)" danger v-if="form.ingressRules.length > 1">
          <MinusOutlined />
        </a-button>
      </div>
      <div v-for="(path, pIdx) in rule.paths" :key="'path'+pIdx" 
           style="display: flex; gap: 8px; margin-bottom: 8px; margin-left: 20px">
        <a-input v-model:value="path.path" placeholder="路径" style="width: 120px" />
        <a-select v-model:value="path.pathType" style="width: 150px">
          <a-select-option value="Prefix">Prefix</a-select-option>
          <a-select-option value="Exact">Exact</a-select-option>
          <a-select-option value="ImplementationSpecific">ImplementationSpecific</a-select-option>
        </a-select>
        <a-input v-model:value="path.serviceName" placeholder="服务名称" style="width: 150px" />
        <a-input-number v-model:value="path.servicePort" placeholder="端口" :min="1" :max="65535" style="width: 100px" />
        <a-button @click="removeIngressPath(rule, pIdx)" danger size="small" v-if="rule.paths.length > 1">
          <MinusOutlined />
        </a-button>
      </div>
      <a-button @click="addIngressPath(rule)" type="dashed" size="small" style="margin-left: 20px">
        <PlusOutlined /> 添加路径
      </a-button>
    </div>
    <a-button @click="addIngressRule" type="dashed" block><PlusOutlined /> 添加规则</a-button>

    <a-divider orientation="left">注解</a-divider>
    <div v-for="(item, index) in form.annotationItems" :key="'ann'+index" style="display: flex; gap: 8px; margin-bottom: 8px">
      <a-input v-model:value="item.key" placeholder="Key" style="width: 250px" />
      <a-input v-model:value="item.value" placeholder="Value" style="flex: 1" />
      <a-button @click="removeAnnotationItem(index)" danger size="small"><MinusOutlined /></a-button>
    </div>
    <a-button @click="addAnnotationItem" type="dashed" size="small"><PlusOutlined /> 添加注解</a-button>
  </a-form>
</template>

<script setup lang="ts">
import { PlusOutlined, MinusOutlined } from '@ant-design/icons-vue'
import type { IngressFormData } from './types'

const props = defineProps<{ form: IngressFormData }>()

const addTlsConfig = () => props.form.tlsConfigs.push({ hosts: '', secretName: '' })
const removeTlsConfig = (index: number) => props.form.tlsConfigs.splice(index, 1)

const addIngressRule = () => props.form.ingressRules.push({ 
  host: '', 
  paths: [{ path: '/', pathType: 'Prefix', serviceName: '', servicePort: 80 }] 
})
const removeIngressRule = (index: number) => props.form.ingressRules.splice(index, 1)

const addIngressPath = (rule: any) => rule.paths.push({ path: '/', pathType: 'Prefix', serviceName: '', servicePort: 80 })
const removeIngressPath = (rule: any, index: number) => rule.paths.splice(index, 1)

const addAnnotationItem = () => props.form.annotationItems.push({ key: '', value: '' })
const removeAnnotationItem = (index: number) => props.form.annotationItems.splice(index, 1)
</script>
