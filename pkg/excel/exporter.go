package excel

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// ExcelExporter Excel 导出工具
type ExcelExporter struct {
	file      *excelize.File
	sheetName string
	rowIndex  int
}

// NewExporter 创建 Excel 导出器
func NewExporter() *ExcelExporter {
	f := excelize.NewFile()
	return &ExcelExporter{
		file:      f,
		sheetName: "Sheet1",
		rowIndex:  1,
	}
}

// AddSheet 添加工作表
func (e *ExcelExporter) AddSheet(name string) *ExcelExporter {
	index, _ := e.file.NewSheet(name)
	e.file.SetActiveSheet(index)
	e.sheetName = name
	e.rowIndex = 1
	return e
}

// SetHeaders 设置表头
func (e *ExcelExporter) SetHeaders(headers []string) *ExcelExporter {
	for i, header := range headers {
		cell := fmt.Sprintf("%s%d", columnName(i), e.rowIndex)
		e.file.SetCellValue(e.sheetName, cell, header)
	}

	// 应用表头样式
	e.ApplyHeaderStyle(len(headers))
	e.rowIndex++
	return e
}

// AddRow 添加数据行
func (e *ExcelExporter) AddRow(data []interface{}) *ExcelExporter {
	for i, value := range data {
		cell := fmt.Sprintf("%s%d", columnName(i), e.rowIndex)
		e.file.SetCellValue(e.sheetName, cell, value)
	}
	e.rowIndex++
	return e
}

// SetColumnWidth 设置列宽
func (e *ExcelExporter) SetColumnWidth(col string, width float64) *ExcelExporter {
	e.file.SetColWidth(e.sheetName, col, col, width)
	return e
}

// SetColumnWidthRange 设置列宽范围
func (e *ExcelExporter) SetColumnWidthRange(startCol, endCol string, width float64) *ExcelExporter {
	e.file.SetColWidth(e.sheetName, startCol, endCol, width)
	return e
}

// ApplyHeaderStyle 应用表头样式
func (e *ExcelExporter) ApplyHeaderStyle(colCount int) *ExcelExporter {
	// 创建表头样式
	style, _ := e.file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 12,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#4472C4"},
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})

	// 应用样式到表头行
	for i := 0; i < colCount; i++ {
		cell := fmt.Sprintf("%s1", columnName(i))
		e.file.SetCellStyle(e.sheetName, cell, cell, style)
	}

	return e
}

// AutoFilter 添加自动筛选
func (e *ExcelExporter) AutoFilter(startCol, endCol string, rowCount int) *ExcelExporter {
	filterRange := fmt.Sprintf("%s1:%s%d", startCol, endCol, rowCount)
	e.file.AutoFilter(e.sheetName, filterRange, []excelize.AutoFilterOptions{})
	return e
}

// SaveToWriter 保存到 Writer
func (e *ExcelExporter) SaveToWriter(w io.Writer) error {
	return e.file.Write(w)
}

// Close 关闭文件
func (e *ExcelExporter) Close() error {
	return e.file.Close()
}

// columnName 将列索引转换为列名 (0 -> A, 1 -> B, ..., 25 -> Z, 26 -> AA)
func columnName(index int) string {
	name := ""
	for index >= 0 {
		name = string(rune('A'+(index%26))) + name
		index = index/26 - 1
	}
	return name
}
