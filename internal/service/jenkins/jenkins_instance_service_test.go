package jenkins

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestJenkinsConnection(t *testing.T) {
	// 测试一个不存在的 Jenkins 地址
	testURL := "http://10.8.2.192:30008"
	
	httpClient := &http.Client{Timeout: 10 * time.Second}
	apiURL := strings.TrimSuffix(testURL, "/") + "/api/json"
	
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}
	
	req.SetBasicAuth("admin", "test-token")
	
	t.Logf("测试连接: %s", apiURL)
	startTime := time.Now()
	
	resp, err := httpClient.Do(req)
	elapsed := time.Since(startTime)
	
	t.Logf("耗时: %v", elapsed)
	
	if err != nil {
		errStr := strings.ToLower(err.Error())
		t.Logf("连接错误: %v", err)
		
		switch {
		case strings.Contains(errStr, "connection refused"):
			t.Log("友好提示: 连接被拒绝，请检查 Jenkins 地址和端口是否正确")
		case strings.Contains(errStr, "no such host"):
			t.Log("友好提示: 无法解析主机名，请检查 Jenkins 地址是否正确")
		case strings.Contains(errStr, "timeout"), strings.Contains(errStr, "deadline exceeded"):
			t.Log("友好提示: 连接超时，请检查网络或 Jenkins 服务是否正常")
		case strings.Contains(errStr, "certificate"):
			t.Log("友好提示: SSL证书验证失败，请检查 HTTPS 配置")
		default:
			t.Logf("友好提示: %s", err.Error())
		}
		return
	}
	defer resp.Body.Close()
	
	t.Logf("响应状态码: %d", resp.StatusCode)
	t.Logf("Jenkins 版本: %s", resp.Header.Get("X-Jenkins"))
	
	switch resp.StatusCode {
	case http.StatusOK:
		t.Log("连接成功!")
	case http.StatusUnauthorized:
		t.Log("友好提示: 认证失败，请检查用户名和 API Token 是否正确")
	case http.StatusForbidden:
		t.Log("友好提示: 权限不足，请检查用户是否有访问权限")
	default:
		t.Logf("友好提示: Jenkins 返回错误状态码: %d", resp.StatusCode)
	}
}
