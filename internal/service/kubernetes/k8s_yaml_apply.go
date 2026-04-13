package kubernetes

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/yaml"

	apperrors "devops/pkg/errors"
)

// gvrMap 常用资源的 GVR 映射
var gvrMap = map[string]schema.GroupVersionResource{
	"Deployment":            {Group: "apps", Version: "v1", Resource: "deployments"},
	"StatefulSet":           {Group: "apps", Version: "v1", Resource: "statefulsets"},
	"DaemonSet":             {Group: "apps", Version: "v1", Resource: "daemonsets"},
	"ReplicaSet":            {Group: "apps", Version: "v1", Resource: "replicasets"},
	"Service":               {Group: "", Version: "v1", Resource: "services"},
	"ConfigMap":             {Group: "", Version: "v1", Resource: "configmaps"},
	"Secret":                {Group: "", Version: "v1", Resource: "secrets"},
	"Pod":                   {Group: "", Version: "v1", Resource: "pods"},
	"Namespace":             {Group: "", Version: "v1", Resource: "namespaces"},
	"Node":                  {Group: "", Version: "v1", Resource: "nodes"},
	"PersistentVolume":      {Group: "", Version: "v1", Resource: "persistentvolumes"},
	"PersistentVolumeClaim": {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
	"ServiceAccount":        {Group: "", Version: "v1", Resource: "serviceaccounts"},
	"Ingress":               {Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"},
	"Job":                   {Group: "batch", Version: "v1", Resource: "jobs"},
	"CronJob":               {Group: "batch", Version: "v1", Resource: "cronjobs"},
	"StorageClass":          {Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"},
}

// isNamespaced 判断资源是否是命名空间级别
func isNamespaced(kind string) bool {
	clusterScoped := map[string]bool{
		"Namespace":        true,
		"Node":             true,
		"PersistentVolume": true,
		"StorageClass":     true,
		"ClusterRole":      true,
		"ClusterRoleBinding": true,
	}
	return !clusterScoped[kind]
}

// ApplyYAMLSimple 简化版本的 Apply，使用预定义的 GVR 映射
func (s *K8sYAMLService) ApplyYAMLSimple(ctx context.Context, clusterID uint, yamlContent string) error {
	config, err := s.clientMgr.GetConfig(ctx, clusterID)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "获取集群配置失败")
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建dynamic client失败")
	}

	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(yamlContent), &obj.Object); err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInvalidParams, "解析YAML失败: "+err.Error())
	}

	kind := obj.GetKind()
	if kind == "" {
		return apperrors.New(apperrors.ErrCodeInvalidParams, "YAML中缺少kind")
	}

	name := obj.GetName()
	if name == "" {
		return apperrors.New(apperrors.ErrCodeInvalidParams, "YAML中缺少metadata.name")
	}

	gvr, ok := gvrMap[kind]
	if !ok {
		return apperrors.New(apperrors.ErrCodeInvalidParams, "不支持的资源类型: "+kind)
	}

	namespace := obj.GetNamespace()
	if namespace == "" && isNamespaced(kind) {
		namespace = "default"
		obj.SetNamespace(namespace)
	}

	// 清理不需要的字段
	unstructured.RemoveNestedField(obj.Object, "metadata", "resourceVersion")
	unstructured.RemoveNestedField(obj.Object, "metadata", "uid")
	unstructured.RemoveNestedField(obj.Object, "metadata", "creationTimestamp")
	unstructured.RemoveNestedField(obj.Object, "metadata", "managedFields")
	unstructured.RemoveNestedField(obj.Object, "metadata", "generation")
	unstructured.RemoveNestedField(obj.Object, "status")

	var dr dynamic.ResourceInterface
	if isNamespaced(kind) {
		dr = dynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		dr = dynamicClient.Resource(gvr)
	}

	// 使用传统的 Create/Update 方式
	return s.createOrUpdate(ctx, dr, obj)
}

// createOrUpdate 传统的创建或更新方式
func (s *K8sYAMLService) createOrUpdate(ctx context.Context, dr dynamic.ResourceInterface, obj *unstructured.Unstructured) error {
	name := obj.GetName()
	
	existing, err := dr.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		// 不存在，创建
		_, err = dr.Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建资源失败")
		}
		return nil
	}

	// 存在，更新
	obj.SetResourceVersion(existing.GetResourceVersion())
	_, err = dr.Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "更新资源失败")
	}
	return nil
}
