<template>
  <div class="knowledge-manage">
    <a-card title="AI 知识库管理">
      <template #extra>
        <a-button type="primary" @click="showCreateModal">
          <template #icon><PlusOutlined /></template>
          添加知识
        </a-button>
      </template>

      <!-- 搜索和筛选 -->
      <div class="filter-bar">
        <a-input-search
          v-model:value="searchKeyword"
          placeholder="搜索知识标题或内容"
          style="width: 300px"
          @search="handleSearch"
        />
        <a-select
          v-model:value="filterCategory"
          placeholder="选择分类"
          style="width: 150px"
          allowClear
          @change="handleSearch"
        >
          <a-select-option v-for="cat in categories" :key="cat.value" :value="cat.value">
            {{ cat.label }}
          </a-select-option>
        </a-select>
      </div>

      <!-- 知识列表 -->
      <a-table
        :columns="columns"
        :data-source="knowledgeList"
        :loading="loading"
        :pagination="pagination"
        row-key="id"
        @change="handleTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'title'">
            <a @click="showDetail(record)">{{ record.title }}</a>
            <a-tag v-if="record.is_system" color="blue" style="margin-left: 8px">系统</a-tag>
          </template>
          <template v-else-if="column.key === 'category'">
            <a-tag>{{ getCategoryLabel(record.category) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'tags'">
            <a-tag v-for="tag in record.tags" :key="tag" style="margin: 2px">{{ tag }}</a-tag>
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button type="link" size="small" @click="showEditModal(record)">编辑</a-button>
              <a-popconfirm
                title="确定删除此知识条目？"
                @confirm="handleDelete(record.id)"
              >
                <a-button type="link" size="small" danger :disabled="record.is_system">删除</a-button>
              </a-popconfirm>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>

    <!-- 创建/编辑弹窗 -->
    <a-modal
      v-model:open="modalVisible"
      :title="isEdit ? '编辑知识' : '添加知识'"
      width="800px"
      @ok="handleSubmit"
      :confirmLoading="submitting"
    >
      <a-form :model="formData" :label-col="{ span: 4 }" :wrapper-col="{ span: 20 }">
        <a-form-item label="标题" required>
          <a-input v-model:value="formData.title" placeholder="请输入知识标题" />
        </a-form-item>
        <a-form-item label="分类" required>
          <a-select v-model:value="formData.category" placeholder="请选择分类">
            <a-select-option v-for="cat in categories" :key="cat.value" :value="cat.value">
              {{ cat.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="标签">
          <a-select
            v-model:value="formData.tags"
            mode="tags"
            placeholder="输入标签后回车"
          />
        </a-form-item>
        <a-form-item label="内容" required>
          <a-textarea
            v-model:value="formData.content"
            placeholder="请输入知识内容（支持 Markdown 格式）"
            :rows="12"
          />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 详情弹窗 -->
    <a-modal
      v-model:open="detailVisible"
      :title="detailData?.title"
      width="800px"
      :footer="null"
    >
      <div class="knowledge-detail">
        <div class="detail-meta">
          <a-tag>{{ getCategoryLabel(detailData?.category) }}</a-tag>
          <span>浏览次数: {{ detailData?.view_count }}</span>
          <span>更新时间: {{ formatDate(detailData?.updated_at) }}</span>
        </div>
        <div class="detail-tags" v-if="detailData?.tags?.length">
          <a-tag v-for="tag in detailData.tags" :key="tag">{{ tag }}</a-tag>
        </div>
        <a-divider />
        <div class="detail-content" v-html="renderedContent"></div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { PlusOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { marked } from 'marked'
import { knowledgeApi } from '../../services/ai'
import type { AIKnowledge } from '../../types/ai'

const loading = ref(false)
const submitting = ref(false)
const knowledgeList = ref<AIKnowledge[]>([])
const categories = ref<{ value: string; label: string }[]>([])
const searchKeyword = ref('')
const filterCategory = ref<string | undefined>()
const modalVisible = ref(false)
const detailVisible = ref(false)
const isEdit = ref(false)
const editId = ref<number | null>(null)
const detailData = ref<AIKnowledge | null>(null)

const pagination = reactive({
  current: 1,
  pageSize: 10,
  total: 0,
})

const formData = reactive({
  title: '',
  content: '',
  category: '',
  tags: [] as string[],
})

const columns = [
  { title: '标题', dataIndex: 'title', key: 'title' },
  { title: '分类', dataIndex: 'category', key: 'category', width: 120 },
  { title: '标签', dataIndex: 'tags', key: 'tags', width: 200 },
  { title: '浏览次数', dataIndex: 'view_count', key: 'view_count', width: 100 },
  { title: '更新时间', dataIndex: 'updated_at', key: 'updated_at', width: 180 },
  { title: '操作', key: 'action', width: 150 },
]

const renderedContent = computed(() => {
  if (!detailData.value?.content) return ''
  try {
    return marked.parse(detailData.value.content, { breaks: true })
  } catch {
    return detailData.value.content
  }
})

const getCategoryLabel = (value?: string) => {
  const cat = categories.value.find(c => c.value === value)
  return cat?.label || value || '-'
}

const formatDate = (date?: string) => {
  if (!date) return '-'
  return new Date(date).toLocaleString('zh-CN')
}

const loadCategories = async () => {
  try {
    const res = await knowledgeApi.getCategories()
    if (res.data) {
      categories.value = res.data
    }
  } catch (e) {
    console.error('Load categories error:', e)
  }
}

const loadData = async () => {
  loading.value = true
  try {
    const res = await knowledgeApi.list({
      page: pagination.current,
      page_size: pagination.pageSize,
      category: filterCategory.value,
      keyword: searchKeyword.value,
    })
    if (res.data) {
      knowledgeList.value = res.data.list || []
      pagination.total = res.data.total
    }
  } catch (e) {
    console.error('Load data error:', e)
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.current = 1
  loadData()
}

const handleTableChange = (pag: any) => {
  pagination.current = pag.current
  pagination.pageSize = pag.pageSize
  loadData()
}

const showCreateModal = () => {
  isEdit.value = false
  editId.value = null
  Object.assign(formData, { title: '', content: '', category: '', tags: [] })
  modalVisible.value = true
}

const showEditModal = (record: AIKnowledge) => {
  isEdit.value = true
  editId.value = record.id
  Object.assign(formData, {
    title: record.title,
    content: record.content,
    category: record.category,
    tags: record.tags || [],
  })
  modalVisible.value = true
}

const showDetail = (record: AIKnowledge) => {
  detailData.value = record
  detailVisible.value = true
}

const handleSubmit = async () => {
  if (!formData.title || !formData.content || !formData.category) {
    message.warning('请填写完整信息')
    return
  }

  submitting.value = true
  try {
    if (isEdit.value && editId.value) {
      await knowledgeApi.update(editId.value, formData)
      message.success('更新成功')
    } else {
      await knowledgeApi.create(formData)
      message.success('创建成功')
    }
    modalVisible.value = false
    loadData()
  } catch (e: any) {
    message.error(e.message || '操作失败')
  } finally {
    submitting.value = false
  }
}

const handleDelete = async (id: number) => {
  try {
    await knowledgeApi.delete(id)
    message.success('删除成功')
    loadData()
  } catch (e: any) {
    message.error(e.message || '删除失败')
  }
}

onMounted(() => {
  loadCategories()
  loadData()
})
</script>

<style scoped>
.filter-bar {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.knowledge-detail {
  padding: 16px 0;
}

.detail-meta {
  display: flex;
  align-items: center;
  gap: 16px;
  color: #999;
  font-size: 13px;
}

.detail-tags {
  margin-top: 12px;
}

.detail-content {
  line-height: 1.8;
}

.detail-content :deep(pre) {
  background: #f5f5f5;
  padding: 12px;
  border-radius: 4px;
  overflow-x: auto;
}

.detail-content :deep(code) {
  font-family: 'Fira Code', monospace;
}
</style>
