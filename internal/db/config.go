package db

// Config 数据库相关配置
type Config struct {
	// 数据库连接配置
	Dns string `json:"dns,omitempty" mapstructure:"dns"`
}
