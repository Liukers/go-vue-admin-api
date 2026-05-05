package conf

import (
	"fmt"
	"net/url"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type Server struct {
	System System `mapstructure:"system" json:"system" yaml:"system"`
	Mysql  Mysql  `mapstructure:"mysql" json:"mysql" yaml:"mysql"`
	JWT    JWT    `mapstructure:"jwt" json:"jwt" yaml:"jwt"`
	Zap    Zap    `mapstructure:"zap" json:"zap" yaml:"zap"`
	Casbin Casbin `mapstructure:"casbin" json:"casbin" yaml:"casbin"`
	Cors   Cors   `mapstructure:"cors" json:"cors" yaml:"cors"`
}

type System struct {
	Addr         int    `mapstructure:"addr" json:"addr" yaml:"addr"`
	Mode         string `mapstructure:"mode" json:"mode" yaml:"mode"`
	DbType       string `mapstructure:"db-type" json:"db-type" yaml:"db-type"`
	CasbinConfig string `mapstructure:"casbin-config" json:"casbin-config" yaml:"casbin-config"`
	TrustProxy   bool   `mapstructure:"trust-proxy" json:"trust-proxy" yaml:"trust-proxy"`
}

type Mysql struct {
	Path         string `mapstructure:"path" json:"path" yaml:"path"`
	Port         string `mapstructure:"port" json:"port" yaml:"port"`
	Config       string `mapstructure:"config" json:"config" yaml:"config"`
	DbName       string `mapstructure:"db-name" json:"db-name" yaml:"db-name"`
	Username     string `mapstructure:"username" json:"username" yaml:"username"`
	Password     string `mapstructure:"password" json:"password" yaml:"password"`
	MaxIdleConns int    `mapstructure:"max-idle-conns" json:"max-idle-conns" yaml:"max-idle-conns"`
	MaxOpenConns int    `mapstructure:"max-open-conns" json:"max-open-conns" yaml:"max-open-conns"`
	LogMode      string `mapstructure:"log-mode" json:"log-mode" yaml:"log-mode"`
	LogZap       bool   `mapstructure:"log-zap" json:"log-zap" yaml:"log-zap"`
}

// Dsn 构建MySQL连接字符串（对用户名密码进行URL编码）
func (m *Mysql) Dsn() string {
	// 对用户名和密码进行URL编码，防止特殊字符导致DSN解析失败
	encodedUsername := url.QueryEscape(m.Username)
	encodedPassword := url.QueryEscape(m.Password)
	
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s",
		encodedUsername, encodedPassword, m.Path, m.Port, m.DbName, m.Config)
}

type JWT struct {
	SigningKey  string `mapstructure:"signing-key" json:"signing-key" yaml:"signing-key"`
	ExpiresTime int64  `mapstructure:"expires-time" json:"expires-time" yaml:"expires-time"`
	BufferTime  int64  `mapstructure:"buffer-time" json:"buffer-time" yaml:"buffer-time"`
	Issuer      string `mapstructure:"issuer" json:"issuer" yaml:"issuer"`
}

type Zap struct {
	Level         string `mapstructure:"level" json:"level" yaml:"level"`
	Format        string `mapstructure:"format" json:"format" yaml:"format"`
	Prefix        string `mapstructure:"prefix" json:"prefix" yaml:"prefix"`
	Director      string `mapstructure:"director" json:"director" yaml:"director"`
	ShowLine      bool   `mapstructure:"show-line" json:"show-line" yaml:"show-line"`
	EncodeLevel   string `mapstructure:"encode-level" json:"encode-level" yaml:"encode-level"`
	StacktraceKey string `mapstructure:"stacktrace-key" json:"stacktrace-key" yaml:"stacktrace-key"`
	LogInConsole  bool   `mapstructure:"log-in-console" json:"log-in-console" yaml:"log-in-console"`
}

type Casbin struct {
	ModelPath string `mapstructure:"model-path" json:"model-path" yaml:"model-path"`
}

type CORSWhitelist struct {
	AllowOrigin      string `mapstructure:"allow-origin" json:"allow-origin" yaml:"allow-origin"`
	AllowMethods     string `mapstructure:"allow-methods" json:"allow-methods" yaml:"allow-methods"`
	AllowHeaders     string `mapstructure:"allow-headers" json:"allow-headers" yaml:"allow-headers"`
	ExposeHeaders    string `mapstructure:"expose-headers" json:"expose-headers" yaml:"expose-headers"`
	AllowCredentials bool   `mapstructure:"allow-credentials" json:"allow-credentials" yaml:"allow-credentials"`
}

type Cors struct {
	Mode       string          `mapstructure:"mode" json:"mode" yaml:"mode"`
	Whitelist  []CORSWhitelist `mapstructure:"whitelist" json:"whitelist" yaml:"whitelist"`
	AllowAll   bool            `mapstructure:"allow-all" json:"allow-all" yaml:"allow-all"`
}

var ServerConfig = new(Server)

func InitConfig(path string) (*Server, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	if err := v.Unmarshal(ServerConfig); err != nil {
		return nil, err
	}

	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		fmt.Printf("配置文件发生变化: %s\n", e.Name)
		if err := v.Unmarshal(ServerConfig); err != nil {
			fmt.Printf("重新加载配置失败: %v\n", err)
		}
	})

	return ServerConfig, nil
}
