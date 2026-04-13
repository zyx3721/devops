package pipeline

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"devops/internal/models"
	"devops/pkg/dto"
	"devops/pkg/logger"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// WebhookService Webhook 服务
type WebhookService struct {
	db         *gorm.DB
	runService *RunService
}

// NewWebhookService 创建 Webhook 服务
func NewWebhookService(db *gorm.DB, runService *RunService) *WebhookService {
	return &WebhookService{
		db:         db,
		runService: runService,
	}
}

// WebhookPayload Webhook 载荷
type WebhookPayload struct {
	Provider    string            `json:"provider"`     // github, gitlab, gitee
	Event       string            `json:"event"`        // push, pull_request, tag
	Ref         string            `json:"ref"`          // refs/heads/main
	Branch      string            `json:"branch"`       // main
	Tag         string            `json:"tag"`          // v1.0.0
	CommitSHA   string            `json:"commit_sha"`   // abc123
	CommitMsg   string            `json:"commit_msg"`   // commit message
	Author      string            `json:"author"`       // author name
	AuthorEmail string            `json:"author_email"` // author email
	RepoURL     string            `json:"repo_url"`     // repository URL
	RepoName    string            `json:"repo_name"`    // repository name
	CloneURL    string            `json:"clone_url"`    // clone URL
	PRNumber    int               `json:"pr_number"`    // PR number
	PRTitle     string            `json:"pr_title"`     // PR title
	PRAction    string            `json:"pr_action"`    // opened, closed, merged
	RawPayload  map[string]any    `json:"raw_payload"`  // 原始载荷
	Headers     map[string]string `json:"headers"`      // 请求头
}

// HandleGitHubWebhook 处理 GitHub Webhook
func (s *WebhookService) HandleGitHubWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookPayload, error) {
	log := logger.L().WithField("provider", "github")
	log.Info("处理 GitHub Webhook")

	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("解析 payload 失败: %w", err)
	}

	event := headers["X-GitHub-Event"]
	wp := &WebhookPayload{
		Provider:   "github",
		Event:      event,
		RawPayload: raw,
		Headers:    headers,
	}

	switch event {
	case "push":
		s.parseGitHubPush(raw, wp)
	case "pull_request":
		s.parseGitHubPR(raw, wp)
	case "create":
		if raw["ref_type"] == "tag" {
			wp.Event = "tag"
			wp.Tag = getString(raw, "ref")
		}
	default:
		log.WithField("event", event).Warn("未知的 GitHub 事件类型")
	}

	return wp, nil
}

func (s *WebhookService) parseGitHubPush(raw map[string]any, wp *WebhookPayload) {
	wp.Ref = getString(raw, "ref")
	wp.Branch = strings.TrimPrefix(wp.Ref, "refs/heads/")

	if strings.HasPrefix(wp.Ref, "refs/tags/") {
		wp.Event = "tag"
		wp.Tag = strings.TrimPrefix(wp.Ref, "refs/tags/")
	}

	if repo, ok := raw["repository"].(map[string]any); ok {
		wp.RepoURL = getString(repo, "html_url")
		wp.RepoName = getString(repo, "full_name")
		wp.CloneURL = getString(repo, "clone_url")
	}

	if headCommit, ok := raw["head_commit"].(map[string]any); ok {
		wp.CommitSHA = getString(headCommit, "id")
		wp.CommitMsg = getString(headCommit, "message")
		if author, ok := headCommit["author"].(map[string]any); ok {
			wp.Author = getString(author, "name")
			wp.AuthorEmail = getString(author, "email")
		}
	}
}

func (s *WebhookService) parseGitHubPR(raw map[string]any, wp *WebhookPayload) {
	wp.PRAction = getString(raw, "action")

	if pr, ok := raw["pull_request"].(map[string]any); ok {
		wp.PRNumber = getInt(pr, "number")
		wp.PRTitle = getString(pr, "title")

		if head, ok := pr["head"].(map[string]any); ok {
			wp.Branch = getString(head, "ref")
			wp.CommitSHA = getString(head, "sha")
		}
	}
}

// HandleGitLabWebhook 处理 GitLab Webhook
func (s *WebhookService) HandleGitLabWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookPayload, error) {
	log := logger.L().WithField("provider", "gitlab")
	log.Info("处理 GitLab Webhook")

	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("解析 payload 失败: %w", err)
	}

	event := getString(raw, "object_kind")
	wp := &WebhookPayload{
		Provider:   "gitlab",
		Event:      event,
		RawPayload: raw,
		Headers:    headers,
	}

	switch event {
	case "push":
		s.parseGitLabPush(raw, wp)
	case "merge_request":
		wp.Event = "pull_request"
		s.parseGitLabMR(raw, wp)
	case "tag_push":
		wp.Event = "tag"
		s.parseGitLabTag(raw, wp)
	}

	return wp, nil
}

