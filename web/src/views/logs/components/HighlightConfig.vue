<template>
  <div class="highlight-config">
    <div class="section">
      <div class="section-header">
        <span>预设规则</span>
      </div>
      <div class="rule-list">
        <div v-for="rule in presetRules" :key="rule.id" class="rule-item">
          <a-switch v-model:checked="rule.enabled" size="small" @change="toggleRule(rule)" />
          <div class="rule-preview" :style="{ color: rule.fg_color, backgroundColor: rule.bg_color }">
            {{ rule.name }}
          </div>
          <span class="rule-type">{{ rule.match_type }}</span>
        </div>
      </div>
    </div>

    <a-divider />

    <div class="section">
      <div class="section-header">
        <span>自定义规则</span>
        <a-button type="primary" size="small" @click="showAddDialog = true">
          <template #icon><PlusOutlined /></template>
          添加
        </a-button>
      </div>
      <div class="rule-list">
        <div v-for="rule in customRules" :key="rule.id" class="rule-item">
          <a-switch v-model:checked="rule.enabled" size="small" @change="toggleRule(rule)" />
          <div class="rule-preview" :style="{ color: rule.fg_color, backgroundColor: rule.bg_color }">
            {{ rule.name }}
          </div>
          <span class="rule-type">{{ rule.match_type }}</span>
          <div class="rule-actions">
            <a-button type="link" size="small" @click="editRule(rule)">
              <template #icon><EditOutlined /></template>
            </a-button>
            <a-button type="link" size="small" danger @click="deleteRule(rule)">
              <template #icon><DeleteOutlined /></template>
            </a-button>
          </div>
        </div>
        <a-empty v-if="customRules.length === 0" description="暂无自定义规则" :image-size="60" />
      </div>
    </div>

    <!-- 添加/编辑对话框 -->
    <a-modal v-model:open="showAddDialog" :title="editingRule ? '编辑规则' : '添加规则'" width="450px" @ok="saveRule">
      <a-form :model="ruleForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
        <a-form-item label="规则名称" required>
          <a-input v-model:value="ruleForm.name" placeholder="输入规则名称" />
        </a-form-item>
        <a-form-item label="匹配类型" required>
          <a-radio-group v-model:value="ruleForm.match_type">
            <a-radio value="keyword">关键词</a-radio>
            <a-radio value="regex">正则</a-radio>
            <a-radio value="level">级别</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="匹配值" required>
          <a-input v-if="ruleForm.match_type !== 'level'" v-model:value="ruleForm.match_value" placeholder="输入匹配值" />
          <a-select v-else v-model:value="ruleForm.match_value" style="width: 100%">
            <a-select-option value="FATAL">FATAL</a-select-option>
            <a-select-option value="ERROR">ERROR</a-select-option>
            <a-select-option value="WARN">WARN</a-select-option>
            <a-select-option value="INFO">INFO</a-select-option>
            <a-select-option value="DEBUG">DEBUG</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="前景色">
          <a-color-picker v-model:value="ruleForm.fg_color" show-text />
        </a-form-item>
        <a-form-item label="背景色">
          <a-color-picker v-model:value="ruleForm.bg_color" show-text />
        </a-form-item>
        <a-form-item label="优先级">
          <a-input-number v-model:value="ruleForm.priority" :min="0" :max="100" />
          <span style="margin-left: 10px; color: rgba(0, 0, 0, 0.45)">数值越小优先级越高</span>
        </a-form-item>
        <a-form-item label="预览">
          <div class="rule-preview-large" :style="{ color: typeof ruleForm.fg_color === 'string' ? ruleForm.fg_color : ruleForm.fg_color?.toHexString(), backgroundColor: typeof ruleForm.bg_color === 'string' ? ruleForm.bg_color : ruleForm.bg_color?.toHexString() }">
            示例日志内容 - {{ ruleForm.match_value || '匹配值' }}
          </div>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons-vue'
import { message, Modal } from 'ant-design-vue'
import { logApi, type HighlightRule } from '@/services/logs'

const props = defineProps<{
  rules: HighlightRule[]
}>()

const emit = defineEmits<{
  (e: 'update:rules', rules: HighlightRule[]): void
}>()

