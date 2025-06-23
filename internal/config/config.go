package config

import (
	"github.com/nekoimi/get-magnet/internal/pkg/apptools"
	log "github.com/sirupsen/logrus"
	"os"
)

type Config struct {
	// http服务端口
	Port int
	// Jwt secret
	JwtSecret string
	// 日志级别
	WorkerNum int
	// 数据库配置
	DB *Database
	// aria2参数
	Aria2Ops *Aria2Ops
}

// Database 数据库相关配置
type Database struct {
	// 数据库连接配置
	Dns string

	// 连接池最大连接数
	MaxOpenConnNum int

	// 连接池最大空闲数
	MaxIdleConnNum int
}

type Aria2Ops struct {
	// jsonRpc
	JsonRpc string
	// 验证token
	Secret string
}

const UIDir = "/workspace/ui"
const UIAriaNgDir = "/workspace/ui/aria-ng"

var cfg *Config

func init() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
	log.SetOutput(os.Stdout)
	log.SetReportCaller(false)

	logLevel := apptools.Getenv("LOG_LEVEL", "debug")
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}
	log.SetLevel(level)
}

func Default() *Config {
	cfg = new(Config)
	cfg.Port = 8080
	cfg.DB = &Database{
		Dns:            "",
		MaxOpenConnNum: 16,
		MaxIdleConnNum: 8,
	}
	cfg.Aria2Ops = &Aria2Ops{
		JsonRpc: "",
		Secret:  "",
	}
	return cfg
}

func (c *Config) LoadFromEnv() {
	cfg.Port = apptools.GetenvInt("PORT", 8093)
	cfg.JwtSecret = apptools.Getenv("JWT_SECRET", "get-magnet")
	cfg.WorkerNum = apptools.GetenvInt("WORKER_NUM", 4)
	cfg.DB.Dns = apptools.Getenv("DB_DSN", "")
	cfg.Aria2Ops.JsonRpc = apptools.Getenv("ARIA2_JSONRPC", "")
	cfg.Aria2Ops.Secret = apptools.Getenv("ARIA2_SECRET", "")

	log.Debugln("加载配置完成")
}

func Get() *Config {
	return cfg
}
