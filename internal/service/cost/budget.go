package cost

import (
	"context"
	"time"

	"devops/internal/models"
	"devops/pkg/dto"
)

// GetBudgetList 获取预算列表
func (s *CostService) GetBudgetList(ctx context.Context, clusterID uint) (*dto.CostBudgetListResponse, error) {
	var budgets []models.CostBudget
	query := s.db.Model(&models.CostBudget{})
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	query.Find(&budgets)

	items := make([]dto.CostBudgetResponse, len(budgets))
	var totalBudget, totalUsed float64
	var overBudget, atRisk int

	for i, b := range budgets {
		items[i] = dto.CostBudgetResponse{
			ID:             b.ID,
			ClusterID:      b.ClusterID,
			Namespace:      b.Namespace,
			MonthlyBudget:  b.MonthlyBudget,
			CurrentCost:    b.CurrentCost,
			UsagePercent:   b.UsagePercent,
			AlertThreshold: b.AlertThreshold,
			Status:         b.Status,
		}
		totalBudget += b.MonthlyBudget
		totalUsed += b.CurrentCost
		if b.Status == "exceeded" {
			overBudget++
		} else if b.Status == "warning" {
			atRisk++
		}
	}

	return &dto.CostBudgetListResponse{
		Items:       items,
		TotalBudget: totalBudget,
		TotalUsed:   totalUsed,
		OverBudget:  overBudget,
		AtRisk:      atRisk,
	}, nil
}

// SaveBudget 保存预算
func (s *CostService) SaveBudget(ctx context.Context, req *dto.CostBudgetRequest) error {
	budget := models.CostBudget{
		ClusterID:      req.ClusterID,
		Namespace:      req.Namespace,
		MonthlyBudget:  req.MonthlyBudget,
		AlertThreshold: req.AlertThreshold,
		Status:         "normal",
	}
	if budget.AlertThreshold <= 0 {
		budget.AlertThreshold = 80
	}
	return s.db.Where("cluster_id = ? AND namespace = ?", req.ClusterID, req.Namespace).Assign(budget).FirstOrCreate(&budget).Error
}

// GetComparison 获取成本对比分析
func (s *CostService) GetComparison(ctx context.Context, req *dto.CostComparisonRequest) (*dto.CostComparisonResponse, error) {
	period1Start, _ := time.Parse("2006-01-02", req.Period1Start)
	period1End, _ := time.Parse("2006-01-02", req.Period1End)
	period2Start, _ := time.Parse("2006-01-02", req.Period2Start)
	period2End, _ := time.Parse("2006-01-02", req.Period2End)

	getCostData := func(start, end time.Time) *dto.PeriodCostData {
		var result struct {
			TotalCost   float64
			CPUCost     float64
			MemoryCost  float64
			StorageCost float64
			AvgCPU      float64
			AvgMemory   float64
		}
		query := s.db.Model(&models.ResourceCost{}).Where("recorded_at BETWEEN ? AND ?", start, end)
		if req.ClusterID > 0 {
			query = query.Where("cluster_id = ?", req.ClusterID)
		}
		query.Select(`
			COALESCE(SUM(total_cost), 0) as total_cost,
			COALESCE(SUM(cpu_cost), 0) as cpu_cost,
			COALESCE(SUM(memory_cost), 0) as memory_cost,
			COALESCE(SUM(storage_cost), 0) as storage_cost,
			COALESCE(AVG(CASE WHEN cpu_request > 0 THEN cpu_usage / cpu_request * 100 ELSE 0 END), 0) as avg_cpu,
			COALESCE(AVG(CASE WHEN memory_request > 0 THEN memory_usage / memory_request * 100 ELSE 0 END), 0) as avg_memory
		`).Scan(&result)
		return &dto.PeriodCostData{
			TotalCost:      result.TotalCost,
			CPUCost:        result.CPUCost,
			MemoryCost:     result.MemoryCost,
			StorageCost:    result.StorageCost,
			AvgCPUUsage:    result.AvgCPU,
			AvgMemoryUsage: result.AvgMemory,
		}
	}

	period1Data := getCostData(period1Start, period1End)
	period2Data := getCostData(period2Start, period2End)

	calcChange := func(v1, v2 float64) (change, rate float64) {
		change = v2 - v1
		if v1 > 0 {
			rate = change / v1 * 100
		}
		return
	}

	totalChange, totalRate := calcChange(period1Data.TotalCost, period2Data.TotalCost)
	cpuChange, cpuRate := calcChange(period1Data.CPUCost, period2Data.CPUCost)
	memChange, memRate := calcChange(period1Data.MemoryCost, period2Data.MemoryCost)
	storageChange, storageRate := calcChange(period1Data.StorageCost, period2Data.StorageCost)

	var nsComparison []dto.NamespaceComparison
	var ns1Data, ns2Data []struct {
		Namespace string
		TotalCost float64
	}

	q1 := s.db.Model(&models.ResourceCost{}).Where("recorded_at BETWEEN ? AND ?", period1Start, period1End)
	if req.ClusterID > 0 {
		q1 = q1.Where("cluster_id = ?", req.ClusterID)
	}
	q1.Select("namespace, SUM(total_cost) as total_cost").Group("namespace").Scan(&ns1Data)

	q2 := s.db.Model(&models.ResourceCost{}).Where("recorded_at BETWEEN ? AND ?", period2Start, period2End)
	if req.ClusterID > 0 {
		q2 = q2.Where("cluster_id = ?", req.ClusterID)
	}
	q2.Select("namespace, SUM(total_cost) as total_cost").Group("namespace").Scan(&ns2Data)

	nsMap := make(map[string]*dto.NamespaceComparison)
	for _, ns := range ns1Data {
		nsMap[ns.Namespace] = &dto.NamespaceComparison{Namespace: ns.Namespace, Period1Cost: ns.TotalCost}
	}
	for _, ns := range ns2Data {
		if existing, ok := nsMap[ns.Namespace]; ok {
			existing.Period2Cost = ns.TotalCost
		} else {
			nsMap[ns.Namespace] = &dto.NamespaceComparison{Namespace: ns.Namespace, Period2Cost: ns.TotalCost}
		}
	}
	for _, ns := range nsMap {
		ns.Change, ns.ChangeRate = calcChange(ns.Period1Cost, ns.Period2Cost)
		nsComparison = append(nsComparison, *ns)
	}

	return &dto.CostComparisonResponse{
		Period1:             dto.PeriodInfo{Start: req.Period1Start, End: req.Period1End, Data: *period1Data},
		Period2:             dto.PeriodInfo{Start: req.Period2Start, End: req.Period2End, Data: *period2Data},
		TotalChange:         totalChange,
		TotalChangeRate:     totalRate,
		CPUChange:           cpuChange,
		CPUChangeRate:       cpuRate,
		MemoryChange:        memChange,
		MemoryChangeRate:    memRate,
		StorageChange:       storageChange,
		StorageChangeRate:   storageRate,
		NamespaceComparison: nsComparison,
	}, nil
}
