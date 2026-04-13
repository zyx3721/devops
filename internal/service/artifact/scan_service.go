// Package artifact 制品管理服务
// 本文件实现制品扫描服务
package artifact

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"devops/internal/models/artifact"
	"devops/pkg/logger"
)

// ScanService 扫描服务
type ScanService struct {
	db *gorm.DB
}

// NewScanService 创建扫描服务
func NewScanService(db *gorm.DB) *ScanService {
	return &ScanService{db: db}
}

// ScanRequest 扫描请求
type ScanRequest struct {
	VersionID uint64   `json:"version_id"`
	ScanTypes []string `json:"scan_types"` // vulnerability, license, quality
	Scanner   string   `json:"scanner"`    // trivy, sonarqube, etc.
	Async     bool     `json:"async"`
}

// StartScan 开始扫描
func (s *ScanService) StartScan(ctx context.Context, req *ScanRequest) error {
	log := logger.L().WithField("version_id", req.VersionID)

	// 更新版本扫描状态
	s.db.Model(&artifact.ArtifactVersion{}).Where("id = ?", req.VersionID).
		Update("scan_status", "scanning")

	if req.Async {
		// 异步扫描
		go s.doScan(context.Background(), req)
		log.Info("开始异步扫描")
		return nil
	}

	// 同步扫描
	return s.doScan(ctx, req)
}

// doScan 执行扫描
func (s *ScanService) doScan(ctx context.Context, req *ScanRequest) error {
	log := logger.L().WithField("version_id", req.VersionID)

	var overallStatus = "passed"
	var totalCritical, totalHigh, totalMedium, totalLow int

	for _, scanType := range req.ScanTypes {
		result, err := s.performScan(ctx, req.VersionID, scanType, req.Scanner)
		if err != nil {
			log.WithError(err).WithField("scan_type", scanType).Error("扫描失败")
			continue
		}

		// 保存扫描结果
		if err := s.db.Create(result).Error; err != nil {
			log.WithError(err).WithField("scan_type", scanType).Error("保存扫描结果失败")
			continue
		}

		// 汇总结果
		totalCritical += result.CriticalCount
		totalHigh += result.HighCount
		totalMedium += result.MediumCount
		totalLow += result.LowCount

		if result.Status == "failed" {
			overallStatus = "failed"
		} else if result.Status == "warning" && overallStatus != "failed" {
			overallStatus = "warning"
		}
	}

	// 更新版本扫描状态
	scanResult := map[string]any{
		"critical": totalCritical,
		"high":     totalHigh,
		"medium":   totalMedium,
		"low":      totalLow,
	}
	scanResultJSON, _ := json.Marshal(scanResult)

	s.db.Model(&artifact.ArtifactVersion{}).Where("id = ?", req.VersionID).
		Updates(map[string]any{
			"scan_status": overallStatus,
			"scan_result": string(scanResultJSON),
		})

	log.WithField("status", overallStatus).Info("扫描完成")
	return nil
}

// performScan 执行单项扫描
func (s *ScanService) performScan(ctx context.Context, versionID uint64, scanType, scanner string) (*artifact.ArtifactScanResult, error) {
	result := &artifact.ArtifactScanResult{
		VersionID: versionID,
		ScanType:  scanType,
		Scanner:   scanner,
		ScannedAt: time.Now(),
		CreatedAt: time.Now(),
	}

	// 根据扫描类型执行不同的扫描逻辑
	switch scanType {
	case "vulnerability":
		return s.scanVulnerability(ctx, result)
	case "license":
		return s.scanLicense(ctx, result)
	case "quality":
		return s.scanQuality(ctx, result)
	default:
		return nil, fmt.Errorf("不支持的扫描类型: %s", scanType)
	}
}

// scanVulnerability 漏洞扫描
func (s *ScanService) scanVulnerability(ctx context.Context, result *artifact.ArtifactScanResult) (*artifact.ArtifactScanResult, error) {
	// TODO: 集成 Trivy 或其他漏洞扫描工具
	// 这里模拟扫描结果
	result.Status = "passed"
	result.CriticalCount = 0
	result.HighCount = 0
	result.MediumCount = 2
	result.LowCount = 5

	details := map[string]any{
		"scanner_version": "0.48.0",
		"scan_duration":   "15s",
		"vulnerabilities": []map[string]any{
			{"id": "CVE-2023-1234", "severity": "MEDIUM", "package": "openssl", "version": "1.1.1"},
			{"id": "CVE-2023-5678", "severity": "MEDIUM", "package": "curl", "version": "7.88.0"},
		},
	}
	detailsJSON, _ := json.Marshal(details)
	result.Details = string(detailsJSON)

	if result.CriticalCount > 0 || result.HighCount > 0 {
		result.Status = "failed"
	} else if result.MediumCount > 0 {
		result.Status = "warning"
	}

	return result, nil
}

