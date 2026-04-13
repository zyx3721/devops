<template>
  <div class="admin-dashboard">
    <!-- 统计概览 -->
    <a-row :gutter="16">
      <a-col :span="6">
        <a-card>
          <a-statistic title="用户总数" :value="stats.total_users" :value-style="{ color: '#1890ff' }">
            <template #prefix><TeamOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="K8s 集群" :value="stats.total_k8s_clusters" :value-style="{ color: '#52c41a' }">
            <template #prefix><CloudServerOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="Jenkins 实例" :value="stats.total_jenkins_instances">
            <template #prefix><SettingOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="流水线" :value="stats.total_pipelines">
            <template #prefix><RocketOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <a-row :gutter="16" style="margin-top: 16px">
      <a-col :span="6">
        <a-card>
          <a-statistic title="应用数量" :value="stats.total_applications">
            <template #prefix><AppstoreOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>

<script setup lang="ts">
import { reactive, onMounted } from 'vue'
import {
  TeamOutlined, CloudServerOutlined, SettingOutlined, RocketOutlined, AppstoreOutlined
} from '@ant-design/icons-vue'
import { adminApi } from '@/services/admin'

const stats = reactive({
  total_users: 0,
  total_k8s_clusters: 0,
  total_jenkins_instances: 0,
  total_pipelines: 0,
  total_applications: 0,
})

const fetchStats = async () => {
  try {
    const res = await adminApi.getStats()
    Object.assign(stats, (res as any).data || res)
  } catch (e) {
    // 错误已统一处理
  }
}

onMounted(() => {
  fetchStats()
})
</script>

<style scoped>
.admin-dashboard {
  padding: 0;
}
</style>
