package repository

import (
	"context"

	"gorm.io/gorm"

	"devops/internal/models"
)

// OADataRepository OA数据仓储
type OADataRepository struct {
	db *gorm.DB
}

func NewOADataRepository(db *gorm.DB) *OADataRepository {
	return &OADataRepository{db: db}
}

func (r *OADataRepository) Create(ctx context.Context, data *models.OAData) error {
	return r.db.WithContext(ctx).Create(data).Error
}

func (r *OADataRepository) GetByUniqueID(ctx context.Context, uniqueID string) (*models.OAData, error) {
	var data models.OAData
	err := r.db.WithContext(ctx).Where("unique_id = ?", uniqueID).First(&data).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *OADataRepository) GetByID(ctx context.Context, id uint) (*models.OAData, error) {
	var data models.OAData
	err := r.db.WithContext(ctx).First(&data, id).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *OADataRepository) List(ctx context.Context, page, pageSize int) ([]models.OAData, int64, error) {
	var dataList []models.OAData
	var total int64

	query := r.db.WithContext(ctx).Model(&models.OAData{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&dataList).Error; err != nil {
		return nil, 0, err
	}

	return dataList, total, nil
}

func (r *OADataRepository) ListBySource(ctx context.Context, source string, page, pageSize int) ([]models.OAData, int64, error) {
	var dataList []models.OAData
	var total int64

	query := r.db.WithContext(ctx).Model(&models.OAData{})
	if source != "" {
		query = query.Where("source LIKE ?", "%"+source+"%")
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&dataList).Error; err != nil {
		return nil, 0, err
	}

	return dataList, total, nil
}

func (r *OADataRepository) GetLatest(ctx context.Context) (*models.OAData, error) {
	var data models.OAData
	err := r.db.WithContext(ctx).Order("created_at DESC").First(&data).Error
	if err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *OADataRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.OAData{}, id).Error
}

// OAAddressRepository OA地址仓储
type OAAddressRepository struct {
	db *gorm.DB
}

func NewOAAddressRepository(db interface{}) *OAAddressRepository {
	if gormDB, ok := db.(*gorm.DB); ok {
		return &OAAddressRepository{db: gormDB}
	}
	return nil
}

func (r *OAAddressRepository) Create(ctx context.Context, addr *models.OAAddress) error {
	return r.db.WithContext(ctx).Create(addr).Error
}

func (r *OAAddressRepository) GetByID(ctx context.Context, id uint) (*models.OAAddress, error) {
	var addr models.OAAddress
	err := r.db.WithContext(ctx).First(&addr, id).Error
	if err != nil {
		return nil, err
	}
	return &addr, nil
}

func (r *OAAddressRepository) List(ctx context.Context, page, pageSize int) ([]models.OAAddress, int64, error) {
	var list []models.OAAddress
	var total int64

	query := r.db.WithContext(ctx).Model(&models.OAAddress{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *OAAddressRepository) Update(ctx context.Context, addr *models.OAAddress) error {
	return r.db.WithContext(ctx).Model(addr).Where("id = ?", addr.ID).Updates(map[string]interface{}{
		"name":        addr.Name,
		"url":         addr.URL,
		"type":        addr.Type,
		"description": addr.Description,
		"status":      addr.Status,
		"is_default":  addr.IsDefault,
		"created_by":  addr.CreatedBy,
	}).Error
}

func (r *OAAddressRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.OAAddress{}, id).Error
}

func (r *OAAddressRepository) GetDefault(ctx context.Context) (*models.OAAddress, error) {
	var addr models.OAAddress
	err := r.db.WithContext(ctx).Where("is_default = ? AND status = ?", true, "active").First(&addr).Error
	if err != nil {
		return nil, err
	}
	return &addr, nil
}

// OANotifyConfigRepository OA通知配置仓储
type OANotifyConfigRepository struct {
	db *gorm.DB
}

func NewOANotifyConfigRepository(db *gorm.DB) *OANotifyConfigRepository {
	return &OANotifyConfigRepository{db: db}
}

func (r *OANotifyConfigRepository) Create(ctx context.Context, config *models.OANotifyConfig) error {
	return r.db.WithContext(ctx).Create(config).Error
}

func (r *OANotifyConfigRepository) GetByID(ctx context.Context, id uint) (*models.OANotifyConfig, error) {
	var config models.OANotifyConfig
	err := r.db.WithContext(ctx).First(&config, id).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *OANotifyConfigRepository) List(ctx context.Context, page, pageSize int) ([]models.OANotifyConfig, int64, error) {
	var list []models.OANotifyConfig
	var total int64

	query := r.db.WithContext(ctx).Model(&models.OANotifyConfig{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *OANotifyConfigRepository) ListActive(ctx context.Context) ([]models.OANotifyConfig, error) {
	var list []models.OANotifyConfig
	err := r.db.WithContext(ctx).Where("status = ?", "active").Find(&list).Error
	return list, err
}

func (r *OANotifyConfigRepository) Update(ctx context.Context, config *models.OANotifyConfig) error {
	return r.db.WithContext(ctx).Model(config).Where("id = ?", config.ID).Updates(map[string]interface{}{
		"name":            config.Name,
		"app_id":          config.AppID,
		"receive_id":      config.ReceiveID,
		"receive_id_type": config.ReceiveIDType,
		"description":     config.Description,
		"status":          config.Status,
		"is_default":      config.IsDefault,
	}).Error
}

func (r *OANotifyConfigRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.OANotifyConfig{}, id).Error
}

func (r *OANotifyConfigRepository) GetDefault(ctx context.Context) (*models.OANotifyConfig, error) {
	var config models.OANotifyConfig
	err := r.db.WithContext(ctx).Where("is_default = ? AND status = ?", true, "active").First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *OANotifyConfigRepository) SetDefault(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Model(&models.OANotifyConfig{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Model(&models.OANotifyConfig{}).Where("id = ?", id).Update("is_default", true).Error
}
