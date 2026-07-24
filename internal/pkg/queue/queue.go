package queue

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type Queue[T any] struct {
	name     string
	items    []T
	mux      *sync.Mutex
	capacity int
	notify   chan struct{}
}

// NewQueue 创建一个新队列
func NewQueue[T any](name string, capacity int) *Queue[T] {
	q := new(Queue[T])
	q.name = name
	q.items = make([]T, 0)
	q.mux = &sync.Mutex{}
	q.capacity = capacity
	q.notify = make(chan struct{}, 1)
	return q
}

// Push 添加元素到队尾
func (q *Queue[T]) Push(item T) {
	for {
		q.mux.Lock()
		if q.capacity < 0 || len(q.items) < q.capacity {
			q.items = append(q.items, item)
			q.mux.Unlock()
			q.signal()
			return
		}
		q.mux.Unlock()
		<-time.After(10 * time.Millisecond)
	}
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
	q.signal()
	return item, true
}

// PopWaitTimeout 从队头获取一个元素，如果队列为空，则阻塞等待一定的超时时间
func (q *Queue[T]) PopWaitTimeout(timeout time.Duration) (T, bool) {
	var empty T
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		if item, ok := q.Pop(); ok {
			log.Debugf("[%s] queue-size: %d", q.name, q.Len())
			return item, true
		}
		select {
		case <-q.notify:
		case <-timer.C:
			return empty, false
		}
	}
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

func (q *Queue[T]) signal() {
	select {
	case q.notify <- struct{}{}:
	default:
	}
}
