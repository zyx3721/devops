package deploy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "k8s.io/client-go/kubernetes"

	"devops/internal/models"
	"devops/internal/repository"
	"devops/internal/service/approval"
	"devops/internal/service/jenkins"
	"devops/internal/service/kubernetes"
	"devops/pkg/logger"
)

var log = logger.L().WithField("module", "deploy")

// 状态常量
const (
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusRunning   = "running"
	StatusSuccess   = "success"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"

	DeployTypeDeploy   = "deploy"
	DeployTypeRollback = "rollback"

	DeployMethodJenkins = "jenkins"
	DeployMethodK8s     = "k8s"

	LockTimeout = 30 * time.Minute
)

// 业务错误
var (
	ErrRecordNotFound        = errors.New("部署记录不存在")
	ErrInvalidStatus         = errors.New("当前状态不允许此操作")
	ErrLockExists            = errors.New("应用环境已被锁定")
	ErrLockNotFound          = errors.New("锁定记录不存在")
	ErrNoRollbackVersion     = errors.New("没有可回滚的版本")
	ErrSelfApprovalForbidden = errors.New("不能审批自己的请求")
	ErrApplicationNotFound   = errors.New("应用不存在")
)

// 需要审批的环境
var requireApprovalEnvs = map[string]bool{
	"prod":       true,
	"production": true,
}

// 发布窗口错误
var (
	ErrOutsideDeployWindow = errors.New("当前不在发布窗口期内")
	ErrEmergencyRequired   = errors.New("窗口期外发布需要紧急发布权限")
)

// Service 发布服务
type Service struct {
	recordRepo      *repository.DeployRecordRepository
	lockRepo        *repository.DeployLockRepository
	approvalRepo    *repository.ApprovalRecordRepository
	appRepo         *repository.ApplicationRepository
	jenkins         *jenkins.Client
	k8sManager      *kubernetes.K8sClientManager
	chainService    *approval.ChainService
	instanceService *approval.InstanceService
	ruleService     *approval.RuleService
	windowService   *approval.WindowService
}

// NewService 创建发布服务
func NewService(
	recordRepo *repository.DeployRecordRepository,
	lockRepo *repository.DeployLockRepository,
	approvalRepo *repository.ApprovalRecordRepository,
	appRepo *repository.ApplicationRepository,
	jenkins *jenkins.Client,
	k8sManager *kubernetes.K8sClientManager,
) *Service {
	return &Service{
		recordRepo:   recordRepo,
		lockRepo:     lockRepo,
		approvalRepo: approvalRepo,
		appRepo:      appRepo,
		jenkins:      jenkins,
		k8sManager:   k8sManager,
	}
}

// SetApprovalChainServices 设置审批链服务（用于依赖注入）
func (s *Service) SetApprovalChainServices(chainService *approval.ChainService, instanceService *approval.InstanceService) {
	s.chainService = chainService
	s.instanceService = instanceService
}

// SetApprovalServices 设置审批规则和发布窗口服务（用于依赖注入）
func (s *Service) SetApprovalServices(ruleService *approval.RuleService, windowService *approval.WindowService) {
	s.ruleService = ruleService
	s.windowService = windowService
}

// CheckDeployWindow 检查发布窗口
// 返回: 是否在窗口内, 是否允许紧急发布, 窗口信息, 错误
func (s *Service) CheckDeployWindow(ctx context.Context, appID uint, envName string) (bool, bool, *models.DeployWindow, error) {
	if s.windowService == nil {
		// 未配置窗口服务，默认允许
		return true, true, nil, nil
	}

	inWindow, allowEmergency, err := s.windowService.IsInWindow(appID, envName)
	if err != nil {
		return true, true, nil, nil
	}

	var windowInfo *models.DeployWindow
	if !inWindow {
		windowInfo, _ = s.windowService.GetWindowInfo(appID, envName)
	}

	return inWindow, allowEmergency, windowInfo, nil
}

