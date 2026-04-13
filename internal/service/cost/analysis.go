package cost

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"devops/internal/models"
	"devops/pkg/dto"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetAppCost 获取应用维度成本分析
func (s *CostService) GetAppCost(ctx context.Context, req *dto.AppCostRequest) (*dto.AppCostResponse, error) {
	var startTime, endTime time.Time
	if req.StartTime != "" {
		startTime, _ = time.Parse("2006-01-02", req.StartTime)
	} else {
		startTime = time.Now().AddDate(0, 0, -30)
	}
	if req.EndTime != "" {
		endTime, _ = time.Parse("2006-01-02", req.EndTime)
	} else {
		endTime = time.Now()
	}
	if req.TopN <= 0 {
		req.TopN = 20
	}

	var results []struct {
		AppName       string
		Namespace     string
		ResourceCount int64
		CPURequest    float64
		CPUUsage      float64
		MemoryRequest float64
		MemoryUsage   float64
		CPUCost       float64
		MemoryCost    float64
		StorageCost   float64
		TotalCost     float64
	}

	query := s.db.Model(&models.ResourceCost{}).
		Where("recorded_at BETWEEN ? AND ?", startTime, endTime).
		Where("app_name != ''")
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}

	query.Select(`
		app_name,
		namespace,
		COUNT(DISTINCT resource_name) as resource_count,
		SUM(cpu_request) as cpu_request,
		SUM(cpu_usage) as cpu_usage,
		SUM(memory_request) as memory_request,
		SUM(memory_usage) as memory_usage,
		SUM(cpu_cost) as cpu_cost,
		SUM(memory_cost) as memory_cost,
		SUM(storage_cost) as storage_cost,
		SUM(total_cost) as total_cost
	`).Group("app_name, namespace").Order("total_cost DESC").Limit(req.TopN).Scan(&results)

	var totalCost float64
	for _, r := range results {
		totalCost += r.TotalCost
	}

	items := make([]dto.AppCostItem, len(results))
	topApps := make([]string, 0)
	for i, r := range results {
		cpuUsageRate := 0.0
		if r.CPURequest > 0 {
			cpuUsageRate = r.CPUUsage / r.CPURequest * 100
		}
		memUsageRate := 0.0
		if r.MemoryRequest > 0 {
			memUsageRate = r.MemoryUsage / r.MemoryRequest * 100
		}
		efficiency := (cpuUsageRate + memUsageRate) / 2

		percentage := 0.0
		if totalCost > 0 {
			percentage = r.TotalCost / totalCost * 100
		}

		items[i] = dto.AppCostItem{
			AppName:         r.AppName,
			Namespace:       r.Namespace,
			ResourceCount:   int(r.ResourceCount),
			CPURequest:      r.CPURequest,
			CPUUsage:        r.CPUUsage,
			CPUUsageRate:    cpuUsageRate,
			MemoryRequest:   r.MemoryRequest,
			MemoryUsage:     r.MemoryUsage,
			MemoryUsageRate: memUsageRate,
			CPUCost:         r.CPUCost,
			MemoryCost:      r.MemoryCost,
			StorageCost:     r.StorageCost,
			TotalCost:       r.TotalCost,
			Percentage:      percentage,
			Efficiency:      efficiency,
		}
		if i < 5 {
			topApps = append(topApps, r.AppName)
		}
	}

	avgCost := 0.0
	if len(items) > 0 {
		avgCost = totalCost / float64(len(items))
	}

	return &dto.AppCostResponse{
		Items:       items,
		TotalCost:   totalCost,
		TotalApps:   len(items),
		AvgCost:     avgCost,
		TopCostApps: topApps,
	}, nil
}

