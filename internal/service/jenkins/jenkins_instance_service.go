package jenkins

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bndr/gojenkins"
	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
	"devops/pkg/httpclient"
	"devops/pkg/logger"
)

// 保留 gojenkins 和 httpclient 的引用，其他地方可能用到
var (
	_ = gojenkins.CreateJenkins
	_ = httpclient.CreateClient
)

type JenkinsInstanceService interface {
	CreateJenkinsInstance(ctx context.Context, req *dto.CreateJenkinsInstanceRequest) (*dto.JenkinsInstanceResponse, error)
	GetJenkinsInstance(ctx context.Context, id uint) (*dto.JenkinsInstanceResponse, error)
	GetJenkinsInstanceList(ctx context.Context, req *dto.JenkinsInstanceListRequest) (*dto.JenkinsInstanceListResponse, error)
	UpdateJenkinsInstance(ctx context.Context, id uint, req *dto.UpdateJenkinsInstanceRequest) (*dto.JenkinsInstanceResponse, error)
	DeleteJenkinsInstance(ctx context.Context, id uint) error
	SetDefaultJenkinsInstance(ctx context.Context, id uint) error
	GetDefaultJenkinsInstance(ctx context.Context) (*dto.JenkinsInstanceResponse, error)
	GetFeishuApps(ctx context.Context, id uint) ([]dto.FeishuAppSimple, error)
	BindFeishuApps(ctx context.Context, id uint, appIDs []uint) error
	TestConnection(ctx context.Context, id uint) (*dto.ConnectionTestResult, error)
}

type jenkinsInstanceService struct {
	db  *gorm.DB
	log *logger.Logger
}

func NewJenkinsInstanceService(db *gorm.DB) JenkinsInstanceService {
	return &jenkinsInstanceService{db: db, log: logger.NewLogger("info")}
}

func (s *jenkinsInstanceService) CreateJenkinsInstance(ctx context.Context, req *dto.CreateJenkinsInstanceRequest) (*dto.JenkinsInstanceResponse, error) {
	instance := &models.JenkinsInstance{
		Name: req.Name, URL: req.URL, Username: req.Username, APIToken: req.APIToken,
		Description: req.Description, Status: req.Status, IsDefault: req.IsDefault,
	}

	if req.IsDefault {
		s.db.Model(&models.JenkinsInstance{}).Where("is_default = ?", true).Update("is_default", false)
	}

	if err := s.db.Create(instance).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建Jenkins实例失败")
	}

	return s.buildResponse(instance), nil
}

func (s *jenkinsInstanceService) GetJenkinsInstance(ctx context.Context, id uint) (*dto.JenkinsInstanceResponse, error) {
	var instance models.JenkinsInstance
	if err := s.db.First(&instance, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询Jenkins实例失败")
	}
	return s.buildResponse(&instance), nil
}

func (s *jenkinsInstanceService) GetJenkinsInstanceList(ctx context.Context, req *dto.JenkinsInstanceListRequest) (*dto.JenkinsInstanceListResponse, error) {
	query := s.db.Model(&models.JenkinsInstance{})

	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		query = query.Where("name LIKE ? OR url LIKE ? OR description LIKE ?", keyword, keyword, keyword)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询Jenkins实例总数失败")
	}

	var instances []models.JenkinsInstance
	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Order("created_at DESC").Find(&instances).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询Jenkins实例列表失败")
	}

	items := make([]dto.JenkinsInstanceResponse, len(instances))
	for i, instance := range instances {
		items[i] = *s.buildResponse(&instance)
	}

	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize != 0 {
		totalPages++
	}

	return &dto.JenkinsInstanceListResponse{Total: total, Page: req.Page, PageSize: req.PageSize, TotalPages: totalPages, Items: items}, nil
}

func (s *jenkinsInstanceService) UpdateJenkinsInstance(ctx context.Context, id uint, req *dto.UpdateJenkinsInstanceRequest) (*dto.JenkinsInstanceResponse, error) {
	var instance models.JenkinsInstance
	if err := s.db.First(&instance, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询Jenkins实例失败")
	}

	if req.IsDefault {
		s.db.Model(&models.JenkinsInstance{}).Where("is_default = ? AND id != ?", true, id).Update("is_default", false)
	}

	instance.Name = req.Name
	instance.URL = req.URL
	instance.Username = req.Username
	instance.APIToken = req.APIToken
	instance.Description = req.Description
	instance.Status = req.Status
	instance.IsDefault = req.IsDefault

	if err := s.db.Save(&instance).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "更新Jenkins实例失败")
	}

	return s.buildResponse(&instance), nil
}

func (s *jenkinsInstanceService) DeleteJenkinsInstance(ctx context.Context, id uint) error {
	var instance models.JenkinsInstance
	if err := s.db.First(&instance, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询Jenkins实例失败")
	}

	if err := s.db.Delete(&instance).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "删除Jenkins实例失败")
	}
	return nil
}

func (s *jenkinsInstanceService) SetDefaultJenkinsInstance(ctx context.Context, id uint) error {
	var instance models.JenkinsInstance
	if err := s.db.First(&instance, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询Jenkins实例失败")
	}

	s.db.Model(&models.JenkinsInstance{}).Where("is_default = ?", true).Update("is_default", false)
	instance.IsDefault = true
	if err := s.db.Save(&instance).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "设置默认Jenkins实例失败")
	}
	return nil
}

