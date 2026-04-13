<template>
  <div class="registry-page">
    <a-page-header title="镜像仓库管理" sub-title="配置和管理镜像仓库连接">
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <PlusOutlined /> 添加仓库
        </a-button>
      </template>
    </a-page-header>

    <a-card :bordered="false">
      <a-table
        :columns="columns"
        :data-source="registries"
        :loading="loading"
        :pagination="false"
        row-key="id"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'name'">
            <span>{{ record.name }}</span>
            <a-tag v-if="record.is_default" color="blue" style="margin-left: 8px">默认</a-tag>
          </template>
          <template v-else-if="column.key === 'type'">
            <a-tag :color="getTypeColor(record.type)">{{ getTypeLabel(record.type) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'url'">
            <a-typography-text copyable>{{ record.url }}</a-typography-text>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatDate(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="handleTestConnection(record)">
                测试连接
              </a-button>
              <a-button type="link" size="small" @click="handleBrowseImages(record)">
                浏览镜像
              </a-button>
              <a-button type="link" size="small" @click="showEditModal(record)">编辑</a-button>
              <a-popconfirm
                title="确定要删除此仓库吗？"
                @confirm="handleDelete(record.id)"
              >
                <a-button type="link" size="small" danger>删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 创建/编辑弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      :title="isEdit ? '编辑仓库' : '添加仓库'"
      :confirm-loading="submitting"
      @ok="handleSubmit"
      width="600px"
    >
      <a-form
        ref="formRef"
        :model="form"
        :rules="rules"
        :label-col="{ span: 5 }"
        :wrapper-col="{ span: 18 }"
      >
        <a-form-item label="仓库名称" name="name">
          <a-input v-model:value="form.name" placeholder="请输入仓库名称" />
        </a-form-item>
        <a-form-item label="仓库类型" name="type">
          <a-select v-model:value="form.type" placeholder="请选择仓库类型">
            <a-select-option value="harbor">Harbor</a-select-option>
            <a-select-option value="dockerhub">Docker Hub</a-select-option>
            <a-select-option value="acr">阿里云 ACR</a-select-option>
            <a-select-option value="gcr">Google GCR</a-select-option>
            <a-select-option value="ecr">AWS ECR</a-select-option>
            <a-select-option value="custom">自定义</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="仓库地址" name="url">
          <a-input v-model:value="form.url" :placeholder="getUrlPlaceholder(form.type)" />
        </a-form-item>
        <a-form-item label="用户名" name="username">
          <a-input v-model:value="form.username" placeholder="请输入用户名（可选）" />
        </a-form-item>
        <a-form-item label="密码" name="password">
          <a-input-password v-model:value="form.password" :placeholder="isEdit ? '留空则不修改' : '请输入密码（可选）'" />
        </a-form-item>
        <a-form-item label="设为默认" name="is_default">
          <a-switch v-model:checked="form.is_default" />
          <span class="form-hint" style="margin-left: 8px">扫描镜像时默认使用此仓库</span>
        </a-form-item>
      </a-form>

      <template #footer>
        <a-button @click="modalVisible = false">取消</a-button>
        <a-button type="default" :loading="testing" @click="handleTestBeforeSave">
          测试连接
        </a-button>
        <a-button type="primary" :loading="submitting" @click="handleSubmit">
          确定
        </a-button>
      </template>
    </a-modal>

    <!-- 镜像浏览抽屉 -->
    <a-drawer
      v-model:open="imageDrawerVisible"
      :title="`镜像列表 - ${currentRegistry?.name || ''}`"
      width="600"
    >
      <a-spin :spinning="loadingImages">
        <a-list
          :data-source="images"
          :locale="{ emptyText: '暂无镜像' }"
        >
          <template #renderItem="{ item }">
            <a-list-item>
              <a-list-item-meta :title="item.name">
                <template #description>
                  <a-space wrap>
                    <a-tag v-for="tag in item.tags.slice(0, 5)" :key="tag">{{ tag }}</a-tag>
                    <span v-if="item.tags.length > 5" class="more-tags">+{{ item.tags.length - 5 }} 更多</span>
                  </a-space>
                </template>
              </a-list-item-meta>
              <template #actions>
                <a-button type="link" size="small" @click="handleScanImage(item)">
                  扫描
                </a-button>
              </template>
            </a-list-item>
          </template>
        </a-list>
      </a-spin>
    </a-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { useRouter } from 'vue-router'
import {
  getRegistries,
  createRegistry,
  updateRegistry,
  deleteRegistry,
  testRegistryConnection,
  getRegistryImages
} from '@/services/security'
import dayjs from 'dayjs'

interface Registry {
  id: number
  name: string
  type: string
  url: string
  username: string
  is_default: boolean
  created_at: string
}

interface RegistryImage {
  name: string
  tags: string[]
}

const router = useRouter()
const loading = ref(false)
const submitting = ref(false)
const testing = ref(false)
const modalVisible = ref(false)
const isEdit = ref(false)
const registries = ref<Registry[]>([])
const formRef = ref()

const imageDrawerVisible = ref(false)
const loadingImages = ref(false)
const currentRegistry = ref<Registry | null>(null)
const images = ref<RegistryImage[]>([])

const form = reactive({
  id: 0,
  name: '',
  type: 'harbor',
  url: '',
  username: '',
  password: '',
  is_default: false
})

const rules = {
  name: [{ required: true, message: '请输入仓库名称' }],
  type: [{ required: true, message: '请选择仓库类型' }],
  url: [{ required: true, message: '请输入仓库地址' }]
}

const columns = [
  { title: '名称', key: 'name' },
  { title: '类型', dataIndex: 'type', key: 'type', width: 120 },
  { title: '地址', dataIndex: 'url', key: 'url', ellipsis: true },
  { title: '用户名', dataIndex: 'username', key: 'username', width: 120 },
  { title: '创建时间', dataIndex: 'created_at', key: 'created_at', width: 180 },
  { title: '操作', key: 'action', width: 280 }
]

const getTypeColor = (type: string) => {
  const colors: Record<string, string> = {
    harbor: 'blue',
    dockerhub: 'cyan',
    acr: 'orange',
    gcr: 'green',
    ecr: 'purple',
    custom: 'default'
  }
  return colors[type] || 'default'
}

const getTypeLabel = (type: string) => {
  const labels: Record<string, string> = {
    harbor: 'Harbor',
    dockerhub: 'Docker Hub',
    acr: '阿里云 ACR',
    gcr: 'Google GCR',
    ecr: 'AWS ECR',
    custom: '自定义'
  }
  return labels[type] || type
}

const getUrlPlaceholder = (type: string) => {
  const placeholders: Record<string, string> = {
    harbor: 'https://harbor.example.com',
    dockerhub: 'https://hub.docker.com',
    acr: 'https://registry.cn-hangzhou.aliyuncs.com',
    gcr: 'https://gcr.io',
    ecr: 'https://123456789.dkr.ecr.us-east-1.amazonaws.com',
    custom: 'https://registry.example.com'
  }
  return placeholders[type] || 'https://registry.example.com'
}

const formatDate = (date: string) => {
  return dayjs(date).format('YYYY-MM-DD HH:mm:ss')
}

const loadRegistries = async () => {
  loading.value = true
  try {
    const res = await getRegistries()
    registries.value = res?.data || []
  } catch (error) {
    console.error('加载仓库失败:', error)
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  form.id = 0
  form.name = ''
  form.type = 'harbor'
  form.url = ''
  form.username = ''
  form.password = ''
  form.is_default = false
}

const showCreateModal = () => {
  resetForm()
  isEdit.value = false
  modalVisible.value = true
}

const showEditModal = (record: Registry) => {
  resetForm()
  form.id = record.id
  form.name = record.name
  form.type = record.type
  form.url = record.url
  form.username = record.username
  form.is_default = record.is_default
  isEdit.value = true
  modalVisible.value = true
}

const handleTestConnection = async (record: Registry) => {
  const hide = message.loading('正在测试连接...', 0)
  try {
    await testRegistryConnection({
      type: record.type,
      url: record.url,
      username: record.username,
      password: '' // 使用已保存的密码
    })
    message.success('连接成功')
  } catch (error: any) {
    message.error(error?.message || '连接失败')
  } finally {
    hide()
  }
}

const handleTestBeforeSave = async () => {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  testing.value = true
  try {
    await testRegistryConnection({
      type: form.type,
      url: form.url,
      username: form.username,
      password: form.password
    })
    message.success('连接成功')
  } catch (error: any) {
    message.error(error?.message || '连接失败')
  } finally {
    testing.value = false
  }
}

const handleSubmit = async () => {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    const data = {
      name: form.name,
      type: form.type,
      url: form.url,
      username: form.username,
      password: form.password || undefined,
      is_default: form.is_default
    }

    if (isEdit.value) {
      await updateRegistry(form.id, data)
      message.success('更新成功')
    } else {
      await createRegistry(data)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadRegistries()
  } catch (error: any) {
    message.error(error?.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: number) => {
  try {
    await deleteRegistry(id)
    message.success('删除成功')
    loadRegistries()
  } catch (error: any) {
    message.error(error?.message || '删除失败')
  }
}

const handleBrowseImages = async (record: Registry) => {
  currentRegistry.value = record
  imageDrawerVisible.value = true
  loadingImages.value = true
  
  try {
    const res = await getRegistryImages(record.id)
    images.value = res?.data || []
  } catch (error: any) {
    message.error(error?.message || '获取镜像列表失败')
    images.value = []
  } finally {
    loadingImages.value = false
  }
}

const handleScanImage = (image: RegistryImage) => {
  // 跳转到镜像扫描页面
  const fullImage = `${currentRegistry.value?.url}/${image.name}:${image.tags[0] || 'latest'}`
  router.push({
    path: '/security/image-scan',
    query: { image: fullImage, registry_id: currentRegistry.value?.id }
  })
}

onMounted(() => {
  loadRegistries()
})
</script>

<style scoped>
.registry-page {
  padding: 0;
}

.form-hint {
  font-size: 12px;
  color: #999;
}

.more-tags {
  color: #999;
  font-size: 12px;
}
</style>