// GetTeamCost 获取团队维度成本分析
func (s *CostService) GetTeamCost(ctx context.Context, req *dto.TeamCostRequest) (*dto.TeamCostResponse, error) {
	var startTime, endTime time.Time
	if req.StartTime != "" {
		startTime, _ = time.Parse("2006-01-02", req.StartTime)
	} else {
		startTime = time.Now().AddDate(0, 0, -30)
	}
	if req.EndTime != "" {
		endTime, _ = time.Parse("2006-01-02", req.EndTime)
	} else {
		endTime = time.Now()
	}

	var results []struct {
		TeamName      string
		AppCount      int64
		ResourceCount int64
		CPUCost       float64
		MemoryCost    float64
		StorageCost   float64
		TotalCost     float64
		AvgCPUUsage   float64
		AvgMemUsage   float64
	}

	query := s.db.Model(&models.ResourceCost{}).
		Where("recorded_at BETWEEN ? AND ?", startTime, endTime).
		Where("team_name != ''")
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}

	query.Select(`
		team_name,
		COUNT(DISTINCT app_name) as app_count,
		COUNT(DISTINCT resource_name) as resource_count,
		SUM(cpu_cost) as cpu_cost,
		SUM(memory_cost) as memory_cost,
		SUM(storage_cost) as storage_cost,
		SUM(total_cost) as total_cost,
		AVG(CASE WHEN cpu_request > 0 THEN cpu_usage / cpu_request * 100 ELSE 0 END) as avg_cpu_usage,
		AVG(CASE WHEN memory_request > 0 THEN memory_usage / memory_request * 100 ELSE 0 END) as avg_mem_usage
	`).Group("team_name").Order("total_cost DESC").Scan(&results)

	// 计算未分配到团队的成本
	var sharedCost float64
	sharedQuery := s.db.Model(&models.ResourceCost{}).
		Where("recorded_at BETWEEN ? AND ?", startTime, endTime).
		Where("team_name = '' OR team_name IS NULL")
	if req.ClusterID > 0 {
		sharedQuery = sharedQuery.Where("cluster_id = ?", req.ClusterID)
	}
	sharedQuery.Select("COALESCE(SUM(total_cost), 0)").Scan(&sharedCost)

	var totalCost float64
	for _, r := range results {
		totalCost += r.TotalCost
	}
	totalCost += sharedCost

	// 获取团队预算
	var budgets []models.CostBudget
	s.db.Where("team_name != ''").Find(&budgets)
	budgetMap := make(map[string]float64)
	for _, b := range budgets {
		budgetMap[b.TeamName] = b.MonthlyBudget
	}

	items := make([]dto.TeamCostItem, len(results))
	for i, r := range results {
		percentage := 0.0
		if totalCost > 0 {
			percentage = r.TotalCost / totalCost * 100
		}
		avgEfficiency := (r.AvgCPUUsage + r.AvgMemUsage) / 2
		wastedCost := r.TotalCost * (1 - avgEfficiency/100) * 0.5

		budgetUsed := 0.0
		monthlyBudget := budgetMap[r.TeamName]
		if monthlyBudget > 0 {
			budgetUsed = r.TotalCost / monthlyBudget * 100
		}

		items[i] = dto.TeamCostItem{
			TeamName:      r.TeamName,
			AppCount:      int(r.AppCount),
			ResourceCount: int(r.ResourceCount),
			CPUCost:       r.CPUCost,
			MemoryCost:    r.MemoryCost,
			StorageCost:   r.StorageCost,
			TotalCost:     r.TotalCost,
			Percentage:    percentage,
			AvgEfficiency: avgEfficiency,
			WastedCost:    wastedCost,
			BudgetUsed:    budgetUsed,
			MonthlyBudget: monthlyBudget,
		}
	}

	return &dto.TeamCostResponse{
		Items:      items,
		TotalCost:  totalCost,
		TotalTeams: len(items),
		SharedCost: sharedCost,
	}, nil
}

