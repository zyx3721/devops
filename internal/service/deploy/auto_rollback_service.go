package deploy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"devops/internal/models"
	"devops/internal/service/kubernetes"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
)

// AutoRollbackService 自动回滚服务
type AutoRollbackService struct {
	db        *gorm.DB
	clientMgr *kubernetes.K8sClientManager
}

// NewAutoRollbackService 创建自动回滚服务
func NewAutoRollbackService(db *gorm.DB, clientMgr *kubernetes.K8sClientManager) *AutoRollbackService {
	return &AutoRollbackService{db: db, clientMgr: clientMgr}
}

// RollbackConfigRecord 回滚配置记录
type RollbackConfigRecord struct {
	ID                uint       `gorm:"primarykey" json:"id"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
	DeployRecordID    uint       `gorm:"uniqueIndex" json:"deploy_record_id"`
	Enabled           bool       `json:"enabled"`
	HealthCheckPeriod int        `json:"health_check_period"`
	FailureThreshold  int        `json:"failure_threshold"`
	SuccessThreshold  int        `json:"success_threshold"`
	ConsecutiveFails  int        `json:"consecutive_fails"`
	LastCheckTime     *time.Time `json:"last_check_time"`
}

// GetHealthStatus 获取部署健康状态
func (s *AutoRollbackService) GetHealthStatus(ctx context.Context, recordID uint) (*dto.DeployHealthStatus, error) {
	var record models.DeployRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "部署记录不存在")
	}

	var app models.Application
	if err := s.db.First(&app, record.ApplicationID).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeNotFound, "应用不存在")
	}

	status := &dto.DeployHealthStatus{
		DeployRecordID: recordID,
		Status:         "unknown",
		LastCheckTime:  time.Now().Format("2006-01-02 15:04:05"),
	}

	if app.K8sClusterID == nil {
		status.Status = "no_cluster"
		return status, nil
	}

	client, err := s.clientMgr.GetClient(ctx, *app.K8sClusterID)
	if err != nil {
		status.Status = "disconnected"
		return status, nil
	}

	// 获取 Deployment 状态
	deploy, err := client.AppsV1().Deployments(app.K8sNamespace).Get(ctx, app.K8sDeployment, metav1.GetOptions{})
	if err != nil {
		status.Status = "not_found"
		return status, nil
	}

	if deploy.Spec.Replicas != nil {
		status.DesiredReplicas = *deploy.Spec.Replicas
	}
	status.ReadyReplicas = deploy.Status.ReadyReplicas
	status.UnavailableCount = deploy.Status.UnavailableReplicas

	// 获取 Pod 重启次数
	pods, err := client.CoreV1().Pods(app.K8sNamespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", app.K8sDeployment),
	})
	if err == nil {
		for _, pod := range pods.Items {
			for _, cs := range pod.Status.ContainerStatuses {
				status.RestartCount += cs.RestartCount
			}
		}
	}

	// 判断健康状态
	if status.ReadyReplicas == status.DesiredReplicas && status.UnavailableCount == 0 {
		status.Status = "healthy"
	} else if status.ReadyReplicas > 0 {
		status.Status = "degraded"
	} else {
		status.Status = "unhealthy"
	}

	// 获取回滚配置
	var config RollbackConfigRecord
	if err := s.db.Where("deploy_record_id = ?", recordID).First(&config).Error; err == nil {
		status.ConsecutiveFails = config.ConsecutiveFails
		if config.Enabled && status.Status == "unhealthy" && config.ConsecutiveFails >= config.FailureThreshold {
			status.ShouldRollback = true
			status.RollbackReason = fmt.Sprintf("连续 %d 次健康检查失败", config.ConsecutiveFails)
		}
	}

	return status, nil
}

// UpdateConfig 更新自动回滚配置
func (s *AutoRollbackService) UpdateConfig(ctx context.Context, recordID uint, cfg *dto.AutoRollbackConfig) error {
	var record models.DeployRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "部署记录不存在")
	}

	// 查找或创建配置
	var config RollbackConfigRecord
	err := s.db.Where("deploy_record_id = ?", recordID).First(&config).Error
	if err == gorm.ErrRecordNotFound {
		config = RollbackConfigRecord{
			DeployRecordID:    recordID,
			Enabled:           cfg.Enabled,
			HealthCheckPeriod: cfg.HealthCheckPeriod,
			FailureThreshold:  cfg.FailureThreshold,
			SuccessThreshold:  cfg.SuccessThreshold,
		}
		return s.db.Create(&config).Error
	}

	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询配置失败")
	}

	return s.db.Model(&config).Updates(map[string]interface{}{
		"enabled":             cfg.Enabled,
		"health_check_period": cfg.HealthCheckPeriod,
		"failure_threshold":   cfg.FailureThreshold,
		"success_threshold":   cfg.SuccessThreshold,
	}).Error
}

// CheckAndRollback 检查并执行自动回滚（由定时任务调用）
func (s *AutoRollbackService) CheckAndRollback(ctx context.Context, recordID uint) error {
	status, err := s.GetHealthStatus(ctx, recordID)
	if err != nil {
		return err
	}

	if !status.ShouldRollback {
		return nil
	}

	// 获取部署记录
	var record models.DeployRecord
	if err := s.db.First(&record, recordID).Error; err != nil {
		return err
	}

	// 查找上一个成功的部署记录
	var lastSuccess models.DeployRecord
	err = s.db.Where("application_id = ? AND env_name = ? AND status = ? AND id < ?",
		record.ApplicationID, record.EnvName, "success", recordID).
		Order("id DESC").First(&lastSuccess).Error
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "找不到可回滚的版本")
	}

	// 执行回滚
	return s.executeRollback(ctx, &record, &lastSuccess, status.RollbackReason)
}

// executeRollback 执行回滚
func (s *AutoRollbackService) executeRollback(ctx context.Context, current, target *models.DeployRecord, reason string) error {
	var app models.Application
	if err := s.db.First(&app, current.ApplicationID).Error; err != nil {
		return err
	}

	if app.K8sClusterID == nil {
		return apperrors.New(apperrors.ErrCodeInvalidParams, "应用未配置K8s集群")
	}

	client, err := s.clientMgr.GetClient(ctx, *app.K8sClusterID)
	if err != nil {
		return err
	}

	// 获取 Deployment
	deploy, err := client.AppsV1().Deployments(app.K8sNamespace).Get(ctx, app.K8sDeployment, metav1.GetOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeNotFound, "Deployment不存在")
	}

	// 更新镜像为目标版本
	if len(deploy.Spec.Template.Spec.Containers) > 0 {
		deploy.Spec.Template.Spec.Containers[0].Image = target.ImageTag
	}

	_, err = client.AppsV1().Deployments(app.K8sNamespace).Update(ctx, deploy, metav1.UpdateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "回滚失败")
	}

	// 创建回滚记录
	now := time.Now()
	rollbackRecord := &models.DeployRecord{
		ApplicationID: current.ApplicationID,
		AppName:       app.Name,
		EnvName:       current.EnvName,
		ImageTag:      target.ImageTag,
		DeployType:    "auto_rollback",
		Status:        "success",
		Description:   fmt.Sprintf("自动回滚: %s, 从 %s 回滚到 %s", reason, current.ImageTag, target.ImageTag),
		RollbackFrom:  &current.ID,
		StartedAt:     &now,
		FinishedAt:    &now,
	}
	s.db.Create(rollbackRecord)

	// 更新原记录状态
	s.db.Model(current).Updates(map[string]interface{}{
		"status":    "rolled_back",
		"error_msg": reason,
	})

	return nil
}

// RecordHealthCheck 记录健康检查结果
func (s *AutoRollbackService) RecordHealthCheck(ctx context.Context, recordID uint, healthy bool) error {
	var config RollbackConfigRecord
	err := s.db.Where("deploy_record_id = ?", recordID).First(&config).Error
	if err != nil {
		return nil // 没有配置，忽略
	}

	now := time.Now()
	updates := map[string]interface{}{
		"last_check_time": &now,
	}

	if healthy {
		updates["consecutive_fails"] = 0
	} else {
		updates["consecutive_fails"] = gorm.Expr("consecutive_fails + 1")
	}

	return s.db.Model(&config).Updates(updates).Error
}

// GetRollbackConfig 获取回滚配置
func (s *AutoRollbackService) GetRollbackConfig(ctx context.Context, recordID uint) (*dto.AutoRollbackConfig, error) {
	var config RollbackConfigRecord
	err := s.db.Where("deploy_record_id = ?", recordID).First(&config).Error
	if err == gorm.ErrRecordNotFound {
		// 返回默认配置
		return &dto.AutoRollbackConfig{
			Enabled:           false,
			HealthCheckPeriod: 30,
			FailureThreshold:  3,
			SuccessThreshold:  1,
		}, nil
	}
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询配置失败")
	}

	return &dto.AutoRollbackConfig{
		Enabled:           config.Enabled,
		HealthCheckPeriod: config.HealthCheckPeriod,
		FailureThreshold:  config.FailureThreshold,
		SuccessThreshold:  config.SuccessThreshold,
	}, nil
}

// SaveRollbackTriggers 保存回滚触发条件
func (s *AutoRollbackService) SaveRollbackTriggers(ctx context.Context, appID uint, triggers []dto.RollbackTrigger) error {
	triggersJSON, err := json.Marshal(triggers)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "序列化失败")
	}

	// 保存到应用配置
	return s.db.Model(&models.Application{}).Where("id = ?", appID).
		Update("config", string(triggersJSON)).Error
}
