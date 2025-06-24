package singleton

import "sync"

type singleton[T any] struct {
	once sync.Once
	inst T
	init func() T
}

func New[T any](initFunc func() T) *singleton[T] {
	return &singleton[T]{
		once: sync.Once{},
		init: initFunc,
	}
}

func (s *singleton[T]) Get() T {
	s.once.Do(func() {
		s.inst = s.init()
	})
	return s.inst
}
