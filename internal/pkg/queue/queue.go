package queue

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Queue[T any] struct {
	name     string
	items    []T
	mux      *sync.Mutex
	capacity int
}

// NewQueue 创建一个新队列
func NewQueue[T any](name string, capacity int) *Queue[T] {
	q := new(Queue[T])
	q.name = name
	q.items = make([]T, 0)
	q.mux = &sync.Mutex{}
	q.capacity = capacity
	return q
}

// Push 添加元素到队尾
func (q *Queue[T]) Push(item T) {
	q.mux.Lock()
	defer q.mux.Unlock()

	// 检查容量限制
	if q.capacity >= 0 {
		for len(q.items) >= q.capacity {
			// 等待一会儿 继续检查容量
			<-time.After(1 * time.Second)
		}
	}

	q.items = append(q.items, item)
}

// Pop 从队头获取一个元素, 返回元素和元素存在状态
// 如果队列为空, 元素存在状态为false
func (q *Queue[T]) Pop() (T, bool) {
	q.mux.Lock()
	defer q.mux.Unlock()

	var empty T
	if len(q.items) == 0 {
		return empty, false
	}

	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

// PopWaitTimeout 从队头获取一个元素，如果队列为空，则阻塞等待一定的超时时间
func (q *Queue[T]) PopWaitTimeout(timeout time.Duration) (T, bool) {
	q.mux.Lock()
	timeoutCh := time.After(timeout)
	checkTicker := time.NewTicker(1 * time.Second)
	defer func() {
		checkTicker.Stop()
		q.mux.Unlock()
	}()

	var empty T

	for len(q.items) == 0 {
		select {
		case <-checkTicker.C:
			// 继续检查元素
		case <-timeoutCh:
			return empty, false
		}
	}

	item := q.items[0]
	q.items = q.items[1:]
	log.Debugf("[%s] queue-size: %d", q.name, len(q.items))
	return item, true
}

// Len 获取队列中元素的数量
func (q *Queue[T]) Len() int {
	q.mux.Lock()
	defer q.mux.Unlock()

	return len(q.items)
}

// IsEmpty 判断队列是否为空
func (q *Queue[T]) IsEmpty() bool {
	return q.Len() == 0
}