// GetNodeCost 获取节点成本分析
func (s *CostService) GetNodeCost(ctx context.Context, req *dto.NodeCostRequest) (*dto.NodeCostResponse, error) {
	// 获取成本配置
	var config models.CostConfig
	if err := s.db.Where("cluster_id = ?", req.ClusterID).First(&config).Error; err != nil {
		// 使用默认配置
		config = models.CostConfig{
			CPUPricePerCore:   0.1,
			MemoryPricePerGB:  0.05,
			StoragePricePerGB: 0.5,
		}
	}

	// 获取 K8s 客户端
	client, err := s.clientMgr.GetClient(ctx, req.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("获取K8s客户端失败: %v", err)
	}

	// 获取节点列表
	nodeList, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取节点列表失败: %v", err)
	}

	// 获取所有 Pod 用于计算已请求资源
	podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("获取Pod列表失败: %v", err)
	}

	// 按节点聚合 Pod 资源请求
	nodeRequested := make(map[string]struct {
		CPURequested    float64
		MemoryRequested float64
		PodCount        int
	})
	for _, pod := range podList.Items {
		if pod.Spec.NodeName == "" || pod.Status.Phase == "Succeeded" || pod.Status.Phase == "Failed" {
			continue
		}
		nodeName := pod.Spec.NodeName
		nodeData := nodeRequested[nodeName]
		nodeData.PodCount++
		for _, container := range pod.Spec.Containers {
			if container.Resources.Requests != nil {
				if cpu := container.Resources.Requests.Cpu(); cpu != nil {
					nodeData.CPURequested += float64(cpu.MilliValue()) / 1000
				}
				if mem := container.Resources.Requests.Memory(); mem != nil {
					nodeData.MemoryRequested += float64(mem.Value()) / (1024 * 1024 * 1024)
				}
			}
		}
		nodeRequested[nodeName] = nodeData
	}

	// 尝试获取节点指标（从 metrics-server）
	nodeMetrics := make(map[string]struct {
		CPUUsage    float64
		MemoryUsage float64
	})
	restConfig, _ := s.clientMgr.GetConfig(ctx, req.ClusterID)
	if restConfig != nil {
		metricsCollector := NewMetricsCollector()
		metrics, _ := metricsCollector.GetNodeMetrics(ctx, restConfig)
		for nodeName, m := range metrics {
			nodeMetrics[nodeName] = struct {
				CPUUsage    float64
				MemoryUsage float64
			}{
				CPUUsage:    m.CPUUsage,
				MemoryUsage: m.MemoryUsage,
			}
		}
	}

	items := make([]dto.NodeCostItem, 0, len(nodeList.Items))
	var totalCPU, totalMemory, totalCost float64
	var totalCPUUsage, totalMemoryUsage float64
	var underutilNodes int

	for _, node := range nodeList.Items {
		// 获取节点状态
		status := "Unknown"
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" {
				if cond.Status == "True" {
					status = "Ready"
				} else {
					status = "NotReady"
				}
				break
			}
		}

		// 获取节点角色
		nodeType := "worker"
		for label := range node.Labels {
			if label == "node-role.kubernetes.io/master" || label == "node-role.kubernetes.io/control-plane" {
				nodeType = "master"
				break
			}
		}

		// 获取节点 IP
		nodeIP := ""
		for _, addr := range node.Status.Addresses {
			if addr.Type == "InternalIP" {
				nodeIP = addr.Address
				break
			}
		}

		// 获取实例类型（从标签）
		instanceType := node.Labels["node.kubernetes.io/instance-type"]
		if instanceType == "" {
			instanceType = node.Labels["beta.kubernetes.io/instance-type"]
		}
		if instanceType == "" {
			instanceType = "standard"
		}

		// 解析资源容量
		cpuCapacity := float64(node.Status.Capacity.Cpu().MilliValue()) / 1000
		memoryCapacity := float64(node.Status.Capacity.Memory().Value()) / (1024 * 1024 * 1024)
		cpuAllocatable := float64(node.Status.Allocatable.Cpu().MilliValue()) / 1000
		memoryAllocatable := float64(node.Status.Allocatable.Memory().Value()) / (1024 * 1024 * 1024)
		podCapacity := int(node.Status.Capacity.Pods().Value())

		// 获取已请求资源
		requested := nodeRequested[node.Name]
		cpuRequested := requested.CPURequested
		memoryRequested := requested.MemoryRequested
		podCount := requested.PodCount

		// 获取实际使用率
		var cpuUsage, memoryUsage float64
		if m, ok := nodeMetrics[node.Name]; ok {
			cpuUsage = m.CPUUsage
			memoryUsage = m.MemoryUsage
		} else {
			// 如果没有指标，使用请求量估算
			cpuUsage = cpuRequested * 0.6
			memoryUsage = memoryRequested * 0.7
		}

		// 计算利用率
		cpuUsageRate := 0.0
		if cpuAllocatable > 0 {
			cpuUsageRate = cpuUsage / cpuAllocatable * 100
		}
		memoryUsageRate := 0.0
		if memoryAllocatable > 0 {
			memoryUsageRate = memoryUsage / memoryAllocatable * 100
		}

		// 计算节点成本（按小时）
		estimatedCost := cpuCapacity*config.CPUPricePerCore + memoryCapacity*config.MemoryPricePerGB

		// 计算效率评分
		efficiency := (cpuUsageRate + memoryUsageRate) / 2

		// 统计低利用率节点
		if efficiency < 30 {
			underutilNodes++
		}

		items = append(items, dto.NodeCostItem{
			NodeName:          node.Name,
			NodeIP:            nodeIP,
			NodeType:          nodeType,
			InstanceType:      instanceType,
			CPUCapacity:       cpuCapacity,
			CPUAllocatable:    cpuAllocatable,
			CPURequested:      cpuRequested,
			CPUUsage:          cpuUsage,
			CPUUsageRate:      cpuUsageRate,
			MemoryCapacity:    memoryCapacity,
			MemoryAllocatable: memoryAllocatable,
			MemoryRequested:   memoryRequested,
			MemoryUsage:       memoryUsage,
			MemoryUsageRate:   memoryUsageRate,
			PodCount:          podCount,
			PodCapacity:       podCapacity,
			EstimatedCost:     estimatedCost,
			Efficiency:        efficiency,
			Status:            status,
		})

		totalCPU += cpuCapacity
		totalMemory += memoryCapacity
		totalCost += estimatedCost
		totalCPUUsage += cpuUsageRate
		totalMemoryUsage += memoryUsageRate
	}

	// 计算平均利用率
	avgCPUUsage := 0.0
	avgMemoryUsage := 0.0
	if len(items) > 0 {
		avgCPUUsage = totalCPUUsage / float64(len(items))
		avgMemoryUsage = totalMemoryUsage / float64(len(items))
	}

	// 按成本排序
	sort.Slice(items, func(i, j int) bool {
		return items[i].EstimatedCost > items[j].EstimatedCost
	})

	return &dto.NodeCostResponse{
		Items:          items,
		TotalNodes:     len(items),
		TotalCPU:       totalCPU,
		TotalMemory:    totalMemory,
		AvgCPUUsage:    avgCPUUsage,
		AvgMemoryUsage: avgMemoryUsage,
		TotalCost:      totalCost,
		UnderutilNodes: underutilNodes,
	}, nil
}

