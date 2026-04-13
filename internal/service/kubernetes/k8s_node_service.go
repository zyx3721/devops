package kubernetes

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
)

// 随机字符串生成器
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func generateRandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// calculateCAHash 计算 CA 证书的 SHA256 哈希
func calculateCAHash(caCert []byte) string {
	hash := sha256.Sum256(caCert)
	return hex.EncodeToString(hash[:])
}

// extractCAFromKubeconfig 从 kubeconfig 中提取 CA 证书
func extractCAFromKubeconfig(kubeconfigData string) ([]byte, error) {
	// 简单解析 YAML 获取 certificate-authority-data
	lines := strings.Split(kubeconfigData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "certificate-authority-data:") {
			caData := strings.TrimPrefix(line, "certificate-authority-data:")
			caData = strings.TrimSpace(caData)
			decoded, err := base64.StdEncoding.DecodeString(caData)
			if err != nil {
				return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "解码 CA 证书失败")
			}
			return decoded, nil
		}
	}
	return nil, apperrors.New(apperrors.ErrCodeInternalError, "未找到 CA 证书数据")
}

// K8sNodeService 节点管理服务
type K8sNodeService struct {
	clientMgr *K8sClientManager
}

// NewK8sNodeService 创建节点管理服务
func NewK8sNodeService(clientMgr *K8sClientManager) *K8sNodeService {
	return &K8sNodeService{clientMgr: clientMgr}
}

// GetNodes 获取节点列表
func (s *K8sNodeService) GetNodes(ctx context.Context, clusterID uint) ([]dto.K8sNode, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	nodeList, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取节点失败")
	}

	result := make([]dto.K8sNode, len(nodeList.Items))
	for i, node := range nodeList.Items {
		// 获取节点状态
		status := "Unknown"
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady {
				if cond.Status == corev1.ConditionTrue {
					status = "Ready"
				} else {
					status = "NotReady"
				}
				break
			}
		}

		// 检查是否可调度
		schedulable := !node.Spec.Unschedulable

		// 获取角色
		roles := []string{}
		for label := range node.Labels {
			if label == "node-role.kubernetes.io/master" || label == "node-role.kubernetes.io/control-plane" {
				roles = append(roles, "master")
			}
			if label == "node-role.kubernetes.io/worker" {
				roles = append(roles, "worker")
			}
		}
		if len(roles) == 0 {
			roles = append(roles, "worker")
		}

		// 获取资源信息
		cpuCapacity := node.Status.Capacity.Cpu().String()
		memCapacity := node.Status.Capacity.Memory().String()
		cpuAllocatable := node.Status.Allocatable.Cpu().String()
		memAllocatable := node.Status.Allocatable.Memory().String()
		podCapacity := node.Status.Capacity.Pods().String()

		// 获取地址
		internalIP := ""
		hostname := ""
		for _, addr := range node.Status.Addresses {
			switch addr.Type {
			case corev1.NodeInternalIP:
				internalIP = addr.Address
			case corev1.NodeHostName:
				hostname = addr.Address
			}
		}

		// 获取污点
		taints := make([]dto.K8sNodeTaint, len(node.Spec.Taints))
		for j, taint := range node.Spec.Taints {
			taints[j] = dto.K8sNodeTaint{
				Key:    taint.Key,
				Value:  taint.Value,
				Effect: string(taint.Effect),
			}
		}

		// 获取系统信息
		nodeInfo := node.Status.NodeInfo

		result[i] = dto.K8sNode{
			Name:              node.Name,
			Status:            status,
			Roles:             roles,
			InternalIP:        internalIP,
			Hostname:          hostname,
			CPUCapacity:       cpuCapacity,
			MemoryCapacity:    memCapacity,
			CPUAllocatable:    cpuAllocatable,
			MemoryAllocatable: memAllocatable,
			PodCapacity:       podCapacity,
			Schedulable:       schedulable,
			Taints:            taints,
			Labels:            node.Labels,
			KubeletVersion:    nodeInfo.KubeletVersion,
			ContainerRuntime:  nodeInfo.ContainerRuntimeVersion,
			OSImage:           nodeInfo.OSImage,
			KernelVersion:     nodeInfo.KernelVersion,
			Architecture:      nodeInfo.Architecture,
			CreatedAt:         node.CreationTimestamp.Format("2006-01-02 15:04:05"),
		}
	}
	return result, nil
}

