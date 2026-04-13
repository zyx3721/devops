<template>
  <div class="message-channel">
    <a-tabs v-model:activeKey="activeTab" @change="handleTabChange">
      <a-tab-pane key="feishu" tab="飞书">
        <FeishuPanel />
      </a-tab-pane>
      <a-tab-pane key="dingtalk" tab="钉钉">
        <DingtalkPanel />
      </a-tab-pane>
      <a-tab-pane key="wechatwork" tab="企业微信">
        <WechatWorkPanel />
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import FeishuPanel from './panels/FeishuPanel.vue'
import DingtalkPanel from './panels/DingtalkPanel.vue'
import WechatWorkPanel from './panels/WechatWorkPanel.vue'

const route = useRoute()
const router = useRouter()

const activeTab = ref((route.query.tab as string) || 'feishu')

function handleTabChange(key: string) {
  router.replace({ query: { tab: key } })
}
</script>

<style scoped>
.message-channel {
  padding: 0;
}
</style>
