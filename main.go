package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"go-vue-admin/core"
	"go-vue-admin/docs"
	"go-vue-admin/flag"
	"go-vue-admin/global"
	"go-vue-admin/middleware"
	"go-vue-admin/router/v1"
	servicesv1 "go-vue-admin/services/v1"

	"github.com/gin-gonic/gin"
)

func main() {
	opt := flag.Parse()

	// 初始化配置（必须在数据库初始化之前）
	core.InitConf("./setting.yaml")

	core.InitLogrus()
	global.Log.Info("日志初始化成功")

	// 配置安全检查（弱密钥/弱密码告警，release模式下命中弱JWT密钥将拒绝启动）
	core.CheckConfigSecurity()

	core.InitValidator()
	global.Log.Info("验证器初始化成功")

	global.DB = core.InitGorm()
	if global.DB == nil {
		global.Log.Fatal("数据库连接失败，请检查配置文件中的数据库配置")
	}
	global.Log.Info("数据库连接成功")

	// 初始化token黑名单表（失败即退出，黑名单不可用时不应继续运行）
	if err := middleware.InitTokenBlacklistTable(); err != nil {
		global.Log.Fatalf("初始化token黑名单表失败: %v", err)
	}

	// 鉴权组件不可用时继续运行会导致中间件nil指针panic，必须fail-fast
	global.Casbin = core.InitCasbin()
	if global.Casbin == nil {
		global.Log.Fatal("Casbin权限管理初始化失败，服务终止启动，请检查数据库连接及casbin配置")
	}
	global.Log.Info("Casbin权限管理初始化成功")

	// 放在数据库初始化之后，这样数据库操作才能正常执行
	if opt.ResetDB {
		flag.ResetDB()
		os.Exit(0)
	}

	if opt.DB {
		flag.MigrateDB()
		os.Exit(0)
	}

	if opt.Help {
		flag.Usage()
		os.Exit(0)
	}

	if opt.ResetPwd {
		flag.ResetAdminPassword()
		os.Exit(0)
	}

	if global.Config.System.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// 配置受信任的代理（生产环境应配置具体IP，开发环境设为nil禁用警告）
	// 参考: https://pkg.go.dev/github.com/gin-gonic/gin#readme-don-t-trust-all-proxies
	if global.Config.System.Mode == "release" {
		// 生产环境：配置你的反向代理服务器IP或CIDR
		r.SetTrustedProxies([]string{"127.0.0.1", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"})
	} else {
		// 开发环境：禁用代理信任（消除警告）
		r.SetTrustedProxies(nil)
	}

	docs.InitSwagger(r)
	global.Log.Infof("Swagger 文档初始化成功，访问地址: http://localhost:%d/swagger/index.html", global.Config.System.Addr)

	v1.InitRouter(r)
	global.Log.Info("路由初始化成功")

	// 定期清理过期token黑名单并清扫过期刷新锁（每天一次）
	cleanupCtx, stopCleanup := context.WithCancel(context.Background())
	defer stopCleanup()
	go middleware.StartTokenBlacklistCleanup(cleanupCtx, 24*time.Hour)

	addr := fmt.Sprintf(":%d", global.Config.System.Addr)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	// 在单独协程中监听，主协程等待退出信号
	go func() {
		global.Log.Infof("服务器启动成功，监听地址: %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			global.Log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号，优雅停机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	global.Log.Info("收到退出信号，正在优雅停机...")

	// 给在途请求最多10秒完成
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		global.Log.Errorf("服务器优雅停机失败: %v", err)
	}

	// 等待日志队列消费完毕，避免进程退出时丢失日志
	servicesv1.WaitLoginLogsFlushed(3 * time.Second)
	middleware.WaitOperationLogsFlushed(3 * time.Second)

	global.Log.Info("服务器已安全退出")
}
