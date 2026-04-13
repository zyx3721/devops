package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/utils/ptr"

	"gorm.io/gorm"

	"devops/internal/models"
	k8sservice "devops/internal/service/kubernetes"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// K8sBuildExecutor K8s 构建执行器
type K8sBuildExecutor struct {
	db        *gorm.DB
	clientMgr *k8sservice.K8sClientManager
}

// NewK8sBuildExecutor 创建 K8s 构建执行器
func NewK8sBuildExecutor(db *gorm.DB) *K8sBuildExecutor {
	return &K8sBuildExecutor{
		db:        db,
		clientMgr: k8sservice.NewK8sClientManager(db),
	}
}

// CreateBuildJob 创建构建 Job
func (e *K8sBuildExecutor) CreateBuildJob(ctx context.Context, config *dto.BuildJobConfig) (*models.BuildJob, error) {
	log := logger.L().WithField("step", config.StepName)
	log.Info("创建构建 Job")

	// 生成 Job 名称
	jobName := fmt.Sprintf("build-%d-%s-%d", config.PipelineRunID, config.StepID, time.Now().Unix())
	jobName = strings.ToLower(jobName)
	if len(jobName) > 63 {
		jobName = jobName[:63]
	}

	// 获取 K8s 客户端
	client, err := e.clientMgr.GetClient(ctx, config.ClusterID)
	if err != nil {
		return nil, fmt.Errorf("获取 K8s 客户端失败: %v", err)
	}

	// 构建环境变量
	envVars := e.buildEnvVars(config)

	// 构建资源限制
	resources := e.buildResources(config.Resources)

	// 构建命令
	command := []string{"/bin/sh", "-c"}
	args := []string{strings.Join(config.Commands, " && ")}

	// 构建 Volume 和 VolumeMount
	volumes, volumeMounts := e.buildVolumes(config)

	// 构建 Init Container（Git Clone）
	initContainers := e.buildInitContainers(config, volumeMounts)

	// 创建 Job
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: config.Namespace,
			Labels: map[string]string{
				"app":             "devops-build",
				"pipeline-run-id": fmt.Sprintf("%d", config.PipelineRunID),
				"step-id":         config.StepID,
			},
		},
		Spec: batchv1.JobSpec{
			TTLSecondsAfterFinished: ptr.To(int32(3600)), // 1小时后清理
			BackoffLimit:            ptr.To(int32(0)),    // 不重试
			ActiveDeadlineSeconds:   e.getDeadline(config.Timeout),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":             "devops-build",
						"pipeline-run-id": fmt.Sprintf("%d", config.PipelineRunID),
						"step-id":         config.StepID,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy:  corev1.RestartPolicyNever,
					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name:         "build",
							Image:        config.Image,
							Command:      command,
							Args:         args,
							WorkingDir:   config.WorkDir,
							Env:          envVars,
							Resources:    resources,
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}

	// 创建 Job
	createdJob, err := client.BatchV1().Jobs(config.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("创建 Job 失败: %v", err)
	}

	// 保存构建任务记录
	commandsJSON, _ := json.Marshal(config.Commands)
	envVarsJSON, _ := json.Marshal(config.EnvVars)
	resourcesJSON, _ := json.Marshal(config.Resources)

	buildJob := &models.BuildJob{
		PipelineRunID: config.PipelineRunID,
		StepID:        config.StepID,
		StepName:      config.StepName,
		JobName:       createdJob.Name,
		Namespace:     config.Namespace,
		ClusterID:     config.ClusterID,
		Image:         config.Image,
		Commands:      string(commandsJSON),
		WorkDir:       config.WorkDir,
		EnvVars:       string(envVarsJSON),
		Resources:     string(resourcesJSON),
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	if err := e.db.Create(buildJob).Error; err != nil {
		// 清理已创建的 Job
		client.BatchV1().Jobs(config.Namespace).Delete(ctx, createdJob.Name, metav1.DeleteOptions{})
		return nil, fmt.Errorf("保存构建任务记录失败: %v", err)
	}

	log.WithField("job_name", jobName).Info("构建 Job 创建成功")
	return buildJob, nil
}

