package logs

import (
	"time"

	"devops/internal/models"
	"devops/pkg/dto"

	"gorm.io/gorm"
)

// SavedQueryService 快捷查询服务
type SavedQueryService struct {
	db *gorm.DB
}

// NewSavedQueryService 创建快捷查询服务
func NewSavedQueryService(db *gorm.DB) *SavedQueryService {
	return &SavedQueryService{db: db}
}

// List 获取用户的快捷查询列表
func (s *SavedQueryService) List(userID int64, includeShared bool) ([]dto.SavedQueryResponse, error) {
	var queries []models.LogSavedQuery

	query := s.db.Where("user_id = ?", userID)
	if includeShared {
		query = query.Or("is_shared = ?", true)
	}

	if err := query.Order("use_count DESC, updated_at DESC").Find(&queries).Error; err != nil {
		return nil, err
	}

	result := make([]dto.SavedQueryResponse, len(queries))
	for i, q := range queries {
		params := map[string]interface{}(q.QueryParams)
		if params == nil {
			params = make(map[string]interface{})
		}
		result[i] = dto.SavedQueryResponse{
			ID:          q.ID,
			UserID:      q.UserID,
			Name:        q.Name,
			Description: q.Description,
			QueryParams: params,
			IsShared:    q.IsShared,
			UseCount:    q.UseCount,
			LastUsedAt:  q.LastUsedAt,
			CreatedAt:   q.CreatedAt,
		}
	}

	return result, nil
}

// Create 创建快捷查询
func (s *SavedQueryService) Create(userID int64, req *dto.SavedQueryRequest) (*dto.SavedQueryResponse, error) {
	query := &models.LogSavedQuery{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		QueryParams: models.JSONObject(req.QueryParams),
		IsShared:    req.IsShared,
		UseCount:    0,
	}

	if err := s.db.Create(query).Error; err != nil {
		return nil, err
	}

	return &dto.SavedQueryResponse{
		ID:          query.ID,
		UserID:      query.UserID,
		Name:        query.Name,
		Description: query.Description,
		QueryParams: req.QueryParams,
		IsShared:    query.IsShared,
		UseCount:    query.UseCount,
		CreatedAt:   query.CreatedAt,
	}, nil
}

// Update 更新快捷查询
func (s *SavedQueryService) Update(userID, queryID int64, req *dto.SavedQueryRequest) (*dto.SavedQueryResponse, error) {
	var query models.LogSavedQuery
	if err := s.db.Where("id = ? AND user_id = ?", queryID, userID).First(&query).Error; err != nil {
		return nil, err
	}

	query.Name = req.Name
	query.Description = req.Description
	query.QueryParams = models.JSONObject(req.QueryParams)
	query.IsShared = req.IsShared

	if err := s.db.Save(&query).Error; err != nil {
		return nil, err
	}

	return &dto.SavedQueryResponse{
		ID:          query.ID,
		UserID:      query.UserID,
		Name:        query.Name,
		Description: query.Description,
		QueryParams: req.QueryParams,
		IsShared:    query.IsShared,
		UseCount:    query.UseCount,
		LastUsedAt:  query.LastUsedAt,
		CreatedAt:   query.CreatedAt,
	}, nil
}

// Delete 删除快捷查询
func (s *SavedQueryService) Delete(userID, queryID int64) error {
	return s.db.Where("id = ? AND user_id = ?", queryID, userID).Delete(&models.LogSavedQuery{}).Error
}

// Use 使用快捷查询（增加使用次数）
func (s *SavedQueryService) Use(queryID int64) error {
	now := time.Now()
	return s.db.Model(&models.LogSavedQuery{}).
		Where("id = ?", queryID).
		Updates(map[string]interface{}{
			"use_count":    gorm.Expr("use_count + 1"),
			"last_used_at": &now,
		}).Error
}

// Get 获取单个快捷查询
func (s *SavedQueryService) Get(queryID int64) (*dto.SavedQueryResponse, error) {
	var query models.LogSavedQuery
	if err := s.db.First(&query, queryID).Error; err != nil {
		return nil, err
	}

	params := map[string]interface{}(query.QueryParams)
	if params == nil {
		params = make(map[string]interface{})
	}

	return &dto.SavedQueryResponse{
		ID:          query.ID,
		UserID:      query.UserID,
		Name:        query.Name,
		Description: query.Description,
		QueryParams: params,
		IsShared:    query.IsShared,
		UseCount:    query.UseCount,
		LastUsedAt:  query.LastUsedAt,
		CreatedAt:   query.CreatedAt,
	}, nil
}
