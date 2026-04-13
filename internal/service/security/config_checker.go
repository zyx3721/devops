package security

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"

	"devops/internal/models"
	"devops/internal/service/kubernetes"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// ConfigCheckerService 配置检查服务
type ConfigCheckerService struct {
	db        *gorm.DB
	clientMgr *kubernetes.K8sClientManager
}

// NewConfigCheckerService 创建配置检查服务
func NewConfigCheckerService(db *gorm.DB) *ConfigCheckerService {
	return &ConfigCheckerService{
		db:        db,
		clientMgr: kubernetes.NewK8sClientManager(db),
	}
}

// RunCheck 运行配置检查
func (s *ConfigCheckerService) RunCheck(ctx context.Context, req *dto.ConfigCheckRequest) (*dto.ConfigCheckResultResponse, error) {
	log := logger.L().WithField("cluster_id", req.ClusterID).WithField("namespace", req.Namespace)
	log.Info("开始配置检查")

	// 创建检查记录
	check := &models.ConfigCheck{
		ClusterID: req.ClusterID,
		Namespace: req.Namespace,
		Status:    "running",
		CreatedAt: time.Now(),
	}
	if err := s.db.Create(check).Error; err != nil {
		return nil, err
	}

	// 获取K8s客户端
	client, err := s.clientMgr.GetClient(ctx, req.ClusterID)
	if err != nil {
		check.Status = "failed"
		s.db.Save(check)
		return nil, err
	}

	// 获取启用的规则
	var rules []models.ComplianceRule
	ruleQuery := s.db.Where("enabled = ?", true)
	if len(req.RuleIDs) > 0 {
		ruleQuery = ruleQuery.Where("id IN ?", req.RuleIDs)
	}
	ruleQuery.Find(&rules)

	// 执行检查
	issues := s.checkResources(ctx, client, req.Namespace, rules)

	// 统计结果
	var critical, high, medium, low, passed int
	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			critical++
		case "high":
			high++
		case "medium":
			medium++
		case "low":
			low++
		}
	}
	passed = len(rules) - (critical + high + medium + low)
	if passed < 0 {
		passed = 0
	}

	// 更新检查记录
	now := time.Now()
	check.Status = "completed"
	check.CriticalCount = critical
	check.HighCount = high
	check.MediumCount = medium
	check.LowCount = low
	check.PassedCount = passed
	check.CheckedAt = &now

	issuesJSON, _ := json.Marshal(issues)
	check.ResultJSON = string(issuesJSON)

	if err := s.db.Save(check).Error; err != nil {
		return nil, err
	}

	log.WithField("issues", len(issues)).Info("配置检查完成")

	return &dto.ConfigCheckResultResponse{
		ID:        check.ID,
		ClusterID: check.ClusterID,
		Namespace: check.Namespace,
		Status:    check.Status,
		Summary: dto.ConfigCheckSummary{
			Critical: critical,
			High:     high,
			Medium:   medium,
			Low:      low,
			Passed:   passed,
			Total:    critical + high + medium + low,
		},
		Issues:    issues,
		CheckedAt: check.CheckedAt,
	}, nil
}

// checkResources 检查资源
func (s *ConfigCheckerService) checkResources(ctx context.Context, client *k8sclient.Clientset, namespace string, rules []models.ComplianceRule) []dto.ConfigIssue {
	issues := make([]dto.ConfigIssue, 0)

	// 获取Deployments
	deployments, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, deploy := range deployments.Items {
			issues = append(issues, s.checkDeployment(&deploy, rules)...)
		}
	}

	// 获取Pods
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err == nil {
		for _, pod := range pods.Items {
			issues = append(issues, s.checkPod(&pod, rules)...)
		}
	}

	return issues
}

// checkDeployment 检查Deployment
func (s *ConfigCheckerService) checkDeployment(deploy *appsv1.Deployment, rules []models.ComplianceRule) []dto.ConfigIssue {
	issues := make([]dto.ConfigIssue, 0)

	for _, container := range deploy.Spec.Template.Spec.Containers {
		issues = append(issues, s.checkContainer(container, deploy.Namespace, "Deployment", deploy.Name, rules)...)
	}

	return issues
}

// checkPod 检查Pod
func (s *ConfigCheckerService) checkPod(pod *corev1.Pod, rules []models.ComplianceRule) []dto.ConfigIssue {
	issues := make([]dto.ConfigIssue, 0)

	// 跳过由Deployment管理的Pod
	if len(pod.OwnerReferences) > 0 {
		for _, ref := range pod.OwnerReferences {
			if ref.Kind == "ReplicaSet" {
				return issues
			}
		}
	}

	for _, container := range pod.Spec.Containers {
		issues = append(issues, s.checkContainer(container, pod.Namespace, "Pod", pod.Name, rules)...)
	}

	return issues
}

