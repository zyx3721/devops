package jenkins

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bndr/gojenkins"

	"devops/internal/config"
	"devops/pkg/httpclient"
)

func init() {
	// Initialize gojenkins loggers to prevent panic
	gojenkins.Error = log.New(os.Stderr, "JENKINS ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	gojenkins.Warning = log.New(os.Stdout, "JENKINS WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	gojenkins.Info = log.New(os.Stdout, "JENKINS INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Client Jenkins客户端
type Client struct {
	jenkins *gojenkins.Jenkins
}

// JobSupervisor Job监控器
type JobSupervisor struct {
	Client *Client
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewJobSupervisor 创建 JobSupervisor
func NewJobSupervisor(client *Client) *JobSupervisor {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobSupervisor{
		Client: client,
		ctx:    ctx,
		cancel: cancel,
	}
}

// NewClient 创建 Jenkins 客户端
func NewClient() *Client {
	httpClient := httpclient.CreateClient()

	clientCopy := *httpClient
	clientCopy.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("无法加载配置: %v", err)
		return nil
	}

	jenkins := gojenkins.CreateJenkins(&clientCopy, cfg.JenkinsURL, cfg.JenkinsUser, cfg.JenkinsToken)
	return &Client{
		jenkins: jenkins,
	}
}

// BuildRequest 结构体定义
type BuildRequest struct {
	JobName      string `json:"job_name"`
	Branch       string `json:"branch"`
	DeployType   string `json:"deploy_type"`
	ImageVersion string `json:"image_version"`
}

// Build 触发构建
func (c *Client) Build(ctx context.Context, req BuildRequest) (int64, error) {
	jenkins := c.jenkins

	ctx, cancel := context.WithTimeout(ctx, 30*1e9)
	defer cancel()

	job, err := jenkins.GetJob(ctx, req.JobName)
	if err != nil {
		log.Printf("无法获取 Job '%s': %v", req.JobName, err)
		return 0, err
	}

	params := map[string]string{
		"BRANCH":        req.Branch,
		"DEPLOY_TYPE":   req.DeployType,
		"IMAGE_VERSION": req.ImageVersion,
	}

	var invokeErr error
	var queueID int64
	backoff := []int{1, 2, 4}

	type NameValue struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	type ParameterHolder struct {
		Parameter []NameValue `json:"parameter"`
	}
	holder := ParameterHolder{}
	for k, v := range params {
		holder.Parameter = append(holder.Parameter, NameValue{Name: k, Value: v})
	}
	jsonBytes, _ := json.Marshal(holder)

	data := url.Values{}
	data.Set("json", string(jsonBytes))

	for i := 0; i < len(backoff); i++ {
		endpoint := job.Base + "/build"
		payload := strings.NewReader(data.Encode())
		resp, err := jenkins.Requester.Post(ctx, endpoint, payload, nil, nil)

		if err == nil {
			if (resp.StatusCode >= 200 && resp.StatusCode < 300) || resp.StatusCode == 302 {
				location := resp.Header.Get("Location")
				if location == "" {
					location = resp.Header.Get("location")
				}

				if location == "" {
					if resp.StatusCode == 200 && resp.Request != nil && resp.Request.URL != nil {
						location = resp.Request.URL.String()
					}
				}

				if location == "" {
					invokeErr = fmt.Errorf("jenkins did not return Location header (status: %d)", resp.StatusCode)
				} else {
					if !strings.Contains(location, "/queue/item/") {
						log.Printf("Warning: Location header '%s' does not contain '/queue/item/'. Trying to find queue item by scanning queue...", location)

						time.Sleep(2 * time.Second)

						queue, err := jenkins.GetQueue(ctx)
						if err == nil {
							var foundID int64
							for _, item := range queue.Raw.Items {
								if item.Task.Name == req.JobName {
									if item.ID > foundID {
										foundID = item.ID
									}
								}
							}
							if foundID > 0 {
								queueID = foundID
								invokeErr = nil
								break
							}
						}

						lastBuild, err := job.GetLastBuild(ctx)
						if err == nil && lastBuild != nil {
							invokeErr = fmt.Errorf("failed to parse queue id from location '%s' and could not find in queue", location)
						} else {
							invokeErr = fmt.Errorf("failed to parse queue id from location '%s'", location)
						}

					} else {
						parts := strings.Split(strings.TrimRight(location, "/"), "/")
						if len(parts) > 0 {
							idStr := parts[len(parts)-1]
							queueID, err = strconv.ParseInt(idStr, 10, 64)
							if err != nil {
								invokeErr = fmt.Errorf("failed to parse queue id from location '%s': %v", location, err)
							} else {
								invokeErr = nil
								break
							}
						} else {
							invokeErr = fmt.Errorf("invalid Location header format: %s", location)
						}
					}
				}
			} else {
				invokeErr = fmt.Errorf("jenkins returned status code: %d", resp.StatusCode)
			}
		} else {
			invokeErr = err
		}

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}
		time.Sleep(time.Duration(backoff[i]) * time.Second)
	}
	if invokeErr != nil {
		log.Printf("无法启动 Job '%s' 的构建: %v", req.JobName, invokeErr)
		return 0, invokeErr
	}

	log.Printf("Build triggered for job '%s' with branch '%s' (DeployType: %s). Queue ID: %d", req.JobName, req.Branch, req.DeployType, queueID)
	return queueID, nil
}

// WaitForBuildToStart 等待构建开始并返回构建号
func (c *Client) WaitForBuildToStart(ctx context.Context, queueID int64) (int64, error) {
	jenkins := c.jenkins
	backoff := 1 * time.Second
	maxRetries := 180

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		task, err := jenkins.GetQueueItem(ctx, queueID)
		if err != nil {
			log.Printf("Failed to get queue item %d: %v", queueID, err)
			time.Sleep(backoff)
			continue
		}

		if task.Raw.Executable.Number != 0 {
			return task.Raw.Executable.Number, nil
		}

		time.Sleep(backoff)
	}

	return 0, fmt.Errorf("timeout waiting for build to start (queue id: %d)", queueID)
}

// GetJobBuildInfo 获取 Job 的构建信息
func (c *Client) GetJobBuildInfo(ctx context.Context, jobName string, buildNumber int) (*gojenkins.Build, error) {
	job, err := c.jenkins.GetJob(ctx, jobName)
	if err != nil {
		return nil, err
	}

	build, err := job.GetBuild(ctx, int64(buildNumber))
	if err != nil {
		return nil, err
	}

	return build, nil
}

// GetAllJobs 获取所有Job
func (c *Client) GetAllJobs(ctx context.Context) ([]*gojenkins.Job, error) {
	return c.jenkins.GetAllJobs(ctx)
}

// GetQueue 获取队列
func (c *Client) GetQueue(ctx context.Context) (*gojenkins.Queue, error) {
	return c.jenkins.GetQueue(ctx)
}
