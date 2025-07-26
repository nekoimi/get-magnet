package config

import (
	"github.com/nekoimi/get-magnet/internal/crawler"
	"github.com/nekoimi/get-magnet/internal/crawler/providers/javdb"
	"github.com/nekoimi/get-magnet/internal/db"
	"github.com/nekoimi/get-magnet/internal/downloader/aria2_downloader"
	"github.com/nekoimi/get-magnet/internal/pkg/apptools"
	"github.com/nekoimi/get-magnet/internal/pkg/rod_browser"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
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
	Browser *rod_browser.Config `json:"browser,omitempty" mapstructure:"browser"`
	// arai2下载配置
	Aria2 *aria2_downloader.Config `json:"aria2,omitempty" mapstructure:"aria2"`
	// 采集配置
	Crawler *crawler.Config `json:"crawler,omitempty" mapstructure:"crawler"`
	// JavDB 配置
	JavDB *javdb.Config `json:"javdb,omitempty" mapstructure:"javdb"`
	// 数据库配置
	DB *db.Config `json:"db,omitempty" mapstructure:"db"`
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

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	cfg := new(Config)
	if err := v.Unmarshal(cfg); err != nil {
		panic(err)
	}
	return cfg
}

func (c *Config) String() string {
	return util.ToJson(c)
}
