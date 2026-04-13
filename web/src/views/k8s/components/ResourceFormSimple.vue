<template>
  <a-form :label-col="{ span: 5 }" :wrapper-col="{ span: 18 }">
    <a-alert message="该资源类型配置较复杂，建议使用 YAML 创建" type="info" show-icon style="margin-bottom: 16px" />
    
    <a-form-item label="名称" required>
      <a-input v-model:value="form.name" placeholder="资源名称" />
    </a-form-item>

    <!-- StatefulSet / DaemonSet -->
    <template v-if="['statefulsets', 'daemonsets'].includes(resourceType)">
      <a-form-item label="镜像" required>
        <a-input v-model:value="form.image" placeholder="如 nginx:latest" />
      </a-form-item>
      <a-form-item v-if="resourceType === 'statefulsets'" label="副本数">
        <a-input-number v-model:value="form.replicas" :min="1" :max="100" />
      </a-form-item>
      <a-form-item v-if="resourceType === 'statefulsets'" label="服务名">
        <a-input v-model:value="form.serviceName" placeholder="Headless Service 名称" />
      </a-form-item>
      <a-form-item label="容器端口">
        <a-input-number v-model:value="form.containerPort" :min="1" :max="65535" />
      </a-form-item>
    </template>

    <!-- Job / CronJob -->
    <template v-if="['jobs', 'cronjobs'].includes(resourceType)">
      <a-form-item label="镜像" required>
        <a-input v-model:value="form.image" placeholder="如 busybox" />
      </a-form-item>
      <a-form-item label="命令">
        <a-input v-model:value="form.command" placeholder="逗号分隔，如: echo,hello" />
      </a-form-item>
      <a-form-item v-if="resourceType === 'jobs'" label="重试次数">
        <a-input-number v-model:value="form.backoffLimit" :min="0" :max="10" />
      </a-form-item>
      <a-form-item v-if="resourceType === 'cronjobs'" label="调度" required>
        <a-input v-model:value="form.schedule" placeholder="*/5 * * * *" />
        <div style="color: #999; font-size: 12px; margin-top: 4px">
          Cron 表达式: 分 时 日 月 周
        </div>
      </a-form-item>
    </template>

    <!-- PVC -->
    <template v-if="resourceType === 'pvcs'">
      <a-form-item label="存储大小" required>
        <a-input v-model:value="form.storage" placeholder="如 1Gi, 10Gi" />
      </a-form-item>
      <a-form-item label="访问模式">
        <a-select v-model:value="form.accessMode" style="width: 200px">
          <a-select-option value="ReadWriteOnce">ReadWriteOnce (单节点读写)</a-select-option>
          <a-select-option value="ReadOnlyMany">ReadOnlyMany (多节点只读)</a-select-option>
          <a-select-option value="ReadWriteMany">ReadWriteMany (多节点读写)</a-select-option>
        </a-select>
      </a-form-item>
      <a-form-item label="StorageClass">
        <a-input v-model:value="form.storageClass" placeholder="留空使用默认 StorageClass" />
      </a-form-item>
    </template>
  </a-form>
</template>

<script setup lang="ts">
import type { SimpleFormData } from './types'

defineProps<{ 
  form: SimpleFormData
  resourceType: string 
}>()
</script>
