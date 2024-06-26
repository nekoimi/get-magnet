package aria2

import (
	"github.com/siku2/arigo"
	"sort"
)

// SortBy 自定义排序方法
// a < b => asc	  升序
// a > b => desc  降序
type SortBy func(a, b *arigo.File) bool

// fileWrapper impl sort.Interface
type fileWrapper struct {
	files  []arigo.File
	sortBy SortBy
}

func (fw *fileWrapper) Len() int {
	return len(fw.files)
}

func (fw *fileWrapper) Less(i, j int) bool {
	return fw.sortBy(&fw.files[i], &fw.files[j])
}

func (fw *fileWrapper) Swap(i, j int) {
	fw.files[i], fw.files[j] = fw.files[j], fw.files[i]
}

func Sort(files []arigo.File, sortBy SortBy) {
	sort.Sort(&fileWrapper{
		files:  files,
		sortBy: sortBy,
	})
}
