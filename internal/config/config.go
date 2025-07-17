package config

import (
	"github.com/nekoimi/get-magnet/internal/pkg/apptools"
	log "github.com/sirupsen/logrus"
	"os"
	"runtime"
	"strings"
)

type Config struct {
	// http服务端口
	Port int
	// Jwt secret
	JwtSecret string
	// worker数量
	WorkerNum int
	// OCR启动路径
	OcrBin string
	// Rod启动路径
	RodBin string
	// Rod调试模式
	RodHeadless bool
	// Rod浏览器数据存储目录
	RodDataDir string
	// 数据库配置
	DB *Database
	// aria2参数
	Aria2Ops *Aria2Ops
	// javdb 账号
	JavDBAuth *Auth
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

type Auth struct {
	// 账号
	Username string
	// 密码
	Password string
}

const PackageName = "github.com/nekoimi/get-magnet"
const UIDir = "/workspace/ui"
const UIAriaNgDir = "/workspace/ui/aria-ng"
const OcrPort = 9898

var cfg *Config

func init() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:            true,
		FullTimestamp:          true,
		TimestampFormat:        "2006-01-02 15:04:05",
		DisableLevelTruncation: true,
		PadLevelText:           true,
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			return strings.ReplaceAll(frame.Function, PackageName+"/", " "), ""
		},
	})
	log.SetOutput(os.Stdout)
	log.SetReportCaller(true)

	logLevel := apptools.Getenv("LOG_LEVEL", "debug")
	level, err := log.ParseLevel(logLevel)
	if err != nil {
		panic(err)
	}
	log.SetLevel(level)

	// 文件输出：不带颜色
	fileFormatter := &log.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			return strings.ReplaceAll(frame.Function, PackageName+"/", " "), ""
		},
	}

	log.AddHook(NewLevelHook(apptools.Getenv("LOG_PATH", "logs"), fileFormatter))
}

func Default() *Config {
	cfg = new(Config)
	cfg.Port = 8080
	cfg.RodHeadless = true
	cfg.RodDataDir = "/var/lib/rod-data"
	cfg.DB = &Database{
		Dns:            "",
		MaxOpenConnNum: 16,
		MaxIdleConnNum: 8,
	}
	cfg.Aria2Ops = new(Aria2Ops)
	cfg.JavDBAuth = new(Auth)

	// 加载环境变量配置
	cfg.loadEnv()

	return cfg
}

func (c *Config) loadEnv() {
	cfg.Port = apptools.GetenvInt("PORT", 8093)
	cfg.JwtSecret = apptools.Getenv("JWT_SECRET", "get-magnet")
	cfg.WorkerNum = apptools.GetenvInt("WORKER_NUM", 4)
	cfg.OcrBin = apptools.Getenv("OCR_BIN_PATH", "")
	cfg.RodBin = apptools.Getenv("ROD_BROWSER_PATH", "")
	cfg.RodHeadless = apptools.GetenvBool("ROD_HEADLESS", true)
	cfg.RodDataDir = apptools.Getenv("ROD_DATA_DIR", "/var/lib/rod-data")
	cfg.DB.Dns = apptools.Getenv("DB_DSN", "")
	cfg.Aria2Ops.JsonRpc = apptools.Getenv("ARIA2_JSONRPC", "")
	cfg.Aria2Ops.Secret = apptools.Getenv("ARIA2_SECRET", "")
	cfg.JavDBAuth.Username = apptools.Getenv("JAVDB_USERNAME", "")
	cfg.JavDBAuth.Password = apptools.Getenv("JAVDB_PASSWORD", "")

	log.Debugln("加载配置完成")
}

func Get() *Config {
	return cfg
}
