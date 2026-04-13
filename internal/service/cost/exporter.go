package cost

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/logger"
)

// CostExporter 成本报表导出器
type CostExporter struct {
	db  *gorm.DB
	log *logger.Logger
}

// NewCostExporter 创建成本报表导出器
func NewCostExporter(db *gorm.DB) *CostExporter {
	return &CostExporter{
		db:  db,
		log: logger.NewLogger("CostExporter"),
	}
}

// ExportOverviewReport 导出成本概览报表
func (e *CostExporter) ExportOverviewReport(ctx context.Context, clusterID uint, startTime, endTime time.Time) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	// 创建概览 Sheet
	sheetName := "成本概览"
	f.SetSheetName("Sheet1", sheetName)

	// 设置标题样式
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 11},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#D9E2F3"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "#000000", Style: 1},
			{Type: "right", Color: "#000000", Style: 1},
			{Type: "top", Color: "#000000", Style: 1},
			{Type: "bottom", Color: "#000000", Style: 1},
		},
	})

	// 报表标题
	f.MergeCell(sheetName, "A1", "G1")
	f.SetCellValue(sheetName, "A1", fmt.Sprintf("成本分析报表 (%s ~ %s)", startTime.Format("2006-01-02"), endTime.Format("2006-01-02")))
	f.SetCellStyle(sheetName, "A1", "G1", titleStyle)

	// 汇总数据
	var summary struct {
		TotalCost   float64
		CPUCost     float64
		MemoryCost  float64
		StorageCost float64
	}
	query := e.db.Model(&models.ResourceCost{}).
		Where("recorded_at BETWEEN ? AND ?", startTime, endTime)
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	query.Select(`
		COALESCE(SUM(total_cost), 0) as total_cost,
		COALESCE(SUM(cpu_cost), 0) as cpu_cost,
		COALESCE(SUM(memory_cost), 0) as memory_cost,
		COALESCE(SUM(storage_cost), 0) as storage_cost
	`).Scan(&summary)

	// 汇总信息
	f.SetCellValue(sheetName, "A3", "总成本")
	f.SetCellValue(sheetName, "B3", fmt.Sprintf("¥%.2f", summary.TotalCost))
	f.SetCellValue(sheetName, "C3", "CPU成本")
	f.SetCellValue(sheetName, "D3", fmt.Sprintf("¥%.2f", summary.CPUCost))
	f.SetCellValue(sheetName, "E3", "内存成本")
	f.SetCellValue(sheetName, "F3", fmt.Sprintf("¥%.2f", summary.MemoryCost))

	// 按命名空间分布
	f.SetCellValue(sheetName, "A5", "命名空间")
	f.SetCellValue(sheetName, "B5", "总成本")
	f.SetCellValue(sheetName, "C5", "CPU成本")
	f.SetCellValue(sheetName, "D5", "内存成本")
	f.SetCellValue(sheetName, "E5", "存储成本")
	f.SetCellValue(sheetName, "F5", "占比")
	f.SetCellStyle(sheetName, "A5", "F5", headerStyle)

	var nsCosts []struct {
		Namespace   string
		TotalCost   float64
		CPUCost     float64
		MemoryCost  float64
		StorageCost float64
	}
	nsQuery := e.db.Model(&models.ResourceCost{}).
		Where("recorded_at BETWEEN ? AND ?", startTime, endTime)
	if clusterID > 0 {
		nsQuery = nsQuery.Where("cluster_id = ?", clusterID)
	}
	nsQuery.Select(`
		namespace,
		SUM(total_cost) as total_cost,
		SUM(cpu_cost) as cpu_cost,
		SUM(memory_cost) as memory_cost,
		SUM(storage_cost) as storage_cost
	`).Group("namespace").Order("total_cost DESC").Scan(&nsCosts)

	row := 6
	for _, ns := range nsCosts {
		percent := 0.0
		if summary.TotalCost > 0 {
			percent = ns.TotalCost / summary.TotalCost * 100
		}
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), ns.Namespace)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("¥%.2f", ns.TotalCost))
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), fmt.Sprintf("¥%.2f", ns.CPUCost))
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("¥%.2f", ns.MemoryCost))
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), fmt.Sprintf("¥%.2f", ns.StorageCost))
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), fmt.Sprintf("%.1f%%", percent))
		row++
	}

	// 创建趋势 Sheet
	e.addTrendSheet(f, clusterID, startTime, endTime)

	// 创建资源明细 Sheet
	e.addDetailSheet(f, clusterID, startTime, endTime)

	// 创建优化建议 Sheet
	e.addSuggestionSheet(f, clusterID)

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 20)
	f.SetColWidth(sheetName, "B", "F", 15)

	// 写入 buffer
	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}

	return buf, nil
}

