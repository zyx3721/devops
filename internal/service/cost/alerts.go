package cost

import (
	"context"
	"time"

	"devops/internal/models"
	"devops/pkg/dto"
)

// GetAlerts 获取成本告警列表
func (s *CostService) GetAlerts(ctx context.Context, clusterID uint, status string) ([]dto.CostAlertItem, error) {
	var alerts []models.CostAlert
	query := s.db.Model(&models.CostAlert{})
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Order("created_at DESC").Limit(100).Find(&alerts)

	items := make([]dto.CostAlertItem, len(alerts))
	for i, a := range alerts {
		items[i] = dto.CostAlertItem{
			ID:          a.ID,
			ClusterID:   a.ClusterID,
			AlertType:   a.AlertType,
			Severity:    a.Severity,
			Title:       a.Title,
			Message:     a.Message,
			Threshold:   a.Threshold,
			ActualValue: a.ActualValue,
			Status:      a.Status,
			CreatedAt:   a.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}
	return items, nil
}

// AcknowledgeAlert 确认告警
func (s *CostService) AcknowledgeAlert(ctx context.Context, alertID uint, userID uint) error {
	now := time.Now()
	return s.db.Model(&models.CostAlert{}).Where("id = ?", alertID).Updates(map[string]interface{}{
		"status":          "acknowledged",
		"acknowledged_at": now,
		"acknowledged_by": userID,
	}).Error
}
