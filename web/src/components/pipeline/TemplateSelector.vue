<template>
  <div class="template-selector">
    <div class="template-header">
      <h3>选择模板</h3>
      <el-input
        v-model="searchKeyword"
        placeholder="搜索模板..."
        prefix-icon="Search"
        clearable
        style="width: 240px"
      />
    </div>

    <div class="template-categories">
      <el-radio-group v-model="selectedCategory" size="small">
        <el-radio-button label="">全部</el-radio-button>
        <el-radio-button 
          v-for="cat in categories" 
          :key="cat.value" 
          :label="cat.value"
        >
          {{ cat.label }}
        </el-radio-button>
      </el-radio-group>
    </div>

    <div class="template-grid" v-loading="loading">
      <div
        v-for="template in filteredTemplates"
        :key="template.id"
        class="template-card"
        :class="{ selected: selectedTemplate?.id === template.id }"
        @click="selectTemplate(template)"
      >
        <div class="template-icon">
          <el-icon :size="32">
            <component :is="getTemplateIcon(template.category)" />
          </el-icon>
        </div>
        <div class="template-info">
          <h4>{{ template.name }}</h4>
          <p>{{ template.description }}</p>
          <div class="template-tags">
            <el-tag v-for="tag in template.tags" :key="tag" size="small" type="info">
              {{ tag }}
            </el-tag>
          </div>
        </div>
        <div class="template-actions">
          <el-button type="primary" link @click.stop="previewTemplate(template)">
            预览
          </el-button>
        </div>
      </div>

      <div v-if="filteredTemplates.length === 0" class="empty-state">
        <el-empty description="暂无匹配的模板" />
      </div>
    </div>

    <div class="template-footer">
      <el-button @click="$emit('cancel')">取消</el-button>
      <el-button type="primary" :disabled="!selectedTemplate" @click="confirmSelect">
        使用此模板
      </el-button>
    </div>

    <!-- 模板预览对话框 -->
    <el-dialog v-model="previewDialog.visible" title="模板预览" width="70%">
      <div v-if="previewDialog.template" class="template-preview">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="模板名称">
            {{ previewDialog.template.name }}
          </el-descriptions-item>
          <el-descriptions-item label="分类">
            {{ getCategoryLabel(previewDialog.template.category) }}
          </el-descriptions-item>
          <el-descriptions-item label="描述" :span="2">
            {{ previewDialog.template.description }}
          </el-descriptions-item>
        </el-descriptions>

        <h4 style="margin: 16px 0 8px">流水线配置</h4>
        <el-collapse>
          <el-collapse-item 
            v-for="(stage, index) in previewDialog.template.stages" 
            :key="index"
            :title="`阶段 ${index + 1}: ${stage.name}`"
          >
            <div class="stage-preview">
              <p v-if="stage.description">{{ stage.description }}</p>
              <div class="steps-preview">
                <div v-for="(step, stepIndex) in stage.steps" :key="stepIndex" class="step-preview">
                  <el-tag size="small">{{ step.type }}</el-tag>
                  <span>{{ step.name }}</span>
                </div>
              </div>
            </div>
          </el-collapse-item>
        </el-collapse>
      </div>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { 
  Document, 
  Box, 
  Monitor, 
  Setting, 
  Connection,
  Search
} from '@element-plus/icons-vue'
import { pipelineApi } from '@/services/pipeline'

const emit = defineEmits(['select', 'cancel'])

const loading = ref(false)
const templates = ref([])
const searchKeyword = ref('')
const selectedCategory = ref('')
const selectedTemplate = ref(null)

const previewDialog = ref({
  visible: false,
  template: null
})

const categories = [
  { value: 'build', label: '构建' },
  { value: 'deploy', label: '部署' },
  { value: 'test', label: '测试' },
  { value: 'release', label: '发布' },
  { value: 'custom', label: '自定义' }
]

const filteredTemplates = computed(() => {
  let result = templates.value

  if (selectedCategory.value) {
    result = result.filter(t => t.category === selectedCategory.value)
  }

  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    result = result.filter(t => 
      t.name.toLowerCase().includes(keyword) ||
      t.description.toLowerCase().includes(keyword) ||
      t.tags?.some(tag => tag.toLowerCase().includes(keyword))
    )
  }

  return result
})

const fetchTemplates = async () => {
  loading.value = true
  try {
    const res = await pipelineApi.listTemplates()
    templates.value = res.data || []
  } catch (error) {
    ElMessage.error('获取模板列表失败')
  } finally {
    loading.value = false
  }
}

const selectTemplate = (template) => {
  selectedTemplate.value = template
}

const previewTemplate = (template) => {
  previewDialog.value.template = template
  previewDialog.value.visible = true
}

const confirmSelect = () => {
  if (selectedTemplate.value) {
    emit('select', selectedTemplate.value)
  }
}

const getTemplateIcon = (category) => {
  const iconMap = {
    build: Box,
    deploy: Monitor,
    test: Document,
    release: Connection,
    custom: Setting
  }
  return iconMap[category] || Document
}

const getCategoryLabel = (category) => {
  const cat = categories.find(c => c.value === category)
  return cat?.label || category
}

onMounted(() => {
  fetchTemplates()
})
</script>

<style scoped>
.template-selector {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.template-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.template-header h3 {
  margin: 0;
}

.template-categories {
  margin-bottom: 16px;
}

.template-grid {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
  overflow-y: auto;
  padding: 4px;
}

.template-card {
  display: flex;
  flex-direction: column;
  padding: 16px;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}

.template-card:hover {
  border-color: #409eff;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.template-card.selected {
  border-color: #409eff;
  background: #ecf5ff;
}

.template-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  background: #f5f7fa;
  border-radius: 8px;
  margin-bottom: 12px;
  color: #409eff;
}

.template-info h4 {
  margin: 0 0 8px;
  font-size: 16px;
}

.template-info p {
  margin: 0 0 8px;
  color: #909399;
  font-size: 13px;
  line-height: 1.4;
}

.template-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.template-actions {
  margin-top: auto;
  padding-top: 12px;
  text-align: right;
}

.template-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding-top: 16px;
  border-top: 1px solid #e4e7ed;
  margin-top: 16px;
}

.empty-state {
  grid-column: 1 / -1;
}

.template-preview {
  padding: 0 20px;
}

.stage-preview {
  padding: 8px 0;
}

.steps-preview {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-top: 8px;
}

.step-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
}
</style>
