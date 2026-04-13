package kubernetes

import (
	"context"

	"gorm.io/gorm"

	"devops/pkg/dto"
)

// K8sResourceService K8s 资源服务接口
type K8sResourceService interface {
	// 命名空间
	GetNamespaces(ctx context.Context, clusterID uint) ([]dto.K8sNamespace, error)
	CreateNamespace(ctx context.Context, clusterID uint, name string, labels map[string]string) error
	DeleteNamespace(ctx context.Context, clusterID uint, name string) error
	// 工作负载
	GetDeployments(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sDeployment, error)
	GetStatefulSets(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sStatefulSet, error)
	GetDaemonSets(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sDaemonSet, error)
	GetJobs(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sJob, error)
	GetCronJobs(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sCronJob, error)
	GetPods(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sPod, error)
	GetPodLogs(ctx context.Context, clusterID uint, namespace, podName, container string, tailLines int64) (string, error)
	DeletePod(ctx context.Context, clusterID uint, namespace, podName string) error
	RestartDeployment(ctx context.Context, clusterID uint, namespace, deploymentName string) error
	ScaleDeployment(ctx context.Context, clusterID uint, namespace, deploymentName string, replicas int32) error
	// 网络
	GetServices(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sService, error)
	GetIngresses(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sIngress, error)
	GetEndpoints(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sEndpoint, error)
	// 配置
	GetConfigMaps(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sConfigMap, error)
	GetSecrets(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sSecret, error)
	GetServiceAccounts(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sServiceAccount, error)
	// 存储
	GetPVCs(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sPVC, error)
	GetPVs(ctx context.Context, clusterID uint) ([]dto.K8sPV, error)
	GetStorageClasses(ctx context.Context, clusterID uint) ([]dto.K8sStorageClass, error)
	// 节点
	GetNodes(ctx context.Context, clusterID uint) ([]dto.K8sNode, error)
	GetNodeDetail(ctx context.Context, clusterID uint, nodeName string) (*dto.K8sNodeDetail, error)
	CordonNode(ctx context.Context, clusterID uint, nodeName string) error
	UncordonNode(ctx context.Context, clusterID uint, nodeName string) error
	AddNodeTaint(ctx context.Context, clusterID uint, nodeName string, taint dto.K8sNodeTaint) error
	RemoveNodeTaint(ctx context.Context, clusterID uint, nodeName, taintKey, taintEffect string) error
	UpdateNodeLabels(ctx context.Context, clusterID uint, nodeName string, labels map[string]string) error
	GetEvents(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sEvent, error)
	GetJoinCommand(ctx context.Context, clusterID uint) (string, error)
	// YAML 操作
	GetResourceYAML(ctx context.Context, clusterID uint, resourceType, namespace, name string) (string, error)
	ApplyResourceYAML(ctx context.Context, clusterID uint, yaml string) error
	DeleteResource(ctx context.Context, clusterID uint, resourceType, namespace, name string) error
	// 资源详情和关联
	GetResourceDetail(ctx context.Context, clusterID uint, resourceType, namespace, name string) (any, error)
	GetResourceEvents(ctx context.Context, clusterID uint, resourceType, namespace, name string) ([]dto.K8sEvent, error)
	GetRelatedPods(ctx context.Context, clusterID uint, ownerType, namespace, ownerName string) ([]dto.K8sPod, error)
	GetServicePods(ctx context.Context, clusterID uint, namespace, serviceName string) ([]dto.K8sPod, error)
}

// k8sResourceService 聚合服务实现
type k8sResourceService struct {
	clientMgr   *K8sClientManager
	workloadSvc *K8sWorkloadService
	networkSvc  *K8sNetworkService
	configSvc   *K8sConfigService
	storageSvc  *K8sStorageService
	nodeSvc     *K8sNodeService
	yamlSvc     *K8sYAMLService
}

