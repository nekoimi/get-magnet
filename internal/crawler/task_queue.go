package crawler

type TaskQueue struct {
	queue chan CrawlerTask
}

func NewCrawlerTaskQueue(size int) *TaskQueue {
	return &TaskQueue{queue: make(chan CrawlerTask, size)}
}

func (q *TaskQueue) Submit(t CrawlerTask) {
	q.queue <- t
}

func (q *TaskQueue) Chan() <-chan CrawlerTask {
	return q.queue
}