// CheckApprovalRequired 检查是否需要审批
// 返回: 是否需要审批, 审批人ID列表, 错误
func (s *Service) CheckApprovalRequired(ctx context.Context, appID uint, envName string) (bool, []uint, error) {
	// 优先使用审批链服务
	if s.chainService != nil {
		needApproval, chain, err := s.chainService.NeedApproval(ctx, appID, envName)
		if err != nil {
			return false, nil, err
		}
		if needApproval && chain != nil {
			// 从审批链获取审批人（这里简化处理，实际应该从节点获取）
			return true, nil, nil
		}
		return needApproval, nil, nil
	}

	// 使用审批规则服务
	if s.ruleService != nil {
		return s.ruleService.NeedApproval(appID, envName)
	}

	// 默认使用环境判断
	return requireApprovalEnvs[envName], nil, nil
}

// CreateDeploy 创建部署记录
func (s *Service) CreateDeploy(ctx context.Context, record *models.DeployRecord) error {
	// 验证应用存在
	app, err := s.appRepo.GetByID(ctx, record.ApplicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrApplicationNotFound
		}
		return err
	}

	record.AppName = app.Name
	record.Status = StatusPending

	// 检查发布窗口
	inWindow, allowEmergency, _, err := s.CheckDeployWindow(ctx, record.ApplicationID, record.EnvName)
	if err != nil {
		log.WithError(err).Warn("检查发布窗口失败，继续执行")
	} else if !inWindow {
		if !allowEmergency {
			return ErrOutsideDeployWindow
		}
		// 窗口外但允许紧急发布，需要通过 CreateDeployWithEmergency 方法
		return ErrEmergencyRequired
	}

	// 检查是否需要审批（优先使用审批链，否则使用旧逻辑）
	if s.chainService != nil {
		needApproval, chain, err := s.chainService.NeedApproval(ctx, record.ApplicationID, record.EnvName)
		if err != nil {
			return err
		}
		record.NeedApproval = needApproval
		if needApproval && chain != nil {
			record.ApprovalChainID = &chain.ID
		}
	} else if s.ruleService != nil {
		needApproval, _, err := s.ruleService.NeedApproval(record.ApplicationID, record.EnvName)
		if err != nil {
			log.WithError(err).Warn("检查审批规则失败，使用默认逻辑")
			record.NeedApproval = requireApprovalEnvs[record.EnvName]
		} else {
			record.NeedApproval = needApproval
		}
	} else {
		record.NeedApproval = requireApprovalEnvs[record.EnvName]
	}

	// 默认部署方式
	if record.DeployMethod == "" {
		if app.JenkinsJobName != "" {
			record.DeployMethod = DeployMethodJenkins
		} else {
			record.DeployMethod = DeployMethodK8s
		}
	}

	// 创建部署记录
	if err := s.recordRepo.Create(ctx, record); err != nil {
		return err
	}

	// 如果需要审批且有审批链，创建审批实例
	if record.NeedApproval && record.ApprovalChainID != nil && s.instanceService != nil {
		chain, err := s.chainService.GetWithNodes(ctx, *record.ApprovalChainID)
		if err != nil {
			return err
		}
		_, err = s.instanceService.Create(ctx, record.ID, chain)
		if err != nil {
			return err
		}
	}

	return nil
}

