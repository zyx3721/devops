package cost

import (
	"context"
	"fmt"
	"math"
	"time"

	"devops/internal/models"
	"devops/pkg/dto"
)

// GetOverview 获取成本概览
func (s *CostService) GetOverview(ctx context.Context, clusterID uint, days int) (*dto.CostOverviewResponse, error) {
	if days <= 0 {
		days = 30
	}

	// 尝试从缓存获取
	cacheKey := fmt.Sprintf("overview:%d:%d", clusterID, days)
	if cached, ok := s.getCache(cacheKey); ok {
		return cached.(*dto.CostOverviewResponse), nil
	}

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)
	prevStartTime := startTime.AddDate(0, 0, -days)
	yoyStartTime := startTime.AddDate(-1, 0, 0)
	yoyEndTime := endTime.AddDate(-1, 0, 0)

	var currentCost struct {
		TotalCost   float64
		CPUCost     float64
		MemoryCost  float64
		StorageCost float64
		AvgCPU      float64
		AvgMemory   float64
	}

	query := s.db.Model(&models.ResourceCost{}).
		Where("recorded_at BETWEEN ? AND ?", startTime, endTime)
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}

	query.Select(`
		COALESCE(SUM(total_cost), 0) as total_cost,
		COALESCE(SUM(cpu_cost), 0) as cpu_cost,
		COALESCE(SUM(memory_cost), 0) as memory_cost,
		COALESCE(SUM(storage_cost), 0) as storage_cost,
		COALESCE(AVG(CASE WHEN cpu_request > 0 THEN cpu_usage / cpu_request * 100 ELSE 0 END), 0) as avg_cpu,
		COALESCE(AVG(CASE WHEN memory_request > 0 THEN memory_usage / memory_request * 100 ELSE 0 END), 0) as avg_memory
	`).Scan(&currentCost)

	var prevCost struct{ TotalCost float64 }
	prevQuery := s.db.Model(&models.ResourceCost{}).Where("recorded_at BETWEEN ? AND ?", prevStartTime, startTime)
	if clusterID > 0 {
		prevQuery = prevQuery.Where("cluster_id = ?", clusterID)
	}
	prevQuery.Select("COALESCE(SUM(total_cost), 0) as total_cost").Scan(&prevCost)

	var yoyCost struct{ TotalCost float64 }
	yoyQuery := s.db.Model(&models.ResourceCost{}).Where("recorded_at BETWEEN ? AND ?", yoyStartTime, yoyEndTime)
	if clusterID > 0 {
		yoyQuery = yoyQuery.Where("cluster_id = ?", clusterID)
	}
	yoyQuery.Select("COALESCE(SUM(total_cost), 0) as total_cost").Scan(&yoyCost)

	costChange := currentCost.TotalCost - prevCost.TotalCost
	var costChangeRate float64
	if prevCost.TotalCost > 0 {
		costChangeRate = costChange / prevCost.TotalCost * 100
	}

	yoyCostChange := currentCost.TotalCost - yoyCost.TotalCost
	var yoyCostChangeRate float64
	if yoyCost.TotalCost > 0 {
		yoyCostChangeRate = yoyCostChange / yoyCost.TotalCost * 100
	}

	var suggestionCount int64
	var potentialSavings float64
	suggestionQuery := s.db.Model(&models.CostSuggestion{}).Where("status = ?", "pending")
	if clusterID > 0 {
		suggestionQuery = suggestionQuery.Where("cluster_id = ?", clusterID)
	}
	suggestionQuery.Count(&suggestionCount)
	suggestionQuery.Select("COALESCE(SUM(savings), 0)").Scan(&potentialSavings)

	var wasteData struct {
		WastedCost    float64
		IdleResources int64
	}
	wasteQuery := s.db.Model(&models.ResourceCost{}).Where("recorded_at >= ?", time.Now().AddDate(0, 0, -1))
	if clusterID > 0 {
		wasteQuery = wasteQuery.Where("cluster_id = ?", clusterID)
	}
	wasteQuery.Select(`
		COALESCE(SUM(CASE WHEN cpu_request > 0 AND cpu_usage / cpu_request < 0.1 THEN total_cost * 0.8 ELSE 0 END), 0) as wasted_cost,
		COUNT(CASE WHEN cpu_request > 0 AND cpu_usage / cpu_request < 0.1 THEN 1 END) as idle_resources
	`).Scan(&wasteData)

	var zombieCount int64
	s.db.Model(&models.ResourceActivity{}).Where("is_zombie = ?", true).
		Where("cluster_id = ? OR ? = 0", clusterID, clusterID).Count(&zombieCount)

	var budgetData struct {
		TotalBudget float64
		UsedBudget  float64
	}
	budgetQuery := s.db.Model(&models.CostBudget{})
	if clusterID > 0 {
		budgetQuery = budgetQuery.Where("cluster_id = ?", clusterID)
	}
	budgetQuery.Select(`COALESCE(SUM(monthly_budget), 0) as total_budget, COALESCE(SUM(current_cost), 0) as used_budget`).Scan(&budgetData)

	var budgetPercent float64
	if budgetData.TotalBudget > 0 {
		budgetPercent = budgetData.UsedBudget / budgetData.TotalBudget * 100
	}

	daysPassed := time.Now().Day()
	daysInMonth := time.Date(time.Now().Year(), time.Now().Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	var predictedCost float64
	if daysPassed > 0 {
		dailyAvg := currentCost.TotalCost / float64(days)
		predictedCost = dailyAvg * float64(daysInMonth)
	}

	var wastedPercentage float64
	if currentCost.TotalCost > 0 {
		wastedPercentage = wasteData.WastedCost / currentCost.TotalCost * 100
	}

	result := &dto.CostOverviewResponse{
		TotalCost:         currentCost.TotalCost,
		CPUCost:           currentCost.CPUCost,
		MemoryCost:        currentCost.MemoryCost,
		StorageCost:       currentCost.StorageCost,
		CostChange:        costChange,
		CostChangeRate:    costChangeRate,
		YoYCostChange:     yoyCostChange,
		YoYCostChangeRate: yoyCostChangeRate,
		AvgCPUUsage:       currentCost.AvgCPU,
		AvgMemoryUsage:    currentCost.AvgMemory,
		WastedCost:        wasteData.WastedCost,
		WastedPercentage:  wastedPercentage,
		IdleResources:     int(wasteData.IdleResources),
		ZombieResources:   int(zombieCount),
		PotentialSavings:  potentialSavings,
		SuggestionCount:   int(suggestionCount),
		BudgetTotal:       budgetData.TotalBudget,
		BudgetUsed:        budgetData.UsedBudget,
		BudgetPercent:     budgetPercent,
		PredictedCost:     predictedCost,
		StartTime:         startTime,
		EndTime:           endTime,
	}

	// 缓存5分钟
	s.setCache(cacheKey, result, 5*time.Minute)
	return result, nil
}

