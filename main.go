package main

import (
	"fmt"
	"os"
	"go-vue-admin/core"
	"go-vue-admin/docs"
	"go-vue-admin/flag"
	"go-vue-admin/global"
	"go-vue-admin/router/v1"

	"github.com/gin-gonic/gin"
)

func main() {
	// 解析命令行参数
	opt := flag.Parse()

	// 初始化配置（必须在数据库初始化之前）
	core.InitConf("./setting.yaml")

	// 初始化日志
	core.InitLogrus()
	global.Log.Info("日志初始化成功")

	// 初始化自定义验证器
	core.InitValidator()
	global.Log.Info("验证器初始化成功")

	// 初始化数据库
	global.DB = core.InitGorm()
	if global.DB == nil {
		global.Log.Fatal("数据库连接失败，请检查配置文件中的数据库配置")
		os.Exit(1)
	}
	global.Log.Info("数据库连接成功")

	// 初始化Casbin权限管理
	global.Casbin = core.InitCasbin()
	if global.Casbin == nil {
		global.Log.Warn("Casbin权限管理初始化失败，将继续运行但权限控制将不可用")
	} else {
		global.Log.Info("Casbin权限管理初始化成功")
	}

	// 执行命令行操作（如 -db 初始化数据库）
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

	// 设置运行模式
	if global.Config.System.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建Gin引擎
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

	// 初始化 Swagger 文档
	docs.InitSwagger(r)
	global.Log.Info("Swagger 文档初始化成功，访问地址: http://localhost:8080/swagger/index.html")

	// 初始化路由
	v1.InitRouter(r)
	global.Log.Info("路由初始化成功")

	// 初始化操作日志表
	initOperationLogTable()

	// 启动服务
	addr := fmt.Sprintf(":%d", global.Config.System.Addr)
	global.Log.Infof("服务器启动成功，监听地址: %s", addr)
	
	if err := r.Run(addr); err != nil {
		global.Log.Fatalf("服务器启动失败: %v", err)
		os.Exit(1)
	}
}

// initOperationLogTable 初始化操作日志表
func initOperationLogTable() {
	if global.DB == nil {
		return
	}
	
	global.Log.Info("操作日志表检查完成")
}