// scanLicense 许可证扫描
func (s *ScanService) scanLicense(ctx context.Context, result *artifact.ArtifactScanResult) (*artifact.ArtifactScanResult, error) {
	// TODO: 集成许可证扫描工具
	result.Status = "passed"

	details := map[string]any{
		"licenses": []map[string]any{
			{"name": "MIT", "count": 45, "risk": "low"},
			{"name": "Apache-2.0", "count": 30, "risk": "low"},
			{"name": "GPL-3.0", "count": 2, "risk": "high"},
		},
	}
	detailsJSON, _ := json.Marshal(details)
	result.Details = string(detailsJSON)

	return result, nil
}

// scanQuality 质量扫描
func (s *ScanService) scanQuality(ctx context.Context, result *artifact.ArtifactScanResult) (*artifact.ArtifactScanResult, error) {
	// TODO: 集成 SonarQube 或其他代码质量工具
	result.Status = "passed"

	details := map[string]any{
		"code_smells":     15,
		"bugs":            2,
		"vulnerabilities": 0,
		"coverage":        78.5,
		"duplications":    3.2,
	}
	detailsJSON, _ := json.Marshal(details)
	result.Details = string(detailsJSON)

	return result, nil
}

// GetScanResults 获取扫描结果
func (s *ScanService) GetScanResults(ctx context.Context, versionID uint64) ([]artifact.ArtifactScanResult, error) {
	var results []artifact.ArtifactScanResult
	err := s.db.Where("version_id = ?", versionID).Order("scanned_at DESC").Find(&results).Error
	return results, err
}

// GetLatestScanResult 获取最新扫描结果
func (s *ScanService) GetLatestScanResult(ctx context.Context, versionID uint64, scanType string) (*artifact.ArtifactScanResult, error) {
	var result artifact.ArtifactScanResult
	err := s.db.Where("version_id = ? AND scan_type = ?", versionID, scanType).
		Order("scanned_at DESC").First(&result).Error
	return &result, err
}

// GetScanStats 获取扫描统计
func (s *ScanService) GetScanStats(ctx context.Context, repoID uint64) (*ScanStats, error) {
	stats := &ScanStats{}

	// 获取仓库下所有版本
	var versionIDs []uint64
	s.db.Model(&artifact.ArtifactVersion{}).
		Joins("JOIN artifacts ON artifacts.id = artifact_versions.artifact_id").
		Where("artifacts.repository_id = ?", repoID).
		Pluck("artifact_versions.id", &versionIDs)

	if len(versionIDs) == 0 {
		return stats, nil
	}

	// 统计扫描状态
	s.db.Model(&artifact.ArtifactVersion{}).
		Where("id IN ?", versionIDs).
		Where("scan_status = ?", "passed").
		Count(&stats.PassedCount)

	s.db.Model(&artifact.ArtifactVersion{}).
		Where("id IN ?", versionIDs).
		Where("scan_status = ?", "failed").
		Count(&stats.FailedCount)

	s.db.Model(&artifact.ArtifactVersion{}).
		Where("id IN ?", versionIDs).
		Where("scan_status = ?", "warning").
		Count(&stats.WarningCount)

	s.db.Model(&artifact.ArtifactVersion{}).
		Where("id IN ?", versionIDs).
		Where("scan_status = ?", "pending").
		Count(&stats.PendingCount)

	// 统计漏洞数
	s.db.Model(&artifact.ArtifactScanResult{}).
		Where("version_id IN ? AND scan_type = ?", versionIDs, "vulnerability").
		Select("COALESCE(SUM(critical_count), 0)").Scan(&stats.TotalCritical)

	s.db.Model(&artifact.ArtifactScanResult{}).
		Where("version_id IN ? AND scan_type = ?", versionIDs, "vulnerability").
		Select("COALESCE(SUM(high_count), 0)").Scan(&stats.TotalHigh)

	return stats, nil
}

// ScanStats 扫描统计
type ScanStats struct {
	PassedCount   int64 `json:"passed_count"`
	FailedCount   int64 `json:"failed_count"`
	WarningCount  int64 `json:"warning_count"`
	PendingCount  int64 `json:"pending_count"`
	TotalCritical int   `json:"total_critical"`
	TotalHigh     int   `json:"total_high"`
}