// GetPVCCost 获取 PVC 存储成本分析
func (s *CostService) GetPVCCost(ctx context.Context, req *dto.PVCCostRequest) (*dto.PVCCostResponse, error) {
	// 从 ResourceCost 中提取存储相关数据
	var results []struct {
		Namespace    string
		ResourceName string
		StorageSize  float64
		StorageCost  float64
	}

	query := s.db.Model(&models.ResourceCost{}).
		Where("storage_size > 0").
		Where("recorded_at >= ?", time.Now().AddDate(0, 0, -1))
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}
	if req.Namespace != "" {
		query = query.Where("namespace = ?", req.Namespace)
	}

	query.Select(`
		namespace,
		resource_name,
		MAX(storage_size) as storage_size,
		MAX(storage_cost) as storage_cost
	`).Group("namespace, resource_name").Order("storage_cost DESC").Scan(&results)

	var totalCapacity, totalCost float64
	items := make([]dto.PVCCostItem, len(results))
	for i, r := range results {
		monthlyCost := r.StorageCost * 720 // 转换为月成本
		items[i] = dto.PVCCostItem{
			Namespace:   r.Namespace,
			PVCName:     r.ResourceName + "-pvc",
			Capacity:    r.StorageSize,
			MonthlyCost: monthlyCost,
			Status:      "Bound",
		}
		totalCapacity += r.StorageSize
		totalCost += monthlyCost
	}

	return &dto.PVCCostResponse{
		Items:         items,
		TotalPVCs:     len(items),
		TotalCapacity: totalCapacity,
		TotalCost:     totalCost,
	}, nil
}

