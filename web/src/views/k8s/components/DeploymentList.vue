<template>
  <div class="deployment-list">
    <el-table
      :data="filteredDeployments"
      stripe
      style="width: 100%"
      v-loading="loading"
    >
      <el-table-column prop="name" label="名称" min-width="150" />
      <el-table-column prop="namespace" label="命名空间" width="120" />
      <el-table-column prop="ready" label="就绪/副本" width="100" />
      <el-table-column prop="up_to_date" label="最新" width="80" />
      <el-table-column prop="available" label="可用" width="80" />
      <el-table-column label="镜像" min-width="200">
        <template #default="{ row }">
          <div v-for="(image, index) in row.images" :key="index" class="image-item">
            {{ image }}
          </div>
        </template>
      </el-table-column>
      <el-table-column prop="age" label="运行时间" width="100" />
      <el-table-column label="操作" width="240" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="handleViewDetail(row)">详情</el-button>
          <el-button size="small" type="warning" @click="handleRestart(row)">重启</el-button>
          <el-button size="small" type="primary" @click="handleScale(row)">扩缩容</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 扩缩容对话框 -->
    <el-dialog
      v-model="scaleDialogVisible"
      title="扩缩容"
      width="400px"
    >
      <el-form :model="scaleForm" label-width="100px">
        <el-form-item label="Deployment">
          <el-input v-model="currentDeployment.name" disabled />
        </el-form-item>
        <el-form-item label="当前副本数">
          <el-input v-model="currentDeployment.replicas" disabled />
        </el-form-item>
        <el-form-item label="目标副本数" required>
          <el-input-number
            v-model="scaleForm.replicas"
            :min="0"
            :max="100"
            style="width: 100%"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="scaleDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmScale" :loading="scaling">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'

interface DeploymentInfo {
  name: string
  namespace: string
  ready: string
  up_to_date: number
  available: number
  age: string
  images: string[]
  replicas: number
  created_at: string
}

interface Props {
  deployments: DeploymentInfo[]
  loading?: boolean
  searchKeyword?: string
}

interface Emits {
  (e: 'restart', deployment: DeploymentInfo): void
  (e: 'scale', deployment: DeploymentInfo, replicas: number): void
  (e: 'viewDetail', deployment: DeploymentInfo): void
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  searchKeyword: ''
})

const emit = defineEmits<Emits>()

const scaleDialogVisible = ref(false)
const scaling = ref(false)
const currentDeployment = ref<DeploymentInfo>({
  name: '',
  namespace: '',
  ready: '',
  up_to_date: 0,
  available: 0,
  age: '',
  images: [],
  replicas: 0,
  created_at: ''
})
const scaleForm = ref({
  replicas: 0
})

const filteredDeployments = computed(() => {
  if (!props.searchKeyword) {
    return props.deployments
  }
  const keyword = props.searchKeyword.toLowerCase()
  return props.deployments.filter(d => 
    d.name.toLowerCase().includes(keyword)
  )
})

const handleViewDetail = (deployment: DeploymentInfo) => {
  emit('viewDetail', deployment)
}

const handleRestart = async (deployment: DeploymentInfo) => {
  try {
    await ElMessageBox.confirm(
      `确定要重启 Deployment "${deployment.name}" 吗？`,
      '确认操作',
      { type: 'warning' }
    )
    emit('restart', deployment)
  } catch {
    // 用户取消
  }
}

const handleScale = (deployment: DeploymentInfo) => {
  currentDeployment.value = { ...deployment }
  scaleForm.value.replicas = deployment.replicas
  scaleDialogVisible.value = true
}

const confirmScale = () => {
  if (scaleForm.value.replicas < 0 || scaleForm.value.replicas > 100) {
    ElMessage.error('副本数必须在 0-100 之间')
    return
  }
  
  scaling.value = true
  emit('scale', currentDeployment.value, scaleForm.value.replicas)
  
  // 延迟关闭对话框，等待父组件处理完成
  setTimeout(() => {
    scaling.value = false
    scaleDialogVisible.value = false
  }, 500)
}
</script>

<style scoped>
.deployment-list {
  width: 100%;
}

.image-item {
  font-size: 12px;
  color: var(--el-text-color-regular);
  line-height: 1.5;
}
</style>
