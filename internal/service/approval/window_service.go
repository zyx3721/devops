package approval

import (
	"devops/internal/models"
	"devops/internal/repository"
	"strconv"
	"strings"
	"time"
)

type WindowService struct {
	repo *repository.DeployWindowRepository
}

func NewWindowService(repo *repository.DeployWindowRepository) *WindowService {
	return &WindowService{repo: repo}
}

// Create 创建发布窗口
func (s *WindowService) Create(window *models.DeployWindow) error {
	return s.repo.Create(window)
}

// Update 更新发布窗口
func (s *WindowService) Update(window *models.DeployWindow) error {
	return s.repo.Update(window)
}

// Delete 删除发布窗口
func (s *WindowService) Delete(id uint) error {
	return s.repo.Delete(id)
}

// GetByID 根据ID获取窗口
func (s *WindowService) GetByID(id uint) (*models.DeployWindow, error) {
	return s.repo.GetByID(id)
}

// List 获取窗口列表
func (s *WindowService) List(appID *uint) ([]models.DeployWindow, error) {
	return s.repo.List(appID)
}

// IsInWindow 检查当前时间是否在发布窗口内
// 返回: 是否在窗口内, 是否允许紧急发布, 错误
func (s *WindowService) IsInWindow(appID uint, env string) (bool, bool, error) {
	window, err := s.repo.GetByAppEnv(appID, env)
	if err != nil {
		// 没有找到窗口配置，默认允许发布
		return true, true, nil
	}

	if !window.Enabled {
		return true, true, nil
	}

	now := time.Now()
	
	// 检查星期
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // 周日转为7
	}
	if !isWeekdayAllowed(window.Weekdays, weekday) {
		return false, window.AllowEmergency, nil
	}

	// 检查时间
	currentTime := now.Format("15:04")
	if !isTimeInRange(currentTime, window.StartTime, window.EndTime) {
		return false, window.AllowEmergency, nil
	}

	return true, window.AllowEmergency, nil
}

// GetWindowInfo 获取窗口信息
func (s *WindowService) GetWindowInfo(appID uint, env string) (*models.DeployWindow, error) {
	return s.repo.GetByAppEnv(appID, env)
}

// isWeekdayAllowed 检查星期是否允许
func isWeekdayAllowed(weekdays string, weekday int) bool {
	if weekdays == "" {
		return true
	}
	parts := strings.Split(weekdays, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		w, err := strconv.Atoi(p)
		if err == nil && w == weekday {
			return true
		}
	}
	return false
}

// isTimeInRange 检查时间是否在范围内
func isTimeInRange(current, start, end string) bool {
	if start == "" || end == "" {
		return true
	}
	return current >= start && current <= end
}