// GetCostAllocationReport 获取成本分摊报表
func (s *CostService) GetCostAllocationReport(ctx context.Context, req *dto.CostAllocationReportRequest) (*dto.CostAllocationReportResponse, error) {
	startTime, _ := time.Parse("2006-01-02", req.StartTime)
	endTime, _ := time.Parse("2006-01-02", req.EndTime)
	if req.GroupBy == "" {
		req.GroupBy = "team"
	}

	var groupField string
	switch req.GroupBy {
	case "team":
		groupField = "team_name"
	case "namespace":
		groupField = "namespace"
	case "app":
		groupField = "app_name"
	default:
		groupField = "team_name"
	}

	var results []struct {
		Name          string
		CPUCost       float64
		MemoryCost    float64
		StorageCost   float64
		TotalCost     float64
		ResourceCount int64
		AvgCPUUsage   float64
		AvgMemUsage   float64
	}

	query := s.db.Model(&models.ResourceCost{}).
		Where("recorded_at BETWEEN ? AND ?", startTime, endTime)
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}

	query.Select(fmt.Sprintf(`
		%s as name,
		SUM(cpu_cost) as cpu_cost,
		SUM(memory_cost) as memory_cost,
		SUM(storage_cost) as storage_cost,
		SUM(total_cost) as total_cost,
		COUNT(DISTINCT resource_name) as resource_count,
		AVG(CASE WHEN cpu_request > 0 THEN cpu_usage / cpu_request * 100 ELSE 0 END) as avg_cpu_usage,
		AVG(CASE WHEN memory_request > 0 THEN memory_usage / memory_request * 100 ELSE 0 END) as avg_mem_usage
	`, groupField)).Group(groupField).Order("total_cost DESC").Scan(&results)

	// 计算总成本和未分配成本
	var totalDirectCost, unallocatedCost float64
	for _, r := range results {
		if r.Name == "" {
			unallocatedCost += r.TotalCost
		} else {
			totalDirectCost += r.TotalCost
		}
	}
	totalCost := totalDirectCost + unallocatedCost

	// 计算公共成本分摊
	sharedCost := 0.0
	if req.IncludeShared && unallocatedCost > 0 && len(results) > 0 {
		sharedCost = unallocatedCost
	}

	items := make([]dto.CostAllocationReportItem, 0)
	for _, r := range results {
		if r.Name == "" {
			continue // 跳过未分配的
		}
		percentage := 0.0
		if totalCost > 0 {
			percentage = r.TotalCost / totalCost * 100
		}

		// 按比例分摊公共成本
		allocatedShared := 0.0
		if req.IncludeShared && totalDirectCost > 0 {
			allocatedShared = sharedCost * (r.TotalCost / totalDirectCost)
		}

		avgEfficiency := (r.AvgCPUUsage + r.AvgMemUsage) / 2

		items = append(items, dto.CostAllocationReportItem{
			Name:          r.Name,
			DirectCost:    r.TotalCost,
			SharedCost:    allocatedShared,
			TotalCost:     r.TotalCost + allocatedShared,
			CPUCost:       r.CPUCost,
			MemoryCost:    r.MemoryCost,
			StorageCost:   r.StorageCost,
			Percentage:    percentage,
			ResourceCount: int(r.ResourceCount),
			AvgEfficiency: avgEfficiency,
		})
	}

	return &dto.CostAllocationReportResponse{
		StartTime:       req.StartTime,
		EndTime:         req.EndTime,
		GroupBy:         req.GroupBy,
		TotalCost:       totalCost,
		DirectCost:      totalDirectCost,
		SharedCost:      sharedCost,
		Items:           items,
		UnallocatedCost: unallocatedCost,
	}, nil
}

