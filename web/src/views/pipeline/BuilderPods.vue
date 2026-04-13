<template>
  <div class="builder-pods">
    <a-page-header title="构建 Pod 管理" sub-title="管理 K8s 集群中的持久化构建 Pod">
      <template #extra>
        <a-space>
          <a-button @click="loadPods">
            <ReloadOutlined /> 刷新
          </a-button>
          <a-button type="primary" @click="showConfigModal = true">
            <SettingOutlined /> 配置
          </a-button>
        </a-space>
      </template>
    </a-page-header>

    <a-card :bordered="false">
      <a-alert
        message="构建 Pod 说明"
        description="构建 Pod 是持久化运行的容器，用于执行 CI/CD 构建任务。Pod 在空闲超时后会自动销毁，也可以手动删除。不同镜像的步骤通过共享 PVC 传递文件。"
        type="info"
        show-icon
        style="margin-bottom: 16px"
      />

      <a-table
        :columns="columns"
        :data-source="pods"
        :loading="loading"
        :pagination="false"
        row-key="pod_name"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="record.status === 'Running' ? 'green' : 'orange'">{{ record.status }}</a-tag>
          </template>
          <template v-else-if="column.key === 'idle'">
            <span :class="{ 'text-warning': record.idle_seconds > 1200 }">
              {{ formatIdleTime(record.idle_seconds) }}
            </span>
          </template>
          <template v-else-if="column.key === 'last_used_at'">
            {{ formatTime(record.last_used_at) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-popconfirm
              title="确定要删除这个构建 Pod 吗？"
              @confirm="deletePod(record)"
            >
              <a-button type="link" danger size="small">删除</a-button>
            </a-popconfirm>
          </template>
        </template>
      </a-table>

      <a-empty v-if="!loading && pods.length === 0" description="暂无活跃的构建 Pod" />
    </a-card>

    <!-- 配置弹窗 -->
    <a-modal
      v-model:open="showConfigModal"
      title="构建 Pod 配置"
      @ok="saveConfig"
      :confirm-loading="savingConfig"
      width="600px"
    >
      <a-form :label-col="{ span: 8 }" :wrapper-col="{ span: 14 }">
        <a-form-item label="空闲超时">
          <a-input-number
            v-model:value="configForm.idle_timeout_minutes"
            :min="1"
            :max="1440"
            addon-after="分钟"
            style="width: 150px"
          />
          <div class="form-help">Pod 空闲超过此时间后自动销毁</div>
        </a-form-item>

        <a-divider orientation="left">存储配置</a-divider>

        <a-form-item label="存储类型">
          <a-radio-group v-model:value="configForm.storage_type">
            <a-radio value="hostpath">HostPath (单节点)</a-radio>
            <a-radio value="pvc">共享 PVC</a-radio>
            <a-radio value="emptydir">EmptyDir</a-radio>
          </a-radio-group>
          <div class="form-help">
            <span v-if="configForm.storage_type === 'hostpath'">HostPath: 所有 Pod 共享宿主机目录，适合单节点测试</span>
            <span v-else-if="configForm.storage_type === 'pvc'">PVC: 需要支持 ReadWriteMany 的存储类（如 NFS）</span>
            <span v-else>EmptyDir: 每个 Pod 独立存储，不支持跨镜像共享</span>
          </div>
        </a-form-item>

        <template v-if="configForm.storage_type === 'hostpath'">
          <a-form-item label="Host Path">
            <a-input
              v-model:value="configForm.host_path"
              placeholder="/tmp/devops-workspace"
              style="width: 300px"
            />
            <div class="form-help">宿主机上的目录路径，所有 Pod 共享此目录</div>
          </a-form-item>
        </template>

        <template v-if="configForm.storage_type === 'pvc'">
          <a-form-item label="PVC 名称">
            <a-input
              v-model:value="configForm.pvc_name"
              placeholder="devops-workspace-shared"
              style="width: 250px"
            />
          </a-form-item>

          <a-form-item label="存储大小">
            <a-input-number
              v-model:value="configForm.pvc_size_gi"
              :min="1"
              :max="1000"
              addon-after="Gi"
              style="width: 150px"
            />
          </a-form-item>

          <a-form-item label="StorageClass">
            <a-input
              v-model:value="configForm.storage_class"
              placeholder="留空使用默认"
              style="width: 250px"
            />
            <div class="form-help">需要支持 ReadWriteMany 的存储类（如 NFS、CephFS）</div>
          </a-form-item>

          <a-form-item label="访问模式">
            <a-select v-model:value="configForm.access_mode" style="width: 200px">
              <a-select-option value="ReadWriteMany">ReadWriteMany</a-select-option>
              <a-select-option value="ReadWriteOnce">ReadWriteOnce</a-select-option>
            </a-select>
          </a-form-item>
        </template>

        <a-divider orientation="left">资源配置</a-divider>

        <a-form-item label="CPU 请求">
          <a-input v-model:value="configForm.cpu_request" placeholder="100m" style="width: 150px" />
        </a-form-item>

        <a-form-item label="CPU 限制">
          <a-input v-model:value="configForm.cpu_limit" placeholder="2" style="width: 150px" />
        </a-form-item>

        <a-form-item label="内存请求">
          <a-input v-model:value="configForm.memory_request" placeholder="256Mi" style="width: 150px" />
        </a-form-item>

        <a-form-item label="内存限制">
          <a-input v-model:value="configForm.memory_limit" placeholder="4Gi" style="width: 150px" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { ReloadOutlined, SettingOutlined } from '@ant-design/icons-vue'
import request from '@/services/api'
import dayjs from 'dayjs'

interface BuilderPod {
  cluster_id: number
  namespace: string
  pod_name: string
  image: string
  last_used_at: string
  idle_seconds: number
}

const loading = ref(false)
const pods = ref<BuilderPod[]>([])
const showConfigModal = ref(false)
const savingConfig = ref(false)

const configForm = reactive({
  idle_timeout_minutes: 30,
  storage_type: 'hostpath',
  pvc_name: 'devops-workspace-shared',
  pvc_size_gi: 10,
  storage_class: '',
  access_mode: 'ReadWriteMany',
  host_path: '/tmp/devops-workspace',
  cpu_request: '100m',
  cpu_limit: '2',
  memory_request: '256Mi',
  memory_limit: '4Gi'
})

const columns = [
  { title: 'Pod Name', dataIndex: 'pod_name', key: 'pod_name' },
  { title: 'Image', dataIndex: 'image', key: 'image', ellipsis: true },
  { title: 'Namespace', dataIndex: 'namespace', key: 'namespace' },
  { title: 'Cluster', dataIndex: 'cluster_name', key: 'cluster_name' },
  { title: 'Status', key: 'status', width: 100 },
  { title: 'Idle Time', key: 'idle', width: 120 },
  { title: 'Last Used', key: 'last_used_at', width: 180 },
  { title: 'Action', key: 'action', width: 100 }
]

const formatIdleTime = (seconds: number) => {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
}

const formatTime = (time: string) => {
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

const loadPods = async () => {
  loading.value = true
  try {
    const res = await request.get('/builders/pods')
    if (res?.data?.items) {
      pods.value = res.data.items
    }
  } catch (error) {
    console.error('Load builder pods failed:', error)
  } finally {
    loading.value = false
  }
}

const loadConfig = async () => {
  try {
    const res = await request.get('/builders/config')
    console.log('loadConfig response:', res)
    if (res?.data) {
      Object.assign(configForm, res.data)
      console.log('configForm after load:', configForm)
    }
  } catch (error) {
    console.error('Load config failed:', error)
  }
}

const saveConfig = async () => {
  savingConfig.value = true
  try {
    await request.put('/builders/config', configForm)
    message.success('Config saved')
    showConfigModal.value = false
  } catch (error: any) {
    message.error(error?.message || 'Save failed')
  } finally {
    savingConfig.value = false
  }
}

const deletePod = async (pod: BuilderPod) => {
  try {
    await request.delete(`/builders/pods/${pod.pod_name}`, {
      params: {
        cluster_id: pod.cluster_id,
        namespace: pod.namespace
      }
    })
    message.success('Deleted')
    loadPods()
  } catch (error: any) {
    message.error(error?.message || 'Delete failed')
  }
}

onMounted(() => {
  loadPods()
  loadConfig()
})
</script>

<style scoped>
.builder-pods {
  padding: 0;
}

.text-warning {
  color: #faad14;
}

.form-help {
  color: #999;
  font-size: 12px;
  margin-top: 4px;
}
</style>