func (s *WebhookService) parseGitLabPush(raw map[string]any, wp *WebhookPayload) {
	wp.Ref = getString(raw, "ref")
	wp.Branch = strings.TrimPrefix(wp.Ref, "refs/heads/")
	wp.CommitSHA = getString(raw, "after")

	if project, ok := raw["project"].(map[string]any); ok {
		wp.RepoURL = getString(project, "web_url")
		wp.RepoName = getString(project, "path_with_namespace")
		wp.CloneURL = getString(project, "git_http_url")
	}

	if commits, ok := raw["commits"].([]any); ok && len(commits) > 0 {
		if lastCommit, ok := commits[len(commits)-1].(map[string]any); ok {
			wp.CommitMsg = getString(lastCommit, "message")
			if author, ok := lastCommit["author"].(map[string]any); ok {
				wp.Author = getString(author, "name")
				wp.AuthorEmail = getString(author, "email")
			}
		}
	}
}

func (s *WebhookService) parseGitLabMR(raw map[string]any, wp *WebhookPayload) {
	if attrs, ok := raw["object_attributes"].(map[string]any); ok {
		wp.PRNumber = getInt(attrs, "iid")
		wp.PRTitle = getString(attrs, "title")
		wp.PRAction = getString(attrs, "action")
		wp.Branch = getString(attrs, "source_branch")
		wp.CommitSHA = getString(attrs, "last_commit")
	}
}

func (s *WebhookService) parseGitLabTag(raw map[string]any, wp *WebhookPayload) {
	wp.Ref = getString(raw, "ref")
	wp.Tag = strings.TrimPrefix(wp.Ref, "refs/tags/")
	wp.CommitSHA = getString(raw, "after")
}

// HandleGiteeWebhook 处理 Gitee Webhook
func (s *WebhookService) HandleGiteeWebhook(ctx context.Context, payload []byte, headers map[string]string) (*WebhookPayload, error) {
	log := logger.L().WithField("provider", "gitee")
	log.Info("处理 Gitee Webhook")

	var raw map[string]any
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, fmt.Errorf("解析 payload 失败: %w", err)
	}

	event := headers["X-Gitee-Event"]
	wp := &WebhookPayload{
		Provider:   "gitee",
		Event:      strings.ToLower(strings.TrimSuffix(event, " Hook")),
		RawPayload: raw,
		Headers:    headers,
	}

	switch wp.Event {
	case "push":
		s.parseGiteePush(raw, wp)
	case "pull request", "merge request":
		wp.Event = "pull_request"
		s.parseGiteePR(raw, wp)
	case "tag push":
		wp.Event = "tag"
		s.parseGiteeTag(raw, wp)
	}

	return wp, nil
}

func (s *WebhookService) parseGiteePush(raw map[string]any, wp *WebhookPayload) {
	wp.Ref = getString(raw, "ref")
	wp.Branch = strings.TrimPrefix(wp.Ref, "refs/heads/")
	wp.CommitSHA = getString(raw, "after")

	if repo, ok := raw["repository"].(map[string]any); ok {
		wp.RepoURL = getString(repo, "html_url")
		wp.RepoName = getString(repo, "full_name")
		wp.CloneURL = getString(repo, "clone_url")
	}

	if headCommit, ok := raw["head_commit"].(map[string]any); ok {
		wp.CommitMsg = getString(headCommit, "message")
		if author, ok := headCommit["author"].(map[string]any); ok {
			wp.Author = getString(author, "name")
			wp.AuthorEmail = getString(author, "email")
		}
	}
}

func (s *WebhookService) parseGiteePR(raw map[string]any, wp *WebhookPayload) {
	wp.PRAction = getString(raw, "action")

	if pr, ok := raw["pull_request"].(map[string]any); ok {
		wp.PRNumber = getInt(pr, "number")
		wp.PRTitle = getString(pr, "title")

		if head, ok := pr["head"].(map[string]any); ok {
			wp.Branch = getString(head, "ref")
			wp.CommitSHA = getString(head, "sha")
		}
	}
}

func (s *WebhookService) parseGiteeTag(raw map[string]any, wp *WebhookPayload) {
	wp.Ref = getString(raw, "ref")
	wp.Tag = strings.TrimPrefix(wp.Ref, "refs/tags/")
	wp.CommitSHA = getString(raw, "after")
}

// VerifyGitHubSignature 验证 GitHub 签名
func (s *WebhookService) VerifyGitHubSignature(payload []byte, signature, secret string) bool {
	if secret == "" {
		return true // 未配置密钥则跳过验证
	}

	signature = strings.TrimPrefix(signature, "sha256=")
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

// VerifyGitLabToken 验证 GitLab Token
func (s *WebhookService) VerifyGitLabToken(token, secret string) bool {
	if secret == "" {
		return true
	}
	return token == secret
}

// VerifyGiteeSignature 验证 Gitee 签名
func (s *WebhookService) VerifyGiteeSignature(payload []byte, signature, secret string) bool {
	if secret == "" {
		return true
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expected))
}

