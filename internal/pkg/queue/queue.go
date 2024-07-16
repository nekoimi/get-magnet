package queue

import "sync"

type Queue[T any] struct {
	mux   sync.Mutex
	cond  *sync.Cond
	items []T
}

// New 创建一个新队列
func New[T any]() *Queue[T] {
	q := new(Queue[T])
	q.cond = sync.NewCond(&q.mux)
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

	var zero T
	if len(q.items) == 0 {
		return zero, false
	}

	item := q.items[0]
	q.items = q.items[1:]
	return item, true
}

// PollWait 从队头获取一个元素，如果队列为空，则阻塞等待
func (q *Queue[T]) PollWait() T {
	q.mux.Lock()
	defer q.mux.Unlock()

	for len(q.items) == 0 {
		q.cond.Wait()
	}

	item := q.items[0]
	q.items = q.items[1:]
	return item
}

// Size 获取队列中元素的数量
func (q *Queue[T]) Size() int {
	q.mux.Lock()
	defer q.mux.Unlock()

	return len(q.items)
}

// IsEmpty 判断队列是否为空
func (q *Queue[T]) IsEmpty() bool {
	return q.Size() <= 0
}
