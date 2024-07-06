package aria2

import (
	"github.com/nekoimi/arigo"
	"github.com/nekoimi/get-magnet/pkg/util"
	"testing"
)

func TestSort(t *testing.T) {
	var files []arigo.File
	files = append(files, arigo.File{
		Length: 636,
	})
	files = append(files, arigo.File{
		Length: 3,
	})
	files = append(files, arigo.File{
		Length: 64536456,
	})
	files = append(files, arigo.File{
		Length: 543534,
	})
	files = append(files, arigo.File{
		Length: 3123,
	})
	files = append(files, arigo.File{
		Length: 34,
	})
	files = append(files, arigo.File{
		Length: 0,
	})

	t.Log("==================================================")
	for _, file := range files {
		t.Log(file)
	}
	t.Log("==================================================")

	util.Sort(files, func(a, b *arigo.File) bool {
		return a.Length > b.Length
	})

	t.Log("==================================================")
	for _, file := range files {
		t.Log(file)
	}
	t.Log("==================================================")
}