// CreateDeployWithEmergency 创建部署记录（支持紧急发布）
func (s *Service) CreateDeployWithEmergency(ctx context.Context, record *models.DeployRecord, isEmergency bool, emergencyReason string) error {
	// 验证应用存在
	app, err := s.appRepo.GetByID(ctx, record.ApplicationID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrApplicationNotFound
		}
		return err
	}

	record.AppName = app.Name
	record.Status = StatusPending

	// 检查发布窗口
	inWindow, allowEmergency, windowInfo, err := s.CheckDeployWindow(ctx, record.ApplicationID, record.EnvName)
	if err != nil {
		log.WithError(err).Warn("检查发布窗口失败，继续执行")
	} else if !inWindow {
		if !isEmergency {
			if !allowEmergency {
				return ErrOutsideDeployWindow
			}
			return ErrEmergencyRequired
		}
		// 紧急发布，记录原因
		if emergencyReason == "" {
			emergencyReason = "紧急发布"
		}
		windowDesc := ""
		if windowInfo != nil {
			windowDesc = fmt.Sprintf("(发布窗口: %s %s-%s)", windowInfo.Weekdays, windowInfo.StartTime, windowInfo.EndTime)
		}
		record.Description = fmt.Sprintf("[紧急发布%s] %s - %s", windowDesc, emergencyReason, record.Description)
	}

	// 紧急发布跳过审批
	if isEmergency {
		record.NeedApproval = false
		log.WithField("app_id", record.ApplicationID).WithField("env", record.EnvName).
			WithField("reason", emergencyReason).Info("紧急发布，跳过审批")
	} else {
		// 检查是否需要审批
		if s.chainService != nil {
			needApproval, chain, err := s.chainService.NeedApproval(ctx, record.ApplicationID, record.EnvName)
			if err != nil {
				return err
			}
			record.NeedApproval = needApproval
			if needApproval && chain != nil {
				record.ApprovalChainID = &chain.ID
			}
		} else if s.ruleService != nil {
			needApproval, _, err := s.ruleService.NeedApproval(record.ApplicationID, record.EnvName)
			if err != nil {
				log.WithError(err).Warn("检查审批规则失败，使用默认逻辑")
				record.NeedApproval = requireApprovalEnvs[record.EnvName]
			} else {
				record.NeedApproval = needApproval
			}
		} else {
			record.NeedApproval = requireApprovalEnvs[record.EnvName]
		}
	}

	// 默认部署方式
	if record.DeployMethod == "" {
		if app.JenkinsJobName != "" {
			record.DeployMethod = DeployMethodJenkins
		} else {
			record.DeployMethod = DeployMethodK8s
		}
	}

	// 创建部署记录
	if err := s.recordRepo.Create(ctx, record); err != nil {
		return err
	}

	// 如果需要审批且有审批链，创建审批实例
	if record.NeedApproval && record.ApprovalChainID != nil && s.instanceService != nil {
		chain, err := s.chainService.GetWithNodes(ctx, *record.ApprovalChainID)
		if err != nil {
			return err
		}
		_, err = s.instanceService.Create(ctx, record.ID, chain)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetRecord 获取部署记录
func (s *Service) GetRecord(ctx context.Context, id uint) (*models.DeployRecord, error) {
	record, err := s.recordRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	return record, nil
}

// ListRecords 获取部署记录列表
func (s *Service) ListRecords(ctx context.Context, filter repository.DeployRecordFilter, page, pageSize int) ([]models.DeployRecord, int64, error) {
	return s.recordRepo.List(ctx, filter, page, pageSize)
}

// CancelDeploy 取消部署
func (s *Service) CancelDeploy(ctx context.Context, id uint) error {
	record, err := s.recordRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRecordNotFound
		}
		return err
	}

	if record.Status != StatusPending {
		return ErrInvalidStatus
	}

	return s.recordRepo.UpdateStatus(ctx, id, StatusCancelled, map[string]any{})
}

// ApproveDeploy 审批通过
func (s *Service) ApproveDeploy(ctx context.Context, id uint, approverID uint, approverName string, comment string) error {
	record, err := s.recordRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRecordNotFound
		}
		return err
	}

	if record.Status != StatusPending {
		return ErrInvalidStatus
	}

	// 不能审批自己的请求
	if record.OperatorID == approverID {
		return ErrSelfApprovalForbidden
	}

	now := time.Now()

	// 创建审批记录
	approval := &models.ApprovalRecord{
		RecordID:     id,
		ApproverID:   approverID,
		ApproverName: approverName,
		Action:       "approve",
		Comment:      comment,
	}
	if err := s.approvalRepo.Create(ctx, approval); err != nil {
		return err
	}

	// 更新记录状态
	return s.recordRepo.UpdateStatus(ctx, id, StatusApproved, map[string]any{
		"approver_id":   &approverID,
		"approver_name": approverName,
		"approved_at":   &now,
	})
}

