package kubernetes

import (
	"context"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
	"devops/pkg/logger"
)

// BuildRestConfig 从集群配置构建 REST 配置
func BuildRestConfig(cluster *models.K8sCluster) (*rest.Config, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Kubeconfig))
	if err != nil {
		return nil, err
	}

	// 如果集群配置了跳过 TLS 验证，则设置 InsecureSkipVerify
	if cluster.InsecureSkipTLS {
		config.TLSClientConfig.Insecure = true
		config.TLSClientConfig.CAData = nil
		config.TLSClientConfig.CAFile = ""
	}

	return config, nil
}

type K8sClusterService interface {
	CreateK8sCluster(ctx context.Context, req *dto.CreateK8sClusterRequest) (*dto.K8sClusterResponse, error)
	GetK8sCluster(ctx context.Context, id uint) (*dto.K8sClusterResponse, error)
	GetK8sClusterList(ctx context.Context, req *dto.K8sClusterListRequest) (*dto.K8sClusterListResponse, error)
	UpdateK8sCluster(ctx context.Context, id uint, req *dto.UpdateK8sClusterRequest) (*dto.K8sClusterResponse, error)
	DeleteK8sCluster(ctx context.Context, id uint) error
	SetDefaultK8sCluster(ctx context.Context, id uint) error
	GetDefaultK8sCluster(ctx context.Context) (*dto.K8sClusterResponse, error)
	GetFeishuApps(ctx context.Context, id uint) ([]dto.FeishuAppSimple, error)
	BindFeishuApps(ctx context.Context, id uint, appIDs []uint) error
	TestConnection(ctx context.Context, id uint) (*dto.ConnectionTestResult, error)
	GetNamespaces(ctx context.Context, clusterID uint) ([]string, error)
	GetPods(ctx context.Context, clusterID uint, namespace string) ([]PodSimple, error)
}

type k8sClusterService struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewK8sClusterService(db *gorm.DB) K8sClusterService {
	return &k8sClusterService{db: db, log: logger.NewLogger("info")}
}

func (s *k8sClusterService) CreateK8sCluster(ctx context.Context, req *dto.CreateK8sClusterRequest) (*dto.K8sClusterResponse, error) {
	cluster := &models.K8sCluster{
		Name:        req.Name,
		Kubeconfig:  req.Kubeconfig,
		Description: req.Description,
		Status:      req.Status,
		IsDefault:   req.IsDefault,
	}

	if req.IsDefault {
		s.db.Model(&models.K8sCluster{}).Where("is_default = ?", true).Update("is_default", false)
	}

	if err := s.db.Create(cluster).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建K8s集群失败")
	}

	return s.buildResponse(cluster), nil
}

func (s *k8sClusterService) GetK8sCluster(ctx context.Context, id uint) (*dto.K8sClusterResponse, error) {
	var cluster models.K8sCluster
	if err := s.db.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群失败")
	}
	return s.buildResponse(&cluster), nil
}

func (s *k8sClusterService) GetK8sClusterList(ctx context.Context, req *dto.K8sClusterListRequest) (*dto.K8sClusterListResponse, error) {
	query := s.db.Model(&models.K8sCluster{})

	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", keyword, keyword)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群总数失败")
	}

	var clusters []models.K8sCluster
	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&clusters).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群列表失败")
	}

	items := make([]dto.K8sClusterResponse, len(clusters))
	for i, cluster := range clusters {
		items[i] = *s.buildResponse(&cluster)
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize != 0 {
		totalPages++
	}

	return &dto.K8sClusterListResponse{Total: total, Page: req.Page, PageSize: req.PageSize, TotalPages: totalPages, Items: items}, nil
}

func (s *k8sClusterService) UpdateK8sCluster(ctx context.Context, id uint, req *dto.UpdateK8sClusterRequest) (*dto.K8sClusterResponse, error) {
	var cluster models.K8sCluster
	if err := s.db.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群失败")
	}

	if req.IsDefault {
		s.db.Model(&models.K8sCluster{}).Where("is_default = ? AND id != ?", true, id).Update("is_default", false)
	}

	cluster.Name = req.Name
	cluster.Kubeconfig = req.Kubeconfig
	cluster.Description = req.Description
	cluster.Status = req.Status
	cluster.IsDefault = req.IsDefault

	if err := s.db.Save(&cluster).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "更新K8s集群失败")
	}

	return s.buildResponse(&cluster), nil
}

func (s *k8sClusterService) DeleteK8sCluster(ctx context.Context, id uint) error {
	var cluster models.K8sCluster
	if err := s.db.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群失败")
	}

	if err := s.db.Delete(&cluster).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "删除K8s集群失败")
	}
	return nil
}

func (s *k8sClusterService) SetDefaultK8sCluster(ctx context.Context, id uint) error {
	var cluster models.K8sCluster
	if err := s.db.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群失败")
	}

	s.db.Model(&models.K8sCluster{}).Where("is_default = ?", true).Update("is_default", false)
	cluster.IsDefault = true
	if err := s.db.Save(&cluster).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "设置默认K8s集群失败")
	}
	return nil
}

func (s *k8sClusterService) GetDefaultK8sCluster(ctx context.Context) (*dto.K8sClusterResponse, error) {
	var cluster models.K8sCluster
	if err := s.db.Where("is_default = ?", true).First(&cluster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if err = s.db.Where("status = ?", "active").First(&cluster).Error; err != nil {
				return nil, apperrors.ErrNotFound
			}
		} else {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询默认K8s集群失败")
		}
	}
	return s.buildResponse(&cluster), nil
}

