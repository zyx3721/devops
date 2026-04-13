package security

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// ImageScannerService 镜像扫描服务
type ImageScannerService struct {
	db      *gorm.DB
	scanner *TrivyScanner
}

// NewImageScannerService 创建镜像扫描服务
func NewImageScannerService(db *gorm.DB) *ImageScannerService {
	return &ImageScannerService{
		db:      db,
		scanner: NewTrivyScanner(),
	}
}

// ScanImage 扫描镜像
func (s *ImageScannerService) ScanImage(ctx context.Context, req *dto.ScanImageRequest) (*dto.ScanResultResponse, error) {
	log := logger.L().WithField("image", req.Image)
	log.Info("开始扫描镜像")

	// 创建扫描记录
	scan := &models.ImageScan{
		Image:      req.Image,
		RegistryID: &req.RegistryID,
		Status:     "scanning",
		CreatedAt:  time.Now(),
	}
	if req.RegistryID == 0 {
		scan.RegistryID = nil
	}

	if err := s.db.Create(scan).Error; err != nil {
		log.WithField("error", err).Error("创建扫描记录失败")
		return nil, err
	}

	// 获取仓库凭证
	var registry *models.ImageRegistry
	if req.RegistryID > 0 {
		registry = &models.ImageRegistry{}
		if err := s.db.First(registry, req.RegistryID).Error; err != nil {
			log.WithField("error", err).Warn("获取仓库配置失败")
		}
	}

	// 执行扫描
	result, err := s.scanner.Scan(ctx, req.Image, registry)
	now := time.Now()

	if err != nil {
		log.WithField("error", err).Error("扫描镜像失败")
		scan.Status = "failed"
		scan.ErrorMessage = err.Error()
		scan.ScannedAt = &now
		s.db.Save(scan)

		return &dto.ScanResultResponse{
			ID:           scan.ID,
			Image:        scan.Image,
			Status:       scan.Status,
			ErrorMessage: scan.ErrorMessage,
			ScannedAt:    scan.ScannedAt,
		}, nil
	}

	// 更新扫描结果
	scan.Status = "completed"
	scan.RiskLevel = result.RiskLevel
	scan.CriticalCount = result.VulnSummary.Critical
	scan.HighCount = result.VulnSummary.High
	scan.MediumCount = result.VulnSummary.Medium
	scan.LowCount = result.VulnSummary.Low
	scan.ScannedAt = &now

	// 保存详细结果
	resultJSON, _ := json.Marshal(result.Vulnerabilities)
	scan.ResultJSON = string(resultJSON)

	if err := s.db.Save(scan).Error; err != nil {
		log.WithField("error", err).Error("保存扫描结果失败")
		return nil, err
	}

	log.WithField("risk_level", result.RiskLevel).Info("镜像扫描完成")

	return &dto.ScanResultResponse{
		ID:              scan.ID,
		Image:           scan.Image,
		Status:          scan.Status,
		RiskLevel:       scan.RiskLevel,
		VulnSummary:     result.VulnSummary,
		Vulnerabilities: result.Vulnerabilities,
		ScannedAt:       scan.ScannedAt,
	}, nil
}

// GetScanHistory 获取扫描历史
func (s *ImageScannerService) GetScanHistory(ctx context.Context, req *dto.ScanHistoryRequest) (*dto.ScanHistoryResponse, error) {
	var scans []models.ImageScan
	var total int64

	query := s.db.Model(&models.ImageScan{})

	if req.Image != "" {
		query = query.Where("image LIKE ?", "%"+req.Image+"%")
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	query.Count(&total)

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&scans)

	items := make([]dto.ImageScanItem, 0, len(scans))
	for _, scan := range scans {
		items = append(items, dto.ImageScanItem{
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

	return &dto.ScanHistoryResponse{
		Total: int(total),
		Items: items,
	}, nil
}

// GetScanResult 获取扫描结果
func (s *ImageScannerService) GetScanResult(ctx context.Context, scanID uint) (*dto.ScanResultResponse, error) {
	var scan models.ImageScan
	if err := s.db.First(&scan, scanID).Error; err != nil {
		return nil, err
	}

	result := &dto.ScanResultResponse{
		ID:        scan.ID,
		Image:     scan.Image,
		Status:    scan.Status,
		RiskLevel: scan.RiskLevel,
		VulnSummary: dto.VulnSummary{
			Critical: scan.CriticalCount,
			High:     scan.HighCount,
			Medium:   scan.MediumCount,
			Low:      scan.LowCount,
			Total:    scan.CriticalCount + scan.HighCount + scan.MediumCount + scan.LowCount,
		},
		ScannedAt:    scan.ScannedAt,
		ErrorMessage: scan.ErrorMessage,
	}

	// 解析漏洞详情
	if scan.ResultJSON != "" {
		var vulns []dto.Vulnerability
		if err := json.Unmarshal([]byte(scan.ResultJSON), &vulns); err == nil {
			result.Vulnerabilities = vulns
		}
	}

	return result, nil
}
