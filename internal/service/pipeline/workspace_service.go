package pipeline

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gorm.io/gorm"

	"devops/internal/models"
	k8sservice "devops/internal/service/kubernetes"
	"devops/pkg/logger"
)

// WorkspaceService 工作空间服务
type WorkspaceService struct {
	db        *gorm.DB
	clientMgr *k8sservice.K8sClientManager
}

// NewWorkspaceService 创建工作空间服务
func NewWorkspaceService(db *gorm.DB) *WorkspaceService {
	return &WorkspaceService{
		db:        db,
		clientMgr: k8sservice.NewK8sClientManager(db),
	}
}

// CreateWorkspace 创建工作空间
func (s *WorkspaceService) CreateWorkspace(ctx context.Context, pipelineRunID, clusterID uint, namespace, storageSize string) (*models.BuildWorkspace, error) {
	log := logger.L().WithField("pipeline_run_id", pipelineRunID)
	log.Info("创建构建工作空间")

	// 生成 PVC 名称
	pvcName := fmt.Sprintf("workspace-%d-%d", pipelineRunID, time.Now().Unix())

	// 获取 K8s 客户端
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, fmt.Errorf("获取 K8s 客户端失败: %v", err)
	}

	// 默认存储大小
	if storageSize == "" {
		storageSize = "10Gi"
	}

	// 创建 PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":             "devops-build",
				"pipeline-run-id": fmt.Sprintf("%d", pipelineRunID),
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(storageSize),
				},
			},
		},
	}

	createdPVC, err := client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("创建 PVC 失败: %v", err)
	}

	// 保存工作空间记录
	workspace := &models.BuildWorkspace{
		PipelineRunID: pipelineRunID,
		ClusterID:     clusterID,
		Namespace:     namespace,
		PVCName:       createdPVC.Name,
		StorageSize:   storageSize,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	if err := s.db.Create(workspace).Error; err != nil {
		// 清理已创建的 PVC
		client.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, createdPVC.Name, metav1.DeleteOptions{})
		return nil, fmt.Errorf("保存工作空间记录失败: %v", err)
	}

	log.WithField("pvc_name", pvcName).Info("工作空间创建成功")
	return workspace, nil
}

// WaitForWorkspaceReady 等待工作空间就绪
func (s *WorkspaceService) WaitForWorkspaceReady(ctx context.Context, workspace *models.BuildWorkspace, timeout time.Duration) error {
	log := logger.L().WithField("pvc_name", workspace.PVCName)
	log.Info("等待工作空间就绪")

	client, err := s.clientMgr.GetClient(ctx, workspace.ClusterID)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pvc, err := client.CoreV1().PersistentVolumeClaims(workspace.Namespace).Get(ctx, workspace.PVCName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if pvc.Status.Phase == corev1.ClaimBound {
			workspace.Status = "ready"
			s.db.Save(workspace)
			log.Info("工作空间已就绪")
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			continue
		}
	}

	return fmt.Errorf("等待工作空间就绪超时")
}

// CleanupWorkspace 清理工作空间
func (s *WorkspaceService) CleanupWorkspace(ctx context.Context, workspace *models.BuildWorkspace) error {
	log := logger.L().WithField("pvc_name", workspace.PVCName)
	log.Info("清理工作空间")

	// 更新状态
	workspace.Status = "cleaning"
	s.db.Save(workspace)

	client, err := s.clientMgr.GetClient(ctx, workspace.ClusterID)
	if err != nil {
		return err
	}

	// 删除 PVC
	err = client.CoreV1().PersistentVolumeClaims(workspace.Namespace).Delete(ctx, workspace.PVCName, metav1.DeleteOptions{})
	if err != nil {
		log.WithError(err).Warn("删除 PVC 失败")
	}

	// 更新状态
	now := time.Now()
	workspace.Status = "deleted"
	workspace.DeletedAt = &now
	s.db.Save(workspace)

	log.Info("工作空间清理完成")
	return nil
}

// GetWorkspace 获取工作空间
func (s *WorkspaceService) GetWorkspace(ctx context.Context, pipelineRunID uint) (*models.BuildWorkspace, error) {
	var workspace models.BuildWorkspace
	if err := s.db.Where("pipeline_run_id = ?", pipelineRunID).First(&workspace).Error; err != nil {
		return nil, err
	}
	return &workspace, nil
}

// CleanupExpiredWorkspaces 清理过期工作空间
func (s *WorkspaceService) CleanupExpiredWorkspaces(ctx context.Context, maxAge time.Duration) error {
	log := logger.L()
	log.Info("开始清理过期工作空间")

	var workspaces []models.BuildWorkspace
	cutoff := time.Now().Add(-maxAge)

	if err := s.db.Where("status IN (?, ?) AND created_at < ?", "ready", "pending", cutoff).Find(&workspaces).Error; err != nil {
		return err
	}

	for _, ws := range workspaces {
		if err := s.CleanupWorkspace(ctx, &ws); err != nil {
			log.WithError(err).WithField("workspace_id", ws.ID).Warn("清理工作空间失败")
		}
	}

	log.WithField("count", len(workspaces)).Info("过期工作空间清理完成")
	return nil
}
