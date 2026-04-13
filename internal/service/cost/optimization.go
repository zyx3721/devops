package cost

import (
	"context"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
)

// GetSuggestions 获取成本优化建议
func (s *CostService) GetSuggestions(ctx context.Context, clusterID uint, status string) (*dto.CostSuggestionListResponse, error) {
	// 获取集群名称映射
	clusterNameMap := make(map[uint]string)
	var clusters []models.K8sCluster
	s.db.Select("id, name").Find(&clusters)
	for _, c := range clusters {
		clusterNameMap[c.ID] = c.Name
	}

	var suggestions []models.CostSuggestion
	query := s.db.Model(&models.CostSuggestion{})
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Order("savings DESC").Find(&suggestions)

	items := make([]dto.CostSuggestionItem, len(suggestions))
	var totalSavings float64
	var highCount, mediumCount, lowCount int

	for i, sg := range suggestions {
		clusterName := clusterNameMap[sg.ClusterID]
		if clusterName == "" {
			clusterName = fmt.Sprintf("集群-%d", sg.ClusterID)
		}
		items[i] = dto.CostSuggestionItem{
			ID:             sg.ID,
			ClusterID:      sg.ClusterID,
			ClusterName:    clusterName,
			Namespace:      sg.Namespace,
			ResourceType:   sg.ResourceType,
			ResourceName:   sg.ResourceName,
			SuggestionType: sg.SuggestionType,
			Severity:       sg.Severity,
			Title:          sg.Title,
			Description:    sg.Description,
			CurrentCost:    sg.CurrentCost,
			OptimizedCost:  sg.OptimizedCost,
			Savings:        sg.Savings,
			SavingsPercent: sg.SavingsPercent,
			Status:         sg.Status,
			CreatedAt:      sg.CreatedAt.Format("2006-01-02 15:04:05"),
		}
		if sg.Status == "pending" {
			totalSavings += sg.Savings
		}
		switch sg.Severity {
		case "high":
			highCount++
		case "medium":
			mediumCount++
		case "low":
			lowCount++
		}
	}

	return &dto.CostSuggestionListResponse{
		Items:        items,
		Total:        int64(len(suggestions)),
		TotalSavings: totalSavings,
		HighCount:    highCount,
		MediumCount:  mediumCount,
		LowCount:     lowCount,
	}, nil
}

// ApplySuggestion 应用优化建议
func (s *CostService) ApplySuggestion(ctx context.Context, suggestionID uint, userID uint) error {
	var suggestion models.CostSuggestion
	if err := s.db.First(&suggestion, suggestionID).Error; err != nil {
		return apperrors.FormatDBError(err, "查询优化建议")
	}
	if suggestion.Status != "pending" {
		return apperrors.New(apperrors.ErrCodeBusiness, "该建议已处理")
	}
	now := time.Now()
	return s.db.Model(&suggestion).Updates(map[string]interface{}{
		"status":     "applied",
		"applied_at": now,
		"applied_by": userID,
	}).Error
}

// IgnoreSuggestion 忽略优化建议
func (s *CostService) IgnoreSuggestion(ctx context.Context, suggestionID uint, userID uint) error {
	var suggestion models.CostSuggestion
	if err := s.db.First(&suggestion, suggestionID).Error; err != nil {
		return apperrors.FormatDBError(err, "查询优化建议")
	}
	if suggestion.Status != "pending" {
		return apperrors.New(apperrors.ErrCodeBusiness, "该建议已处理")
	}
	return s.db.Model(&suggestion).Update("status", "ignored").Error
}