// WatchJobStatus 监控 Job 状态
func (e *K8sBuildExecutor) WatchJobStatus(ctx context.Context, buildJob *models.BuildJob) error {
	log := logger.L().WithField("job_name", buildJob.JobName)
	log.Info("开始监控 Job 状态")

	client, err := e.clientMgr.GetClient(ctx, buildJob.ClusterID)
	if err != nil {
		return err
	}

	// 创建 Watch
	watcher, err := client.BatchV1().Jobs(buildJob.Namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", buildJob.JobName),
	})
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}

			if event.Type == watch.Error {
				return fmt.Errorf("watch error")
			}

			job, ok := event.Object.(*batchv1.Job)
			if !ok {
				continue
			}

			// 更新状态
			e.updateBuildJobStatus(ctx, buildJob, job)

			// 检查是否完成
			if job.Status.Succeeded > 0 {
				log.Info("Job 执行成功")
				return nil
			}
			if job.Status.Failed > 0 {
				log.Error("Job 执行失败")
				return fmt.Errorf("job failed")
			}
		}
	}
}

// GetJobLogs 获取 Job 日志
func (e *K8sBuildExecutor) GetJobLogs(ctx context.Context, buildJob *models.BuildJob) (string, error) {
	client, err := e.clientMgr.GetClient(ctx, buildJob.ClusterID)
	if err != nil {
		return "", err
	}

	// 获取 Pod
	pods, err := client.CoreV1().Pods(buildJob.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", buildJob.JobName),
	})
	if err != nil {
		return "", err
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %s", buildJob.JobName)
	}

	pod := pods.Items[0]

	// 获取日志
	req := client.CoreV1().Pods(buildJob.Namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: "build",
	})

	logs, err := req.DoRaw(ctx)
	if err != nil {
		return "", err
	}

	return string(logs), nil
}

// CleanupJob 清理 Job
func (e *K8sBuildExecutor) CleanupJob(ctx context.Context, buildJob *models.BuildJob) error {
	log := logger.L().WithField("job_name", buildJob.JobName)
	log.Info("清理构建 Job")

	client, err := e.clientMgr.GetClient(ctx, buildJob.ClusterID)
	if err != nil {
		return err
	}

	// 删除 Job（级联删除 Pod）
	propagationPolicy := metav1.DeletePropagationBackground
	err = client.BatchV1().Jobs(buildJob.Namespace).Delete(ctx, buildJob.JobName, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		log.WithError(err).Warn("删除 Job 失败")
	}

	return nil
}

// CancelJob 取消 Job
func (e *K8sBuildExecutor) CancelJob(ctx context.Context, buildJob *models.BuildJob) error {
	log := logger.L().WithField("job_name", buildJob.JobName)
	log.Info("取消构建 Job")

	// 清理 Job
	if err := e.CleanupJob(ctx, buildJob); err != nil {
		log.WithError(err).Warn("清理 Job 失败")
	}

	// 更新状态
	now := time.Now()
	buildJob.Status = "cancelled"
	buildJob.FinishedAt = &now
	return e.db.Save(buildJob).Error
}

// buildEnvVars 构建环境变量
func (e *K8sBuildExecutor) buildEnvVars(config *dto.BuildJobConfig) []corev1.EnvVar {
	envVars := make([]corev1.EnvVar, 0)

	// 添加用户定义的环境变量
	for k, v := range config.EnvVars {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	// 添加内置环境变量
	envVars = append(envVars,
		corev1.EnvVar{Name: "CI", Value: "true"},
		corev1.EnvVar{Name: "CI_PIPELINE_RUN_ID", Value: fmt.Sprintf("%d", config.PipelineRunID)},
		corev1.EnvVar{Name: "CI_STEP_ID", Value: config.StepID},
	)

	return envVars
}

// buildResources 构建资源限制
func (e *K8sBuildExecutor) buildResources(config *dto.BuildResourceConfig) corev1.ResourceRequirements {
	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}

	if config == nil {
		// 默认资源
		resources.Requests[corev1.ResourceCPU] = resource.MustParse("100m")
		resources.Requests[corev1.ResourceMemory] = resource.MustParse("256Mi")
		resources.Limits[corev1.ResourceCPU] = resource.MustParse("1")
		resources.Limits[corev1.ResourceMemory] = resource.MustParse("1Gi")
		return resources
	}

	if config.CPURequest != "" {
		resources.Requests[corev1.ResourceCPU] = resource.MustParse(config.CPURequest)
	}
	if config.MemoryRequest != "" {
		resources.Requests[corev1.ResourceMemory] = resource.MustParse(config.MemoryRequest)
	}
	if config.CPULimit != "" {
		resources.Limits[corev1.ResourceCPU] = resource.MustParse(config.CPULimit)
	}
	if config.MemoryLimit != "" {
		resources.Limits[corev1.ResourceMemory] = resource.MustParse(config.MemoryLimit)
	}

	return resources
}