// NewK8sResourceService 创建 K8s 资源服务
func NewK8sResourceService(db *gorm.DB) K8sResourceService {
	clientMgr := NewK8sClientManager(db)
	return &k8sResourceService{
		clientMgr:   clientMgr,
		workloadSvc: NewK8sWorkloadService(clientMgr),
		networkSvc:  NewK8sNetworkService(clientMgr),
		configSvc:   NewK8sConfigService(clientMgr),
		storageSvc:  NewK8sStorageService(clientMgr),
		nodeSvc:     NewK8sNodeService(clientMgr),
		yamlSvc:     NewK8sYAMLService(clientMgr),
	}
}

// 命名空间
func (s *k8sResourceService) GetNamespaces(ctx context.Context, clusterID uint) ([]dto.K8sNamespace, error) {
	return s.configSvc.GetNamespaces(ctx, clusterID)
}

func (s *k8sResourceService) CreateNamespace(ctx context.Context, clusterID uint, name string, labels map[string]string) error {
	return s.configSvc.CreateNamespace(ctx, clusterID, name, labels)
}

func (s *k8sResourceService) DeleteNamespace(ctx context.Context, clusterID uint, name string) error {
	return s.configSvc.DeleteNamespace(ctx, clusterID, name)
}

// 工作负载
func (s *k8sResourceService) GetDeployments(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sDeployment, error) {
	return s.workloadSvc.GetDeployments(ctx, clusterID, namespace)
}

func (s *k8sResourceService) GetStatefulSets(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sStatefulSet, error) {
	return s.workloadSvc.GetStatefulSets(ctx, clusterID, namespace)
}

func (s *k8sResourceService) GetDaemonSets(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sDaemonSet, error) {
	return s.workloadSvc.GetDaemonSets(ctx, clusterID, namespace)
}

func (s *k8sResourceService) GetJobs(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sJob, error) {
	return s.workloadSvc.GetJobs(ctx, clusterID, namespace)
}

func (s *k8sResourceService) GetCronJobs(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sCronJob, error) {
	return s.workloadSvc.GetCronJobs(ctx, clusterID, namespace)
}

func (s *k8sResourceService) GetPods(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sPod, error) {
	return s.workloadSvc.GetPods(ctx, clusterID, namespace)
}

func (s *k8sResourceService) GetPodLogs(ctx context.Context, clusterID uint, namespace, podName, container string, tailLines int64) (string, error) {
	return s.workloadSvc.GetPodLogs(ctx, clusterID, namespace, podName, container, tailLines)
}

func (s *k8sResourceService) DeletePod(ctx context.Context, clusterID uint, namespace, podName string) error {
	return s.workloadSvc.DeletePod(ctx, clusterID, namespace, podName)
}

func (s *k8sResourceService) RestartDeployment(ctx context.Context, clusterID uint, namespace, deploymentName string) error {
	return s.workloadSvc.RestartDeployment(ctx, clusterID, namespace, deploymentName)
}

func (s *k8sResourceService) ScaleDeployment(ctx context.Context, clusterID uint, namespace, deploymentName string, replicas int32) error {
	return s.workloadSvc.ScaleDeployment(ctx, clusterID, namespace, deploymentName, replicas)
}

// 网络
func (s *k8sResourceService) GetServices(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sService, error) {
	return s.networkSvc.GetServices(ctx, clusterID, namespace)
}

func (s *k8sResourceService) GetIngresses(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sIngress, error) {
	return s.networkSvc.GetIngresses(ctx, clusterID, namespace)
}

func (s *k8sResourceService) GetEndpoints(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sEndpoint, error) {
	return s.networkSvc.GetEndpoints(ctx, clusterID, namespace)
}

// 配置
func (s *k8sResourceService) GetConfigMaps(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sConfigMap, error) {
	return s.configSvc.GetConfigMaps(ctx, clusterID, namespace)
}

func (s *k8sResourceService) GetSecrets(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sSecret, error) {
	return s.configSvc.GetSecrets(ctx, clusterID, namespace)
}

func (s *k8sResourceService) GetServiceAccounts(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sServiceAccount, error) {
	return s.configSvc.GetServiceAccounts(ctx, clusterID, namespace)
}

// 存储
func (s *k8sResourceService) GetPVCs(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sPVC, error) {
	return s.storageSvc.GetPVCs(ctx, clusterID, namespace)
}

