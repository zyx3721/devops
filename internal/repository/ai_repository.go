package repository

import (
	"context"
	"time"

	"gorm.io/gorm"

	"devops/internal/models/ai"
)

// ==================== AI 会话仓库 ====================

// AIConversationRepository AI会话仓库
type AIConversationRepository struct {
	db *gorm.DB
}

// NewAIConversationRepository 创建AI会话仓库
func NewAIConversationRepository(db *gorm.DB) *AIConversationRepository {
	return &AIConversationRepository{db: db}
}

// Create 创建会话
func (r *AIConversationRepository) Create(ctx context.Context, conv *ai.AIConversation) error {
	return r.db.WithContext(ctx).Create(conv).Error
}

// Update 更新会话
func (r *AIConversationRepository) Update(ctx context.Context, conv *ai.AIConversation) error {
	return r.db.WithContext(ctx).Save(conv).Error
}

// Delete 删除会话（软删除）
func (r *AIConversationRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&ai.AIConversation{}, "id = ?", id).Error
}

// GetByID 根据ID获取会话
func (r *AIConversationRepository) GetByID(ctx context.Context, id string) (*ai.AIConversation, error) {
	var conv ai.AIConversation
	if err := r.db.WithContext(ctx).First(&conv, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &conv, nil
}

// GetByIDWithMessages 根据ID获取会话及其消息
func (r *AIConversationRepository) GetByIDWithMessages(ctx context.Context, id string, messageLimit int) (*ai.AIConversation, error) {
	var conv ai.AIConversation
	if err := r.db.WithContext(ctx).First(&conv, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// 获取最近的消息
	var messages []ai.AIMessage
	query := r.db.WithContext(ctx).Where("conversation_id = ?", id).Order("created_at DESC")
	if messageLimit > 0 {
		query = query.Limit(messageLimit)
	}
	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}

	// 反转消息顺序（从旧到新）
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	conv.Messages = messages

	return &conv, nil
}

// GetByUserID 获取用户的会话列表
func (r *AIConversationRepository) GetByUserID(ctx context.Context, userID uint, page, pageSize int) ([]ai.AIConversation, int64, error) {
	var convs []ai.AIConversation
	var total int64

	query := r.db.WithContext(ctx).Model(&ai.AIConversation{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("last_message_at DESC, created_at DESC").Offset(offset).Limit(pageSize).Find(&convs).Error; err != nil {
		return nil, 0, err
	}

	return convs, total, nil
}

// UpdateMessageCount 更新消息数量
func (r *AIConversationRepository) UpdateMessageCount(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&ai.AIConversation{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"message_count":   gorm.Expr("message_count + 1"),
			"last_message_at": now,
		}).Error
}

// UpdateContext 更新会话上下文
func (r *AIConversationRepository) UpdateContext(ctx context.Context, id string, pageCtx *ai.PageContext) error {
	return r.db.WithContext(ctx).Model(&ai.AIConversation{}).
		Where("id = ?", id).
		Update("context", pageCtx).Error
}

// UpdateTitle 更新会话标题
func (r *AIConversationRepository) UpdateTitle(ctx context.Context, id string, title string) error {
	return r.db.WithContext(ctx).Model(&ai.AIConversation{}).
		Where("id = ?", id).
		Update("title", title).Error
}

// ==================== AI 消息仓库 ====================

// AIMessageRepository AI消息仓库
type AIMessageRepository struct {
	db *gorm.DB
}

// NewAIMessageRepository 创建AI消息仓库
func NewAIMessageRepository(db *gorm.DB) *AIMessageRepository {
	return &AIMessageRepository{db: db}
}

// Create 创建消息
func (r *AIMessageRepository) Create(ctx context.Context, msg *ai.AIMessage) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

// Update 更新消息
func (r *AIMessageRepository) Update(ctx context.Context, msg *ai.AIMessage) error {
	return r.db.WithContext(ctx).Save(msg).Error
}

// GetByID 根据ID获取消息
func (r *AIMessageRepository) GetByID(ctx context.Context, id string) (*ai.AIMessage, error) {
	var msg ai.AIMessage
	if err := r.db.WithContext(ctx).First(&msg, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &msg, nil
}

// GetByConversationID 获取会话的消息列表
func (r *AIMessageRepository) GetByConversationID(ctx context.Context, conversationID string, limit int) ([]ai.AIMessage, error) {
	var messages []ai.AIMessage
	query := r.db.WithContext(ctx).Where("conversation_id = ?", conversationID).Order("created_at ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// GetRecentMessages 获取最近的消息（用于上下文压缩）
func (r *AIMessageRepository) GetRecentMessages(ctx context.Context, conversationID string, limit int) ([]ai.AIMessage, error) {
	var messages []ai.AIMessage
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}

	// 反转顺序
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

// CountByConversationID 统计会话消息数量
func (r *AIMessageRepository) CountByConversationID(ctx context.Context, conversationID string) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&ai.AIMessage{}).
		Where("conversation_id = ?", conversationID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// UpdateStatus 更新消息状态
func (r *AIMessageRepository) UpdateStatus(ctx context.Context, id string, status ai.MessageStatus, errorMsg string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}
	return r.db.WithContext(ctx).Model(&ai.AIMessage{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateContent 更新消息内容（用于流式响应）
func (r *AIMessageRepository) UpdateContent(ctx context.Context, id string, content string, tokenCount int) error {
	return r.db.WithContext(ctx).Model(&ai.AIMessage{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"content":     content,
			"token_count": tokenCount,
			"status":      ai.StatusComplete,
		}).Error
}

// DeleteOldMessages 删除旧消息（用于会话压缩）
func (r *AIMessageRepository) DeleteOldMessages(ctx context.Context, conversationID string, keepCount int) error {
	// 获取要保留的消息ID
	var keepIDs []string
	if err := r.db.WithContext(ctx).Model(&ai.AIMessage{}).
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Limit(keepCount).
		Pluck("id", &keepIDs).Error; err != nil {
		return err
	}

	if len(keepIDs) == 0 {
		return nil
	}

	// 删除其他消息
	return r.db.WithContext(ctx).
		Where("conversation_id = ? AND id NOT IN ?", conversationID, keepIDs).
		Delete(&ai.AIMessage{}).Error
}

// UpdateFeedback 更新消息反馈
func (r *AIMessageRepository) UpdateFeedback(ctx context.Context, id string, rating string, comment string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&ai.AIMessage{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"feedback_rating":  rating,
			"feedback_comment": comment,
			"feedback_at":      &now,
		}).Error
}

// ==================== AI 知识库仓库 ====================

// AIKnowledgeRepository AI知识库仓库
type AIKnowledgeRepository struct {
	db *gorm.DB
}

// NewAIKnowledgeRepository 创建AI知识库仓库
func NewAIKnowledgeRepository(db *gorm.DB) *AIKnowledgeRepository {
	return &AIKnowledgeRepository{db: db}
}

// Create 创建知识条目
func (r *AIKnowledgeRepository) Create(ctx context.Context, knowledge *ai.AIKnowledge) error {
	return r.db.WithContext(ctx).Create(knowledge).Error
}

// Update 更新知识条目
func (r *AIKnowledgeRepository) Update(ctx context.Context, knowledge *ai.AIKnowledge) error {
	return r.db.WithContext(ctx).Save(knowledge).Error
}

// Delete 删除知识条目（软删除）
func (r *AIKnowledgeRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&ai.AIKnowledge{}, id).Error
}

// GetByID 根据ID获取知识条目
func (r *AIKnowledgeRepository) GetByID(ctx context.Context, id uint) (*ai.AIKnowledge, error) {
	var knowledge ai.AIKnowledge
	if err := r.db.WithContext(ctx).First(&knowledge, id).Error; err != nil {
		return nil, err
	}
	return &knowledge, nil
}

// List 获取知识列表
func (r *AIKnowledgeRepository) List(ctx context.Context, filter AIKnowledgeFilter, page, pageSize int) ([]ai.AIKnowledge, int64, error) {
	var items []ai.AIKnowledge
	var total int64

	query := r.db.WithContext(ctx).Model(&ai.AIKnowledge{})

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Keyword != "" {
		query = query.Where("title LIKE ? OR content LIKE ?", "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}
	if filter.IsActive != nil {
		query = query.Where("is_active = ?", *filter.IsActive)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}

// Search 搜索知识（全文搜索）
func (r *AIKnowledgeRepository) Search(ctx context.Context, query string, limit int) ([]ai.KnowledgeItem, error) {
	var items []ai.AIKnowledge

	// 使用 MATCH AGAINST 进行全文搜索（如果支持）
	// 否则使用 LIKE 搜索
	searchQuery := r.db.WithContext(ctx).Model(&ai.AIKnowledge{}).
		Where("is_active = ?", true).
		Where("title LIKE ? OR content LIKE ?", "%"+query+"%", "%"+query+"%")

	if limit > 0 {
		searchQuery = searchQuery.Limit(limit)
	}

	if err := searchQuery.Find(&items).Error; err != nil {
		return nil, err
	}

	// 转换为 KnowledgeItem
	result := make([]ai.KnowledgeItem, len(items))
	for i, item := range items {
		result[i] = ai.KnowledgeItem{
			ID:       item.ID,
			Title:    item.Title,
			Content:  item.Content,
			Category: item.Category,
			Score:    1.0, // 简单实现，后续可以添加相关性评分
		}
	}

	return result, nil
}

// GetByCategory 按分类获取知识
func (r *AIKnowledgeRepository) GetByCategory(ctx context.Context, category ai.KnowledgeCategory) ([]ai.AIKnowledge, error) {
	var items []ai.AIKnowledge
	if err := r.db.WithContext(ctx).
		Where("category = ? AND is_active = ?", category, true).
		Order("created_at DESC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// GetAllCategories 获取所有分类
func (r *AIKnowledgeRepository) GetAllCategories(ctx context.Context) ([]string, error) {
	var categories []string
	if err := r.db.WithContext(ctx).Model(&ai.AIKnowledge{}).
		Distinct("category").
		Where("is_active = ?", true).
		Pluck("category", &categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// IncrementViewCount 增加查看次数
func (r *AIKnowledgeRepository) IncrementViewCount(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Model(&ai.AIKnowledge{}).
		Where("id = ?", id).
		Update("view_count", gorm.Expr("view_count + 1")).Error
}

// AIKnowledgeFilter 知识过滤条件
type AIKnowledgeFilter struct {
	Category string
	Keyword  string
	IsActive *bool
}

// ==================== AI 操作日志仓库 ====================

// AIOperationLogRepository AI操作日志仓库
type AIOperationLogRepository struct {
	db *gorm.DB
}

// NewAIOperationLogRepository 创建AI操作日志仓库
func NewAIOperationLogRepository(db *gorm.DB) *AIOperationLogRepository {
	return &AIOperationLogRepository{db: db}
}

// Create 创建操作日志
func (r *AIOperationLogRepository) Create(ctx context.Context, log *ai.AIOperationLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

// GetByID 根据ID获取操作日志
func (r *AIOperationLogRepository) GetByID(ctx context.Context, id uint) (*ai.AIOperationLog, error) {
	var log ai.AIOperationLog
	if err := r.db.WithContext(ctx).First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// List 获取操作日志列表
func (r *AIOperationLogRepository) List(ctx context.Context, filter AIOperationLogFilter, page, pageSize int) ([]ai.AIOperationLog, int64, error) {
	var logs []ai.AIOperationLog
	var total int64

	query := r.db.WithContext(ctx).Model(&ai.AIOperationLog{})

	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.TargetType != "" {
		query = query.Where("target_type = ?", filter.TargetType)
	}
	if filter.Success != nil {
		query = query.Where("success = ?", *filter.Success)
	}
	if !filter.StartTime.IsZero() {
		query = query.Where("created_at >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query = query.Where("created_at <= ?", filter.EndTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetByConversationID 获取会话的操作日志
func (r *AIOperationLogRepository) GetByConversationID(ctx context.Context, conversationID string) ([]ai.AIOperationLog, error) {
	var logs []ai.AIOperationLog
	if err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// GetStats 获取操作统计
func (r *AIOperationLogRepository) GetStats(ctx context.Context, filter AIOperationLogFilter) (*AIOperationStats, error) {
	var stats AIOperationStats

	query := r.db.WithContext(ctx).Model(&ai.AIOperationLog{})

	if filter.UserID > 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if !filter.StartTime.IsZero() {
		query = query.Where("created_at >= ?", filter.StartTime)
	}
	if !filter.EndTime.IsZero() {
		query = query.Where("created_at <= ?", filter.EndTime)
	}

	// 总数
	if err := query.Count(&stats.Total).Error; err != nil {
		return nil, err
	}

	// 成功数
	var successCount int64
	r.db.WithContext(ctx).Model(&ai.AIOperationLog{}).Where("success = ?", true).Count(&successCount)
	stats.Success = successCount

	// 失败数
	stats.Failed = stats.Total - stats.Success

	// 成功率
	if stats.Total > 0 {
		stats.SuccessRate = float64(stats.Success) / float64(stats.Total) * 100
	}

	return &stats, nil
}

// AIOperationLogFilter 操作日志过滤条件
type AIOperationLogFilter struct {
	UserID     uint
	Action     string
	TargetType string
	Success    *bool
	StartTime  time.Time
	EndTime    time.Time
}

// AIOperationStats 操作统计
type AIOperationStats struct {
	Total       int64   `json:"total"`
	Success     int64   `json:"success"`
	Failed      int64   `json:"failed"`
	SuccessRate float64 `json:"success_rate"`
}

// ==================== AI LLM 配置仓库 ====================

// AILLMConfigRepository AI LLM配置仓库
type AILLMConfigRepository struct {
	db *gorm.DB
}

// NewAILLMConfigRepository 创建AI LLM配置仓库
func NewAILLMConfigRepository(db *gorm.DB) *AILLMConfigRepository {
	return &AILLMConfigRepository{db: db}
}

// Create 创建配置
func (r *AILLMConfigRepository) Create(ctx context.Context, config *ai.AILLMConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

// Update 更新配置
func (r *AILLMConfigRepository) Update(ctx context.Context, config *ai.AILLMConfig) error {
	return r.db.WithContext(ctx).Save(config).Error
}

// Delete 删除配置（软删除）
func (r *AILLMConfigRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&ai.AILLMConfig{}, id).Error
}

// GetByID 根据ID获取配置
func (r *AILLMConfigRepository) GetByID(ctx context.Context, id uint) (*ai.AILLMConfig, error) {
	var config ai.AILLMConfig
	if err := r.db.WithContext(ctx).First(&config, id).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// GetByName 根据名称获取配置
func (r *AILLMConfigRepository) GetByName(ctx context.Context, name string) (*ai.AILLMConfig, error) {
	var config ai.AILLMConfig
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// GetDefault 获取默认配置
func (r *AILLMConfigRepository) GetDefault(ctx context.Context) (*ai.AILLMConfig, error) {
	var config ai.AILLMConfig
	if err := r.db.WithContext(ctx).
		Where("is_default = ? AND is_active = ?", true, true).
		First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// List 获取配置列表
func (r *AILLMConfigRepository) List(ctx context.Context) ([]ai.AILLMConfig, error) {
	var configs []ai.AILLMConfig
	if err := r.db.WithContext(ctx).Order("is_default DESC, created_at DESC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// GetActive 获取所有启用的配置
func (r *AILLMConfigRepository) GetActive(ctx context.Context) ([]ai.AILLMConfig, error) {
	var configs []ai.AILLMConfig
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("is_default DESC, created_at DESC").
		Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// SetDefault 设置默认配置
func (r *AILLMConfigRepository) SetDefault(ctx context.Context, id uint) error {
	// 先取消所有默认
	if err := r.db.WithContext(ctx).Model(&ai.AILLMConfig{}).
		Where("is_default = ?", true).
		Update("is_default", false).Error; err != nil {
		return err
	}

	// 设置新的默认
	return r.db.WithContext(ctx).Model(&ai.AILLMConfig{}).
		Where("id = ?", id).
		Update("is_default", true).Error
}

// ==================== AI 消息反馈仓库 ====================

// AIMessageFeedbackRepository AI消息反馈仓库
type AIMessageFeedbackRepository struct {
	db *gorm.DB
}

// NewAIMessageFeedbackRepository 创建AI消息反馈仓库
func NewAIMessageFeedbackRepository(db *gorm.DB) *AIMessageFeedbackRepository {
	return &AIMessageFeedbackRepository{db: db}
}

// Create 创建反馈
func (r *AIMessageFeedbackRepository) Create(ctx context.Context, feedback *ai.AIMessageFeedback) error {
	return r.db.WithContext(ctx).Create(feedback).Error
}

// Upsert 创建或更新反馈
func (r *AIMessageFeedbackRepository) Upsert(ctx context.Context, feedback *ai.AIMessageFeedback) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND message_id = ?", feedback.UserID, feedback.MessageID).
		Assign(map[string]interface{}{
			"feedback_type":   feedback.FeedbackType,
			"comment":         feedback.Comment,
			"conversation_id": feedback.ConversationID,
		}).
		FirstOrCreate(feedback).Error
}

// GetByMessageID 获取消息的反馈
func (r *AIMessageFeedbackRepository) GetByMessageID(ctx context.Context, messageID string) ([]ai.AIMessageFeedback, error) {
	var feedbacks []ai.AIMessageFeedback
	if err := r.db.WithContext(ctx).Where("message_id = ?", messageID).Find(&feedbacks).Error; err != nil {
		return nil, err
	}
	return feedbacks, nil
}

// GetByUserAndMessage 获取用户对消息的反馈
func (r *AIMessageFeedbackRepository) GetByUserAndMessage(ctx context.Context, userID uint, messageID string) (*ai.AIMessageFeedback, error) {
	var feedback ai.AIMessageFeedback
	if err := r.db.WithContext(ctx).
		Where("user_id = ? AND message_id = ?", userID, messageID).
		First(&feedback).Error; err != nil {
		return nil, err
	}
	return &feedback, nil
}

// GetStats 获取反馈统计
func (r *AIMessageFeedbackRepository) GetStats(ctx context.Context, conversationID string) (*AIFeedbackStats, error) {
	var stats AIFeedbackStats

	query := r.db.WithContext(ctx).Model(&ai.AIMessageFeedback{})
	if conversationID != "" {
		query = query.Where("conversation_id = ?", conversationID)
	}

	// 点赞数
	var likeCount int64
	r.db.WithContext(ctx).Model(&ai.AIMessageFeedback{}).
		Where("feedback_type = ?", ai.FeedbackLike).
		Count(&likeCount)
	stats.Likes = likeCount

	// 点踩数
	var dislikeCount int64
	r.db.WithContext(ctx).Model(&ai.AIMessageFeedback{}).
		Where("feedback_type = ?", ai.FeedbackDislike).
		Count(&dislikeCount)
	stats.Dislikes = dislikeCount

	stats.Total = stats.Likes + stats.Dislikes

	return &stats, nil
}

// AIFeedbackStats 反馈统计
type AIFeedbackStats struct {
	Total    int64 `json:"total"`
	Likes    int64 `json:"likes"`
	Dislikes int64 `json:"dislikes"`
}
