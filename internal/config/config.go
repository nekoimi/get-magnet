package config

import (
	xormLog "xorm.io/xorm/log"
)

type Config struct {
	// http服务端口
	Port int
	// Jwt secret
	JwtSecret string
	// 数据库配置
	DB *Database
	// aria2参数
	Aria2Ops *Aria2Ops
}

// Database 数据库相关配置
type Database struct {
	// 数据库连接配置
	Dns string

	// 是否打印SQL语句
	ShowSQL bool

	// 日志级别
	LogLevel xormLog.LogLevel

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

func Default() *Config {
	cfg = new(Config)
	cfg.Port = 8080
	cfg.DB = &Database{
		Dns:            "",
		ShowSQL:        true,
		LogLevel:       xormLog.LOG_DEBUG,
		MaxOpenConnNum: 16,
		MaxIdleConnNum: 8,
	}
	cfg.Aria2Ops = &Aria2Ops{
		JsonRpc: "",
		Secret:  "",
	}
	return cfg
}

func Get() *Config {
	return cfg
}
