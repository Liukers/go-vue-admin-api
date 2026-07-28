package core

import (
	"go-vue-admin/global"
)

// 已知弱JWT密钥黑名单，命中时release模式拒绝启动
var weakJWTKeys = []string{
	"go-vue-admin-secret-key",
	"please-change-me-use-openssl-rand-base64-48",
	"secret",
	"secret-key",
	"123456",
	"admin123",
}

// CheckConfigSecurity 配置安全检查
// debug模式：发现弱配置打印警告
// release模式：JWT使用已知弱密钥时直接拒绝启动
func CheckConfigSecurity() {
	cfg := global.Config
	if cfg == nil {
		return
	}

	releaseMode := cfg.System.Mode == "release"

	// JWT密钥检查
	key := cfg.JWT.SigningKey
	if key == "" {
		global.Log.Fatal("JWT signing-key 未配置，服务拒绝启动")
		return
	}
	for _, weak := range weakJWTKeys {
		if key == weak {
			if releaseMode {
				global.Log.Fatalf("JWT signing-key 使用了已知弱密钥[%s]，release模式拒绝启动，请更换为强随机密钥（如: openssl rand -base64 48）", weak)
				return
			}
			global.Log.Warnf("JWT signing-key 使用了已知弱密钥[%s]，仅供开发调试，部署前务必更换", weak)
			break
		}
	}
	if len(key) < 32 {
		global.Log.Warn("JWT signing-key 长度不足32字节，存在被暴力破解的风险，建议更换为更长的随机密钥")
	}

	// 数据库弱密码检查（本地开发库常见弱密码，仅警告）
	switch cfg.Mysql.Password {
	case "", "123456", "root", "password", "admin123":
		global.Log.Warn("数据库使用了弱密码或空密码，生产环境请务必修改")
	}
}
