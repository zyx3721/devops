package deploy

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"devops/internal/models"
	"devops/internal/service/kubernetes"
	"devops/pkg/dto"
)

// DeployCheckService 部署检查服务
type DeployCheckService struct {
	db        *gorm.DB
	clientMgr *kubernetes.K8sClientManager
}

// NewDeployCheckService 创建部署检查服务
func NewDeployCheckService(db *gorm.DB, clientMgr *kubernetes.K8sClientManager) *DeployCheckService {
	return &DeployCheckService{db: db, clientMgr: clientMgr}
}

// PreCheck 部署前置检查
func (s *DeployCheckService) PreCheck(ctx context.Context, req *dto.DeployPreCheckRequest) (*dto.DeployPreCheckResponse, error) {
	resp := &dto.DeployPreCheckResponse{
		CanDeploy: true,
		Checks:    []dto.PreCheckItem{},
		Warnings:  []string{},
		Errors:    []string{},
	}

	// 获取应用信息
	var app models.Application
	if err := s.db.First(&app, req.ApplicationID).Error; err != nil {
		resp.CanDeploy = false
		resp.Errors = append(resp.Errors, "应用不存在")
		return resp, nil
	}

	// 1. 检查应用状态
	resp.Checks = append(resp.Checks, s.checkAppStatus(&app))

	// 2. 检查 K8s 集群连接
	if app.K8sClusterID != nil {
		resp.Checks = append(resp.Checks, s.checkK8sConnection(ctx, *app.K8sClusterID))
	}

	// 3. 检查部署锁
	resp.Checks = append(resp.Checks, s.checkDeployLock(req.ApplicationID, req.EnvName))

	// 4. 检查发布窗口
	resp.Checks = append(resp.Checks, s.checkDeployWindow(req.ApplicationID, req.EnvName))

	// 5. 检查当前 Pod 健康状态
	if app.K8sClusterID != nil && app.K8sNamespace != "" && app.K8sDeployment != "" {
		check, warning := s.checkPodHealth(ctx, *app.K8sClusterID, app.K8sNamespace, app.K8sDeployment)
		resp.Checks = append(resp.Checks, check)
		if warning != "" {
			resp.Warnings = append(resp.Warnings, warning)
		}
	}

	// 6. 检查资源配额
	if app.K8sClusterID != nil && app.K8sNamespace != "" {
		check, warning := s.checkResourceQuota(ctx, *app.K8sClusterID, app.K8sNamespace)
		resp.Checks = append(resp.Checks, check)
		if warning != "" {
			resp.Warnings = append(resp.Warnings, warning)
		}
	}

	// 7. 检查是否有未完成的部署
	resp.Checks = append(resp.Checks, s.checkPendingDeploy(req.ApplicationID, req.EnvName))

	// 汇总结果
	for _, check := range resp.Checks {
		if check.Status == "failed" {
			resp.CanDeploy = false
			resp.Errors = append(resp.Errors, check.Message)
		}
	}

	return resp, nil
}

// checkAppStatus 检查应用状态
func (s *DeployCheckService) checkAppStatus(app *models.Application) dto.PreCheckItem {
	if app.Status != "active" {
		return dto.PreCheckItem{
			Name:    "应用状态",
			Status:  "failed",
			Message: "应用已禁用",
		}
	}
	return dto.PreCheckItem{
		Name:    "应用状态",
		Status:  "passed",
		Message: "应用状态正常",
	}
}

// checkK8sConnection 检查 K8s 连接
func (s *DeployCheckService) checkK8sConnection(ctx context.Context, clusterID uint) dto.PreCheckItem {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return dto.PreCheckItem{
			Name:    "K8s集群连接",
			Status:  "failed",
			Message: "无法连接到K8s集群",
			Detail:  err.Error(),
		}
	}

	// 尝试获取版本信息验证连接
	_, err = client.Discovery().ServerVersion()
	if err != nil {
		return dto.PreCheckItem{
			Name:    "K8s集群连接",
			Status:  "failed",
			Message: "K8s集群连接异常",
			Detail:  err.Error(),
		}
	}

	return dto.PreCheckItem{
		Name:    "K8s集群连接",
		Status:  "passed",
		Message: "K8s集群连接正常",
	}
}

// checkDeployLock 检查部署锁
func (s *DeployCheckService) checkDeployLock(appID uint, envName string) dto.PreCheckItem {
	var lock models.DeployLock
	err := s.db.Where("application_id = ? AND env_name = ? AND status = ? AND expires_at > ?",
		appID, envName, "active", time.Now()).First(&lock).Error

	if err == nil {
		return dto.PreCheckItem{
			Name:    "部署锁",
			Status:  "failed",
			Message: fmt.Sprintf("存在部署锁，锁定者: %s", lock.LockedByName),
			Detail:  fmt.Sprintf("锁定时间: %s, 过期时间: %s", lock.CreatedAt.Format("2006-01-02 15:04:05"), lock.ExpiresAt.Format("2006-01-02 15:04:05")),
		}
	}

	return dto.PreCheckItem{
		Name:    "部署锁",
		Status:  "passed",
		Message: "无部署锁",
	}
}