// buildVolumes 构建 Volume
func (e *K8sBuildExecutor) buildVolumes(config *dto.BuildJobConfig) ([]corev1.Volume, []corev1.VolumeMount) {
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}

	// 工作空间 PVC
	if config.WorkspacePVC != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: config.WorkspacePVC,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "workspace",
			MountPath: "/workspace",
		})
	} else {
		// 使用 EmptyDir
		volumes = append(volumes, corev1.Volume{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "workspace",
			MountPath: "/workspace",
		})
	}

	// 缓存 PVC 挂载
	if config.CachePVC != "" {
		volumes = append(volumes, corev1.Volume{
			Name: "cache",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: config.CachePVC,
				},
			},
		})
		// 挂载缓存目录
		for i, cachePath := range config.CachePaths {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      "cache",
				MountPath: cachePath,
				SubPath:   fmt.Sprintf("cache-%d", i),
			})
		}
	}

	return volumes, volumeMounts
}

// buildInitContainers 构建 Init Container
func (e *K8sBuildExecutor) buildInitContainers(config *dto.BuildJobConfig, volumeMounts []corev1.VolumeMount) []corev1.Container {
	if config.GitURL == "" {
		return nil
	}

	// Git Clone Init Container
	gitCloneCmd := fmt.Sprintf("git clone --depth=1 -b %s %s /workspace", config.GitBranch, config.GitURL)

	return []corev1.Container{
		{
			Name:         "git-clone",
			Image:        "alpine/git:latest",
			Command:      []string{"/bin/sh", "-c"},
			Args:         []string{gitCloneCmd},
			VolumeMounts: volumeMounts,
		},
	}
}

// getDeadline 获取超时时间
func (e *K8sBuildExecutor) getDeadline(timeout int) *int64 {
	if timeout <= 0 {
		timeout = 3600 // 默认 1 小时
	}
	deadline := int64(timeout)
	return &deadline
}

// updateBuildJobStatus 更新构建任务状态
func (e *K8sBuildExecutor) updateBuildJobStatus(ctx context.Context, buildJob *models.BuildJob, job *batchv1.Job) {
	// 获取 Pod 信息
	client, err := e.clientMgr.GetClient(ctx, buildJob.ClusterID)
	if err == nil {
		pods, err := client.CoreV1().Pods(buildJob.Namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", buildJob.JobName),
		})
		if err == nil && len(pods.Items) > 0 {
			pod := pods.Items[0]
			buildJob.PodName = pod.Name
			buildJob.NodeName = pod.Spec.NodeName
		}
	}

	// 更新状态
	if job.Status.Active > 0 {
		buildJob.Status = "running"
		if buildJob.StartedAt == nil {
			now := time.Now()
			buildJob.StartedAt = &now
		}
	}
	if job.Status.Succeeded > 0 {
		buildJob.Status = "success"
		now := time.Now()
		buildJob.FinishedAt = &now
		exitCode := 0
		buildJob.ExitCode = &exitCode
	}
	if job.Status.Failed > 0 {
		buildJob.Status = "failed"
		now := time.Now()
		buildJob.FinishedAt = &now
		exitCode := 1
		buildJob.ExitCode = &exitCode

		// 获取失败原因
		for _, condition := range job.Status.Conditions {
			if condition.Type == batchv1.JobFailed {
				buildJob.ErrorMessage = condition.Message
				break
			}
		}
	}

	e.db.Save(buildJob)
}
