<template>
  <div class="deploy-check">
    <a-page-header title="部署检查" sub-title="部署前置检查">
      <template #extra>
        <a-button type="primary" @click="$router.push('/canary/list')">
          <RocketOutlined /> 灰度发布
        </a-button>
      </template>
    </a-page-header>

    <a-card>
      <a-form layout="inline" @finish="runPreCheck">
        <a-form-item label="应用">
          <a-select v-model:value="checkForm.application_id" style="width: 200px" placeholder="选择应用" show-search>
            <a-select-option v-for="app in applications" :key="app.id" :value="app.id">{{ app.name }}</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="环境">
          <a-select v-model:value="checkForm.env_name" style="width: 120px">
            <a-select-option value="dev">开发</a-select-option>
            <a-select-option value="test">测试</a-select-option>
            <a-select-option value="staging">预发</a-select-option>
            <a-select-option value="prod">生产</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="镜像标签">
          <a-input v-model:value="checkForm.image_tag" placeholder="可选" style="width: 200px" />
        </a-form-item>
        <a-form-item>
          <a-button type="primary" html-type="submit" :loading="checking">执行检查</a-button>
        </a-form-item>
      </a-form>

      <a-divider />

      <div v-if="checkResult" class="check-result">
        <a-result :status="checkResult.can_deploy ? 'success' : 'warning'" :title="checkResult.can_deploy ? '可以部署' : '存在问题'">
          <template #extra>
            <a-space v-if="checkResult.can_deploy">
              <a-button type="primary" @click="goToDeploy">去部署</a-button>
              <a-button @click="goToCanary">灰度发布</a-button>
            </a-space>
          </template>
        </a-result>

        <a-list :data-source="checkResult.checks" bordered>
          <template #renderItem="{ item }">
            <a-list-item>
              <a-list-item-meta :title="item.name" :description="item.message">
                <template #avatar>
                  <CheckCircleOutlined v-if="item.status === 'passed'" style="color: #52c41a; font-size: 20px" />
                  <WarningOutlined v-else-if="item.status === 'warning'" style="color: #faad14; font-size: 20px" />
                  <CloseCircleOutlined v-else-if="item.status === 'failed'" style="color: #ff4d4f; font-size: 20px" />
                  <MinusCircleOutlined v-else style="color: #999; font-size: 20px" />
                </template>
              </a-list-item-meta>
              <template #actions>
                <a-tag :color="getStatusColor(item.status)">{{ item.status }}</a-tag>
              </template>
            </a-list-item>
          </template>
        </a-list>

        <a-alert v-if="checkResult.warnings && checkResult.warnings.length > 0" type="warning" show-icon style="margin-top: 16px">
          <template #message>警告</template>
          <template #description>
            <ul>
              <li v-for="(w, i) in checkResult.warnings" :key="i">{{ w }}</li>
            </ul>
          </template>
        </a-alert>

        <a-alert v-if="checkResult.errors && checkResult.errors.length > 0" type="error" show-icon style="margin-top: 16px">
          <template #message>错误</template>
          <template #description>
            <ul>
              <li v-for="(e, i) in checkResult.errors" :key="i">{{ e }}</li>
            </ul>
          </template>
        </a-alert>
      </div>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  CheckCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
  MinusCircleOutlined,
  RocketOutlined
} from '@ant-design/icons-vue'
import { applicationApi } from '@/services/application'
import { deployCheckApi, type DeployPreCheckResponse } from '@/services/deploy'

const router = useRouter()
const checking = ref(false)

const applications = ref<any[]>([])
const checkResult = ref<DeployPreCheckResponse | null>(null)

const checkForm = reactive({
  application_id: undefined as number | undefined,
  env_name: 'dev',
  image_tag: ''
})

const fetchApplications = async () => {
  try {
    const res = await applicationApi.list({ page: 1, page_size: 1000 })
    if (res?.code === 0) {
      applications.value = res.data?.list || []
    }
  } catch (error) {
    console.error('获取应用列表失败')
  }
}

const runPreCheck = async () => {
  if (!checkForm.application_id) {
    message.error('请选择应用')
    return
  }

  checking.value = true
  try {
    const res = await deployCheckApi.preCheck({
      application_id: checkForm.application_id,
      env_name: checkForm.env_name,
      image_tag: checkForm.image_tag || undefined
    })
    if (res?.code === 0) {
      checkResult.value = res.data || null
    } else {
      message.error(res?.message || '检查失败')
    }
  } catch (error) {
    message.error('检查失败')
  } finally {
    checking.value = false
  }
}

const goToDeploy = () => {
  router.push('/deploy/requests')
}

const goToCanary = () => {
  router.push('/canary/list')
}

const getStatusColor = (status: string) => {
  const map: Record<string, string> = {
    passed: 'green',
    warning: 'orange',
    failed: 'red',
    skipped: 'default'
  }
  return map[status] || 'default'
}

onMounted(() => {
  fetchApplications()
})
</script>

<style scoped>
.deploy-check {
  padding: 16px;
}

.check-result {
  margin-top: 16px;
}
</style>
