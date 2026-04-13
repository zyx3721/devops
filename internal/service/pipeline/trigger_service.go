package pipeline

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	"devops/pkg/logger"
)

// TriggerService 触发器服务
type TriggerService struct {
	db         *gorm.DB
	runService *RunService
	cron       *cron.Cron
	cronJobs   map[uint]cron.EntryID // pipelineID -> cronEntryID
	mu         sync.RWMutex
}

// NewTriggerService 创建触发器服务
func NewTriggerService(db *gorm.DB, runService *RunService) *TriggerService {
	s := &TriggerService{
		db:         db,
		runService: runService,
		cron:       cron.New(cron.WithSeconds()),
		cronJobs:   make(map[uint]cron.EntryID),
	}
	s.cron.Start()
	return s
}

// Start 启动触发器服务，加载所有定时任务
func (s *TriggerService) Start(ctx context.Context) error {
	log := logger.L().WithField("service", "trigger")
	log.Info("启动触发器服务")

	// 加载所有启用定时触发的流水线
	var pipelines []models.Pipeline
	if err := s.db.Where("status = ?", "active").Find(&pipelines).Error; err != nil {
		return fmt.Errorf("加载流水线失败: %w", err)
	}

	for _, p := range pipelines {
		if p.TriggerConfigJSON == "" {
			continue
		}

		var triggerConfig dto.TriggerConfig
		if err := json.Unmarshal([]byte(p.TriggerConfigJSON), &triggerConfig); err != nil {
			log.WithField("pipeline_id", p.ID).WithError(err).Warn("解析触发器配置失败")
			continue
		}

		if triggerConfig.Scheduled != nil && triggerConfig.Scheduled.Enabled {
			if err := s.AddScheduledTrigger(p.ID, triggerConfig.Scheduled.Cron); err != nil {
				log.WithField("pipeline_id", p.ID).WithError(err).Warn("添加定时触发失败")
			}
		}
	}

	log.WithField("count", len(s.cronJobs)).Info("定时触发器加载完成")
	return nil
}

// Stop 停止触发器服务
func (s *TriggerService) Stop() {
	s.cron.Stop()
}

// AddScheduledTrigger 添加定时触发
func (s *TriggerService) AddScheduledTrigger(pipelineID uint, cronExpr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log := logger.L().WithField("pipeline_id", pipelineID).WithField("cron", cronExpr)

	// 移除旧的定时任务
	if entryID, exists := s.cronJobs[pipelineID]; exists {
		s.cron.Remove(entryID)
		delete(s.cronJobs, pipelineID)
	}

	// 添加新的定时任务
	entryID, err := s.cron.AddFunc(cronExpr, func() {
		s.triggerPipeline(pipelineID, "scheduled", "system")
	})
	if err != nil {
		log.WithError(err).Error("添加定时任务失败")
		return fmt.Errorf("无效的 Cron 表达式: %w", err)
	}

	s.cronJobs[pipelineID] = entryID
	log.Info("定时触发器已添加")
	return nil
}

// RemoveScheduledTrigger 移除定时触发
func (s *TriggerService) RemoveScheduledTrigger(pipelineID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.cronJobs[pipelineID]; exists {
		s.cron.Remove(entryID)
		delete(s.cronJobs, pipelineID)
		logger.L().WithField("pipeline_id", pipelineID).Info("定时触发器已移除")
	}
}

// triggerPipeline 触发流水线执行
func (s *TriggerService) triggerPipeline(pipelineID uint, triggerType, triggerBy string) {
	log := logger.L().WithField("pipeline_id", pipelineID).WithField("trigger_type", triggerType)
	log.Info("触发流水线执行")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req := &dto.RunPipelineRequest{}
	_, err := s.runService.Run(ctx, pipelineID, req, triggerType, triggerBy)
	if err != nil {
		log.WithError(err).Error("触发流水线失败")
	}
}

// UpdateTriggerConfig 更新触发器配置
func (s *TriggerService) UpdateTriggerConfig(ctx context.Context, pipelineID uint, config *dto.TriggerConfig) error {
	log := logger.L().WithField("pipeline_id", pipelineID)

	// 更新定时触发
	if config.Scheduled != nil && config.Scheduled.Enabled && config.Scheduled.Cron != "" {
		if err := s.AddScheduledTrigger(pipelineID, config.Scheduled.Cron); err != nil {
			return err
		}
	} else {
		s.RemoveScheduledTrigger(pipelineID)
	}

	// 生成 Webhook URL 和 Secret
	if config.Webhook != nil && config.Webhook.Enabled {
		if config.Webhook.Secret == "" {
			config.Webhook.Secret = generateSecret(32)
		}
		config.Webhook.URL = fmt.Sprintf("/app/api/v1/pipelines/%d/webhook", pipelineID)
	}

	// 保存配置到数据库
	configJSON, err := json.Marshal(config)
	if err != nil {
		return err
	}

	if err := s.db.Model(&models.Pipeline{}).Where("id = ?", pipelineID).
		Update("trigger_config_json", string(configJSON)).Error; err != nil {
		log.WithError(err).Error("保存触发器配置失败")
		return err
	}

	log.Info("触发器配置已更新")
	return nil
}

// GetTriggerConfig 获取触发器配置
func (s *TriggerService) GetTriggerConfig(ctx context.Context, pipelineID uint) (*dto.TriggerConfig, error) {
	var pipeline models.Pipeline
	if err := s.db.First(&pipeline, pipelineID).Error; err != nil {
		return nil, err
	}

	config := &dto.TriggerConfig{Manual: true}
	if pipeline.TriggerConfigJSON != "" {
		if err := json.Unmarshal([]byte(pipeline.TriggerConfigJSON), config); err != nil {
			return nil, err
		}
	}

	return config, nil
}

// GetScheduledTriggers 获取所有定时触发器状态
func (s *TriggerService) GetScheduledTriggers() []map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]map[string]interface{}, 0, len(s.cronJobs))
	for pipelineID, entryID := range s.cronJobs {
		entry := s.cron.Entry(entryID)
		result = append(result, map[string]interface{}{
			"pipeline_id": pipelineID,
			"next_run":    entry.Next,
			"prev_run":    entry.Prev,
		})
	}
	return result
}

// ManualTrigger 手动触发流水线
func (s *TriggerService) ManualTrigger(ctx context.Context, pipelineID uint, params map[string]string, username string) (*dto.PipelineRunItem, error) {
	req := &dto.RunPipelineRequest{Parameters: params}
	return s.runService.Run(ctx, pipelineID, req, "manual", username)
}

// generateSecret 生成随机密钥
func generateSecret(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
