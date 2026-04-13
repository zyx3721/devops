<template>
  <a-form :label-col="{ span: 5 }" :wrapper-col="{ span: 18 }">
    <a-divider orientation="left">基本信息</a-divider>
    <a-form-item label="名称" required>
      <a-input v-model:value="form.name" placeholder="资源名称" />
    </a-form-item>
    <a-form-item label="类型">
      <a-select v-model:value="form.secretType" style="width: 250px">
        <a-select-option value="Opaque">Opaque (通用)</a-select-option>
        <a-select-option value="kubernetes.io/tls">TLS 证书</a-select-option>
        <a-select-option value="kubernetes.io/dockerconfigjson">Docker Registry</a-select-option>
        <a-select-option value="kubernetes.io/basic-auth">Basic Auth</a-select-option>
        <a-select-option value="kubernetes.io/ssh-auth">SSH Auth</a-select-option>
      </a-select>
    </a-form-item>

    <a-divider orientation="left">数据</a-divider>
    
    <!-- TLS 类型 -->
    <template v-if="form.secretType === 'kubernetes.io/tls'">
      <a-form-item label="TLS 证书" required>
        <a-textarea v-model:value="form.tlsCert" placeholder="-----BEGIN CERTIFICATE-----" :rows="4" />
      </a-form-item>
      <a-form-item label="TLS 私钥" required>
        <a-textarea v-model:value="form.tlsKey" placeholder="-----BEGIN PRIVATE KEY-----" :rows="4" />
      </a-form-item>
    </template>

    <!-- Docker Registry 类型 -->
    <template v-else-if="form.secretType === 'kubernetes.io/dockerconfigjson'">
      <a-form-item label="Registry 地址" required>
        <a-input v-model:value="form.dockerServer" placeholder="如 https://index.docker.io/v1/" />
      </a-form-item>
      <a-form-item label="用户名" required>
        <a-input v-model:value="form.dockerUsername" />
      </a-form-item>
      <a-form-item label="密码" required>
        <a-input-password v-model:value="form.dockerPassword" />
      </a-form-item>
      <a-form-item label="邮箱">
        <a-input v-model:value="form.dockerEmail" />
      </a-form-item>
    </template>

    <!-- Basic Auth 类型 -->
    <template v-else-if="form.secretType === 'kubernetes.io/basic-auth'">
      <a-form-item label="用户名" required>
        <a-input v-model:value="form.basicUsername" />
      </a-form-item>
      <a-form-item label="密码" required>
        <a-input-password v-model:value="form.basicPassword" />
      </a-form-item>
    </template>

    <!-- SSH Auth 类型 -->
    <template v-else-if="form.secretType === 'kubernetes.io/ssh-auth'">
      <a-form-item label="SSH 私钥" required>
        <a-textarea v-model:value="form.sshPrivateKey" placeholder="-----BEGIN RSA PRIVATE KEY-----" :rows="6" />
      </a-form-item>
    </template>

    <!-- Opaque 类型 -->
    <template v-else>
      <div v-for="(item, index) in form.dataItems" :key="index" 
           style="margin-bottom: 12px; border: 1px solid #f0f0f0; padding: 12px; border-radius: 4px">
        <div style="display: flex; justify-content: space-between; margin-bottom: 8px">
          <a-input v-model:value="item.key" placeholder="Key" style="width: 300px" />
          <a-button @click="removeDataItem(index)" danger size="small"><MinusOutlined /></a-button>
        </div>
        <a-input-password v-model:value="item.value" placeholder="Value" />
      </div>
      <a-button @click="addDataItem" type="dashed" block><PlusOutlined /> 添加数据</a-button>
    </template>
  </a-form>
</template>

<script setup lang="ts">
import { PlusOutlined, MinusOutlined } from '@ant-design/icons-vue'
import type { SecretFormData } from './types'

const props = defineProps<{ form: SecretFormData }>()

const addDataItem = () => props.form.dataItems.push({ key: '', value: '' })
const removeDataItem = (index: number) => {
  props.form.dataItems.splice(index, 1)
  if (props.form.dataItems.length === 0) props.form.dataItems.push({ key: '', value: '' })
}
</script>