func (s *jenkinsInstanceService) GetDefaultJenkinsInstance(ctx context.Context) (*dto.JenkinsInstanceResponse, error) {
	var instance models.JenkinsInstance
	if err := s.db.Where("is_default = ?", true).First(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			if err = s.db.Where("status = ?", "active").First(&instance).Error; err != nil {
				return nil, apperrors.ErrNotFound
			}
		} else {
			return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询默认Jenkins实例失败")
		}
	}
	return s.buildResponse(&instance), nil
}

func (s *jenkinsInstanceService) buildResponse(instance *models.JenkinsInstance) *dto.JenkinsInstanceResponse {
	return &dto.JenkinsInstanceResponse{
		ID: instance.ID, Name: instance.Name, URL: instance.URL, Username: instance.Username,
		Description: instance.Description, Status: instance.Status, IsDefault: instance.IsDefault,
		CreatedAt: instance.CreatedAt.Format("2006-01-02 15:04:05"), UpdatedAt: instance.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

func (s *jenkinsInstanceService) GetFeishuApps(ctx context.Context, id uint) ([]dto.FeishuAppSimple, error) {
	var instance models.JenkinsInstance
	if err := s.db.First(&instance, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询Jenkins实例失败")
	}

	var bindings []models.JenkinsFeishuApp
	if err := s.db.Preload("FeishuApp").Where("jenkins_instance_id = ?", id).Find(&bindings).Error; err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询绑定的飞书应用失败")
	}

	apps := make([]dto.FeishuAppSimple, len(bindings))
	for i, b := range bindings {
		apps[i] = dto.FeishuAppSimple{ID: b.FeishuApp.ID, Name: b.FeishuApp.Name, AppID: b.FeishuApp.AppID, Project: b.FeishuApp.Project}
	}
	return apps, nil
}

func (s *jenkinsInstanceService) BindFeishuApps(ctx context.Context, id uint, appIDs []uint) error {
	var instance models.JenkinsInstance
	if err := s.db.First(&instance, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperrors.ErrNotFound
		}
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询Jenkins实例失败")
	}

	// 删除旧的绑定
	if err := s.db.Where("jenkins_instance_id = ?", id).Delete(&models.JenkinsFeishuApp{}).Error; err != nil {
		return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "删除旧绑定失败")
	}

	// 创建新的绑定
	for _, appID := range appIDs {
		binding := &models.JenkinsFeishuApp{JenkinsInstanceID: id, FeishuAppID: appID}
		if err := s.db.Create(binding).Error; err != nil {
			return apperrors.Wrap(err, apperrors.ErrCodeInternalError, "创建绑定失败")
		}
	}
	return nil
}

func (s *jenkinsInstanceService) TestConnection(ctx context.Context, id uint) (*dto.ConnectionTestResult, error) {
	var instance models.JenkinsInstance
	if err := s.db.First(&instance, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询Jenkins实例失败")
	}

	startTime := time.Now()
	result := &dto.ConnectionTestResult{Connected: false}

	// 直接用 HTTP 请求测试连接，不用 gojenkins 库
	httpClient := &http.Client{Timeout: 5 * time.Second}

	// 构建请求 URL
	apiURL := strings.TrimSuffix(instance.URL, "/") + "/api/json"
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		result.Error = "创建请求失败"
		result.ResponseTimeMs = time.Since(startTime).Milliseconds()
		return result, nil
	}

	// 设置 Basic Auth
	if instance.Username != "" && instance.APIToken != "" {
		req.SetBasicAuth(instance.Username, instance.APIToken)
	}

	resp, err := httpClient.Do(req)
	result.ResponseTimeMs = time.Since(startTime).Milliseconds()

	if err != nil {
		// 解析常见错误，提供友好提示
		errStr := strings.ToLower(err.Error())
		switch {
		case strings.Contains(errStr, "connection refused"):
			result.Error = "连接被拒绝，请检查 Jenkins 地址和端口是否正确"
		case strings.Contains(errStr, "no such host"):
			result.Error = "无法解析主机名，请检查 Jenkins 地址是否正确"
		case strings.Contains(errStr, "timeout"), strings.Contains(errStr, "deadline exceeded"):
			result.Error = "连接超时，请检查网络或 Jenkins 服务是否正常"
		case strings.Contains(errStr, "certificate"):
			result.Error = "SSL证书验证失败，请检查 HTTPS 配置"
		default:
			result.Error = err.Error()
		}
		return result, nil
	}
	defer resp.Body.Close()

	// 检查响应状态码
	switch resp.StatusCode {
	case http.StatusOK:
		result.Connected = true
		// 尝试获取版本号
		if v := resp.Header.Get("X-Jenkins"); v != "" {
			result.Version = v
		}
	case http.StatusUnauthorized:
		result.Error = "Jenkins 认证失败，请检查用户名和 API Token 是否正确"
	case http.StatusForbidden:
		result.Error = "Jenkins 用户权限不足，请检查该用户在 Jenkins 中是否有 API 访问权限"
	default:
		result.Error = fmt.Sprintf("Jenkins 返回错误状态码: %d", resp.StatusCode)
	}

	return result, nil
}
