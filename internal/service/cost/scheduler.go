package cost

import (
	"context"
	"sync"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/internal/service/kubernetes"
	"devops/pkg/logger"
)

// CostScheduler 成本任务调度器
type CostScheduler struct {
	db        *gorm.DB
	collector *CostCollector
	alerter   *CostAlerter
	log       *logger.Logger
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

// NewCostScheduler 创建成本任务调度器
func NewCostScheduler(db *gorm.DB, clientMgr *kubernetes.K8sClientManager) *CostScheduler {
	return &CostScheduler{
		db:        db,
		collector: NewCostCollector(db, clientMgr),
		alerter:   NewCostAlerter(db),
		log:       logger.NewLogger("CostScheduler"),
		stopCh:    make(chan struct{}),
	}
}

// Start 启动调度器
func (s *CostScheduler) Start() {
	s.log.Info("成本任务调度器启动")

	// 启动成本采集任务（每小时）
	s.wg.Add(1)
	go s.runCollectTask()

	// 启动预算检查任务（每30分钟）
	s.wg.Add(1)
	go s.runBudgetCheckTask()

	// 启动僵尸资源检测任务（每天凌晨2点）
	s.wg.Add(1)
	go s.runZombieDetectionTask()

	// 启动数据清理任务（每天凌晨3点）
	s.wg.Add(1)
	go s.runCleanupTask()
}

// Stop 停止调度器
func (s *CostScheduler) Stop() {
	s.log.Info("成本任务调度器停止中...")
	close(s.stopCh)
	s.wg.Wait()
	s.log.Info("成本任务调度器已停止")
}

// runCollectTask 运行成本采集任务
func (s *CostScheduler) runCollectTask() {
	defer s.wg.Done()
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	// 启动时立即执行一次
	s.doCollect()

	for {
		select {
		case <-ticker.C:
			s.doCollect()
		case <-s.stopCh:
			return
		}
	}
}

func (s *CostScheduler) doCollect() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	s.log.Info("开始采集成本数据")
	if err := s.collector.CollectAll(ctx); err != nil {
		s.log.WithField("error", err.Error()).Error("采集成本数据失败")
	} else {
		s.log.Info("采集成本数据完成")
	}
}

// runBudgetCheckTask 运行预算检查任务
func (s *CostScheduler) runBudgetCheckTask() {
	defer s.wg.Done()
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.doBudgetCheck()
		case <-s.stopCh:
			return
		}
	}
}

func (s *CostScheduler) doBudgetCheck() {
	ctx := context.Background()
	s.log.Info("开始检查预算")

	var budgets []models.CostBudget
	s.db.Find(&budgets)

	for _, budget := range budgets {
		// 计算当月成本
		monthStart := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.UTC)
		var currentCost float64
		query := s.db.Model(&models.ResourceCost{}).
			Where("recorded_at >= ?", monthStart).
			Where("cluster_id = ?", budget.ClusterID)
		if budget.Namespace != "" {
			query = query.Where("namespace = ?", budget.Namespace)
		}
		query.Select("COALESCE(SUM(total_cost), 0)").Scan(&currentCost)

		// 更新预算状态
		usagePercent := 0.0
		if budget.MonthlyBudget > 0 {
			usagePercent = currentCost / budget.MonthlyBudget * 100
		}

		status := "normal"
		if usagePercent >= 100 {
			status = "exceeded"
		} else if usagePercent >= budget.AlertThreshold {
			status = "warning"
		}

		// 状态变化时发送告警
		if status != budget.Status {
			if status == "warning" {
				s.alerter.SendBudgetWarning(ctx, &budget, currentCost, usagePercent)
			} else if status == "exceeded" {
				s.alerter.SendBudgetExceeded(ctx, &budget, currentCost, usagePercent)
			}
		}

		// 更新数据库
		s.db.Model(&budget).Updates(map[string]interface{}{
			"current_cost":  currentCost,
			"usage_percent": usagePercent,
			"status":        status,
		})
	}

	s.log.Info("预算检查完成")
}

// runZombieDetectionTask 运行僵尸资源检测任务
func (s *CostScheduler) runZombieDetectionTask() {
	defer s.wg.Done()

	// 计算到凌晨2点的时间
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 2, 0, 0, 0, now.Location())
	timer := time.NewTimer(next.Sub(now))

	for {
		select {
		case <-timer.C:
			s.doZombieDetection()
			// 重置定时器到下一个凌晨2点
			next = next.Add(24 * time.Hour)
			timer.Reset(next.Sub(time.Now()))
		case <-s.stopCh:
			timer.Stop()
			return
		}
	}
}

func (s *CostScheduler) doZombieDetection() {
	s.log.Info("开始检测僵尸资源")

	// 查找7天内没有活动的资源
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	// 更新资源活跃度
	s.db.Model(&models.ResourceActivity{}).
		Where("last_active_at < ? OR last_active_at IS NULL", sevenDaysAgo).
		Where("cpu_usage_avg < 1 AND memory_usage_avg < 1").
		Updates(map[string]interface{}{
			"is_zombie": true,
			"idle_days": gorm.Expr("DATEDIFF(NOW(), COALESCE(last_active_at, created_at))"),
		})

	s.log.Info("僵尸资源检测完成")
}

// runCleanupTask 运行数据清理任务
func (s *CostScheduler) runCleanupTask() {
	defer s.wg.Done()

	// 计算到凌晨3点的时间
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day()+1, 3, 0, 0, 0, now.Location())
	timer := time.NewTimer(next.Sub(now))

	for {
		select {
		case <-timer.C:
			s.doCleanup()
			next = next.Add(24 * time.Hour)
			timer.Reset(next.Sub(time.Now()))
		case <-s.stopCh:
			timer.Stop()
			return
		}
	}
}

func (s *CostScheduler) doCleanup() {
	s.log.Info("开始清理历史数据")

	// 删除90天前的详细数据
	ninetyDaysAgo := time.Now().AddDate(0, 0, -90)
	result := s.db.Where("recorded_at < ?", ninetyDaysAgo).Delete(&models.ResourceCost{})
	s.log.WithField("deleted", result.RowsAffected).Info("清理资源成本数据")

	// 删除365天前的汇总数据
	oneYearAgo := time.Now().AddDate(-1, 0, 0)
	result = s.db.Where("period_start < ?", oneYearAgo).Delete(&models.CostSummary{})
	s.log.WithField("deleted", result.RowsAffected).Info("清理成本汇总数据")

	// 删除30天前已处理的建议
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	result = s.db.Where("status IN (?, ?) AND updated_at < ?", "applied", "ignored", thirtyDaysAgo).
		Delete(&models.CostSuggestion{})
	s.log.WithField("deleted", result.RowsAffected).Info("清理优化建议数据")

	s.log.Info("历史数据清理完成")
}