// GetTrend 获取成本趋势
func (s *CostService) GetTrend(ctx context.Context, req *dto.CostTrendRequest) (*dto.CostTrendResponse, error) {
	if req.Days <= 0 {
		req.Days = 30
	}

	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -req.Days)

	var results []struct {
		Date        string
		TotalCost   float64
		CPUCost     float64
		MemoryCost  float64
		StorageCost float64
	}

	query := s.db.Model(&models.ResourceCost{}).Where("recorded_at BETWEEN ? AND ?", startTime, endTime)
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}

	query.Select(`
		DATE(recorded_at) as date,
		SUM(total_cost) as total_cost,
		SUM(cpu_cost) as cpu_cost,
		SUM(memory_cost) as memory_cost,
		SUM(storage_cost) as storage_cost
	`).Group("DATE(recorded_at)").Order("date").Scan(&results)

	items := make([]dto.CostTrendItem, len(results))
	var totalCost float64
	for i, r := range results {
		items[i] = dto.CostTrendItem{
			Date:        r.Date,
			TotalCost:   r.TotalCost,
			CPUCost:     r.CPUCost,
			MemoryCost:  r.MemoryCost,
			StorageCost: r.StorageCost,
		}
		totalCost += r.TotalCost
	}

	trendDirection := "stable"
	var trendPercentage float64
	if len(items) >= 7 {
		recent := 0.0
		previous := 0.0
		for i := len(items) - 7; i < len(items); i++ {
			recent += items[i].TotalCost
		}
		for i := len(items) - 14; i < len(items)-7 && i >= 0; i++ {
			previous += items[i].TotalCost
		}
		if previous > 0 {
			trendPercentage = (recent - previous) / previous * 100
			if trendPercentage > 5 {
				trendDirection = "up"
			} else if trendPercentage < -5 {
				trendDirection = "down"
			}
		}
	}

	prediction := make([]dto.CostTrendItem, 0)
	if len(items) >= 7 {
		var recentAvg float64
		for i := len(items) - 7; i < len(items); i++ {
			recentAvg += items[i].TotalCost
		}
		recentAvg /= 7

		var slope float64
		if len(items) >= 14 {
			var prevAvg float64
			for i := len(items) - 14; i < len(items)-7; i++ {
				prevAvg += items[i].TotalCost
			}
			prevAvg /= 7
			slope = (recentAvg - prevAvg) / 7
		}

		for i := 1; i <= 7; i++ {
			futureDate := endTime.AddDate(0, 0, i)
			predictedCost := recentAvg + slope*float64(i)
			if predictedCost < 0 {
				predictedCost = 0
			}
			prediction = append(prediction, dto.CostTrendItem{
				Date:          futureDate.Format("2006-01-02"),
				PredictedCost: predictedCost,
			})
		}
	}

	anomalies := make([]dto.CostAnomaly, 0)
	if len(items) > 1 {
		avgCost := totalCost / float64(len(items))
		var sumSquares float64
		for _, item := range items {
			diff := item.TotalCost - avgCost
			sumSquares += diff * diff
		}
		stdDev := 0.0
		if len(items) > 1 {
			variance := sumSquares / float64(len(items)-1)
			if variance > 0 {
				stdDev = math.Sqrt(variance)
			}
		}

		threshold := avgCost + 2*stdDev
		for _, item := range items {
			if item.TotalCost > threshold && stdDev > 0 {
				deviation := (item.TotalCost - avgCost) / avgCost * 100
				anomalies = append(anomalies, dto.CostAnomaly{
					Date:         item.Date,
					ActualCost:   item.TotalCost,
					ExpectedCost: avgCost,
					Deviation:    deviation,
					Reason:       "成本异常偏高，可能存在资源突增或配置变更",
				})
			}
		}
	}

	return &dto.CostTrendResponse{
		Items:           items,
		TrendDirection:  trendDirection,
		TrendPercentage: trendPercentage,
		Prediction:      prediction,
		Anomalies:       anomalies,
	}, nil
}

