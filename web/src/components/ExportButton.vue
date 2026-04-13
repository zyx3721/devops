<template>
  <a-dropdown>
    <a-button>
      <template #icon><ExportOutlined /></template>
      导出
    </a-button>
    <template #overlay>
      <a-menu @click="handleExport">
        <a-menu-item key="csv">
          <FileTextOutlined /> 导出 CSV
        </a-menu-item>
        <a-menu-item key="excel">
          <FileExcelOutlined /> 导出 Excel
        </a-menu-item>
        <a-menu-item key="json">
          <FileOutlined /> 导出 JSON
        </a-menu-item>
      </a-menu>
    </template>
  </a-dropdown>
</template>

<script setup lang="ts">
import { ExportOutlined, FileTextOutlined, FileExcelOutlined, FileOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { exportToCSV, exportToJSON, exportToExcel } from '@/utils/export'

const props = defineProps<{
  data: any[]
  columns: { title: string; dataIndex: string }[]
  filename?: string
}>()

const handleExport = ({ key }: { key: string }) => {
  if (!props.data || props.data.length === 0) {
    message.warning('没有数据可导出')
    return
  }

  const filename = props.filename || `export_${new Date().toISOString().slice(0, 10)}`

  switch (key) {
    case 'csv':
      exportToCSV(props.data, props.columns, filename)
      message.success('CSV 导出成功')
      break
    case 'excel':
      exportToExcel(props.data, props.columns, filename)
      message.success('Excel 导出成功')
      break
    case 'json':
      exportToJSON(props.data, filename)
      message.success('JSON 导出成功')
      break
  }
}
</script>
