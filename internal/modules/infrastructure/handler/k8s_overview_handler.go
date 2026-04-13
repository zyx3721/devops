package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"devops/internal/config"
	"devops/internal/service/kubernetes"
	apperrors "devops/pkg/errors"
	"devops/pkg/ioc"
	"devops/pkg/middleware"
)

func init() {
	ioc.Api.RegisterContainer("K8sOverviewHandler", &K8sOverviewApiHandler{})
}

type K8sOverviewApiHandler struct {
	handler *K8sOverviewHandler
}

func (h *K8sOverviewApiHandler) Init() error {
	cfg, _ := config.LoadConfig()
	db := cfg.GetDB()
	clientMgr := kubernetes.NewK8sClientManager(db)
	svc := kubernetes.NewK8sOverviewService(clientMgr, db)
	h.handler = NewK8sOverviewHandler(svc)

	root := cfg.Application.GinRootRouter().Group("k8s")
	root.Use(middleware.AuthMiddleware())
	h.Register(root)

	return nil
}

func (h *K8sOverviewApiHandler) Register(r gin.IRouter) {
	r.GET("/overview", h.handler.GetMultiClusterOverview)
	r.GET("/clusters/:id/overview", h.handler.GetClusterOverview)
}

type K8sOverviewHandler struct {
	svc *kubernetes.K8sOverviewService
}

func NewK8sOverviewHandler(svc *kubernetes.K8sOverviewService) *K8sOverviewHandler {
	return &K8sOverviewHandler{svc: svc}
}

// GetClusterOverview 获取单个集群概览
func (h *K8sOverviewHandler) GetClusterOverview(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": apperrors.ErrCodeInvalidParams, "message": "集群ID格式错误"})
		return
	}

	result, err := h.svc.GetClusterOverview(c.Request.Context(), uint(clusterID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取集群概览失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}

// GetMultiClusterOverview 获取多集群概览
func (h *K8sOverviewHandler) GetMultiClusterOverview(c *gin.Context) {
	result, err := h.svc.GetMultiClusterOverview(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": apperrors.ErrCodeInternalError, "message": "获取多集群概览失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": apperrors.Success, "message": "success", "data": result})
}
