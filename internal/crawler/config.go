package crawler

type Config struct {
	// 启动立即执行
	ExecOnStartup bool `json:"exec_on_startup,omitempty" mapstructure:"exec_on_startup"`
	// worker数量
	WorkerNum int `json:"worker_num,omitempty" mapstructure:"worker_num"`
	// ocr服务可执行文件路径
	OcrBin string `json:"ocr_bin,omitempty" mapstructure:"ocr_bin"`
}
