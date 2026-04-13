// Package infrastructure 定义基础设施相关的数据模型
//
// 本包包含与基础设施管理相关的所有数据模型，包括：
//   - Jenkins：实例配置、构建任务、凭证管理
//   - Kubernetes：集群配置、节点信息、资源管理
//   - CronHPA：定时弹性伸缩配置
//
// 使用示例:
//
//	import "devops/internal/models/infrastructure"
//
//	// 创建 K8s 集群配置
//	cluster := &infrastructure.K8sCluster{
//	    Name:       "prod-cluster",
//	    APIServer:  "https://k8s.example.com:6443",
//	}
package infrastructure
