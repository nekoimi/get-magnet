package queue

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Queue[T any] struct {
	mux     *sync.RWMutex
	name    string
	cond    *sync.Cond
	items   []T
	opCount uint64
}

// New 创建一个新队列
func New[T any](name string) *Queue[T] {
	q := new(Queue[T])
	q.mux = &sync.RWMutex{}
	q.name = name
	q.cond = sync.NewCond(q.mux)
	q.items = make([]T, 0)
	q.opCount = 0
	return q
}

// Add 添加元素到队尾
func (q *Queue[T]) Add(item T) {
	q.mux.Lock()
	defer q.mux.Unlock()

	q.items = append(q.items, item)
	q.cond.Signal()
}

// Poll 从队头获取一个元素, 返回元素和元素存在状态
// 如果队列为空, 元素存在状态为false
func (q *Queue[T]) Poll() (T, bool) {
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

// PollWaitTimeout 从队头获取一个元素，如果队列为空，则阻塞等待
func (q *Queue[T]) PollWaitTimeout(timeout time.Duration) (T, bool) {
	var empty T
	timer := time.NewTimer(timeout)

	q.mux.Lock()
	q.opCount = q.opCount + 1

	log.Debugf("[%s-%d]init-lock\n", q.name, q.opCount)
	defer func() {
		timer.Stop()
		q.mux.Unlock()
		log.Debugf("[%s-%d]defer-unlock\n", q.name, q.opCount)
	}()

	for len(q.items) == 0 {
		waitCh := make(chan struct{})
		go func() {
			log.Debugf("[%s-%d]wait-unlock\n", q.name, q.opCount)
			q.cond.Wait()
			log.Debugf("[%s-%d]wait-lock\n", q.name, q.opCount)
			close(waitCh)
		}()

		select {
		case <-waitCh:
			log.Debugf("[%s-%d]waitCh\n", q.name, q.opCount)
			continue

		case <-timer.C:
			if q.mux.TryLock() {
				log.Debugf("[%s-%d]timeout-lock\n", q.name, q.opCount)
			} else {
				log.Debugf("[%s-%d]timeout-wait\n", q.name, q.opCount)
			}
			return empty, false
		}
	}

	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

// Size 获取队列中元素的数量
func (q *Queue[T]) Size() int {
	q.mux.RLock()
	defer q.mux.RUnlock()

	return len(q.items)
}

// IsEmpty 判断队列是否为空
func (q *Queue[T]) IsEmpty() bool {
	return q.Size() <= 0
}