// GetDistribution 获取成本分布
func (s *CostService) GetDistribution(ctx context.Context, req *dto.CostDistributionRequest) (*dto.CostDistributionResponse, error) {
	if req.TopN <= 0 {
		req.TopN = 10
	}
	if req.Dimension == "" {
		req.Dimension = "namespace"
	}

	var startTime, endTime time.Time
	var err error
	if req.StartTime != "" {
		startTime, err = time.Parse("2006-01-02", req.StartTime)
		if err != nil {
			return nil, fmt.Errorf("开始时间格式错误")
		}
	} else {
		startTime = time.Now().AddDate(0, 0, -30)
	}
	if req.EndTime != "" {
		endTime, err = time.Parse("2006-01-02", req.EndTime)
		if err != nil {
			return nil, fmt.Errorf("结束时间格式错误")
		}
	} else {
		endTime = time.Now()
	}

	var groupField string
	switch req.Dimension {
	case "namespace":
		groupField = "namespace"
	case "app":
		groupField = "app_name"
	case "team":
		groupField = "team_name"
	case "resource_type":
		groupField = "resource_type"
	default:
		groupField = "namespace"
	}

	var results []struct {
		Name        string
		TotalCost   float64
		CPUCost     float64
		MemoryCost  float64
		StorageCost float64
	}

	query := s.db.Model(&models.ResourceCost{}).Where("recorded_at BETWEEN ? AND ?", startTime, endTime)
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}

	query.Select(fmt.Sprintf(`
		%s as name,
		SUM(total_cost) as total_cost,
		SUM(cpu_cost) as cpu_cost,
		SUM(memory_cost) as memory_cost,
		SUM(storage_cost) as storage_cost
	`, groupField)).Group(groupField).Order("total_cost DESC").Limit(req.TopN).Scan(&results)

	var total float64
	for _, r := range results {
		total += r.TotalCost
	}

	items := make([]dto.CostDistributionItem, len(results))
	for i, r := range results {
		var percentage float64
		if total > 0 {
			percentage = r.TotalCost / total * 100
		}
		items[i] = dto.CostDistributionItem{
			Name:        r.Name,
			TotalCost:   r.TotalCost,
			CPUCost:     r.CPUCost,
			MemoryCost:  r.MemoryCost,
			StorageCost: r.StorageCost,
			Percentage:  percentage,
		}
	}

	return &dto.CostDistributionResponse{
		Dimension: req.Dimension,
		Items:     items,
		Total:     total,
	}, nil
}