// MatchBranchFilter 匹配分支过滤器
func (s *WebhookService) MatchBranchFilter(branch string, filters []string) bool {
	if len(filters) == 0 {
		return true // 无过滤器则匹配所有
	}

	for _, filter := range filters {
		// 支持通配符
		if strings.Contains(filter, "*") {
			pattern := strings.ReplaceAll(filter, "*", ".*")
			if matched, _ := regexp.MatchString("^"+pattern+"$", branch); matched {
				return true
			}
		} else if filter == branch {
			return true
		}
	}

	return false
}

// TriggerPipeline 触发流水线
func (s *WebhookService) TriggerPipeline(ctx context.Context, repoID uint, wp *WebhookPayload) (*models.PipelineRun, error) {
	log := logger.L().WithField("repo_id", repoID).WithField("event", wp.Event)
	log.Info("Webhook 触发流水线")

	// 查找关联的流水线
	var pipelines []models.Pipeline
	if err := s.db.WithContext(ctx).
		Where("git_repo_id = ? AND status = ?", repoID, "active").
		Find(&pipelines).Error; err != nil {
		return nil, fmt.Errorf("查询流水线失败: %w", err)
	}

	if len(pipelines) == 0 {
		log.Warn("未找到关联的流水线")
		return nil, nil
	}

	// 遍历流水线，检查触发条件
	for _, pipeline := range pipelines {
		if s.shouldTrigger(&pipeline, wp) {
			// 构建触发参数
			params := map[string]string{
				"CI_COMMIT_SHA":     wp.CommitSHA,
				"CI_COMMIT_MESSAGE": wp.CommitMsg,
				"CI_COMMIT_BRANCH":  wp.Branch,
				"CI_COMMIT_TAG":     wp.Tag,
				"CI_COMMIT_AUTHOR":  wp.Author,
				"CI_REPOSITORY_URL": wp.RepoURL,
				"CI_TRIGGER_EVENT":  wp.Event,
			}

			// 触发流水线
			req := &dto.RunPipelineRequest{Parameters: params}
			runItem, err := s.runService.Run(ctx, pipeline.ID, req, "webhook", wp.Author)
			if err != nil {
				log.WithField("error", err).Error("触发流水线失败")
				continue
			}

			// 转换为 models.PipelineRun
			run := &models.PipelineRun{
				ID:           runItem.ID,
				PipelineID:   runItem.PipelineID,
				PipelineName: runItem.PipelineName,
				Status:       runItem.Status,
				TriggerType:  runItem.TriggerType,
				TriggerBy:    runItem.TriggerBy,
			}

			log.WithField("run_id", run.ID).Info("流水线已触发")
			return run, nil
		}
	}

	log.Info("无匹配的触发条件")
	return nil, nil
}

// shouldTrigger 检查是否应该触发
func (s *WebhookService) shouldTrigger(pipeline *models.Pipeline, wp *WebhookPayload) bool {
	// 解析触发配置
	var triggerConfig TriggerConfig
	if pipeline.TriggerConfig != "" {
		if err := json.Unmarshal([]byte(pipeline.TriggerConfig), &triggerConfig); err != nil {
			return false
		}
	}

	// 检查事件类型
	switch wp.Event {
	case "push":
		if !triggerConfig.OnPush {
			return false
		}
		return s.MatchBranchFilter(wp.Branch, triggerConfig.Branches)

	case "pull_request":
		if !triggerConfig.OnPR {
			return false
		}
		// 检查 PR 动作
		if len(triggerConfig.PRActions) > 0 {
			matched := false
			for _, action := range triggerConfig.PRActions {
				if action == wp.PRAction {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
		return true

	case "tag":
		if !triggerConfig.OnTag {
			return false
		}
		return s.MatchBranchFilter(wp.Tag, triggerConfig.Tags)
	}

	return false
}

// TriggerConfig 触发配置
type TriggerConfig struct {
	OnPush    bool     `json:"on_push"`
	OnPR      bool     `json:"on_pr"`
	OnTag     bool     `json:"on_tag"`
	Branches  []string `json:"branches"`
	Tags      []string `json:"tags"`
	PRActions []string `json:"pr_actions"`
}

// SaveWebhookLog 保存 Webhook 日志
func (s *WebhookService) SaveWebhookLog(ctx context.Context, repoID uint, wp *WebhookPayload, runID uint, err error) error {
	log := &models.WebhookLog{
		GitRepoID:   repoID,
		Provider:    wp.Provider,
		Event:       wp.Event,
		Ref:         wp.Ref,
		CommitSHA:   wp.CommitSHA,
		PipelineRun: runID,
		ReceivedAt:  time.Now(),
	}

	if err != nil {
		log.Status = "failed"
		log.ErrorMsg = err.Error()
	} else if runID > 0 {
		log.Status = "triggered"
	} else {
		log.Status = "skipped"
	}

	payloadBytes, _ := json.Marshal(wp.RawPayload)
	log.Payload = string(payloadBytes)

	return s.db.WithContext(ctx).Create(log).Error
}

// 辅助函数
func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}
