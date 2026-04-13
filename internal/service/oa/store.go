package oa

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"

	"devops/internal/models"
	"devops/internal/repository"
	"devops/pkg/logger"
)

var (
	Logger = logger.NewLogger("ERROR")
	oaRepo *repository.OADataRepository
)

// InitOAStore 初始化OA存储
func InitOAStore(db *gorm.DB) {
	oaRepo = repository.NewOADataRepository(db)
}

// SaveToDisk 保存数据到数据库（保持原函数名兼容）
func SaveToDisk(id string, req interface{}) error {
	if oaRepo == nil {
		Logger.Error("OA repository not initialized")
		return gorm.ErrInvalidDB
	}

	data, err := json.Marshal(req)
	if err != nil {
		Logger.Error("Failed to marshal request data: %v", err)
		return err
	}

	storedJSON, ok := req.(StoredJSON)
	oaData := &models.OAData{
		UniqueID:     id,
		OriginalData: string(data),
	}
	if ok {
		oaData.IPAddress = storedJSON.IPAddress
		oaData.UserAgent = storedJSON.UserAgent
	}

	err = oaRepo.Create(context.Background(), oaData)
	if err != nil {
		Logger.Error("Failed to save OA data to database: %v", err)
		return err
	}
	return nil
}

// LoadFromDisk 从数据库加载数据（保持原函数名兼容）
func LoadFromDisk(id string) (map[string]interface{}, error) {
	if oaRepo == nil {
		Logger.Error("OA repository not initialized")
		return nil, gorm.ErrInvalidDB
	}

	data, err := oaRepo.GetByUniqueID(context.Background(), id)
	if err != nil {
		Logger.Error("Failed to load OA data from database: %v", err)
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data.OriginalData), &result); err != nil {
		Logger.Error("Failed to unmarshal OA data: %v", err)
		return nil, err
	}
	return result, nil
}

// LoadJsonFromDiskALL 从数据库加载所有数据（保持原函数名兼容）
func LoadJsonFromDiskALL() (map[string]interface{}, error) {
	if oaRepo == nil {
		Logger.Error("OA repository not initialized")
		return nil, gorm.ErrInvalidDB
	}

	dataList, _, err := oaRepo.List(context.Background(), 1, 1000)
	if err != nil {
		Logger.Error("Failed to load all OA data from database: %v", err)
		return nil, err
	}

	result := make(map[string]interface{})
	for _, data := range dataList {
		var item map[string]interface{}
		if err := json.Unmarshal([]byte(data.OriginalData), &item); err != nil {
			Logger.Error("Failed to unmarshal OA data for id %s: %v", data.UniqueID, err)
			continue
		}
		result[data.UniqueID] = item
	}
	return result, nil
}

// GetLatestJsonFileContent 获取最新的数据（保持原函数名兼容）
func GetLatestJsonFileContent() (map[string]interface{}, error) {
	if oaRepo == nil {
		Logger.Error("OA repository not initialized")
		return nil, gorm.ErrInvalidDB
	}

	data, err := oaRepo.GetLatest(context.Background())
	if err != nil {
		Logger.Error("Failed to get latest OA data from database: %v", err)
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(data.OriginalData), &result); err != nil {
		Logger.Error("Failed to unmarshal latest OA data: %v", err)
		return nil, err
	}
	return result, nil
}