// GetResourceUsage 获取资源利用率
func (s *CostService) GetResourceUsage(ctx context.Context, req *dto.ResourceUsageRequest) (*dto.ResourceUsageResponse, error) {
	if req.TopN <= 0 {
		req.TopN = 20
	}

	var results []models.ResourceCost
	query := s.db.Model(&models.ResourceCost{}).Where("recorded_at >= ?", time.Now().AddDate(0, 0, -1))
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}
	if req.Namespace != "" {
		query = query.Where("namespace = ?", req.Namespace)
	}
	query.Order("total_cost DESC").Limit(req.TopN).Find(&results)

	items := make([]dto.ResourceUsageItem, len(results))
	var totalCPUUsage, totalMemoryUsage float64
	var count int

	for i, r := range results {
		var cpuUsageRate, memoryUsageRate float64
		if r.CPURequest > 0 {
			cpuUsageRate = r.CPUUsage / r.CPURequest * 100
			totalCPUUsage += cpuUsageRate
			count++
		}
		if r.MemoryRequest > 0 {
			memoryUsageRate = r.MemoryUsage / r.MemoryRequest * 100
			totalMemoryUsage += memoryUsageRate
		}

		status := "normal"
		if cpuUsageRate < 10 && memoryUsageRate < 10 {
			status = "idle"
		} else if cpuUsageRate < 30 || memoryUsageRate < 30 {
			status = "underutilized"
		} else if cpuUsageRate > 80 || memoryUsageRate > 80 {
			status = "overprovisioned"
		}

		items[i] = dto.ResourceUsageItem{
			Namespace:       r.Namespace,
			ResourceType:    r.ResourceType,
			ResourceName:    r.ResourceName,
			CPURequest:      r.CPURequest,
			CPUUsage:        r.CPUUsage,
			CPUUsageRate:    cpuUsageRate,
			MemoryRequest:   r.MemoryRequest,
			MemoryUsage:     r.MemoryUsage,
			MemoryUsageRate: memoryUsageRate,
			TotalCost:       r.TotalCost,
			Status:          status,
		}
	}

	var avgCPU, avgMemory float64
	if count > 0 {
		avgCPU = totalCPUUsage / float64(count)
		avgMemory = totalMemoryUsage / float64(count)
	}

	return &dto.ResourceUsageResponse{
		Items:          items,
		AvgCPUUsage:    avgCPU,
		AvgMemoryUsage: avgMemory,
	}, nil
}