// RejectDeploy 审批拒绝
func (s *Service) RejectDeploy(ctx context.Context, id uint, approverID uint, approverName string, reason string) error {
	record, err := s.recordRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRecordNotFound
		}
		return err
	}

	if record.Status != StatusPending {
		return ErrInvalidStatus
	}

	// 创建审批记录
	approval := &models.ApprovalRecord{
		RecordID:     id,
		ApproverID:   approverID,
		ApproverName: approverName,
		Action:       "reject",
		Comment:      reason,
	}
	if err := s.approvalRepo.Create(ctx, approval); err != nil {
		return err
	}

	return s.recordRepo.UpdateStatus(ctx, id, StatusRejected, map[string]any{
		"reject_reason": reason,
	})
}

// ExecuteDeploy 执行部署
func (s *Service) ExecuteDeploy(ctx context.Context, id uint, operatorID uint, operatorName string) error {
	record, err := s.recordRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRecordNotFound
		}
		return err
	}

	// 检查状态
	if record.NeedApproval && record.Status != StatusApproved {
		return ErrInvalidStatus
	}
	if !record.NeedApproval && record.Status != StatusPending {
		return ErrInvalidStatus
	}

	// 尝试获取锁
	lock := &models.DeployLock{
		ApplicationID: record.ApplicationID,
		EnvName:       record.EnvName,
		RecordID:      id,
		LockedBy:      operatorID,
		LockedByName:  operatorName,
		ExpiresAt:     time.Now().Add(LockTimeout),
		Status:        "active",
	}

	if err := s.lockRepo.AcquireLock(ctx, lock); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrLockExists
		}
		return err
	}

	now := time.Now()

	// 更新状态为执行中
	if err := s.recordRepo.UpdateStatus(ctx, id, StatusRunning, map[string]any{
		"started_at": &now,
	}); err != nil {
		s.lockRepo.ReleaseLock(ctx, record.ApplicationID, record.EnvName, operatorID, "执行失败")
		return err
	}

	// 异步执行部署（带超时控制）
	go func() {
		// 创建带超时的 context，防止部署无限期运行
		deployCtx, cancel := context.WithTimeout(context.Background(), LockTimeout)
		defer cancel()

		// 监听 context 取消
		done := make(chan struct{})
		go func() {
			s.executeDeploy(record, operatorID)
			close(done)
		}()

		select {
		case <-done:
			// 部署正常完成
		case <-deployCtx.Done():
			// 部署超时，强制释放锁
			log.WithField("record_id", record.ID).Warn("部署执行超时，强制释放锁")
			s.finishDeploy(context.Background(), record.ID, StatusFailed, "部署执行超时", operatorID)
		}
	}()

	return nil
}

// executeDeploy 执行部署（异步）
// 包含 panic recovery 确保锁一定会被释放
func (s *Service) executeDeploy(record *models.DeployRecord, operatorID uint) {
	ctx := context.Background()

	// panic recovery - 确保锁一定会被释放
	defer func() {
		if r := recover(); r != nil {
			log.WithField("record_id", record.ID).
				WithField("panic", r).
				Error("部署执行发生 panic，正在释放锁")
			s.finishDeploy(ctx, record.ID, StatusFailed, fmt.Sprintf("部署执行异常: %v", r), operatorID)
		}
	}()

	// 获取应用信息
	app, err := s.appRepo.GetByID(ctx, record.ApplicationID)
	if err != nil {
		s.finishDeploy(ctx, record.ID, StatusFailed, "获取应用信息失败: "+err.Error(), operatorID)
		return
	}

	// 根据部署方式执行
	switch record.DeployMethod {
	case DeployMethodJenkins:
		s.executeJenkinsDeploy(ctx, record, app, operatorID)
	case DeployMethodK8s:
		s.executeK8sDeploy(ctx, record, app, operatorID)
	default:
		s.finishDeploy(ctx, record.ID, StatusFailed, "未知的部署方式: "+record.DeployMethod, operatorID)
	}
}

