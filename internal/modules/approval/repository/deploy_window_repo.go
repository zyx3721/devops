package repository

import (
	"devops/internal/models"

	"gorm.io/gorm"
)

type DeployWindowRepository struct {
	db *gorm.DB
}

func NewDeployWindowRepository(db *gorm.DB) *DeployWindowRepository {
	return &DeployWindowRepository{db: db}
}

// Create 创建发布窗口
func (r *DeployWindowRepository) Create(window *models.DeployWindow) error {
	return r.db.Create(window).Error
}

// Update 更新发布窗口
func (r *DeployWindowRepository) Update(window *models.DeployWindow) error {
	return r.db.Save(window).Error
}

// Delete 删除发布窗口
func (r *DeployWindowRepository) Delete(id uint) error {
	return r.db.Delete(&models.DeployWindow{}, id).Error
}

// GetByID 根据ID获取窗口
func (r *DeployWindowRepository) GetByID(id uint) (*models.DeployWindow, error) {
	var window models.DeployWindow
	err := r.db.First(&window, id).Error
	if err != nil {
		return nil, err
	}
	return &window, nil
}

// List 获取窗口列表
func (r *DeployWindowRepository) List(appID *uint) ([]models.DeployWindow, error) {
	var windows []models.DeployWindow
	query := r.db.Model(&models.DeployWindow{})
	if appID != nil {
		query = query.Where("app_id = ? OR app_id = 0", *appID)
	}
	err := query.Order("app_id ASC, env ASC").Find(&windows).Error
	return windows, err
}

// GetByAppEnv 根据应用和环境获取窗口（优先应用级别，其次全局）
func (r *DeployWindowRepository) GetByAppEnv(appID uint, env string) (*models.DeployWindow, error) {
	var window models.DeployWindow
	// 先查应用级别的窗口
	err := r.db.Where("app_id = ? AND env = ? AND enabled = ?", appID, env, true).First(&window).Error
	if err == nil {
		return &window, nil
	}
	// 再查全局窗口
	err = r.db.Where("app_id = 0 AND env = ? AND enabled = ?", env, true).First(&window).Error
	if err != nil {
		return nil, err
	}
	return &window, nil
}

// GetGlobalWindows 获取全局窗口
func (r *DeployWindowRepository) GetGlobalWindows() ([]models.DeployWindow, error) {
	var windows []models.DeployWindow
	err := r.db.Where("app_id = 0").Find(&windows).Error
	return windows, err
}