// GetNodeDetail 获取节点详情
func (s *K8sNodeService) GetNodeDetail(ctx context.Context, clusterID uint, nodeName string) (*dto.K8sNodeDetail, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取节点详情失败")
	}

	// 获取节点上的 Pod
	podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + nodeName,
	})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取节点Pod失败")
	}

	pods := make([]dto.K8sNodePod, len(podList.Items))
	for i, pod := range podList.Items {
		pods[i] = dto.K8sNodePod{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
			IP:        pod.Status.PodIP,
		}
	}

	// 获取节点条件
	conditions := make([]dto.K8sNodeCondition, len(node.Status.Conditions))
	for i, cond := range node.Status.Conditions {
		conditions[i] = dto.K8sNodeCondition{
			Type:    string(cond.Type),
			Status:  string(cond.Status),
			Reason:  cond.Reason,
			Message: cond.Message,
		}
	}

	// 获取污点
	taints := make([]dto.K8sNodeTaint, len(node.Spec.Taints))
	for i, taint := range node.Spec.Taints {
		taints[i] = dto.K8sNodeTaint{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: string(taint.Effect),
		}
	}

	return &dto.K8sNodeDetail{
		Name:             node.Name,
		Labels:           node.Labels,
		Annotations:      node.Annotations,
		Taints:           taints,
		Conditions:       conditions,
		Pods:             pods,
		PodCount:         len(pods),
		Schedulable:      !node.Spec.Unschedulable,
		CPUCapacity:      node.Status.Capacity.Cpu().String(),
		MemoryCapacity:   node.Status.Capacity.Memory().String(),
		CPUAllocatable:   node.Status.Allocatable.Cpu().String(),
		MemoryAllocatable: node.Status.Allocatable.Memory().String(),
		CreatedAt:        node.CreationTimestamp.Format("2006-01-02 15:04:05"),
	}, nil
}

// CordonNode 设置节点不可调度
func (s *K8sNodeService) CordonNode(ctx context.Context, clusterID uint, nodeName string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	patch := []byte(`{"spec":{"unschedulable":true}}`)
	_, err = client.CoreV1().Nodes().Patch(ctx, nodeName, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "设置节点不可调度失败")
	}
	return nil
}

// UncordonNode 设置节点可调度
func (s *K8sNodeService) UncordonNode(ctx context.Context, clusterID uint, nodeName string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	patch := []byte(`{"spec":{"unschedulable":false}}`)
	_, err = client.CoreV1().Nodes().Patch(ctx, nodeName, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "设置节点可调度失败")
	}
	return nil
}

// AddNodeTaint 添加节点污点
func (s *K8sNodeService) AddNodeTaint(ctx context.Context, clusterID uint, nodeName string, taint dto.K8sNodeTaint) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取节点失败")
	}

	// 检查污点是否已存在
	for _, t := range node.Spec.Taints {
		if t.Key == taint.Key && t.Effect == corev1.TaintEffect(taint.Effect) {
			return apperrors.New(apperrors.ErrCodeInvalidParams, "污点已存在")
		}
	}

	newTaint := corev1.Taint{
		Key:    taint.Key,
		Value:  taint.Value,
		Effect: corev1.TaintEffect(taint.Effect),
	}
	node.Spec.Taints = append(node.Spec.Taints, newTaint)

	_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "添加污点失败")
	}
	return nil
}

// RemoveNodeTaint 移除节点污点
func (s *K8sNodeService) RemoveNodeTaint(ctx context.Context, clusterID uint, nodeName, taintKey, taintEffect string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取节点失败")
	}

	newTaints := []corev1.Taint{}
	found := false
	for _, t := range node.Spec.Taints {
		if t.Key == taintKey && string(t.Effect) == taintEffect {
			found = true
			continue
		}
		newTaints = append(newTaints, t)
	}

	if !found {
		return apperrors.New(apperrors.ErrCodeNotFound, "污点不存在")
	}

	node.Spec.Taints = newTaints
	_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "移除污点失败")
	}
	return nil
}

// UpdateNodeLabels 更新节点标签
func (s *K8sNodeService) UpdateNodeLabels(ctx context.Context, clusterID uint, nodeName string, labels map[string]string) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	labelsJSON, _ := json.Marshal(labels)
	patch := fmt.Sprintf(`{"metadata":{"labels":%s}}`, string(labelsJSON))

	_, err = client.CoreV1().Nodes().Patch(ctx, nodeName, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "更新节点标签失败")
	}
	return nil
}

// GetEvents 获取事件列表
func (s *K8sNodeService) GetEvents(ctx context.Context, clusterID uint, namespace string) ([]dto.K8sEvent, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	eventList, err := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取事件失败")
	}

	result := make([]dto.K8sEvent, len(eventList.Items))
	for i, event := range eventList.Items {
		lastTimestamp := ""
		if !event.LastTimestamp.IsZero() {
			lastTimestamp = event.LastTimestamp.Format("2006-01-02 15:04:05")
		}
		result[i] = dto.K8sEvent{
			Name:          event.Name,
			Namespace:     event.Namespace,
			Type:          event.Type,
			Reason:        event.Reason,
			Message:       event.Message,
			Object:        event.InvolvedObject.Kind + "/" + event.InvolvedObject.Name,
			Count:         event.Count,
			LastTimestamp: lastTimestamp,
		}
	}
	return result, nil
}