// executeJenkinsDeploy 通过 Jenkins 执行部署
func (s *Service) executeJenkinsDeploy(ctx context.Context, record *models.DeployRecord, app *models.Application, operatorID uint) {
	if s.jenkins == nil {
		s.finishDeploy(ctx, record.ID, StatusFailed, "Jenkins 客户端未初始化", operatorID)
		return
	}

	if app.JenkinsJobName == "" {
		s.finishDeploy(ctx, record.ID, StatusFailed, "应用未配置 Jenkins Job", operatorID)
		return
	}

	// 触发构建
	buildReq := jenkins.BuildRequest{
		JobName:    app.JenkinsJobName,
		Branch:     record.Branch,
		DeployType: record.DeployType,
	}

	queueID, err := s.jenkins.Build(ctx, buildReq)
	if err != nil {
		s.finishDeploy(ctx, record.ID, StatusFailed, "触发构建失败: "+err.Error(), operatorID)
		return
	}

	// 等待构建开始
	buildCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	buildNumber, err := s.jenkins.WaitForBuildToStart(buildCtx, queueID)
	if err != nil {
		s.finishDeploy(ctx, record.ID, StatusFailed, "等待构建开始超时: "+err.Error(), operatorID)
		return
	}

	// 更新构建号
	s.recordRepo.UpdateStatus(ctx, record.ID, StatusRunning, map[string]any{
		"jenkins_build": buildNumber,
	})

	// 轮询构建状态
	s.pollJenkinsBuild(ctx, record, app.JenkinsJobName, int(buildNumber), operatorID)
}

// pollJenkinsBuild 轮询 Jenkins 构建状态
func (s *Service) pollJenkinsBuild(ctx context.Context, record *models.DeployRecord, jobName string, buildNumber int, operatorID uint) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute)

	for {
		select {
		case <-timeout:
			s.finishDeploy(ctx, record.ID, StatusFailed, "构建超时", operatorID)
			return
		case <-ticker.C:
			build, err := s.jenkins.GetJobBuildInfo(ctx, jobName, buildNumber)
			if err != nil {
				continue
			}

			if build.IsRunning(ctx) {
				continue
			}

			result := build.GetResult()
			if result == "SUCCESS" {
				s.finishDeploy(ctx, record.ID, StatusSuccess, "", operatorID)
			} else {
				s.finishDeploy(ctx, record.ID, StatusFailed, fmt.Sprintf("构建结果: %s", result), operatorID)
			}
			return
		}
	}
}

// executeK8sDeploy 通过 K8s 执行部署
func (s *Service) executeK8sDeploy(ctx context.Context, record *models.DeployRecord, app *models.Application, operatorID uint) {
	if s.k8sManager == nil {
		s.finishDeploy(ctx, record.ID, StatusFailed, "K8s 客户端管理器未初始化", operatorID)
		return
	}

	if app.K8sClusterID == nil || app.K8sDeployment == "" {
		s.finishDeploy(ctx, record.ID, StatusFailed, "应用未配置 K8s 集群或 Deployment", operatorID)
		return
	}

	// 获取 K8s 客户端
	client, err := s.k8sManager.GetClient(ctx, *app.K8sClusterID)
	if err != nil {
		s.finishDeploy(ctx, record.ID, StatusFailed, "获取 K8s 客户端失败: "+err.Error(), operatorID)
		return
	}

	// 更新镜像
	imageTag := record.ImageTag
	if imageTag == "" {
		imageTag = record.Version
	}
	if imageTag == "" {
		imageTag = record.Branch
	}

	log.WithField("deployment", app.K8sDeployment).WithField("image", imageTag).Info("开始 K8s 部署")

	// 获取 Deployment 并更新镜像
	namespace := app.K8sNamespace
	if namespace == "" {
		namespace = "default"
	}

	deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, app.K8sDeployment, metav1.GetOptions{})
	if err != nil {
		s.finishDeploy(ctx, record.ID, StatusFailed, "获取 Deployment 失败: "+err.Error(), operatorID)
		return
	}

	// 更新第一个容器的镜像
	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		// 解析当前镜像，替换 tag
		currentImage := deployment.Spec.Template.Spec.Containers[0].Image
		newImage := replaceImageTag(currentImage, imageTag)
		deployment.Spec.Template.Spec.Containers[0].Image = newImage

		_, err = client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			s.finishDeploy(ctx, record.ID, StatusFailed, "更新 Deployment 失败: "+err.Error(), operatorID)
			return
		}
	}

	// 等待 Deployment 就绪
	s.waitForDeploymentReady(ctx, client, namespace, app.K8sDeployment, record.ID, operatorID)
}

