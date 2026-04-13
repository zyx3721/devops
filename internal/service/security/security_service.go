package security

import (
	"context"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
)

// SecurityService 安全服务
type SecurityService struct {
	db            *gorm.DB
	imageScanner  *ImageScannerService
	configChecker *ConfigCheckerService
	auditLogger   *AuditLoggerService
}

// NewSecurityService 创建安全服务
func NewSecurityService(db *gorm.DB) *SecurityService {
	return &SecurityService{
		db:            db,
		imageScanner:  NewImageScannerService(db),
		configChecker: NewConfigCheckerService(db),
		auditLogger:   NewAuditLoggerService(db),
	}
}

// GetDB 获取数据库连接
func (s *SecurityService) GetDB() *gorm.DB {
	return s.db
}

// GetImageScanner 获取镜像扫描服务
func (s *SecurityService) GetImageScanner() *ImageScannerService {
	return s.imageScanner
}

// GetConfigChecker 获取配置检查服务
func (s *SecurityService) GetConfigChecker() *ConfigCheckerService {
	return s.configChecker
}

// GetAuditLogger 获取审计日志服务
func (s *SecurityService) GetAuditLogger() *AuditLoggerService {
	return s.auditLogger
}

// GetOverview 获取安全概览
func (s *SecurityService) GetOverview(ctx context.Context, clusterID uint) (*dto.SecurityOverviewResponse, error) {
	result := &dto.SecurityOverviewResponse{
		RecentScans:  make([]dto.ImageScanItem, 0),
		RecentChecks: make([]dto.ConfigCheckItem, 0),
		TrendData:    make([]dto.SecurityTrendPoint, 0),
	}

	// 获取漏洞统计
	var vulnStats struct {
		Critical int
		High     int
		Medium   int
		Low      int
	}
	s.db.Model(&models.ImageScan{}).
		Select("COALESCE(SUM(critical_count), 0) as critical, COALESCE(SUM(high_count), 0) as high, COALESCE(SUM(medium_count), 0) as medium, COALESCE(SUM(low_count), 0) as low").
		Where("status = ?", "completed").
		Where("scanned_at >= ?", time.Now().AddDate(0, 0, -30)).
		Scan(&vulnStats)

	result.VulnSummary = dto.VulnSummary{
		Critical: vulnStats.Critical,
		High:     vulnStats.High,
		Medium:   vulnStats.Medium,
		Low:      vulnStats.Low,
		Total:    vulnStats.Critical + vulnStats.High + vulnStats.Medium + vulnStats.Low,
	}

	// 获取配置检查统计
	var configStats struct {
		Critical int
		High     int
		Medium   int
		Low      int
		Passed   int
	}
	query := s.db.Model(&models.ConfigCheck{}).
		Select("COALESCE(SUM(critical_count), 0) as critical, COALESCE(SUM(high_count), 0) as high, COALESCE(SUM(medium_count), 0) as medium, COALESCE(SUM(low_count), 0) as low, COALESCE(SUM(passed_count), 0) as passed").
		Where("status = ?", "completed").
		Where("checked_at >= ?", time.Now().AddDate(0, 0, -30))
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	query.Scan(&configStats)

	result.ConfigSummary = dto.ConfigCheckSummary{
		Critical: configStats.Critical,
		High:     configStats.High,
		Medium:   configStats.Medium,
		Low:      configStats.Low,
		Passed:   configStats.Passed,
		Total:    configStats.Critical + configStats.High + configStats.Medium + configStats.Low,
	}

	// 计算安全评分
	result.SecurityScore = s.calculateSecurityScore(result.VulnSummary, result.ConfigSummary)
	result.RiskLevel = s.getRiskLevel(result.SecurityScore)

	// 获取最近扫描记录
	var recentScans []models.ImageScan
	s.db.Where("status = ?", "completed").
		Order("scanned_at DESC").
		Limit(5).
		Find(&recentScans)

	for _, scan := range recentScans {
		result.RecentScans = append(result.RecentScans, dto.ImageScanItem{
			ID:            scan.ID,
			Image:         scan.Image,
			Status:        scan.Status,
			RiskLevel:     scan.RiskLevel,
			CriticalCount: scan.CriticalCount,
			HighCount:     scan.HighCount,
			MediumCount:   scan.MediumCount,
			LowCount:      scan.LowCount,
			ScannedAt:     scan.ScannedAt,
			CreatedAt:     scan.CreatedAt,
		})
	}

	// 获取最近配置检查
	var recentChecks []models.ConfigCheck
	checkQuery := s.db.Where("status = ?", "completed").Order("checked_at DESC").Limit(5)
	if clusterID > 0 {
		checkQuery = checkQuery.Where("cluster_id = ?", clusterID)
	}
	checkQuery.Find(&recentChecks)

	for _, check := range recentChecks {
		result.RecentChecks = append(result.RecentChecks, dto.ConfigCheckItem{
			ID:            check.ID,
			ClusterID:     check.ClusterID,
			Namespace:     check.Namespace,
			Status:        check.Status,
			CriticalCount: check.CriticalCount,
			HighCount:     check.HighCount,
			MediumCount:   check.MediumCount,
			LowCount:      check.LowCount,
			PassedCount:   check.PassedCount,
			CheckedAt:     check.CheckedAt,
		})
	}

	// 获取趋势数据（最近7天）
	for i := 6; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.AddDate(0, 0, 1)

		var vulnCount int64
		s.db.Model(&models.ImageScan{}).
			Where("scanned_at >= ? AND scanned_at < ?", startOfDay, endOfDay).
			Where("status = ?", "completed").
			Select("COALESCE(SUM(critical_count + high_count + medium_count + low_count), 0)").
			Scan(&vulnCount)

		var issueCount int64
		issueQuery := s.db.Model(&models.ConfigCheck{}).
			Where("checked_at >= ? AND checked_at < ?", startOfDay, endOfDay).
			Where("status = ?", "completed")
		if clusterID > 0 {
			issueQuery = issueQuery.Where("cluster_id = ?", clusterID)
		}
		issueQuery.Select("COALESCE(SUM(critical_count + high_count + medium_count + low_count), 0)").Scan(&issueCount)

		result.TrendData = append(result.TrendData, dto.SecurityTrendPoint{
			Date:       dateStr,
			VulnCount:  int(vulnCount),
			IssueCount: int(issueCount),
		})
	}

	return result, nil
}

// calculateSecurityScore 计算安全评分
func (s *SecurityService) calculateSecurityScore(vuln dto.VulnSummary, config dto.ConfigCheckSummary) int {
	score := 100

	// 漏洞扣分
	score -= vuln.Critical * 10
	score -= vuln.High * 5
	score -= vuln.Medium * 2
	score -= vuln.Low * 1

	// 配置问题扣分
	score -= config.Critical * 8
	score -= config.High * 4
	score -= config.Medium * 2
	score -= config.Low * 1

	if score < 0 {
		score = 0
	}
	return score
}

// getRiskLevel 获取风险等级
func (s *SecurityService) getRiskLevel(score int) string {
	switch {
	case score >= 90:
		return "low"
	case score >= 70:
		return "medium"
	case score >= 50:
		return "high"
	default:
		return "critical"
	}
}