// GetEnvCost 获取环境维度成本分析
func (s *CostService) GetEnvCost(ctx context.Context, req *dto.EnvCostRequest) (*dto.EnvCostResponse, error) {
	var startTime, endTime time.Time
	if req.StartTime != "" {
		startTime, _ = time.Parse("2006-01-02", req.StartTime)
	} else {
		startTime = time.Now().AddDate(0, 0, -30)
	}
	if req.EndTime != "" {
		endTime, _ = time.Parse("2006-01-02", req.EndTime)
	} else {
		endTime = time.Now()
	}

	// 根据命名空间推断环境（常见命名规则）
	envPatterns := map[string][]string{
		"dev":     {"dev", "development", "develop"},
		"test":    {"test", "testing", "qa"},
		"staging": {"staging", "uat", "pre", "preprod"},
		"prod":    {"prod", "production", "prd", "live"},
	}

	var results []struct {
		Namespace   string
		AppCount    int64
		CPUCost     float64
		MemoryCost  float64
		StorageCost float64
		TotalCost   float64
		AvgCPUUsage float64
		AvgMemUsage float64
	}

	query := s.db.Model(&models.ResourceCost{}).
		Where("recorded_at BETWEEN ? AND ?", startTime, endTime)
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}

	query.Select(`
		namespace,
		COUNT(DISTINCT app_name) as app_count,
		SUM(cpu_cost) as cpu_cost,
		SUM(memory_cost) as memory_cost,
		SUM(storage_cost) as storage_cost,
		SUM(total_cost) as total_cost,
		AVG(CASE WHEN cpu_request > 0 THEN cpu_usage / cpu_request * 100 ELSE 0 END) as avg_cpu_usage,
		AVG(CASE WHEN memory_request > 0 THEN memory_usage / memory_request * 100 ELSE 0 END) as avg_mem_usage
	`).Group("namespace").Scan(&results)

	// 按环境聚合
	envData := make(map[string]*dto.EnvCostItem)
	for env := range envPatterns {
		envData[env] = &dto.EnvCostItem{Environment: env}
	}
	envData["other"] = &dto.EnvCostItem{Environment: "other"}

	var totalCost float64
	for _, r := range results {
		env := "other"
		nsLower := strings.ToLower(r.Namespace)
		for envName, patterns := range envPatterns {
			for _, p := range patterns {
				if strings.Contains(nsLower, p) {
					env = envName
					break
				}
			}
			if env != "other" {
				break
			}
		}

		item := envData[env]
		item.NamespaceCount++
		item.AppCount += int(r.AppCount)
		item.CPUCost += r.CPUCost
		item.MemoryCost += r.MemoryCost
		item.StorageCost += r.StorageCost
		item.TotalCost += r.TotalCost
		totalCost += r.TotalCost
	}

	items := make([]dto.EnvCostItem, 0)
	for _, item := range envData {
		if item.TotalCost > 0 {
			if totalCost > 0 {
				item.Percentage = item.TotalCost / totalCost * 100
			}
			items = append(items, *item)
		}
	}

	// 按成本排序
	sort.Slice(items, func(i, j int) bool {
		return items[i].TotalCost > items[j].TotalCost
	})

	return &dto.EnvCostResponse{
		Items:     items,
		TotalCost: totalCost,
	}, nil
}