func (s *k8sClusterService) buildResponse(cluster *models.K8sCluster) *dto.K8sClusterResponse {
	return &dto.K8sClusterResponse{
		ID:          cluster.ID,
		Name:        cluster.Name,
		Description: cluster.Description,
		Status:      cluster.Status,
		IsDefault:   cluster.IsDefault,
		CreatedAt:   cluster.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   cluster.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func (s *k8sClusterService) GetFeishuApps(ctx context.Context, id uint) ([]dto.FeishuAppSimple, error) {
	var cluster models.K8sCluster
	if err := s.db.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群失败")
	}

	var bindings []models.K8sClusterFeishuApp
	if err := s.db.Preload("FeishuApp").Where("k8s_cluster_id = ?", id).Find(&bindings).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询绑定的飞书应用失败")
	}

	apps := make([]dto.FeishuAppSimple, len(bindings))
	for i, b := range bindings {
		apps[i] = dto.FeishuAppSimple{ID: b.FeishuApp.ID, Name: b.FeishuApp.Name, AppID: b.FeishuApp.AppID, Project: b.FeishuApp.Project}
	}
	return apps, nil
}

func (s *k8sClusterService) BindFeishuApps(ctx context.Context, id uint, appIDs []uint) error {
	var cluster models.K8sCluster
	if err := s.db.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群失败")
	}

	// 删除旧的绑定
	if err := s.db.Where("k8s_cluster_id = ?", id).Delete(&models.K8sClusterFeishuApp{}).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "删除旧绑定失败")
	}

	// 创建新的绑定
	for _, appID := range appIDs {
		binding := &models.K8sClusterFeishuApp{K8sClusterID: id, FeishuAppID: appID}
		if err := s.db.Create(binding).Error; err != nil {
			return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建绑定失败")
		}
	}
	return nil
}

func (s *k8sClusterService) TestConnection(ctx context.Context, id uint) (*dto.ConnectionTestResult, error) {
	var cluster models.K8sCluster
	if err := s.db.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群失败")
	}

	startTime := time.Now()
	result := &dto.ConnectionTestResult{Connected: false}

	// 使用 BuildRestConfig 构建配置（支持跳过 TLS 验证）
	config, err := BuildRestConfig(&cluster)
	if err != nil {
		result.Error = "Kubeconfig 格式错误，请检查配置内容"
		result.ResponseTimeMs = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// 设置超时
	config.Timeout = 15 * time.Second

	// 创建客户端
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		result.Error = "创建 K8s 客户端失败，请检查 Kubeconfig 配置"
		result.ResponseTimeMs = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// 测试连接 - 获取集群版本
	testCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	version, err := clientset.Discovery().ServerVersion()
	result.ResponseTimeMs = time.Since(startTime).Milliseconds()

	if err != nil {
		// 解析常见错误，提供友好提示
		errStr := strings.ToLower(err.Error())
		switch {
		case strings.Contains(errStr, "connection refused"):
			result.Error = "连接被拒绝，请检查集群地址和端口是否正确"
		case strings.Contains(errStr, "no such host"):
			result.Error = "无法解析主机名，请检查集群地址是否正确"
		case strings.Contains(errStr, "timeout"), strings.Contains(errStr, "deadline exceeded"):
			result.Error = "连接超时，请检查网络或集群是否正常运行"
		case strings.Contains(errStr, "certificate"), strings.Contains(errStr, "x509"):
			result.Error = "证书验证失败，请检查 Kubeconfig 中的证书配置"
		case strings.Contains(errStr, "unauthorized"), strings.Contains(errStr, "401"):
			result.Error = "认证失败，请检查 Kubeconfig 中的认证信息"
		case strings.Contains(errStr, "forbidden"), strings.Contains(errStr, "403"):
			result.Error = "权限不足，请检查用户是否有访问集群的权限"
		default:
			result.Error = err.Error()
		}
		return result, nil
	}

	// 获取节点数量
	nodes, err := clientset.CoreV1().Nodes().List(testCtx, metav1.ListOptions{})
	nodeCount := 0
	if err == nil {
		nodeCount = len(nodes.Items)
	}

	result.Connected = true
	result.ServerVersion = version.GitVersion
	result.NodeCount = nodeCount
	return result, nil
}

// GetNamespaces 获取命名空间列表
func (s *k8sClusterService) GetNamespaces(ctx context.Context, clusterID uint) ([]string, error) {
	var cluster models.K8sCluster
	if err := s.db.First(&cluster, clusterID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群失败")
	}

	config, err := BuildRestConfig(&cluster)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "解析kubeconfig失败")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建k8s客户端失败")
	}

	nsList, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取命名空间失败")
	}

	result := make([]string, len(nsList.Items))
	for i, ns := range nsList.Items {
		result[i] = ns.Name
	}
	return result, nil
}

// PodSimple Pod简要信息
type PodSimple struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// GetPods 获取Pod列表
func (s *k8sClusterService) GetPods(ctx context.Context, clusterID uint, namespace string) ([]PodSimple, error) {
	var cluster models.K8sCluster
	if err := s.db.First(&cluster, clusterID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询K8s集群失败")
	}

	config, err := BuildRestConfig(&cluster)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "解析kubeconfig失败")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建k8s客户端失败")
	}

	podList, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取Pod列表失败")
	}

	result := make([]PodSimple, len(podList.Items))
	for i, pod := range podList.Items {
		result[i] = PodSimple{
			Name:   pod.Name,
			Status: string(pod.Status.Phase),
		}
	}
	return result, nil
}