// GetJoinCommand 获取节点加入命令
// 注意：此功能需要集群支持 bootstrap token，并且需要有权限创建 token
func (s *K8sNodeService) GetJoinCommand(ctx context.Context, clusterID uint) (string, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return "", err
	}

	// 获取 API Server 地址
	// 从 kubeconfig 或集群信息中获取
	config, err := s.clientMgr.GetConfig(ctx, clusterID)
	if err != nil {
		return "", err
	}
	apiServer := config.Host

	// 生成 bootstrap token
	// Token 格式: [a-z0-9]{6}.[a-z0-9]{16}
	tokenID := generateRandomString(6)
	tokenSecret := generateRandomString(16)
	token := tokenID + "." + tokenSecret

	// 创建 bootstrap token secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bootstrap-token-" + tokenID,
			Namespace: "kube-system",
		},
		Type: corev1.SecretTypeBootstrapToken,
		StringData: map[string]string{
			"token-id":                       tokenID,
			"token-secret":                   tokenSecret,
			"usage-bootstrap-authentication": "true",
			"usage-bootstrap-signing":        "true",
			"auth-extra-groups":              "system:bootstrappers:kubeadm:default-node-token",
			// Token 有效期 24 小时
			"expiration": metav1.Now().Add(24 * 60 * 60 * 1e9).Format("2006-01-02T15:04:05Z07:00"),
		},
	}

	_, err = client.CoreV1().Secrets("kube-system").Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return "", apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建 bootstrap token 失败")
	}

	// 获取 CA 证书哈希
	caHash, err := s.getCAHash(ctx, client)
	if err != nil {
		return "", err
	}

	// 构建 kubeadm join 命令
	joinCommand := fmt.Sprintf("kubeadm join %s --token %s --discovery-token-ca-cert-hash sha256:%s", apiServer, token, caHash)

	return joinCommand, nil
}

// GetResourceEvents 获取特定资源的事件
func (s *K8sNodeService) GetResourceEvents(ctx context.Context, clusterID uint, resourceType, namespace, name string) ([]dto.K8sEvent, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	// 根据资源类型确定 Kind
	kindMap := map[string]string{
		"deployment":     "Deployment",
		"statefulset":    "StatefulSet",
		"daemonset":      "DaemonSet",
		"job":            "Job",
		"cronjob":        "CronJob",
		"pod":            "Pod",
		"service":        "Service",
		"ingress":        "Ingress",
		"configmap":      "ConfigMap",
		"secret":         "Secret",
		"pvc":            "PersistentVolumeClaim",
		"pv":             "PersistentVolume",
		"node":           "Node",
		"namespace":      "Namespace",
		"storageclass":   "StorageClass",
		"serviceaccount": "ServiceAccount",
	}

	kind, ok := kindMap[resourceType]
	if !ok {
		kind = resourceType
	}

	// 获取事件，使用 fieldSelector 过滤
	fieldSelector := fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=%s", name, kind)
	
	// 对于集群级资源，不需要命名空间
	var eventList *corev1.EventList
	if resourceType == "node" || resourceType == "pv" || resourceType == "storageclass" || resourceType == "namespace" {
		eventList, err = client.CoreV1().Events("").List(ctx, metav1.ListOptions{
			FieldSelector: fieldSelector,
		})
	} else {
		eventList, err = client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
			FieldSelector: fieldSelector,
		})
	}

	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取资源事件失败")
	}

	result := make([]dto.K8sEvent, len(eventList.Items))
	for i, event := range eventList.Items {
		lastTimestamp := ""
		if !event.LastTimestamp.IsZero() {
			lastTimestamp = event.LastTimestamp.Format("2006-01-02 15:04:05")
		}
		result[i] = dto.K8sEvent{
			Name:          event.Name,
			Namespace:     event.Namespace,
			Type:          event.Type,
			Reason:        event.Reason,
			Message:       event.Message,
			Object:        event.InvolvedObject.Kind + "/" + event.InvolvedObject.Name,
			Count:         event.Count,
			LastTimestamp: lastTimestamp,
		}
	}
	return result, nil
}

// getCAHash 获取 CA 证书哈希
func (s *K8sNodeService) getCAHash(ctx context.Context, client kubernetes.Interface) (string, error) {
	// 从 kube-system 命名空间获取 cluster-info ConfigMap
	cm, err := client.CoreV1().ConfigMaps("kube-public").Get(ctx, "cluster-info", metav1.GetOptions{})
	if err != nil {
		// 如果获取失败，尝试从 kube-root-ca.crt 获取
		cm, err = client.CoreV1().ConfigMaps("kube-system").Get(ctx, "kube-root-ca.crt", metav1.GetOptions{})
		if err != nil {
			return "", apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取 CA 证书失败")
		}
		caCert := cm.Data["ca.crt"]
		return calculateCAHash([]byte(caCert)), nil
	}

	// 从 cluster-info 中解析 kubeconfig
	kubeconfigData := cm.Data["kubeconfig"]
	if kubeconfigData == "" {
		return "", apperrors.New(apperrors.ErrCodeInternalError, "cluster-info 中没有 kubeconfig 数据")
	}

	// 简单解析获取 CA 数据
	// 实际生产中应该使用 yaml 解析
	caCert, err := extractCAFromKubeconfig(kubeconfigData)
	if err != nil {
		return "", err
	}

	return calculateCAHash(caCert), nil
}
