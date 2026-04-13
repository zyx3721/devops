package jenkins

import (
	"context"
	"fmt"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	jobStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "jenkins_job_status",
			Help: "Jenkins Job状态 (0=失败, 1=成功, 2=构建中, 3=未知)",
		},
		[]string{"job_name", "branch", "deploy_type", "image_version"},
	)

	buildDuration = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "jenkins_build_duration_seconds",
			Help: "构建持续时间（秒）",
		},
		[]string{"job_name", "build_number", "branch", "deploy_type", "image_version"},
	)

	buildTimestamp = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "jenkins_build_timestamp",
			Help: "构建时间戳",
		},
		[]string{"job_name", "build_number", "branch", "deploy_type", "image_version"},
	)
)

func init() {
	prometheus.MustRegister(jobStatus)
	prometheus.MustRegister(buildDuration)
	prometheus.MustRegister(buildTimestamp)
}

// UpdateMetrics 更新Prometheus指标
func UpdateMetrics(jobName string, build *gojenkins.Build) {
	var branch, deployType, imageVersion string
	for _, p := range build.GetParameters() {
		switch p.Name {
		case "BRANCH":
			branch = p.Value
		case "DEPLOY_TYPE":
			deployType = p.Value
		case "IMAGE_VERSION":
			imageVersion = p.Value
		}
	}

	var statusValue float64
	switch build.GetResult() {
	case "SUCCESS":
		statusValue = 1
	case "FAILURE", "UNSTABLE":
		statusValue = 0
	case "":
		if build.Raw.Building {
			statusValue = 2
		} else {
			statusValue = 3
		}
	default:
		statusValue = 3
	}

	jobStatus.WithLabelValues(jobName, branch, deployType, imageVersion).Set(statusValue)

	buildDuration.WithLabelValues(jobName, fmt.Sprintf("%d", build.GetBuildNumber()), branch, deployType, imageVersion).Set(float64(build.Raw.Duration) / 1000)

	buildTimestamp.WithLabelValues(jobName, fmt.Sprintf("%d", build.GetBuildNumber()), branch, deployType, imageVersion).Set(float64(build.Raw.Timestamp) / 1000)
}

// MonitorBuildUntilCompletion 监控特定构建直到完成
func (c *Client) MonitorBuildUntilCompletion(ctx context.Context, jobName string, buildNumber int64) (*gojenkins.Build, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			job, err := c.jenkins.GetJob(ctx, jobName)
			if err != nil {
				continue
			}
			build, err := job.GetBuild(ctx, buildNumber)
			if err != nil {
				continue
			}

			UpdateMetrics(jobName, build)

			if !build.IsRunning(ctx) {
				return build, nil
			}
		}
	}
}
