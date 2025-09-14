package config

import (
	"github.com/nekoimi/get-magnet/internal/logger"
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
	// arai2下载配置
	Aria2 *Aria2Config `json:"aria2,omitempty" mapstructure:"aria2"`
	// 采集配置
	Crawler *CrawlerConfig `json:"crawler,omitempty" mapstructure:"crawler"`
	// 数据库配置
	DB *DBConfig `json:"db,omitempty" mapstructure:"db"`
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

	// DrissionRod 设置
	DrissionRodGrpcIp   string `json:"drission_rod_grpc_ip,omitempty" mapstructure:"drission_rod_grpc_ip"`
	DrissionRodGrpcPort int    `json:"drission_rod_grpc_port,omitempty" mapstructure:"drission_rod_grpc_port"`
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
	v.SetDefault("crawler.exec_on_startup", false)
	v.SetDefault("crawler.worker_num", 4)

	v.BindEnv("aria2.jsonrpc")
	v.BindEnv("aria2.secret")
	v.BindEnv("aria2.move_to.javdb_dir")
	v.BindEnv("crawler.exec_on_startup")
	v.BindEnv("crawler.worker_num")
	v.BindEnv("crawler.drission_rod_grpc_ip")
	v.BindEnv("crawler.drission_rod_grpc_port")
	v.BindEnv("db.dsn")

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
