/**
 * 数据导出工具
 */

// 导出为 CSV
export function exportToCSV(data: any[], columns: { title: string; dataIndex: string }[], filename: string) {
  if (!data || data.length === 0) {
    return
  }

  // 生成表头
  const headers = columns.map(col => col.title).join(',')
  
  // 生成数据行
  const rows = data.map(row => {
    return columns.map(col => {
      let value = row[col.dataIndex]
      if (value === null || value === undefined) {
        value = ''
      }
      // 处理包含逗号或换行的值
      if (typeof value === 'string' && (value.includes(',') || value.includes('\n') || value.includes('"'))) {
        value = `"${value.replace(/"/g, '""')}"`
      }
      return value
    }).join(',')
  }).join('\n')

  const csvContent = '\uFEFF' + headers + '\n' + rows // 添加 BOM 以支持中文
  downloadFile(csvContent, `${filename}.csv`, 'text/csv;charset=utf-8')
}

// 导出为 JSON
export function exportToJSON(data: any[], filename: string) {
  const jsonContent = JSON.stringify(data, null, 2)
  downloadFile(jsonContent, `${filename}.json`, 'application/json')
}

// 导出为 Excel (简单的 HTML 表格格式)
export function exportToExcel(data: any[], columns: { title: string; dataIndex: string }[], filename: string) {
  if (!data || data.length === 0) {
    return
  }

  let html = '<html xmlns:o="urn:schemas-microsoft-com:office:office" xmlns:x="urn:schemas-microsoft-com:office:excel">'
  html += '<head><meta charset="UTF-8"></head><body>'
  html += '<table border="1">'
  
  // 表头
  html += '<tr>'
  columns.forEach(col => {
    html += `<th style="background:#f0f0f0;font-weight:bold">${col.title}</th>`
  })
  html += '</tr>'
  
  // 数据行
  data.forEach(row => {
    html += '<tr>'
    columns.forEach(col => {
      const value = row[col.dataIndex] ?? ''
      html += `<td>${value}</td>`
    })
    html += '</tr>'
  })
  
  html += '</table></body></html>'
  
  downloadFile(html, `${filename}.xls`, 'application/vnd.ms-excel')
}

// 下载文件
function downloadFile(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

// 导出按钮组件的配置
export interface ExportConfig {
  data: any[]
  columns: { title: string; dataIndex: string }[]
  filename: string
}
