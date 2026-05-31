package config

import (
	"os"
	"strings"

	"github.com/nekoimi/get-magnet/internal/logger"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
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
	// 网盘驱动中间服务配置
	CloudDriver *CloudDriverConfig `json:"cloud_driver,omitempty" mapstructure:"cloud_driver"`
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

type CloudDriverConfig struct {
	// 中间服务地址
	BaseURL string `json:"base_url,omitempty" mapstructure:"base_url"`
	// 网盘平台
	Platform string `json:"platform,omitempty" mapstructure:"platform"`
	// 浏览器 Profile ID
	ProfileID string `json:"profile_id,omitempty" mapstructure:"profile_id"`
	// 网盘保存根目录
	SaveRoot string `json:"save_root,omitempty" mapstructure:"save_root"`
	// HTTP 超时时间，单位秒
	Timeout int `json:"timeout,omitempty" mapstructure:"timeout"`
	// 轮询未完成任务的 cron 表达式
	PollCron string `json:"poll_cron,omitempty" mapstructure:"poll_cron"`
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
	v.SetDefault("cloud_driver.platform", "115")
	v.SetDefault("cloud_driver.save_root", "/get-magnet")
	v.SetDefault("cloud_driver.timeout", 30)
	v.SetDefault("cloud_driver.poll_cron", "*/10 * * * *")
	v.SetDefault("crawler.exec_on_startup", false)
	v.SetDefault("crawler.worker_num", 4)

	// 加载 YAML 配置文件
	loadYamlFile(v)

	v.BindEnv("aria2.jsonrpc")
	v.BindEnv("aria2.secret")
	v.BindEnv("aria2.move_to.javdb_dir")
	v.BindEnv("cloud_driver.base_url")
	v.BindEnv("cloud_driver.platform")
	v.BindEnv("cloud_driver.profile_id")
	v.BindEnv("cloud_driver.save_root")
	v.BindEnv("cloud_driver.timeout")
	v.BindEnv("cloud_driver.poll_cron")
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

// loadYamlFile 加载环境特定的 YAML 配置文件
// 优先级：CONFIG_FILE 环境变量（指定完整路径）> config/{APP_ENV}.yaml
// 配置文件不存在不是错误，仅格式错误会输出警告
func loadYamlFile(v *viper.Viper) {
	if configFile := os.Getenv("CONFIG_FILE"); configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		env := os.Getenv("APP_ENV")
		if env == "" {
			env = "dev"
		}
		v.SetConfigName(env)
		v.SetConfigType("yaml")
		v.AddConfigPath("config")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Warnf("读取配置文件异常: %s", err.Error())
		}
	} else {
		log.Infof("已加载配置文件: %s", v.ConfigFileUsed())
	}
}

func (c *Config) String() string {
	return util.ToJson(c)
}