// replaceImageTag 替换镜像标签
// 支持以下格式:
// - image:tag -> image:newTag
// - registry:5000/image:tag -> registry:5000/image:newTag
// - image@sha256:... -> image:newTag (digest 格式)
// - image -> image:newTag
func replaceImageTag(image, newTag string) string {
	// 处理 digest 格式 (image@sha256:...)
	if atIdx := strings.LastIndex(image, "@"); atIdx != -1 {
		return image[:atIdx] + ":" + newTag
	}

	// 找到最后一个 / 的位置，用于区分 registry:port 和 image:tag
	lastSlash := strings.LastIndex(image, "/")

	// 在最后一个 / 之后查找 :
	searchStart := lastSlash + 1
	if colonIdx := strings.LastIndex(image[searchStart:], ":"); colonIdx != -1 {
		return image[:searchStart+colonIdx+1] + newTag
	}

	return image + ":" + newTag
}

// waitForDeploymentReady 等待 Deployment 就绪
func (s *Service) waitForDeploymentReady(ctx context.Context, client *k8sclient.Clientset, namespace, name string, recordID uint, operatorID uint) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(10 * time.Minute)

	for {
		select {
		case <-timeout:
			s.finishDeploy(ctx, recordID, StatusFailed, "等待 Deployment 就绪超时", operatorID)
			return
		case <-ticker.C:
			deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				continue
			}

			// 检查是否就绪（添加 nil 检查防止空指针）
			replicas := int32(1)
			if deployment.Spec.Replicas != nil {
				replicas = *deployment.Spec.Replicas
			}
			if deployment.Status.ReadyReplicas == replicas &&
				deployment.Status.UpdatedReplicas == replicas {
				s.finishDeploy(ctx, recordID, StatusSuccess, "", operatorID)
				return
			}
		}
	}
}

// finishDeploy 完成部署
func (s *Service) finishDeploy(ctx context.Context, id uint, status string, errorMsg string, operatorID uint) {
	record, err := s.recordRepo.GetByID(ctx, id)
	if err != nil {
		return
	}

	now := time.Now()
	var duration int
	if record.StartedAt != nil {
		duration = int(now.Sub(*record.StartedAt).Seconds())
	}

	updates := map[string]any{
		"finished_at": &now,
		"duration":    duration,
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}

	s.recordRepo.UpdateStatus(ctx, id, status, updates)

	// 释放锁
	s.lockRepo.ReleaseLock(ctx, record.ApplicationID, record.EnvName, operatorID, "部署完成")
}

// CreateRollback 创建回滚
func (s *Service) CreateRollback(ctx context.Context, appID uint, envName string, operatorID uint, operatorName string) (*models.DeployRecord, error) {
	// 查找最近一次成功的部署
	lastSuccess, err := s.recordRepo.GetLatestSuccess(ctx, appID, envName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoRollbackVersion
		}
		return nil, err
	}

	// 获取应用信息
	app, err := s.appRepo.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}

	// 创建回滚记录
	record := &models.DeployRecord{
		ApplicationID: appID,
		AppName:       app.Name,
		EnvName:       envName,
		Version:       lastSuccess.Version,
		Branch:        lastSuccess.Branch,
		CommitID:      lastSuccess.CommitID,
		ImageTag:      lastSuccess.ImageTag,
		DeployType:    DeployTypeRollback,
		DeployMethod:  lastSuccess.DeployMethod,
		Description:   fmt.Sprintf("回滚到版本 %s", lastSuccess.Version),
		Status:        StatusPending,
		NeedApproval:  false, // 回滚不需要审批
		Operator:      operatorName,
		OperatorID:    operatorID,
		RollbackFrom:  &lastSuccess.ID,
	}

	if err := s.recordRepo.Create(ctx, record); err != nil {
		return nil, err
	}

	return record, nil
}

