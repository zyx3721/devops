package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	"gorm.io/gorm"

	"devops/internal/models"
	k8sservice "devops/internal/service/kubernetes"
	"devops/pkg/logger"
)

// LogService 日志服务
type LogService struct {
	db            *gorm.DB
	clientMgr     *k8sservice.K8sClientManager
	sensitiveKeys []string
	customRules   []SanitizeRule
}

// SanitizeRule 自定义脱敏规则
type SanitizeRule struct {
	Name        string
	Pattern     *regexp.Regexp
	Replacement string
}

// NewLogService 创建日志服务
func NewLogService(db *gorm.DB) *LogService {
	svc := &LogService{
		db:        db,
		clientMgr: k8sservice.NewK8sClientManager(db),
		sensitiveKeys: []string{
			"password", "secret", "token", "key", "credential",
			"api_key", "apikey", "auth", "private", "access_key",
			"secret_key", "aws_secret", "db_password", "mysql_password",
			"postgres_password", "redis_password", "mongo_password",
		},
	}

	// 初始化内置脱敏规则
	svc.initBuiltinRules()
	return svc
}

// initBuiltinRules 初始化内置脱敏规则
func (s *LogService) initBuiltinRules() {
	s.customRules = []SanitizeRule{
		// AWS Access Key ID
		{
			Name:        "AWS Access Key",
			Pattern:     regexp.MustCompile(`(?i)(AKIA|ABIA|ACCA|ASIA)[A-Z0-9]{16}`),
			Replacement: "******AWS_KEY******",
		},
		// AWS Secret Access Key
		{
			Name:        "AWS Secret Key",
			Pattern:     regexp.MustCompile(`(?i)aws_secret_access_key\s*[=:]\s*["']?([A-Za-z0-9/+=]{40})["']?`),
			Replacement: "aws_secret_access_key=******",
		},
		// GitHub Token
		{
			Name:        "GitHub Token",
			Pattern:     regexp.MustCompile(`(ghp_[A-Za-z0-9]{36}|github_pat_[A-Za-z0-9_]{22,})`),
			Replacement: "******GITHUB_TOKEN******",
		},
		// GitLab Token
		{
			Name:        "GitLab Token",
			Pattern:     regexp.MustCompile(`glpat-[A-Za-z0-9\-_]{20,}`),
			Replacement: "******GITLAB_TOKEN******",
		},
		// Docker Registry Password
		{
			Name:        "Docker Password",
			Pattern:     regexp.MustCompile(`(?i)(docker_password|registry_password)\s*[=:]\s*["']?([^"'\s]+)["']?`),
			Replacement: "$1=******",
		},
		// Private Key
		{
			Name:        "Private Key",
			Pattern:     regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----[\s\S]*?-----END (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`),
			Replacement: "******PRIVATE_KEY******",
		},
		// JWT Token
		{
			Name:        "JWT Token",
			Pattern:     regexp.MustCompile(`eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*`),
			Replacement: "******JWT_TOKEN******",
		},
		// Generic API Key (32+ hex chars)
		{
			Name:        "Generic API Key",
			Pattern:     regexp.MustCompile(`(?i)(api[_-]?key)\s*[=:]\s*["']?([a-f0-9]{32,})["']?`),
			Replacement: "$1=******",
		},
		// Connection String with password
		{
			Name:        "Connection String",
			Pattern:     regexp.MustCompile(`(?i)(mysql|postgres|mongodb|redis)://[^:]+:([^@]+)@`),
			Replacement: "$1://***:******@",
		},
		// Slack Webhook
		{
			Name:        "Slack Webhook",
			Pattern:     regexp.MustCompile(`https://hooks\.slack\.com/services/[A-Za-z0-9/]+`),
			Replacement: "******SLACK_WEBHOOK******",
		},
		// Feishu/Lark Webhook
		{
			Name:        "Feishu Webhook",
			Pattern:     regexp.MustCompile(`https://open\.feishu\.cn/open-apis/bot/v2/hook/[A-Za-z0-9-]+`),
			Replacement: "******FEISHU_WEBHOOK******",
		},
		// DingTalk Webhook
		{
			Name:        "DingTalk Webhook",
			Pattern:     regexp.MustCompile(`https://oapi\.dingtalk\.com/robot/send\?access_token=[A-Za-z0-9]+`),
			Replacement: "******DINGTALK_WEBHOOK******",
		},
	}
}

// AddCustomRule 添加自定义脱敏规则
func (s *LogService) AddCustomRule(name, pattern, replacement string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}
	s.customRules = append(s.customRules, SanitizeRule{
		Name:        name,
		Pattern:     re,
		Replacement: replacement,
	})
	return nil
}

// AddSensitiveKey 添加敏感关键字
func (s *LogService) AddSensitiveKey(key string) {
	s.sensitiveKeys = append(s.sensitiveKeys, strings.ToLower(key))
}

// LogStreamCallback 日志流回调函数
type LogStreamCallback func(line string) error

// GetBuildJobLogs 获取构建任务日志
func (s *LogService) GetBuildJobLogs(ctx context.Context, buildJobID uint) (string, error) {
	var buildJob models.BuildJob
	if err := s.db.First(&buildJob, buildJobID).Error; err != nil {
		return "", err
	}

	return s.GetJobLogs(ctx, buildJob.ClusterID, buildJob.Namespace, buildJob.JobName, "build")
}

// GetJobLogs 获取 Job 日志
func (s *LogService) GetJobLogs(ctx context.Context, clusterID uint, namespace, jobName, container string) (string, error) {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return "", err
	}

	// 获取 Pod
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("job-name=%s", jobName),
	})
	if err != nil {
		return "", err
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for job %s", jobName)
	}

	pod := pods.Items[0]

	// 获取日志
	req := client.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
		Container: container,
	})

	logs, err := req.DoRaw(ctx)
	if err != nil {
		return "", err
	}

	// 脱敏处理
	sanitized := s.SanitizeLogs(string(logs))
	return sanitized, nil
}

