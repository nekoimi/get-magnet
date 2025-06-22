package util

import "sort"

// SortBy 自定义排序方法
// a < b => asc	  升序
// a > b => desc  降序
type SortBy[T any] func(a T, b T) bool

// wrapper impl sort.Interface
type sortWrapper[T any] struct {
	objects []T
	sortBy  SortBy[T]
}

func (sw *sortWrapper[T]) Len() int {
	return len(sw.objects)
}

func (sw *sortWrapper[T]) Less(i int, j int) bool {
	return sw.sortBy(sw.objects[i], sw.objects[j])
}

func (sw *sortWrapper[T]) Swap(i, j int) {
	sw.objects[i], sw.objects[j] = sw.objects[j], sw.objects[i]
}

func Sort[T any](objects []T, sortBy SortBy[T]) {
	sort.Sort(&sortWrapper[T]{
		objects: objects,
		sortBy:  sortBy,
	})
}