func (s *k8sResourceService) GetPVs(ctx context.Context, clusterID uint) ([]dto.K8sPV, error) {
	return s.storageSvc.GetPVs(ctx, clusterID)
}

func (s *k8sResourceService) GetStorageClasses(ctx context.Context, clusterID uint) ([]dto.K8sStorageClass, error) {
	return s.storageSvc.GetStorageClasses(ctx, clusterID)
}

// 节点
func (s *k8sResourceService) GetNodes(ctx context.Context, clusterID uint) ([]dto.K8sNode, error) {
	return s.nodeSvc.GetNodes(ctx, clusterID)
}

func (s *k8sResourceService) GetNodeDetail(ctx context.Context, clusterID uint, nodeName string) (*dto.K8sNodeDetail, error) {
	return s.nodeSvc.GetNodeDetail(ctx, clusterID, nodeName)
}

func (s *k8sResourceService) CordonNode(ctx context.Context, clusterID uint, nodeName string) error {
	return s.nodeSvc.CordonNode(ctx, clusterID, nodeName)
}

func (s *k8sResourceService) UncordonNode(ctx context.Context, clusterID uint, nodeName string) error {
	return s.nodeSvc.UncordonNode(ctx, clusterID, nodeName)
}

func (s *k8sResourceService) AddNodeTaint(ctx context.Context, clusterID uint, nodeName string, taint dto.K8sNodeTaint) error {
	return s.nodeSvc.AddNodeTaint(ctx, clusterID, nodeName, taint)
}

func (s *k8sResourceService) RemoveNodeTaint(ctx context.Context, clusterID uint, nodeName, taintKey, taintEffect string) error {
	return s.nodeSvc.RemoveNodeTaint(ctx, clusterID, nodeName, taintKey, taintEffect)
}

func (s *k8sResourceService) UpdateNodeLabels(ctx context.Context, clusterID uint, nodeName string, labels map[string]string) error {
	return s.nodeSvc.UpdateNodeLabels(ctx, clusterID, nodeName, labels)
}

func (s *k8sResourceService) GetEvents(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sEvent, error) {
	return s.nodeSvc.GetEvents(ctx, clusterID, namespace)
}

func (s *k8sResourceService) GetJoinCommand(ctx context.Context, clusterID uint) (string, error) {
	return s.nodeSvc.GetJoinCommand(ctx, clusterID)
}

// YAML 操作
func (s *k8sResourceService) GetResourceYAML(ctx context.Context, clusterID uint, resourceType, namespace, name string) (string, error) {
	return s.yamlSvc.GetResourceYAML(ctx, clusterID, resourceType, namespace, name)
}

func (s *k8sResourceService) ApplyResourceYAML(ctx context.Context, clusterID uint, yaml string) error {
	return s.yamlSvc.ApplyResourceYAML(ctx, clusterID, yaml)
}

func (s *k8sResourceService) DeleteResource(ctx context.Context, clusterID uint, resourceType, namespace, name string) error {
	return s.yamlSvc.DeleteResource(ctx, clusterID, resourceType, namespace, name)
}

// 资源详情和关联
func (s *k8sResourceService) GetResourceDetail(ctx context.Context, clusterID uint, resourceType, namespace, name string) (any, error) {
	return s.yamlSvc.GetResourceDetail(ctx, clusterID, resourceType, namespace, name)
}

func (s *k8sResourceService) GetResourceEvents(ctx context.Context, clusterID uint, resourceType, namespace, name string) ([]dto.K8sEvent, error) {
	return s.nodeSvc.GetResourceEvents(ctx, clusterID, resourceType, namespace, name)
}

func (s *k8sResourceService) GetRelatedPods(ctx context.Context, clusterID uint, ownerType, namespace, ownerName string) ([]dto.K8sPod, error) {
	return s.workloadSvc.GetRelatedPods(ctx, clusterID, ownerType, namespace, ownerName)
}

func (s *k8sResourceService) GetServicePods(ctx context.Context, clusterID uint, namespace, serviceName string) ([]dto.K8sPod, error) {
	return s.networkSvc.GetServicePods(ctx, clusterID, namespace, serviceName)
}
