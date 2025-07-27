package crawler

type Crawler interface {
	// Name 唯一名称
	Name() string

	// CronSpec 定时表达式
	CronSpec() string

	// Run 执行任务
	Run()
}
