package v1

import (
	"go-vue-admin/middleware"

	"github.com/gin-gonic/gin"
)

// InitRouter 初始化路由
func InitRouter(r *gin.Engine) *gin.RouterGroup {
	r.Use(middleware.Cors())

	// 健康检查（无需鉴权，供负载均衡/容器探活使用）
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	apiV1 := r.Group("/api/v1")
	{
		InitSystemRouter(apiV1)
	}

	return apiV1
}