// StreamBuildJobLogs 流式获取构建任务日志
func (s *LogService) StreamBuildJobLogs(ctx context.Context, buildJobID uint, callback LogStreamCallback) error {
	var buildJob models.BuildJob
	if err := s.db.First(&buildJob, buildJobID).Error; err != nil {
		return err
	}

	return s.StreamJobLogs(ctx, buildJob.ClusterID, buildJob.Namespace, buildJob.JobName, "build", callback)
}

// StreamJobLogs 流式获取 Job 日志
func (s *LogService) StreamJobLogs(ctx context.Context, clusterID uint, namespace, jobName, container string, callback LogStreamCallback) error {
	log := logger.L().WithField("job_name", jobName)
	log.Info("开始流式获取日志")

	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	// 等待 Pod 创建
	var podName string
	for i := 0; i < 30; i++ {
		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("job-name=%s", jobName),
		})
		if err == nil && len(pods.Items) > 0 {
			podName = pods.Items[0].Name
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			continue
		}
	}

	if podName == "" {
		return fmt.Errorf("no pods found for job %s", jobName)
	}

	// 等待容器启动
	for i := 0; i < 60; i++ {
		pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		// 检查容器状态
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Name == container && (cs.State.Running != nil || cs.State.Terminated != nil) {
				goto streamLogs
			}
		}

		// 检查 Init Container 状态
		for _, cs := range pod.Status.InitContainerStatuses {
			if cs.State.Running != nil || cs.State.Terminated != nil {
				// 先流式输出 Init Container 日志
				if err := s.streamContainerLogs(ctx, clusterID, namespace, podName, cs.Name, callback); err != nil {
					log.WithError(err).Warn("获取 Init Container 日志失败")
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			continue
		}
	}

streamLogs:
	// 流式获取主容器日志
	return s.streamContainerLogs(ctx, clusterID, namespace, podName, container, callback)
}

// streamContainerLogs 流式获取容器日志
func (s *LogService) streamContainerLogs(ctx context.Context, clusterID uint, namespace, podName, container string, callback LogStreamCallback) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	req := client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: container,
		Follow:    true,
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	reader := bufio.NewReader(stream)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			// 脱敏处理
			line = s.SanitizeLogs(line)

			if err := callback(line); err != nil {
				return err
			}
		}
	}
}

// StreamPodLogs 流式获取 Pod 日志（简化版）
func (s *LogService) StreamPodLogs(ctx context.Context, clusterID uint, namespace, podName, container string, callback LogStreamCallback) error {
	client, err := s.clientMgr.GetClient(ctx, clusterID)
	if err != nil {
		return err
	}

	req := client.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: container,
		Follow:    true,
		TailLines: ptr.To(int64(100)),
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	reader := bufio.NewReader(stream)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			// 脱敏处理
			line = s.SanitizeLogs(line)

			if err := callback(line); err != nil {
				return err
			}
		}
	}
}

