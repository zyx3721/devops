package logs

import (
	"crypto/rand"
	"encoding/hex"
	"time"

	"devops/internal/models"
	"devops/pkg/dto"

	"gorm.io/gorm"
)

// BookmarkService 书签服务
type BookmarkService struct {
	db *gorm.DB
}

// NewBookmarkService 创建书签服务
func NewBookmarkService(db *gorm.DB) *BookmarkService {
	return &BookmarkService{db: db}
}

// List 获取用户书签列表
func (s *BookmarkService) List(userID int64, page, pageSize int) ([]dto.BookmarkResponse, int64, error) {
	var bookmarks []models.LogBookmark
	var total int64

	query := s.db.Model(&models.LogBookmark{}).Where("user_id = ?", userID)
	query.Count(&total)

	if err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&bookmarks).Error; err != nil {
		return nil, 0, err
	}

	result := make([]dto.BookmarkResponse, len(bookmarks))
	for i, b := range bookmarks {
		result[i] = s.toResponse(&b)
	}

	return result, total, nil
}

// Create 创建书签
func (s *BookmarkService) Create(userID int64, req *dto.BookmarkRequest) (*dto.BookmarkResponse, error) {
	var logTimestamp *time.Time
	if req.LogTimestamp != "" {
		t, err := time.Parse(time.RFC3339Nano, req.LogTimestamp)
		if err != nil {
			t, _ = time.Parse(time.RFC3339, req.LogTimestamp)
		}
		logTimestamp = &t
	}

	bookmark := &models.LogBookmark{
		UserID:       userID,
		ClusterID:    req.ClusterID,
		Namespace:    req.Namespace,
		PodName:      req.PodName,
		Container:    req.Container,
		LogTimestamp: logTimestamp,
		Content:      req.Content,
		Note:         req.Note,
	}

	if err := s.db.Create(bookmark).Error; err != nil {
		return nil, err
	}

	resp := s.toResponse(bookmark)
	return &resp, nil
}

// Update 更新书签
func (s *BookmarkService) Update(userID, bookmarkID int64, note string) error {
	return s.db.Model(&models.LogBookmark{}).
		Where("id = ? AND user_id = ?", bookmarkID, userID).
		Update("note", note).Error
}

// Delete 删除书签
func (s *BookmarkService) Delete(userID, bookmarkID int64) error {
	return s.db.Where("id = ? AND user_id = ?", bookmarkID, userID).
		Delete(&models.LogBookmark{}).Error
}

// Share 生成分享链接
func (s *BookmarkService) Share(userID, bookmarkID int64, expiresInDays int) (string, error) {
	var bookmark models.LogBookmark
	if err := s.db.Where("id = ? AND user_id = ?", bookmarkID, userID).First(&bookmark).Error; err != nil {
		return "", err
	}

	// 生成分享 URL
	shareURL := s.generateShareURL()

	// 设置过期时间
	var expiresAt *time.Time
	if expiresInDays > 0 {
		t := time.Now().AddDate(0, 0, expiresInDays)
		expiresAt = &t
	}

	bookmark.ShareToken = shareURL
	bookmark.ShareExpiresAt = expiresAt

	if err := s.db.Save(&bookmark).Error; err != nil {
		return "", err
	}

	return shareURL, nil
}

// GetByShareURL 通过分享链接获取书签
func (s *BookmarkService) GetByShareURL(shareURL string) (*dto.BookmarkResponse, error) {
	var bookmark models.LogBookmark
	if err := s.db.Where("share_token = ?", shareURL).First(&bookmark).Error; err != nil {
		return nil, err
	}

	// 检查是否过期
	if bookmark.ShareExpiresAt != nil && bookmark.ShareExpiresAt.Before(time.Now()) {
		return nil, gorm.ErrRecordNotFound
	}

	resp := s.toResponse(&bookmark)
	return &resp, nil
}

func (s *BookmarkService) generateShareURL() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (s *BookmarkService) toResponse(b *models.LogBookmark) dto.BookmarkResponse {
	return dto.BookmarkResponse{
		ID:             b.ID,
		UserID:         b.UserID,
		ClusterID:      b.ClusterID,
		Namespace:      b.Namespace,
		PodName:        b.PodName,
		Container:      b.Container,
		LogTimestamp:   b.LogTimestamp,
		Content:        b.Content,
		Note:           b.Note,
		ShareURL:       b.ShareToken,
		ShareExpiresAt: b.ShareExpiresAt,
		CreatedAt:      b.CreatedAt,
	}
}