// checkDeployWindow 检查发布窗口
func (s *DeployCheckService) checkDeployWindow(appID uint, envName string) dto.PreCheckItem {
	now := time.Now()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	currentTime := now.Format("15:04")

	// 查找匹配的发布窗口规则
	var window models.DeployWindow
	err := s.db.Where("(app_id = ? OR app_id = 0) AND (env = ? OR env = '*') AND enabled = ?",
		appID, envName, true).Order("app_id DESC").First(&window).Error

	if err != nil {
		// 没有配置发布窗口，默认允许
		return dto.PreCheckItem{
			Name:    "发布窗口",
			Status:  "passed",
			Message: "未配置发布窗口限制",
		}
	}

	// 检查是否在允许的工作日
	weekdayStr := fmt.Sprintf("%d", weekday)
	if !containsWeekday(window.Weekdays, weekdayStr) {
		return dto.PreCheckItem{
			Name:    "发布窗口",
			Status:  "warning",
			Message: fmt.Sprintf("当前不在发布窗口内（允许: 周%s）", window.Weekdays),
			Detail:  "可申请紧急发布",
		}
	}

	// 检查是否在允许的时间段
	if currentTime < window.StartTime || currentTime > window.EndTime {
		return dto.PreCheckItem{
			Name:    "发布窗口",
			Status:  "warning",
			Message: fmt.Sprintf("当前不在发布窗口内（允许: %s-%s）", window.StartTime, window.EndTime),
			Detail:  "可申请紧急发布",
		}
	}

	return dto.PreCheckItem{
		Name:    "发布窗口",
		Status:  "passed",
		Message: "在发布窗口内",
	}
}

// checkPodHealth 检查 Pod 健康状态
func (s *DeployCheckService) checkPodHealth(ctx context.Context, clusterID uint, namespace, deployment string) (dto.PreCheckItem, string) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return dto.PreCheckItem{
			Name:    "Pod健康状态",
			Status:  "skipped",
			Message: "无法获取Pod状态",
		}, ""
	}

	deploy, err := client.AppsV1().Deployments(namespace).Get(ctx, deployment, metav1.GetOptions{})
	if err != nil {
		return dto.PreCheckItem{
			Name:    "Pod健康状态",
			Status:  "skipped",
			Message: "Deployment不存在（首次部署）",
		}, ""
	}

	replicas := int32(1)
	if deploy.Spec.Replicas != nil {
		replicas = *deploy.Spec.Replicas
	}

	if deploy.Status.ReadyReplicas < replicas {
		warning := fmt.Sprintf("当前有 %d/%d 个Pod未就绪", replicas-deploy.Status.ReadyReplicas, replicas)
		return dto.PreCheckItem{
			Name:    "Pod健康状态",
			Status:  "warning",
			Message: warning,
		}, warning
	}

	// 检查是否有重启过多的 Pod
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", deployment),
	})
	if err == nil {
		for _, pod := range pods.Items {
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.RestartCount > 5 {
					warning := fmt.Sprintf("Pod %s 重启次数过多 (%d次)", pod.Name, cs.RestartCount)
					return dto.PreCheckItem{
						Name:    "Pod健康状态",
						Status:  "warning",
						Message: warning,
					}, warning
				}
			}
		}
	}

	return dto.PreCheckItem{
		Name:    "Pod健康状态",
		Status:  "passed",
		Message: fmt.Sprintf("所有Pod运行正常 (%d/%d)", deploy.Status.ReadyReplicas, replicas),
	}, ""
}

// checkResourceQuota 检查资源配额
func (s *DeployCheckService) checkResourceQuota(ctx context.Context, clusterID uint, namespace string) (dto.PreCheckItem, string) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return dto.PreCheckItem{
			Name:    "资源配额",
			Status:  "skipped",
			Message: "无法检查资源配额",
		}, ""
	}

	quotas, err := client.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{})
	if err != nil || len(quotas.Items) == 0 {
		return dto.PreCheckItem{
			Name:    "资源配额",
			Status:  "passed",
			Message: "未配置资源配额限制",
		}, ""
	}

	// 检查是否接近配额限制
	for _, quota := range quotas.Items {
		for resourceName, hard := range quota.Spec.Hard {
			if used, ok := quota.Status.Used[resourceName]; ok {
				usedVal := used.Value()
				hardVal := hard.Value()
				if hardVal > 0 {
					usage := float64(usedVal) / float64(hardVal) * 100
					if usage > 90 {
						warning := fmt.Sprintf("资源 %s 使用率已达 %.1f%%", resourceName, usage)
						return dto.PreCheckItem{
							Name:    "资源配额",
							Status:  "warning",
							Message: warning,
						}, warning
					}
				}
			}
		}
	}

	return dto.PreCheckItem{
		Name:    "资源配额",
		Status:  "passed",
		Message: "资源配额充足",
	}, ""
}

// checkPendingDeploy 检查是否有未完成的部署
func (s *DeployCheckService) checkPendingDeploy(appID uint, envName string) dto.PreCheckItem {
	var count int64
	s.db.Model(&models.DeployRecord{}).Where(
		"application_id = ? AND env_name = ? AND status IN ?",
		appID, envName, []string{"pending", "approved", "running"},
	).Count(&count)

	if count > 0 {
		return dto.PreCheckItem{
			Name:    "进行中的部署",
			Status:  "failed",
			Message: fmt.Sprintf("存在 %d 个未完成的部署", count),
		}
	}

	return dto.PreCheckItem{
		Name:    "进行中的部署",
		Status:  "passed",
		Message: "无进行中的部署",
	}
}

func containsWeekday(weekdays, day string) bool {
	for _, d := range weekdays {
		if string(d) == day {
			return true
		}
	}
	return false
}
