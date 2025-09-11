package config

import (
	"github.com/nekoimi/get-magnet/internal/logger"
	"github.com/nekoimi/get-magnet/internal/pkg/apptools"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"strings"
)

type Config struct {
	// http服务端口
	Port int `json:"port,omitempty" mapstructure:"port"`
	// 日志等级
	LogLevel string `json:"log_level,omitempty" mapstructure:"log_level"`
	// 日志文件夹
	LogDir string `json:"log_dir,omitempty" mapstructure:"log_dir"`
	// Jwt secret
	JwtSecret string `json:"jwt_secret,omitempty" mapstructure:"jwt_secret"`
	// 无头浏览器配置
	Browser *BrowserConfig `json:"browser,omitempty" mapstructure:"browser"`
	// arai2下载配置
	Aria2 *Aria2Config `json:"aria2,omitempty" mapstructure:"aria2"`
	// 采集配置
	Crawler *CrawlerConfig `json:"crawler,omitempty" mapstructure:"crawler"`
	// JavDB 配置
	JavDB *JavDBConfig `json:"javdb,omitempty" mapstructure:"javdb"`
	// 数据库配置
	DB *DBConfig `json:"db,omitempty" mapstructure:"db"`
	// cloudflare pass
	CloudflarePassApi string `json:"cloudflare_pass_api,omitempty" mapstructure:"cloudflare_pass_api"`
}

type BrowserConfig struct {
	// Rod启动路径
	Bin string `json:"bin,omitempty" mapstructure:"bin"`
	// Rod调试模式
	Headless bool `json:"headless,omitempty" mapstructure:"headless"`
	// Rod浏览器数据存储目录
	DataDir string `json:"data_dir,omitempty" mapstructure:"data_dir"`
}

type Aria2Config struct {
	// jsonRpc
	JsonRpc string `json:"jsonrpc,omitempty" mapstructure:"jsonrpc"`
	// 验证token
	Secret string `json:"secret,omitempty" mapstructure:"secret"`
	// 移动文件夹
	MoveTo Aria2MoveToConfig `json:"move_to" mapstructure:"move_to"`
}

type Aria2MoveToConfig struct {
	// javdb 移动目录
	JavDBDir string `json:"javdb_dir,omitempty" mapstructure:"javdb_dir"`
}

type CrawlerConfig struct {
	// 启动立即执行
	ExecOnStartup bool `json:"exec_on_startup,omitempty" mapstructure:"exec_on_startup"`
	// worker数量
	WorkerNum int `json:"worker_num,omitempty" mapstructure:"worker_num"`
	// ocr服务可执行文件路径
	OcrBin string `json:"ocr_bin,omitempty" mapstructure:"ocr_bin"`
}

type JavDBConfig struct {
	// 账号
	Username string `json:"username,omitempty" mapstructure:"username"`
	// 密码
	Password string `json:"password,omitempty" mapstructure:"password"`
}

// DBConfig 数据库相关配置
type DBConfig struct {
	// 数据库连接配置
	Dsn string `json:"dsn,omitempty" mapstructure:"dsn"`
}

func Load() *Config {
	v := viper.New()
	v.SetDefault("port", 8093)
	v.SetDefault("log_level", "debug")
	v.SetDefault("log_dir", "logs")
	v.SetDefault("jwt_secret", "abc123456")
	v.SetDefault("browser.bin", apptools.Getenv("ROD_BROWSER_PATH", ""))
	v.SetDefault("browser.headless", true)
	v.SetDefault("browser.data_dir", apptools.Getenv("ROD_DATA_DIR", ""))
	v.SetDefault("crawler.exec_on_startup", false)
	v.SetDefault("crawler.worker_num", 4)
	v.SetDefault("crawler.ocr_bin", apptools.Getenv("OCR_BIN_PATH", ""))

	v.BindEnv("browser.bin")
	v.BindEnv("browser.headless")
	v.BindEnv("browser.data_dir")
	v.BindEnv("aria2.jsonrpc")
	v.BindEnv("aria2.secret")
	v.BindEnv("aria2.move_to.javdb_dir")
	v.BindEnv("crawler.exec_on_startup")
	v.BindEnv("crawler.worker_num")
	v.BindEnv("crawler.ocr_bin")
	v.BindEnv("javdb.username")
	v.BindEnv("javdb.password")
	v.BindEnv("db.dsn")
	v.BindEnv("cloudflare_pass_api")

	// 从环境变量自动映射配置
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	cfg := new(Config)
	if err := v.Unmarshal(cfg); err != nil {
		panic(err)
	}

	logger.Initialize(cfg.LogLevel, cfg.LogDir)
	log.Infof("配置信息：\n%s", cfg)

	return cfg
}

func (c *Config) String() string {
	return util.ToJson(c)
}
