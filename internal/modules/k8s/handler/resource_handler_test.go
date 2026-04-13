package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"devops/internal/service/kubernetes"
)

// mockNamespaceService 模拟 Namespace 服务
type mockNamespaceService struct {
	listNamespacesFunc func(ctx context.Context, clusterID uint) ([]kubernetes.NamespaceInfo, error)
	getNamespaceFunc   func(ctx context.Context, clusterID uint, name string) (*kubernetes.NamespaceDetail, error)
}

func (m *mockNamespaceService) ListNamespaces(ctx context.Context, clusterID uint) ([]kubernetes.NamespaceInfo, error) {
	if m.listNamespacesFunc != nil {
		return m.listNamespacesFunc(ctx, clusterID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockNamespaceService) GetNamespace(ctx context.Context, clusterID uint, name string) (*kubernetes.NamespaceDetail, error) {
	if m.getNamespaceFunc != nil {
		return m.getNamespaceFunc(ctx, clusterID, name)
	}
	return nil, errors.New("not implemented")
}

// TestListNamespaces_Success 测试成功获取命名空间列表
func TestListNamespaces_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 准备测试数据
	expectedNamespaces := []kubernetes.NamespaceInfo{
		{
			Name:      "default",
			Status:    "Active",
			Age:       "30d",
			Labels:    map[string]string{"env": "prod"},
			CreatedAt: "2024-01-01 00:00:00",
		},
		{
			Name:      "kube-system",
			Status:    "Active",
			Age:       "30d",
			Labels:    map[string]string{},
			CreatedAt: "2024-01-01 00:00:00",
		},
	}

	// 创建 mock 服务
	mockNsSvc := &mockNamespaceService{
		listNamespacesFunc: func(ctx context.Context, clusterID uint) ([]kubernetes.NamespaceInfo, error) {
			assert.Equal(t, uint(1), clusterID)
			return expectedNamespaces, nil
		},
	}

	// 创建 handler
	handler := &K8sResourceHandler{
		namespaceSvc: mockNsSvc,
	}

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("GET", "/k8s/clusters/1/namespaces", nil)

	// 执行请求
	handler.ListNamespaces(c)

	// 验证结果
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 验证返回的数据
	data, ok := response["data"].([]interface{})
	assert.True(t, ok)
	assert.Equal(t, 2, len(data))
}

// TestListNamespaces_InvalidClusterID 测试无效的集群ID
func TestListNamespaces_InvalidClusterID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &K8sResourceHandler{}

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	c.Request = httptest.NewRequest("GET", "/k8s/clusters/invalid/namespaces", nil)

	// 执行请求
	handler.ListNamespaces(c)

	// 验证结果
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "无效的集群ID")
}

// TestListNamespaces_ServiceError 测试服务层错误
func TestListNamespaces_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建 mock 服务
	mockNsSvc := &mockNamespaceService{
		listNamespacesFunc: func(ctx context.Context, clusterID uint) ([]kubernetes.NamespaceInfo, error) {
			return nil, errors.New("service error")
		},
	}

	// 创建 handler
	handler := &K8sResourceHandler{
		namespaceSvc: mockNsSvc,
	}

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Request = httptest.NewRequest("GET", "/k8s/clusters/1/namespaces", nil)

	// 执行请求
	handler.ListNamespaces(c)

	// 验证结果
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "获取命名空间列表失败")
}

// TestGetNamespace_Success 测试成功获取命名空间详情
func TestGetNamespace_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 准备测试数据
	expectedNamespace := &kubernetes.NamespaceDetail{
		NamespaceInfo: kubernetes.NamespaceInfo{
			Name:      "default",
			Status:    "Active",
			Age:       "30d",
			Labels:    map[string]string{"env": "prod"},
			CreatedAt: "2024-01-01 00:00:00",
		},
		Annotations: map[string]string{"description": "Default namespace"},
		ResourceQuota: []kubernetes.ResourceQuotaInfo{
			{
				Name: "default-quota",
				Hard: map[string]string{"cpu": "10", "memory": "10Gi"},
				Used: map[string]string{"cpu": "5", "memory": "5Gi"},
			},
		},
	}

	// 创建 mock 服务
	mockNsSvc := &mockNamespaceService{
		getNamespaceFunc: func(ctx context.Context, clusterID uint, name string) (*kubernetes.NamespaceDetail, error) {
			assert.Equal(t, uint(1), clusterID)
			assert.Equal(t, "default", name)
			return expectedNamespace, nil
		},
	}

	// 创建 handler
	handler := &K8sResourceHandler{
		namespaceSvc: mockNsSvc,
	}

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "name", Value: "default"},
	}
	c.Request = httptest.NewRequest("GET", "/k8s/clusters/1/namespaces/default", nil)

	// 执行请求
	handler.GetNamespace(c)

	// 验证结果
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 验证返回的数据
	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "default", data["name"])
}

