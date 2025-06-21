package crawler

// WorkerTaskHandler 任务处理器
type WorkerTaskHandler interface {
	// Handle worker任务处理器
	Handle(t WorkerTask) ([]WorkerTask, []Magnet, error)
}
