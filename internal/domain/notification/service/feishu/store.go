package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"gorm.io/gorm"

	"devops/internal/domain/notification/repository"
	"devops/internal/models"
)

// RequestStore 用于存储发送的卡片请求数据
type RequestStore struct {
	cache sync.Map
	mu    sync.Mutex
	repo  *repository.FeishuRequestRepository
}

// GlobalStore 全局请求存储
var GlobalStore = &RequestStore{}

// InitFeishuStore 初始化飞书存储
func InitFeishuStore(db *gorm.DB) {
	GlobalStore.repo = repository.NewFeishuRequestRepository(db)
}

// StoredRequest 存储的请求数据
type StoredRequest struct {
	OriginalRequest GrayCardRequest
	DisabledActions map[string]bool
	ActionCounts    map[string]int
}

// Save 保存请求数据
func (s *RequestStore) Save(id string, req GrayCardRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stored := &StoredRequest{
		OriginalRequest: req,
		DisabledActions: make(map[string]bool),
		ActionCounts:    make(map[string]int),
	}
	s.cache.Store(id, stored)

	if s.repo != nil {
		originalReqJSON, _ := json.Marshal(req)
		disabledJSON, _ := json.Marshal(stored.DisabledActions)
		countsJSON, _ := json.Marshal(stored.ActionCounts)

		dbReq := &models.FeishuRequest{
			RequestID:       id,
			OriginalRequest: string(originalReqJSON),
			DisabledActions: string(disabledJSON),
			ActionCounts:    string(countsJSON),
		}
		if err := s.repo.Create(context.Background(), dbReq); err != nil {
			fmt.Printf("Failed to save feishu request to database: %v\n", err)
		}
	}
}

// Delete 删除请求数据
func (s *RequestStore) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache.Delete(id)

	if s.repo != nil {
		if err := s.repo.Delete(context.Background(), id); err != nil {
			fmt.Printf("Failed to delete feishu request from database: %v\n", err)
		}
	}
}

// Get 获取请求数据
func (s *RequestStore) Get(id string) (*StoredRequest, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	val, ok := s.cache.Load(id)
	if ok {
		return val.(*StoredRequest), true
	}

	if s.repo != nil {
		dbReq, err := s.repo.GetByRequestID(context.Background(), id)
		if err == nil {
			stored := &StoredRequest{
				DisabledActions: make(map[string]bool),
				ActionCounts:    make(map[string]int),
			}
			json.Unmarshal([]byte(dbReq.OriginalRequest), &stored.OriginalRequest)
			json.Unmarshal([]byte(dbReq.DisabledActions), &stored.DisabledActions)
			json.Unmarshal([]byte(dbReq.ActionCounts), &stored.ActionCounts)

			s.cache.Store(id, stored)
			return stored, true
		}
	}

	return nil, false
}

// MarkActionDisabled 标记某个动作已禁用
func (s *RequestStore) MarkActionDisabled(id, serviceName, action string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	val, ok := s.cache.Load(id)
	var req *StoredRequest

	if !ok {
		if s.repo != nil {
			dbReq, err := s.repo.GetByRequestID(context.Background(), id)
			if err != nil {
				fmt.Printf("MarkActionDisabled: ID %s not found\n", id)
				return
			}
			req = &StoredRequest{
				DisabledActions: make(map[string]bool),
				ActionCounts:    make(map[string]int),
			}
			json.Unmarshal([]byte(dbReq.OriginalRequest), &req.OriginalRequest)
			json.Unmarshal([]byte(dbReq.DisabledActions), &req.DisabledActions)
			json.Unmarshal([]byte(dbReq.ActionCounts), &req.ActionCounts)
			s.cache.Store(id, req)
		} else {
			return
		}
	} else {
		req = val.(*StoredRequest)
	}

	key := serviceName + ":" + action
	req.DisabledActions[key] = true

	s.saveToDatabase(id, req)
}

// IncrementActionCount 增加动作执行次数
func (s *RequestStore) IncrementActionCount(id, serviceName, action string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	val, ok := s.cache.Load(id)
	var req *StoredRequest

	if !ok {
		if s.repo != nil {
			dbReq, err := s.repo.GetByRequestID(context.Background(), id)
			if err != nil {
				return
			}
			req = &StoredRequest{
				DisabledActions: make(map[string]bool),
				ActionCounts:    make(map[string]int),
			}
			json.Unmarshal([]byte(dbReq.OriginalRequest), &req.OriginalRequest)
			json.Unmarshal([]byte(dbReq.DisabledActions), &req.DisabledActions)
			json.Unmarshal([]byte(dbReq.ActionCounts), &req.ActionCounts)
			s.cache.Store(id, req)
		} else {
			return
		}
	} else {
		req = val.(*StoredRequest)
	}

	key := serviceName + ":" + action
	if req.ActionCounts == nil {
		req.ActionCounts = make(map[string]int)
	}
	req.ActionCounts[key]++

	s.saveToDatabase(id, req)
}

// GetActionCount 获取动作执行次数
func (s *RequestStore) GetActionCount(id, serviceName, action string) int {
	req, ok := s.Get(id)
	if !ok {
		return 0
	}

	key := serviceName + ":" + action
	if req.ActionCounts == nil {
		return 0
	}
	return req.ActionCounts[key]
}

// IsActionDisabled 检查某个动作是否已执行
func (s *RequestStore) IsActionDisabled(id, serviceName, action string) bool {
	req, ok := s.Get(id)
	if !ok {
		return false
	}

	key := serviceName + ":" + action
	return req.DisabledActions[key]
}

// saveToDatabase 保存到数据库
func (s *RequestStore) saveToDatabase(id string, req *StoredRequest) {
	if s.repo == nil {
		return
	}

	dbReq, err := s.repo.GetByRequestID(context.Background(), id)
	if err != nil {
		return
	}

	originalReqJSON, _ := json.Marshal(req.OriginalRequest)
	disabledJSON, _ := json.Marshal(req.DisabledActions)
	countsJSON, _ := json.Marshal(req.ActionCounts)

	dbReq.OriginalRequest = string(originalReqJSON)
	dbReq.DisabledActions = string(disabledJSON)
	dbReq.ActionCounts = string(countsJSON)

	if err := s.repo.Update(context.Background(), dbReq); err != nil {
		fmt.Printf("Failed to update feishu request in database: %v\n", err)
	}
}