// GetAvailableRollback 获取可回滚版本
func (s *Service) GetAvailableRollback(ctx context.Context, appID uint, envName string) (*models.DeployRecord, error) {
	record, err := s.recordRepo.GetLatestSuccess(ctx, appID, envName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoRollbackVersion
		}
		return nil, err
	}
	return record, nil
}

// GetLockStatus 获取锁定状态
func (s *Service) GetLockStatus(ctx context.Context, appID uint, envName string) (*models.DeployLock, error) {
	lock, err := s.lockRepo.GetActiveLock(ctx, appID, envName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return lock, nil
}

// ReleaseLock 手动释放锁
func (s *Service) ReleaseLock(ctx context.Context, appID uint, envName string, operatorID uint, reason string) error {
	lock, err := s.lockRepo.GetActiveLock(ctx, appID, envName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLockNotFound
		}
		return err
	}

	if lock == nil {
		return ErrLockNotFound
	}

	return s.lockRepo.ReleaseLock(ctx, appID, envName, operatorID, reason)
}

// GetStats 获取统计数据
func (s *Service) GetStats(ctx context.Context, filter repository.DeployStatsFilter) (*repository.DeployStats, error) {
	return s.recordRepo.GetStats(ctx, filter)
}

// RequiresApproval 检查是否需要审批
func (s *Service) RequiresApproval(envName string) bool {
	return requireApprovalEnvs[envName]
}

// GetApprovalRecords 获取审批记录
func (s *Service) GetApprovalRecords(ctx context.Context, recordID uint) ([]models.ApprovalRecord, error) {
	return s.approvalRepo.GetByRecordID(ctx, recordID)
}

// ExpireLocks 过期超时的锁（定时任务调用）
func (s *Service) ExpireLocks(ctx context.Context) (int64, error) {
	return s.lockRepo.ExpireLocks(ctx)
}

// TriggerDeployAfterApproval 审批通过后触发部署（实现 DeployTrigger 接口）
func (s *Service) TriggerDeployAfterApproval(ctx context.Context, recordID uint) error {
	record, err := s.recordRepo.GetByID(ctx, recordID)
	if err != nil {
		return fmt.Errorf("获取部署记录失败: %w", err)
	}

	// 检查状态是否为 approved
	if record.Status != StatusApproved && record.Status != StatusPending {
		log.WithField("record_id", recordID).WithField("status", record.Status).
			Warn("部署记录状态不允许自动执行")
		return nil
	}

	// 更新状态为 approved（如果还是 pending）
	if record.Status == StatusPending {
		if err := s.recordRepo.UpdateStatus(ctx, recordID, StatusApproved, map[string]any{}); err != nil {
			return fmt.Errorf("更新状态失败: %w", err)
		}
	}

	// 自动执行部署
	log.WithField("record_id", recordID).Info("审批通过，自动触发部署")
	return s.ExecuteDeploy(ctx, recordID, record.OperatorID, record.Operator)
}

// GetDeployWindowStatus 获取发布窗口状态
func (s *Service) GetDeployWindowStatus(ctx context.Context, appID uint, envName string) (*DeployWindowStatus, error) {
	inWindow, allowEmergency, windowInfo, err := s.CheckDeployWindow(ctx, appID, envName)
	if err != nil {
		return nil, err
	}

	status := &DeployWindowStatus{
		InWindow:       inWindow,
		AllowEmergency: allowEmergency,
	}

	if windowInfo != nil {
		status.WindowInfo = &WindowInfo{
			Weekdays:  windowInfo.Weekdays,
			StartTime: windowInfo.StartTime,
			EndTime:   windowInfo.EndTime,
		}
	}

	return status, nil
}

// DeployWindowStatus 发布窗口状态
type DeployWindowStatus struct {
	InWindow       bool        `json:"in_window"`
	AllowEmergency bool        `json:"allow_emergency"`
	WindowInfo     *WindowInfo `json:"window_info,omitempty"`
}

// WindowInfo 窗口信息
type WindowInfo struct {
	Weekdays  string `json:"weekdays"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}