// addTrendSheet 添加趋势 Sheet
func (e *CostExporter) addTrendSheet(f *excelize.File, clusterID uint, startTime, endTime time.Time) {
	sheetName := "成本趋势"
	f.NewSheet(sheetName)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#D9E2F3"}, Pattern: 1},
	})

	f.SetCellValue(sheetName, "A1", "日期")
	f.SetCellValue(sheetName, "B1", "总成本")
	f.SetCellValue(sheetName, "C1", "CPU成本")
	f.SetCellValue(sheetName, "D1", "内存成本")
	f.SetCellValue(sheetName, "E1", "存储成本")
	f.SetCellStyle(sheetName, "A1", "E1", headerStyle)

	var dailyCosts []struct {
		Date        string
		TotalCost   float64
		CPUCost     float64
		MemoryCost  float64
		StorageCost float64
	}
	query := e.db.Model(&models.ResourceCost{}).
		Where("recorded_at BETWEEN ? AND ?", startTime, endTime)
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	query.Select(`
		DATE(recorded_at) as date,
		SUM(total_cost) as total_cost,
		SUM(cpu_cost) as cpu_cost,
		SUM(memory_cost) as memory_cost,
		SUM(storage_cost) as storage_cost
	`).Group("DATE(recorded_at)").Order("date").Scan(&dailyCosts)

	row := 2
	for _, dc := range dailyCosts {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), dc.Date)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), dc.TotalCost)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), dc.CPUCost)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), dc.MemoryCost)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), dc.StorageCost)
		row++
	}

	f.SetColWidth(sheetName, "A", "E", 15)
}

// addDetailSheet 添加资源明细 Sheet
func (e *CostExporter) addDetailSheet(f *excelize.File, clusterID uint, startTime, endTime time.Time) {
	sheetName := "资源明细"
	f.NewSheet(sheetName)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#D9E2F3"}, Pattern: 1},
	})

	headers := []string{"命名空间", "资源类型", "资源名称", "应用", "团队", "CPU请求", "CPU使用", "内存请求", "内存使用", "总成本"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, h)
	}
	f.SetCellStyle(sheetName, "A1", "J1", headerStyle)

	var costs []models.ResourceCost
	query := e.db.Model(&models.ResourceCost{}).
		Where("recorded_at BETWEEN ? AND ?", startTime, endTime)
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	query.Order("total_cost DESC").Limit(500).Find(&costs)

	row := 2
	for _, c := range costs {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), c.Namespace)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), c.ResourceType)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), c.ResourceName)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), c.AppName)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), c.TeamName)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), fmt.Sprintf("%.2f核", c.CPURequest))
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), fmt.Sprintf("%.2f核", c.CPUUsage))
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), fmt.Sprintf("%.2fGB", c.MemoryRequest))
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), fmt.Sprintf("%.2fGB", c.MemoryUsage))
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), fmt.Sprintf("¥%.2f", c.TotalCost))
		row++
	}

	f.SetColWidth(sheetName, "A", "J", 15)
}

