package resilience

import (
	"devops/internal/config"
	"devops/pkg/ioc"

	"github.com/gin-gonic/gin"
)

func init() {
	ioc.Api.RegisterContainer("ResilienceHandler", &ResilienceApiHandler{})
}

// ResilienceApiHandler IOC容器注册的处理器
type ResilienceApiHandler struct {
	handler *Handler
}

func (h *ResilienceApiHandler) Init() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}

	h.handler = NewHandler()

	root := cfg.Application.GinRootRouter().(*gin.RouterGroup)
	h.handler.RegisterRoutes(root)

	return nil
}
