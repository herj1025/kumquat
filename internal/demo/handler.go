package demo

import (
	"github.com/herj1025/kumquat/pkg/logger"
	"github.com/herj1025/kumquat/pkg/response"

	"github.com/gin-gonic/gin"
)

type handler struct {
	svc *service
}

func newHandler(service *service) *handler {
	return &handler{
		svc: service,
	}
}

func (h *handler) Get(c *gin.Context) {

	// 打印日志，观察是否能自动带出 TraceID, UserID, Accept-Language
	logger.C(c).Info("Processing FindById request")

	// 将 gin.Context (实现了 context.Context) 传递给 Service
	data, err := h.svc.FindById(c)
	if err != nil {
		response.GinError(c, err)
		return
	}
	response.GinSuccess(c, data)
}

// RunTask 处理分布式任务请求
func (h *handler) RunTask(c *gin.Context) {
	logger.C(c).Info("Processing RunTask request")

	data, err := h.svc.RunDistributedTask(c)
	if err != nil {
		response.GinError(c, err)
		return
	}
	response.GinSuccess(c, gin.H{"message": "Task executed successfully", "data": data})
}