// GetWasteDetection 获取资源浪费检测
func (s *CostService) GetWasteDetection(ctx context.Context, clusterID uint, days int) (*dto.WasteDetectionResponse, error) {
	if days <= 0 {
		days = 7
	}
	startTime := time.Now().AddDate(0, 0, -days)

	// 获取集群名称映射
	clusterNameMap := make(map[uint]string)
	var clusters []models.K8sCluster
	s.db.Select("id, name").Find(&clusters)
	for _, c := range clusters {
		clusterNameMap[c.ID] = c.Name
	}

	var resources []models.ResourceCost
	query := s.db.Model(&models.ResourceCost{}).Where("recorded_at >= ?", startTime)
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	query.Order("total_cost DESC").Limit(500).Find(&resources)

	// 使用map去重，key为 cluster_id:namespace:resource_type:resource_name
	idleMap := make(map[string]*dto.WasteItem)
	overprovisionedMap := make(map[string]*dto.WasteItem)
	var summary dto.WasteSummary

	for _, r := range resources {
		cpuUsageRate := 0.0
		memUsageRate := 0.0
		if r.CPURequest > 0 {
			cpuUsageRate = r.CPUUsage / r.CPURequest * 100
		}
		if r.MemoryRequest > 0 {
			memUsageRate = r.MemoryUsage / r.MemoryRequest * 100
		}

		uniqueKey := fmt.Sprintf("%d:%s:%s:%s", r.ClusterID, r.Namespace, r.ResourceType, r.ResourceName)
		clusterName := clusterNameMap[r.ClusterID]
		if clusterName == "" {
			clusterName = fmt.Sprintf("集群-%d", r.ClusterID)
		}

		if cpuUsageRate < 10 && memUsageRate < 10 {
			wastedCost := r.TotalCost * 0.9
			if existing, ok := idleMap[uniqueKey]; ok {
				existing.WasteCost += wastedCost
			} else {
				idleMap[uniqueKey] = &dto.WasteItem{
					ClusterID:    r.ClusterID,
					ClusterName:  clusterName,
					Namespace:    r.Namespace,
					ResourceType: r.ResourceType,
					ResourceName: r.ResourceName,
					WasteType:    "idle",
					WasteCost:    wastedCost,
					CurrentUsage: (cpuUsageRate + memUsageRate) / 2,
					Suggestion:   "建议缩容或删除该资源",
					Impact:       getImpactLevel(wastedCost),
				}
				summary.IdleCount++
			}
			summary.IdleCost += wastedCost
		} else if cpuUsageRate < 30 || memUsageRate < 30 {
			wastedCost := r.TotalCost * 0.5
			if existing, ok := overprovisionedMap[uniqueKey]; ok {
				existing.WasteCost += wastedCost
			} else {
				overprovisionedMap[uniqueKey] = &dto.WasteItem{
					ClusterID:    r.ClusterID,
					ClusterName:  clusterName,
					Namespace:    r.Namespace,
					ResourceType: r.ResourceType,
					ResourceName: r.ResourceName,
					WasteType:    "overprovisioned",
					WasteCost:    wastedCost,
					CurrentUsage: (cpuUsageRate + memUsageRate) / 2,
					Suggestion:   fmt.Sprintf("建议将资源配置降低50%%，当前CPU利用率%.1f%%，内存利用率%.1f%%", cpuUsageRate, memUsageRate),
					Impact:       getImpactLevel(wastedCost),
				}
				summary.OverprovisionedCount++
			}
			summary.OverprovisionedCost += wastedCost
		}
	}

	// 转换map为slice
	idleResources := make([]dto.WasteItem, 0, len(idleMap))
	for _, item := range idleMap {
		item.Impact = getImpactLevel(item.WasteCost)
		idleResources = append(idleResources, *item)
	}
	overprovisioned := make([]dto.WasteItem, 0, len(overprovisionedMap))
	for _, item := range overprovisionedMap {
		item.Impact = getImpactLevel(item.WasteCost)
		overprovisioned = append(overprovisioned, *item)
	}

	// 按浪费成本排序
	sort.Slice(idleResources, func(i, j int) bool { return idleResources[i].WasteCost > idleResources[j].WasteCost })
	sort.Slice(overprovisioned, func(i, j int) bool { return overprovisioned[i].WasteCost > overprovisioned[j].WasteCost })

	var zombies []models.ResourceActivity
	zombieQuery := s.db.Model(&models.ResourceActivity{}).Where("is_zombie = ?", true)
	if clusterID > 0 {
		zombieQuery = zombieQuery.Where("cluster_id = ?", clusterID)
	}
	zombieQuery.Find(&zombies)

	// 僵尸资源去重
	zombieMap := make(map[string]*dto.WasteItem)
	for _, z := range zombies {
		uniqueKey := fmt.Sprintf("%d:%s:%s:%s", z.ClusterID, z.Namespace, z.ResourceType, z.ResourceName)
		if _, ok := zombieMap[uniqueKey]; !ok {
			clusterName := clusterNameMap[z.ClusterID]
			if clusterName == "" {
				clusterName = fmt.Sprintf("集群-%d", z.ClusterID)
			}
			zombieMap[uniqueKey] = &dto.WasteItem{
				ClusterID:    z.ClusterID,
				ClusterName:  clusterName,
				Namespace:    z.Namespace,
				ResourceType: z.ResourceType,
				ResourceName: z.ResourceName,
				WasteType:    "zombie",
				IdleDays:     z.IdleDays,
				Suggestion:   fmt.Sprintf("该资源已闲置%d天，无任何流量，建议删除", z.IdleDays),
				Impact:       "high",
			}
			summary.ZombieCount++
		}
	}
	zombieResources := make([]dto.WasteItem, 0, len(zombieMap))
	for _, item := range zombieMap {
		zombieResources = append(zombieResources, *item)
	}

	totalWaste := summary.IdleCost + summary.OverprovisionedCost + summary.ZombieCost + summary.UnusedVolumeCost

	var totalCost float64
	s.db.Model(&models.ResourceCost{}).Where("recorded_at >= ?", startTime).
		Where("cluster_id = ? OR ? = 0", clusterID, clusterID).
		Select("COALESCE(SUM(total_cost), 0)").Scan(&totalCost)

	var wastePercent float64
	if totalCost > 0 {
		wastePercent = totalWaste / totalCost * 100
	}

	return &dto.WasteDetectionResponse{
		TotalWaste:      totalWaste,
		WastePercent:    wastePercent,
		IdleResources:   idleResources,
		Overprovisioned: overprovisioned,
		ZombieResources: zombieResources,
		UnusedVolumes:   []dto.WasteItem{},
		Summary:         summary,
	}, nil
}

