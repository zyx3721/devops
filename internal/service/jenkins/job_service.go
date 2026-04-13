package jenkins

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/bndr/gojenkins"
	"gorm.io/gorm"

	"devops/internal/models"
	"devops/pkg/dto"
	apperrors "devops/pkg/errors"
	"devops/pkg/httpclient"
)

type JobService interface {
	GetJobs(ctx context.Context, instanceID uint) ([]dto.JenkinsJob, error)
	GetJobBuilds(ctx context.Context, instanceID uint, jobName string, limit int) ([]dto.JenkinsBuild, error)
}

type jobService struct {
	db *gorm.DB
}

func NewJobService(db *gorm.DB) JobService {
	return &jobService{db: db}
}

func (s *jobService) createClient(instance *models.JenkinsInstance) (*gojenkins.Jenkins, error) {
	httpClient := httpclient.CreateClient()
	clientCopy := *httpClient
	clientCopy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	jenkins := gojenkins.CreateJenkins(&clientCopy, instance.URL, instance.Username, instance.APIToken)
	return jenkins, nil
}

func (s *jobService) GetJobs(ctx context.Context, instanceID uint) ([]dto.JenkinsJob, error) {
	var instance models.JenkinsInstance
	if err := s.db.First(&instance, instanceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询Jenkins实例失败")
	}

	log.Printf("连接Jenkins: %s, 用户: %s", instance.URL, instance.Username)

	jenkins, err := s.createClient(&instance)
	if err != nil {
		log.Printf("创建Jenkins客户端失败: %v", err)
		return nil, apperrors.Wrap(err, apperrors.ErrCodeJenkinsAPI, "创建Jenkins客户端失败")
	}

	_, err = jenkins.Init(ctx)
	if err != nil {
		log.Printf("连接Jenkins失败: %v", err)
		return nil, apperrors.Wrap(err, apperrors.ErrCodeJenkinsAPI, fmt.Sprintf("连接Jenkins失败: %v", err))
	}

	jobs, err := jenkins.GetAllJobs(ctx)
	if err != nil {
		log.Printf("获取Job列表失败: %v", err)
		return nil, apperrors.Wrap(err, apperrors.ErrCodeJenkinsAPI, fmt.Sprintf("获取Job列表失败: %v", err))
	}

	log.Printf("获取到 %d 个Job", len(jobs))

	result := make([]dto.JenkinsJob, 0, len(jobs))
	for _, job := range jobs {
		jobInfo := dto.JenkinsJob{
			Name:  job.GetName(),
			URL:   job.Raw.URL,
			Color: job.Raw.Color,
			Class: job.Raw.Class,
		}

		// 获取最后一次构建信息（忽略错误，不影响列表展示）
		lastBuild, err := job.GetLastBuild(ctx)
		if err == nil && lastBuild != nil {
			jobInfo.LastBuildNumber = lastBuild.GetBuildNumber()
			jobInfo.LastBuildResult = lastBuild.GetResult()
			jobInfo.LastBuildTime = lastBuild.GetTimestamp().Format("2006-01-02 15:04:05")
		}

		result = append(result, jobInfo)
	}

	return result, nil
}

func (s *jobService) GetJobBuilds(ctx context.Context, instanceID uint, jobName string, limit int) ([]dto.JenkinsBuild, error) {
	var instance models.JenkinsInstance
	if err := s.db.First(&instance, instanceID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperrors.ErrNotFound
		}
		return nil, apperrors.Wrap(err, apperrors.ErrCodeInternalError, "查询Jenkins实例失败")
	}

	jenkins, err := s.createClient(&instance)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeJenkinsAPI, "创建Jenkins客户端失败")
	}

	_, err = jenkins.Init(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeJenkinsAPI, "连接Jenkins失败")
	}

	job, err := jenkins.GetJob(ctx, jobName)
	if err != nil {
		return nil, apperrors.Wrap(err, apperrors.ErrCodeJenkinsAPI, fmt.Sprintf("获取Job '%s' 失败", jobName))
	}

	buildIDs, err := job.GetAllBuildIds(ctx)
	if err != nil {
		log.Printf("获取构建ID列表失败: %v", err)
		return []dto.JenkinsBuild{}, nil
	}

	if limit <= 0 {
		limit = 20
	}
	if len(buildIDs) > limit {
		buildIDs = buildIDs[:limit]
	}

	result := make([]dto.JenkinsBuild, 0, len(buildIDs))
	for _, buildID := range buildIDs {
		build, err := job.GetBuild(ctx, buildID.Number)
		if err != nil {
			continue
		}

		result = append(result, dto.JenkinsBuild{
			Number:    build.GetBuildNumber(),
			Result:    build.GetResult(),
			Building:  build.IsRunning(ctx),
			Timestamp: build.GetTimestamp().Format("2006-01-02 15:04:05"),
			Duration:  int64(build.GetDuration() / 1000), // 转换为秒
			URL:       build.GetUrl(),
		})
	}

	return result, nil
}
