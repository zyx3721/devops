<template>
  <div class="log-bookmarks">
    <a-card>
      <template #title>
        <div class="card-header">
          <span>我的书签</span>
          <a-tag>{{ total }} 个</a-tag>
        </div>
      </template>

      <a-table :dataSource="bookmarks" :columns="columns" :loading="loading" :pagination="false" rowKey="id">
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'content'">
            <div class="log-content">{{ record.content }}</div>
          </template>
          <template v-else-if="column.key === 'note'">
            <a-input 
              v-if="editingId === record.id"
              v-model:value="editingNote"
              size="small"
              @blur="saveNote(record)"
              @pressEnter="saveNote(record)"
            />
            <span v-else @dblclick="startEdit(record)" class="note-text">
              {{ record.note || '双击添加备注' }}
            </span>
          </template>
          <template v-else-if="column.key === 'created_at'">
            {{ formatTime(record.created_at) }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-button type="link" size="small" @click="viewLog(record)">查看</a-button>
            <a-button type="link" size="small" style="color: #52c41a" @click="shareBookmark(record)">分享</a-button>
            <a-button type="link" danger size="small" @click="deleteBookmark(record)">删除</a-button>
          </template>
        </template>
      </a-table>

      <a-pagination
        v-if="total > 0"
        v-model:current="currentPage"
        :total="total"
        :pageSize="20"
        show-quick-jumper
        @change="loadBookmarks"
        style="margin-top: 15px; text-align: right"
      />
    </a-card>

    <!-- 分享对话框 -->
    <a-modal v-model:open="showShareDialog" title="分享书签" width="400px" :footer="null">
      <a-form :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
        <a-form-item label="有效期">
          <a-radio-group v-model:value="shareExpires">
            <a-radio :value="1">1 天</a-radio>
            <a-radio :value="7">7 天</a-radio>
            <a-radio :value="30">30 天</a-radio>
            <a-radio :value="0">永久</a-radio>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="分享链接" v-if="shareURL">
          <a-input v-model:value="fullShareURL" readonly>
            <template #addonAfter>
              <a-button size="small" type="link" @click="copyShareURL">复制</a-button>
            </template>
          </a-input>
        </a-form-item>
      </a-form>
      <div class="dialog-footer" style="text-align: right; margin-top: 20px">
        <a-button @click="showShareDialog = false" style="margin-right: 8px">关闭</a-button>
        <a-button type="primary" @click="generateShareURL" :loading="sharing" v-if="!shareURL">生成链接</a-button>
      </div>
    </a-modal>

    <!-- 日志上下文 -->
    <LogContext
      v-model:open="showContext"
      :cluster-id="contextBookmark?.cluster_id || 0"
      :namespace="contextBookmark?.namespace || ''"
      :pod-name="contextBookmark?.pod_name || ''"
      :log="contextLog"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { message, Modal } from 'ant-design-vue'
import { logApi } from '@/services/logs'
import LogContext from './components/LogContext.vue'

interface Bookmark {
  id: number
  user_id: number
  cluster_id: number
  namespace: string
  pod_name: string
  container: string
  log_timestamp: string
  content: string
  note: string
  share_url: string
  share_expires_at: string
  created_at: string
}

const columns = [
  {
    title: '日志内容',
    key: 'content',
    width: 400,
  },
  {
    title: 'Pod',
    dataIndex: 'pod_name',
    width: 200,
    ellipsis: true,
  },
  {
    title: '命名空间',
    dataIndex: 'namespace',
    width: 150,
  },
  {
    title: '备注',
    key: 'note',
    width: 200,
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 180,
  },
  {
    title: '操作',
    key: 'action',
    width: 180,
    fixed: 'right',
  },
]

const bookmarks = ref<Bookmark[]>([])
const total = ref(0)
const currentPage = ref(1)
const loading = ref(false)

const editingId = ref<number | null>(null)
const editingNote = ref('')

const showShareDialog = ref(false)
const sharingBookmark = ref<Bookmark | null>(null)
const shareExpires = ref(7)
const shareURL = ref('')
const sharing = ref(false)

const showContext = ref(false)
const contextBookmark = ref<Bookmark | null>(null)
const contextLog = ref<any>(null)

const fullShareURL = computed(() => {
  if (!shareURL.value) return ''
  return `${window.location.origin}/logs/shared/${shareURL.value}`
})

const loadBookmarks = async () => {
  loading.value = true
  try {
    const res = await logApi.getBookmarks(currentPage.value)
    bookmarks.value = res.data?.items || []
    total.value = res.data?.total || 0
  } catch (error) {
    message.error('加载书签失败')
  } finally {
    loading.value = false
  }
}

const startEdit = (bookmark: Bookmark) => {
  editingId.value = bookmark.id
  editingNote.value = bookmark.note || ''
}

const saveNote = async (bookmark: Bookmark) => {
  if (editingNote.value !== bookmark.note) {
    try {
      await logApi.updateBookmark(bookmark.id, { note: editingNote.value })
      bookmark.note = editingNote.value
      message.success('备注已保存')
    } catch (error) {
      message.error('保存失败')
    }
  }
  editingId.value = null
}

const viewLog = (bookmark: Bookmark) => {
  contextBookmark.value = bookmark
  contextLog.value = {
    timestamp: bookmark.log_timestamp,
    content: bookmark.content,
    pod_name: bookmark.pod_name
  }
  showContext.value = true
}

const shareBookmark = (bookmark: Bookmark) => {
  sharingBookmark.value = bookmark
  shareURL.value = bookmark.share_url || ''
  shareExpires.value = 7
  showShareDialog.value = true
}

const generateShareURL = async () => {
  if (!sharingBookmark.value) return

  sharing.value = true
  try {
    const res = await logApi.shareBookmark(sharingBookmark.value.id, shareExpires.value)
    shareURL.value = res.data?.share_url || ''
    sharingBookmark.value.share_url = shareURL.value
    message.success('分享链接已生成')
  } catch (error) {
    message.error('生成分享链接失败')
  } finally {
    sharing.value = false
  }
}

const copyShareURL = () => {
  navigator.clipboard.writeText(fullShareURL.value)
  message.success('链接已复制')
}

const deleteBookmark = async (bookmark: Bookmark) => {
  try {
    await Modal.confirm({
      title: '提示',
      content: '确定删除此书签？',
      onOk: async () => {
        await logApi.deleteBookmark(bookmark.id)
        message.success('删除成功')
        loadBookmarks()
      }
    })
  } catch (error) {
    // Cancelled
  }
}

const formatTime = (ts: string) => {
  if (!ts) return ''
  try {
    return new Date(ts).toLocaleString('zh-CN')
  } catch {
    return ts
  }
}

onMounted(() => {
  loadBookmarks()
})
</script>

<style scoped>
.log-bookmarks {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.log-content {
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  color: rgba(0, 0, 0, 0.85);
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 60px;
  overflow: hidden;
}

.note-text {
  color: rgba(0, 0, 0, 0.45);
  cursor: pointer;
}

.note-text:hover {
  color: #1890ff;
}
</style>