// GetConfig 获取成本配置
func (s *CostService) GetConfig(ctx context.Context, clusterID uint) (*dto.CostConfigResponse, error) {
	var config models.CostConfig
	err := s.db.Where("cluster_id = ?", clusterID).First(&config).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &dto.CostConfigResponse{
				ClusterID:         clusterID,
				CPUPricePerCore:   0.1,
				MemoryPricePerGB:  0.05,
				StoragePricePerGB: 0.5,
				Currency:          "CNY",
			}, nil
		}
		return nil, apperrors.FormatDBError(err, "查询成本配置")
	}
	return &dto.CostConfigResponse{
		ClusterID:         config.ClusterID,
		CPUPricePerCore:   config.CPUPricePerCore,
		MemoryPricePerGB:  config.MemoryPricePerGB,
		StoragePricePerGB: config.StoragePricePerGB,
		Currency:          config.Currency,
	}, nil
}

// SaveConfig 保存成本配置
func (s *CostService) SaveConfig(ctx context.Context, clusterID uint, req *dto.CostConfigRequest) error {
	config := models.CostConfig{
		ClusterID:         clusterID,
		CPUPricePerCore:   req.CPUPricePerCore,
		MemoryPricePerGB:  req.MemoryPricePerGB,
		StoragePricePerGB: req.StoragePricePerGB,
		Currency:          req.Currency,
	}
	if config.Currency == "" {
		config.Currency = "CNY"
	}
	return s.db.Where("cluster_id = ?", clusterID).Assign(config).FirstOrCreate(&config).Error
}