// GetForecast 获取成本预测
func (s *CostService) GetForecast(ctx context.Context, clusterID uint, days int) (*dto.CostForecastResponse, error) {
	if days <= 0 {
		days = 30
	}

	var historicalData []struct {
		Date      string
		TotalCost float64
	}
	query := s.db.Model(&models.ResourceCost{}).Where("recorded_at >= ?", time.Now().AddDate(0, 0, -90))
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	query.Select("DATE(recorded_at) as date, SUM(total_cost) as total_cost").Group("DATE(recorded_at)").Order("date").Scan(&historicalData)

	monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
	var currentMonthCost float64
	monthQuery := s.db.Model(&models.ResourceCost{}).Where("recorded_at >= ?", monthStart)
	if clusterID > 0 {
		monthQuery = monthQuery.Where("cluster_id = ?", clusterID)
	}
	monthQuery.Select("COALESCE(SUM(total_cost), 0)").Scan(&currentMonthCost)

	var dailyAvg float64
	if len(historicalData) > 0 {
		var total float64
		for _, d := range historicalData {
			total += d.TotalCost
		}
		dailyAvg = total / float64(len(historicalData))
	}

	daysPassed := time.Now().Day()
	daysInMonth := time.Date(time.Now().Year(), time.Now().Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	daysRemaining := daysInMonth - daysPassed
	predictedMonthCost := currentMonthCost + dailyAvg*float64(daysRemaining)

	var trendFactor float64 = 1.0
	if len(historicalData) >= 60 {
		var recent, previous float64
		for i := len(historicalData) - 30; i < len(historicalData); i++ {
			recent += historicalData[i].TotalCost
		}
		for i := len(historicalData) - 60; i < len(historicalData)-30; i++ {
			previous += historicalData[i].TotalCost
		}
		if previous > 0 {
			trendFactor = recent / previous
		}
	}
	nextMonthCost := dailyAvg * 30 * trendFactor

	dailyForecast := make([]dto.ForecastItem, 0, days)
	for i := 1; i <= days; i++ {
		futureDate := time.Now().AddDate(0, 0, i)
		predicted := dailyAvg * trendFactor
		dailyForecast = append(dailyForecast, dto.ForecastItem{
			Date:          futureDate.Format("2006-01-02"),
			PredictedCost: predicted,
			LowerBound:    predicted * 0.8,
			UpperBound:    predicted * 1.2,
		})
	}

	factors := []dto.ForecastFactor{
		{Name: "资源增长趋势", Impact: (trendFactor - 1) * 100, Trend: getTrendString(trendFactor)},
	}

	confidence := 70.0
	if len(historicalData) >= 60 {
		confidence = 85.0
	} else if len(historicalData) >= 30 {
		confidence = 75.0
	}

	return &dto.CostForecastResponse{
		CurrentMonthCost:   currentMonthCost,
		PredictedMonthCost: predictedMonthCost,
		NextMonthCost:      nextMonthCost,
		DailyForecast:      dailyForecast,
		Confidence:         confidence,
		Factors:            factors,
	}, nil
}

// GetCostHealthScore 获取成本健康评分
func (s *CostService) GetCostHealthScore(ctx context.Context, clusterID uint) (*dto.CostHealthScoreResponse, error) {
	overview, _ := s.GetOverview(ctx, clusterID, 30)
	waste, _ := s.GetWasteDetection(ctx, clusterID, 7)

	dimensions := []dto.ScoreDimension{}
	totalScore := 0

	utilizationScore := 30
	avgUsage := (overview.AvgCPUUsage + overview.AvgMemoryUsage) / 2
	if avgUsage < 20 {
		utilizationScore = 10
	} else if avgUsage < 40 {
		utilizationScore = 20
	} else if avgUsage < 60 {
		utilizationScore = 25
	}
	dimensions = append(dimensions, dto.ScoreDimension{
		Name:        "资源利用率",
		Score:       utilizationScore,
		MaxScore:    30,
		Description: fmt.Sprintf("平均利用率 %.1f%%", avgUsage),
		Status:      getScoreStatus(utilizationScore, 30),
	})
	totalScore += utilizationScore

	trendScore := 20
	if overview.CostChangeRate > 20 {
		trendScore = 5
	} else if overview.CostChangeRate > 10 {
		trendScore = 10
	} else if overview.CostChangeRate > 0 {
		trendScore = 15
	}
	dimensions = append(dimensions, dto.ScoreDimension{
		Name:        "成本趋势",
		Score:       trendScore,
		MaxScore:    20,
		Description: fmt.Sprintf("环比变化 %.1f%%", overview.CostChangeRate),
		Status:      getScoreStatus(trendScore, 20),
	})
	totalScore += trendScore

	wasteScore := 25
	if waste.WastePercent > 30 {
		wasteScore = 5
	} else if waste.WastePercent > 20 {
		wasteScore = 10
	} else if waste.WastePercent > 10 {
		wasteScore = 18
	}
	dimensions = append(dimensions, dto.ScoreDimension{
		Name:        "浪费控制",
		Score:       wasteScore,
		MaxScore:    25,
		Description: fmt.Sprintf("浪费占比 %.1f%%", waste.WastePercent),
		Status:      getScoreStatus(wasteScore, 25),
	})
	totalScore += wasteScore

	budgetScore := 15
	if overview.BudgetPercent > 100 {
		budgetScore = 0
	} else if overview.BudgetPercent > 90 {
		budgetScore = 5
	} else if overview.BudgetPercent > 80 {
		budgetScore = 10
	}
	dimensions = append(dimensions, dto.ScoreDimension{
		Name:        "预算执行",
		Score:       budgetScore,
		MaxScore:    15,
		Description: fmt.Sprintf("预算使用率 %.1f%%", overview.BudgetPercent),
		Status:      getScoreStatus(budgetScore, 15),
	})
	totalScore += budgetScore

	optimizeScore := 10
	if overview.SuggestionCount > 20 {
		optimizeScore = 3
	} else if overview.SuggestionCount > 10 {
		optimizeScore = 5
	} else if overview.SuggestionCount > 5 {
		optimizeScore = 7
	}
	dimensions = append(dimensions, dto.ScoreDimension{
		Name:        "优化执行",
		Score:       optimizeScore,
		MaxScore:    10,
		Description: fmt.Sprintf("%d 条待处理建议", overview.SuggestionCount),
		Status:      getScoreStatus(optimizeScore, 10),
	})
	totalScore += optimizeScore

	grade := "F"
	if totalScore >= 90 {
		grade = "A"
	} else if totalScore >= 80 {
		grade = "B"
	} else if totalScore >= 70 {
		grade = "C"
	} else if totalScore >= 60 {
		grade = "D"
	}

	recommendations := []string{}
	if utilizationScore < 20 {
		recommendations = append(recommendations, "资源利用率偏低，建议进行资源优化")
	}
	if trendScore < 15 {
		recommendations = append(recommendations, "成本增长过快，建议分析成本增长原因")
	}
	if wasteScore < 18 {
		recommendations = append(recommendations, "存在较多资源浪费，建议清理闲置资源")
	}
	if budgetScore < 10 {
		recommendations = append(recommendations, "预算使用率较高，建议关注成本控制")
	}

	return &dto.CostHealthScoreResponse{
		OverallScore:    totalScore,
		Grade:           grade,
		Dimensions:      dimensions,
		Recommendations: recommendations,
		Trend:           getTrendFromScore(totalScore),
	}, nil
}