// TestGetNamespace_InvalidClusterID 测试无效的集群ID
func TestGetNamespace_InvalidClusterID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &K8sResourceHandler{}

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "invalid"},
		{Key: "name", Value: "default"},
	}
	c.Request = httptest.NewRequest("GET", "/k8s/clusters/invalid/namespaces/default", nil)

	// 执行请求
	handler.GetNamespace(c)

	// 验证结果
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "无效的集群ID")
}

// TestGetNamespace_EmptyName 测试空的命名空间名称
func TestGetNamespace_EmptyName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &K8sResourceHandler{}

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "name", Value: ""},
	}
	c.Request = httptest.NewRequest("GET", "/k8s/clusters/1/namespaces/", nil)

	// 执行请求
	handler.GetNamespace(c)

	// 验证结果
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "命名空间名称不能为空")
}

// TestGetNamespace_ServiceError 测试服务层错误
func TestGetNamespace_ServiceError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建 mock 服务
	mockNsSvc := &mockNamespaceService{
		getNamespaceFunc: func(ctx context.Context, clusterID uint, name string) (*kubernetes.NamespaceDetail, error) {
			return nil, errors.New("service error")
		},
	}

	// 创建 handler
	handler := &K8sResourceHandler{
		namespaceSvc: mockNsSvc,
	}

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "name", Value: "default"},
	}
	c.Request = httptest.NewRequest("GET", "/k8s/clusters/1/namespaces/default", nil)

	// 执行请求
	handler.GetNamespace(c)

	// 验证结果
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "获取命名空间详情失败")
}

// ===== Deployment Handler Tests =====

// TestListDeployments_InvalidClusterID 测试无效的集群ID
func TestListDeployments_InvalidClusterID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &K8sResourceHandler{}

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "id", Value: "invalid"}}
	c.Request = httptest.NewRequest("GET", "/k8s/clusters/invalid/deployments", nil)

	// 执行请求
	handler.ListDeployments(c)

	// 验证结果
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "无效的集群ID")
}

// TestGetDeployment_EmptyName 测试空的 Deployment 名称
func TestGetDeployment_EmptyName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &K8sResourceHandler{}

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "name", Value: ""},
	}
	c.Request = httptest.NewRequest("GET", "/k8s/clusters/1/deployments/", nil)

	// 执行请求
	handler.GetDeployment(c)

	// 验证结果
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Deployment名称不能为空")
}

// TestRestartDeployment_EmptyName 测试空的 Deployment 名称
func TestRestartDeployment_EmptyName(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &K8sResourceHandler{}

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "name", Value: ""},
	}
	c.Request = httptest.NewRequest("POST", "/k8s/clusters/1/deployments//restart?namespace=default", nil)

	// 执行请求
	handler.RestartDeployment(c)

	// 验证结果
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "Deployment名称不能为空")
}

// TestScaleDeployment_InvalidReplicas 测试无效的副本数
func TestScaleDeployment_InvalidReplicas(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &K8sResourceHandler{}

	testCases := []struct {
		name     string
		replicas int32
	}{
		{"负数副本", -1},
		{"超出范围", 101},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建测试请求
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{
				{Key: "id", Value: "1"},
				{Key: "name", Value: "app1"},
			}
			body := fmt.Sprintf(`{"replicas": %d}`, tc.replicas)
			c.Request = httptest.NewRequest("POST", "/k8s/clusters/1/deployments/app1/scale?namespace=default",
				strings.NewReader(body))
			c.Request.Header.Set("Content-Type", "application/json")

			// 执行请求
			handler.ScaleDeployment(c)

			// 验证结果
			assert.Equal(t, http.StatusBadRequest, w.Code)

			var response map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Contains(t, response["message"], "请求参数错误")
		})
	}
}

// TestScaleDeployment_MissingReplicas 测试缺少副本数参数
func TestScaleDeployment_MissingReplicas(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := &K8sResourceHandler{}

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{
		{Key: "id", Value: "1"},
		{Key: "name", Value: "app1"},
	}
	c.Request = httptest.NewRequest("POST", "/k8s/clusters/1/deployments/app1/scale?namespace=default",
		strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")

	// 执行请求
	handler.ScaleDeployment(c)

	// 验证结果
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["message"], "请求参数错误")
}
