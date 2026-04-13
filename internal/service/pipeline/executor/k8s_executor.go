package executor

import (
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sservice "devops/internal/service/kubernetes"
	"devops/pkg/dto"
)

// K8sDeployExecutor K8s部署执行器
type K8sDeployExecutor struct {
	db        *gorm.DB
	clientMgr *k8sservice.K8sClientManager
}

// NewK8sDeployExecutor 创建K8s部署执行器
func NewK8sDeployExecutor(db *gorm.DB) *K8sDeployExecutor {
	return &K8sDeployExecutor{
		db:        db,
		clientMgr: k8sservice.NewK8sClientManager(db),
	}
}

// Execute 执行K8s部署
func (e *K8sDeployExecutor) Execute(ctx context.Context, step *dto.Step, env map[string]string) (*StepResult, error) {
	clusterID, _ := step.Config["cluster_id"].(float64)
	namespace, _ := step.Config["namespace"].(string)
	deploymentName, _ := step.Config["deployment"].(string)
	imageName, _ := step.Config["image"].(string)

	if imageName == "" {
		imageName = env["IMAGE_NAME"]
	}
	if namespace == "" {
		namespace = "default"
	}

	if clusterID == 0 {
		return nil, fmt.Errorf("缺少cluster_id配置")
	}
	if deploymentName == "" {
		return nil, fmt.Errorf("缺少deployment配置")
	}
	if imageName == "" {
		return nil, fmt.Errorf("缺少image配置")
	}

	// 获取K8s客户端
	client, err := e.clientMgr.GetClient(ctx, uint(clusterID))
	if err != nil {
		return nil, fmt.Errorf("获取K8s客户端失败: %v", err)
	}

	var logs strings.Builder

	// 获取现有Deployment
	deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		// 如果不存在，创建新的
		logs.WriteString(fmt.Sprintf("Deployment %s 不存在，创建新的...\n", deploymentName))

		replicas := int32(1)
		if r, ok := step.Config["replicas"].(float64); ok {
			replicas = int32(r)
		}

		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      deploymentName,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": deploymentName,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": deploymentName,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  deploymentName,
								Image: imageName,
							},
						},
					},
				},
			},
		}

		_, err = client.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
		if err != nil {
			return &StepResult{Logs: logs.String(), ExitCode: 1}, fmt.Errorf("创建Deployment失败: %v", err)
		}

		logs.WriteString(fmt.Sprintf("Deployment %s 创建成功\n", deploymentName))
	} else {
		// 更新镜像
		logs.WriteString(fmt.Sprintf("更新Deployment %s 的镜像为 %s\n", deploymentName, imageName))

		for i := range deployment.Spec.Template.Spec.Containers {
			deployment.Spec.Template.Spec.Containers[i].Image = imageName
		}

		_, err = client.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		if err != nil {
			return &StepResult{Logs: logs.String(), ExitCode: 1}, fmt.Errorf("更新Deployment失败: %v", err)
		}

		logs.WriteString(fmt.Sprintf("Deployment %s 更新成功\n", deploymentName))
	}

	return &StepResult{
		Logs:     logs.String(),
		ExitCode: 0,
	}, nil
}

// Validate 验证配置
func (e *K8sDeployExecutor) Validate(config map[string]interface{}) error {
	if _, ok := config["cluster_id"]; !ok {
		return fmt.Errorf("缺少cluster_id配置")
	}
	if _, ok := config["deployment"]; !ok {
		return fmt.Errorf("缺少deployment配置")
	}
	return nil
}

// NotifyExecutor 通知执行器
type NotifyExecutor struct{}

// NewNotifyExecutor 创建通知执行器
func NewNotifyExecutor() *NotifyExecutor {
	return &NotifyExecutor{}
}

// Execute 执行通知
func (e *NotifyExecutor) Execute(ctx context.Context, step *dto.Step, env map[string]string) (*StepResult, error) {
	notifyType, _ := step.Config["type"].(string)
	message, _ := step.Config["message"].(string)

	if message == "" {
		message = "流水线执行完成"
	}

	var logs strings.Builder
	logs.WriteString(fmt.Sprintf("发送%s通知: %s\n", notifyType, message))

	// TODO: 实现实际的通知发送
	// 根据notifyType调用不同的通知服务

	logs.WriteString("通知发送成功\n")

	return &StepResult{
		Logs:     logs.String(),
		ExitCode: 0,
	}, nil
}

// Validate 验证配置
func (e *NotifyExecutor) Validate(config map[string]interface{}) error {
	return nil
}