const allRules = ref<HighlightRule[]>([])
const showAddDialog = ref(false)
const editingRule = ref<HighlightRule | null>(null)

const ruleForm = ref({
  name: '',
  match_type: 'keyword' as 'keyword' | 'regex' | 'level',
  match_value: '',
  fg_color: '#FFFFFF',
  bg_color: '#DC3545',
  priority: 50,
  enabled: true
})

const presetRules = computed(() => allRules.value.filter(r => r.is_preset))
const customRules = computed(() => allRules.value.filter(r => !r.is_preset))

const loadRules = async () => {
  try {
    const res = await logApi.getHighlightRules()
    allRules.value = res.data || []
    emit('update:rules', allRules.value)
  } catch (error) {
    console.error('加载染色规则失败', error)
  }
}

const toggleRule = async (rule: HighlightRule) => {
  try {
    await logApi.toggleHighlightRule(rule.id)
    emit('update:rules', allRules.value)
  } catch (error: any) {
    message.error(error.message || '操作失败')
    rule.enabled = !rule.enabled
  }
}

const editRule = (rule: HighlightRule) => {
  editingRule.value = rule
  ruleForm.value = {
    name: rule.name,
    match_type: rule.match_type,
    match_value: rule.match_value,
    fg_color: rule.fg_color,
    bg_color: rule.bg_color,
    priority: rule.priority,
    enabled: rule.enabled
  }
  showAddDialog.value = true
}

const deleteRule = async (rule: HighlightRule) => {
  try {
    await Modal.confirm({
      title: '确认删除',
      content: '确定要删除这条规则吗？',
      okType: 'danger',
      onOk: async () => {
        await logApi.deleteHighlightRule(rule.id)
        await loadRules()
        message.success('删除成功')
      }
    })
  } catch (error: any) {
    // Cancelled
  }
}

const saveRule = async () => {
  if (!ruleForm.value.name || !ruleForm.value.match_value) {
    message.warning('请填写完整信息')
    return
  }

  // Handle color object from a-color-picker if needed
  // But a-color-picker usually emits string if valueFormat is not set, or object.
  // We should ensure it is string before sending.
  const formToSave = {
    ...ruleForm.value,
    fg_color: typeof ruleForm.value.fg_color === 'string' ? ruleForm.value.fg_color : (ruleForm.value.fg_color as any)?.toHexString(),
    bg_color: typeof ruleForm.value.bg_color === 'string' ? ruleForm.value.bg_color : (ruleForm.value.bg_color as any)?.toHexString()
  }

  try {
    if (editingRule.value) {
      await logApi.updateHighlightRule(editingRule.value.id, formToSave)
      message.success('更新成功')
    } else {
      await logApi.createHighlightRule(formToSave)
      message.success('创建成功')
    }
    showAddDialog.value = false
    editingRule.value = null
    resetForm()
    await loadRules()
  } catch (error: any) {
    message.error(error.message || '保存失败')
  }
}

const resetForm = () => {
  ruleForm.value = {
    name: '',
    match_type: 'keyword',
    match_value: '',
    fg_color: '#FFFFFF',
    bg_color: '#DC3545',
    priority: 50,
    enabled: true
  }
}

onMounted(() => {
  loadRules()
})
</script>

<style scoped>
.highlight-config {
  padding: 16px;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  font-weight: 500;
  font-size: 14px;
  color: rgba(0, 0, 0, 0.85);
}

.rule-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rule-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  background: #fafafa;
  border-radius: 4px;
  border: 1px solid #f0f0f0;
  transition: all 0.3s;
}

.rule-item:hover {
  background: #f5f5f5;
  border-color: #d9d9d9;
}

.rule-preview {
  padding: 4px 12px;
  border-radius: 4px;
  font-size: 13px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}

.rule-preview-large {
  padding: 12px 16px;
  border-radius: 4px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
}

.rule-type {
  color: rgba(0, 0, 0, 0.45);
  font-size: 12px;
  padding: 2px 8px;
  background: #ffffff;
  border-radius: 2px;
}

.rule-actions {
  margin-left: auto;
  display: flex;
  gap: 4px;
}
</style>
