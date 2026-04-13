package handler

import (
	"devops/internal/service/kubernetes"
	"devops/pkg/response"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type K8sDeploymentHandler struct {
	service *kubernetes.K8sDeploymentService
}

func NewK8sDeploymentHandler(svc *kubernetes.K8sDeploymentService) *K8sDeploymentHandler {
	return &K8sDeploymentHandler{service: svc}
}

// ListDeployments 获取 Deployment 列表
// @Summary 获取 Deployment 列表
// @Tags K8s Deployment
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/deployments [get]
func (h *K8sDeploymentHandler) ListDeployments(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")

	deployments, err := h.service.ListDeployments(c.Request.Context(), uint(clusterID), namespace)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, deployments)
}

// GetDeployment 获取 Deployment 详情
// @Summary 获取 Deployment 详情
// @Tags K8s Deployment
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Deployment名称"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/deployments/{name} [get]
func (h *K8sDeploymentHandler) GetDeployment(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")

	deployment, err := h.service.GetDeployment(c.Request.Context(), uint(clusterID), namespace, name)
	if err != nil {
		response.Error(c, http.StatusNotFound, err.Error())
		return
	}

	response.Success(c, deployment)
}

// UpdateImage 更新 Deployment 镜像
// @Summary 更新 Deployment 镜像
// @Tags K8s Deployment
// @Accept json
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Deployment名称"
// @Param body body object true "镜像信息"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/deployments/{name}/image [put]
func (h *K8sDeploymentHandler) UpdateImage(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")

	var req struct {
		Container string `json:"container" binding:"required"`
		Image     string `json:"image" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.service.UpdateImage(c.Request.Context(), uint(clusterID), namespace, name, req.Container, req.Image); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// Scale 扩缩容
// @Summary 扩缩容 Deployment
// @Tags K8s Deployment
// @Accept json
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Deployment名称"
// @Param body body object true "副本数"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/deployments/{name}/scale [put]
func (h *K8sDeploymentHandler) Scale(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")

	var req struct {
		Replicas int32 `json:"replicas"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.service.Scale(c.Request.Context(), uint(clusterID), namespace, name, req.Replicas); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// Restart 重启 Deployment
// @Summary 重启 Deployment
// @Tags K8s Deployment
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Deployment名称"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/deployments/{name}/restart [post]
func (h *K8sDeploymentHandler) Restart(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")

	if err := h.service.Restart(c.Request.Context(), uint(clusterID), namespace, name); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetRevisionHistory 获取版本历史
// @Summary 获取 Deployment 版本历史
// @Tags K8s Deployment
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Deployment名称"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/deployments/{name}/revisions [get]
func (h *K8sDeploymentHandler) GetRevisionHistory(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")

	revisions, err := h.service.GetRevisionHistory(c.Request.Context(), uint(clusterID), namespace, name)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, revisions)
}

// Rollback 回滚
// @Summary 回滚 Deployment
// @Tags K8s Deployment
// @Accept json
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Deployment名称"
// @Param body body object true "版本号"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/deployments/{name}/rollback [post]
func (h *K8sDeploymentHandler) Rollback(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")

	var req struct {
		Revision int64 `json:"revision" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "参数错误: "+err.Error())
		return
	}

	if err := h.service.Rollback(c.Request.Context(), uint(clusterID), namespace, name, req.Revision); err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, nil)
}

// GetUpdateProgress 获取更新进度
// @Summary 获取 Deployment 更新进度
// @Tags K8s Deployment
// @Param id path int true "集群ID"
// @Param ns path string true "命名空间"
// @Param name path string true "Deployment名称"
// @Success 200 {object} response.Response
// @Router /api/v1/k8s/clusters/{id}/namespaces/{ns}/deployments/{name}/progress [get]
func (h *K8sDeploymentHandler) GetUpdateProgress(c *gin.Context) {
	clusterID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	namespace := c.Param("ns")
	name := c.Param("name")

	progress, err := h.service.GetUpdateProgress(c.Request.Context(), uint(clusterID), namespace, name)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	response.Success(c, progress)
}
