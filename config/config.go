package config

import (
	xormLog "xorm.io/xorm/log"
)

type Config struct {
	// http服务端口
	Port int

	// 数据库配置
	DB *Database
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

func Default() *Config {
	cfg := new(Config)
	cfg.Port = 8080
	cfg.DB = &Database{
		Dns:            "",
		ShowSQL:        true,
		LogLevel:       xormLog.LOG_DEBUG,
		MaxOpenConnNum: 16,
		MaxIdleConnNum: 8,
	}
	return cfg
}