// addSuggestionSheet 添加优化建议 Sheet
func (e *CostExporter) addSuggestionSheet(f *excelize.File, clusterID uint) {
	sheetName := "优化建议"
	f.NewSheet(sheetName)

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#D9E2F3"}, Pattern: 1},
	})

	headers := []string{"命名空间", "资源类型", "资源名称", "建议类型", "严重程度", "标题", "当前成本", "优化后成本", "可节省", "状态"}
	for i, h := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, h)
	}
	f.SetCellStyle(sheetName, "A1", "J1", headerStyle)

	var suggestions []models.CostSuggestion
	query := e.db.Model(&models.CostSuggestion{})
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	query.Order("savings DESC").Find(&suggestions)

	row := 2
	for _, s := range suggestions {
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), s.Namespace)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), s.ResourceType)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), s.ResourceName)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), s.SuggestionType)
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), s.Severity)
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), s.Title)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), fmt.Sprintf("¥%.2f", s.CurrentCost))
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", row), fmt.Sprintf("¥%.2f", s.OptimizedCost))
		f.SetCellValue(sheetName, fmt.Sprintf("I%d", row), fmt.Sprintf("¥%.2f", s.Savings))
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", row), s.Status)
		row++
	}

	f.SetColWidth(sheetName, "A", "J", 15)
}

// ExportComparisonReport 导出成本对比报表
func (e *CostExporter) ExportComparisonReport(ctx context.Context, clusterID uint, period1Start, period1End, period2Start, period2End time.Time) (*bytes.Buffer, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "成本对比"
	f.SetSheetName("Sheet1", sheetName)

	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Alignment: &excelize.Alignment{Horizontal: "center"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#D9E2F3"}, Pattern: 1},
	})

	// 标题
	f.MergeCell(sheetName, "A1", "F1")
	f.SetCellValue(sheetName, "A1", "成本对比分析报表")
	f.SetCellStyle(sheetName, "A1", "F1", titleStyle)

	// 表头
	f.SetCellValue(sheetName, "A3", "指标")
	f.SetCellValue(sheetName, "B3", fmt.Sprintf("周期1 (%s~%s)", period1Start.Format("01-02"), period1End.Format("01-02")))
	f.SetCellValue(sheetName, "C3", fmt.Sprintf("周期2 (%s~%s)", period2Start.Format("01-02"), period2End.Format("01-02")))
	f.SetCellValue(sheetName, "D3", "变化金额")
	f.SetCellValue(sheetName, "E3", "变化率")
	f.SetCellStyle(sheetName, "A3", "E3", headerStyle)

	// 获取两个周期的数据
	getCost := func(start, end time.Time) (total, cpu, memory, storage float64) {
		var result struct {
			TotalCost   float64
			CPUCost     float64
			MemoryCost  float64
			StorageCost float64
		}
		query := e.db.Model(&models.ResourceCost{}).
			Where("recorded_at BETWEEN ? AND ?", start, end)
		if clusterID > 0 {
			query = query.Where("cluster_id = ?", clusterID)
		}
		query.Select(`
			COALESCE(SUM(total_cost), 0) as total_cost,
			COALESCE(SUM(cpu_cost), 0) as cpu_cost,
			COALESCE(SUM(memory_cost), 0) as memory_cost,
			COALESCE(SUM(storage_cost), 0) as storage_cost
		`).Scan(&result)
		return result.TotalCost, result.CPUCost, result.MemoryCost, result.StorageCost
	}

	p1Total, p1CPU, p1Mem, p1Storage := getCost(period1Start, period1End)
	p2Total, p2CPU, p2Mem, p2Storage := getCost(period2Start, period2End)

	// 填充数据
	writeRow := func(row int, name string, v1, v2 float64) {
		change := v2 - v1
		rate := 0.0
		if v1 > 0 {
			rate = change / v1 * 100
		}
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), name)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), fmt.Sprintf("¥%.2f", v1))
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), fmt.Sprintf("¥%.2f", v2))
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), fmt.Sprintf("¥%.2f", change))
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), fmt.Sprintf("%.1f%%", rate))
	}

	writeRow(4, "总成本", p1Total, p2Total)
	writeRow(5, "CPU成本", p1CPU, p2CPU)
	writeRow(6, "内存成本", p1Mem, p2Mem)
	writeRow(7, "存储成本", p1Storage, p2Storage)

	f.SetColWidth(sheetName, "A", "E", 20)

	buf := new(bytes.Buffer)
	if err := f.Write(buf); err != nil {
		return nil, err
	}

	return buf, nil
}