// SanitizeLogs 日志脱敏
func (s *LogService) SanitizeLogs(logs string) string {
	result := logs

	// 应用自定义规则
	for _, rule := range s.customRules {
		result = rule.Pattern.ReplaceAllString(result, rule.Replacement)
	}

	// 脱敏敏感键值对
	for _, key := range s.sensitiveKeys {
		// 匹配 key=value 或 key: value 格式（带引号或不带引号）
		patterns := []string{
			fmt.Sprintf(`(?i)(%s)\s*[=:]\s*["']([^"']+)["']`, key),
			fmt.Sprintf(`(?i)(%s)\s*[=:]\s*([^"'\s,;]+)`, key),
		}

		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			result = re.ReplaceAllString(result, "$1=******")
		}
	}

	// 脱敏 Bearer Token
	bearerRe := regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9\-_\.]+`)
	result = bearerRe.ReplaceAllString(result, "$1******")

	// 脱敏 Base64 编码的凭证（常见于 Authorization header）
	base64Re := regexp.MustCompile(`(?i)(Basic\s+)[A-Za-z0-9+/=]+`)
	result = base64Re.ReplaceAllString(result, "$1******")

	// 脱敏环境变量导出语句中的敏感值
	exportRe := regexp.MustCompile(`(?i)(export\s+\w*(password|secret|token|key|credential)\w*\s*=\s*)["']?([^"'\s]+)["']?`)
	result = exportRe.ReplaceAllString(result, "$1******")

	return result
}

// SanitizeLogsWithCredentials 使用指定凭证列表进行脱敏
func (s *LogService) SanitizeLogsWithCredentials(logs string, credentials []string) string {
	result := s.SanitizeLogs(logs)

	// 替换已知凭证值
	for _, cred := range credentials {
		if len(cred) >= 4 { // 只替换长度>=4的凭证，避免误替换
			result = strings.ReplaceAll(result, cred, "******")
		}
	}

	return result
}

// GetStepRunLogs 获取步骤执行日志
func (s *LogService) GetStepRunLogs(ctx context.Context, stepRunID uint) (string, error) {
	var stepRun models.StepRun
	if err := s.db.First(&stepRun, stepRunID).Error; err != nil {
		return "", err
	}

	// 如果有关联的构建任务，获取实时日志
	if stepRun.BuildJobID != nil {
		logs, err := s.GetBuildJobLogs(ctx, *stepRun.BuildJobID)
		if err == nil {
			return logs, nil
		}
	}

	// 返回存储的日志
	return s.SanitizeLogs(stepRun.Logs), nil
}

// SaveStepRunLogs 保存步骤执行日志
func (s *LogService) SaveStepRunLogs(ctx context.Context, stepRunID uint, logs string) error {
	return s.db.Model(&models.StepRun{}).Where("id = ?", stepRunID).Update("logs", logs).Error
}

// AppendStepRunLogs 追加步骤执行日志
func (s *LogService) AppendStepRunLogs(ctx context.Context, stepRunID uint, logs string) error {
	var stepRun models.StepRun
	if err := s.db.First(&stepRun, stepRunID).Error; err != nil {
		return err
	}

	stepRun.Logs = stepRun.Logs + logs
	return s.db.Save(&stepRun).Error
}

// SearchLogs 搜索日志
func (s *LogService) SearchLogs(ctx context.Context, logs, keyword string) []string {
	lines := strings.Split(logs, "\n")
	var results []string

	keyword = strings.ToLower(keyword)
	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), keyword) {
			results = append(results, line)
		}
	}

	return results
}

// HighlightErrors 高亮错误信息
func (s *LogService) HighlightErrors(logs string) string {
	errorPatterns := []string{
		`(?i)\berror\b`,
		`(?i)\bfailed\b`,
		`(?i)\bfailure\b`,
		`(?i)\bexception\b`,
		`(?i)\bpanic\b`,
		`(?i)\bfatal\b`,
	}

	result := logs
	for _, pattern := range errorPatterns {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllStringFunc(result, func(match string) string {
			return fmt.Sprintf("[ERROR]%s[/ERROR]", match)
		})
	}

	return result
}