// checkContainer 检查容器
func (s *ConfigCheckerService) checkContainer(container corev1.Container, namespace, kind, name string, rules []models.ComplianceRule) []dto.ConfigIssue {
	issues := make([]dto.ConfigIssue, 0)

	for _, rule := range rules {
		issue := s.applyRule(container, namespace, kind, name, rule)
		if issue != nil {
			issues = append(issues, *issue)
		}
	}

	return issues
}

// applyRule 应用规则
func (s *ConfigCheckerService) applyRule(container corev1.Container, namespace, kind, name string, rule models.ComplianceRule) *dto.ConfigIssue {
	var condition struct {
		Field    string      `json:"field"`
		Operator string      `json:"operator"`
		Value    interface{} `json:"value"`
	}

	if err := json.Unmarshal([]byte(rule.ConditionJSON), &condition); err != nil {
		return nil
	}

	var violated bool

	switch condition.Field {
	case "securityContext.runAsNonRoot":
		if container.SecurityContext == nil || container.SecurityContext.RunAsNonRoot == nil || !*container.SecurityContext.RunAsNonRoot {
			violated = true
		}

	case "resources.limits.cpu":
		if container.Resources.Limits.Cpu().IsZero() {
			violated = true
		}

	case "resources.limits.memory":
		if container.Resources.Limits.Memory().IsZero() {
			violated = true
		}

	case "securityContext.privileged":
		if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
			violated = true
		}

	case "securityContext.allowPrivilegeEscalation":
		if container.SecurityContext != nil && container.SecurityContext.AllowPrivilegeEscalation != nil && *container.SecurityContext.AllowPrivilegeEscalation {
			violated = true
		}

	case "livenessProbe":
		if container.LivenessProbe == nil {
			violated = true
		}

	case "image":
		if condition.Operator == "endsWith" {
			if val, ok := condition.Value.(string); ok {
				if len(container.Image) >= len(val) && container.Image[len(container.Image)-len(val):] == val {
					violated = true
				}
			}
		}
	}

	if violated {
		return &dto.ConfigIssue{
			RuleID:       rule.ID,
			RuleName:     rule.Name,
			Severity:     rule.Severity,
			ResourceKind: kind,
			ResourceName: name,
			Namespace:    namespace,
			Message:      rule.Description,
			Remediation:  rule.Remediation,
		}
	}

	return nil
}

// GetCheckHistory 获取检查历史
func (s *ConfigCheckerService) GetCheckHistory(ctx context.Context, clusterID uint, page, pageSize int) (*dto.ConfigCheckHistoryResponse, error) {
	var checks []models.ConfigCheck
	var total int64

	query := s.db.Model(&models.ConfigCheck{})
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}

	query.Count(&total)

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&checks)

	items := make([]dto.ConfigCheckItem, 0, len(checks))
	for _, check := range checks {
		// 获取集群名称
		var clusterName string
		s.db.Table("k8s_clusters").Select("name").Where("id = ?", check.ClusterID).Scan(&clusterName)

		items = append(items, dto.ConfigCheckItem{
			ID:            check.ID,
			ClusterID:     check.ClusterID,
			ClusterName:   clusterName,
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

	return &dto.ConfigCheckHistoryResponse{
		Total: int(total),
		Items: items,
	}, nil
}

// GetCheckResult 获取检查结果
func (s *ConfigCheckerService) GetCheckResult(ctx context.Context, checkID uint) (*dto.ConfigCheckResultResponse, error) {
	var check models.ConfigCheck
	if err := s.db.First(&check, checkID).Error; err != nil {
		return nil, err
	}

	result := &dto.ConfigCheckResultResponse{
		ID:        check.ID,
		ClusterID: check.ClusterID,
		Namespace: check.Namespace,
		Status:    check.Status,
		Summary: dto.ConfigCheckSummary{
			Critical: check.CriticalCount,
			High:     check.HighCount,
			Medium:   check.MediumCount,
			Low:      check.LowCount,
			Passed:   check.PassedCount,
			Total:    check.CriticalCount + check.HighCount + check.MediumCount + check.LowCount,
		},
		CheckedAt: check.CheckedAt,
	}

	// 解析问题详情
	if check.ResultJSON != "" {
		var issues []dto.ConfigIssue
		if err := json.Unmarshal([]byte(check.ResultJSON), &issues); err == nil {
			result.Issues = issues
		}
	}

	return result, nil
}
