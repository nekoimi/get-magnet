package file_storage

import (
	"fmt"
	"log"
	"sync"
	"testing"
)

func TestOutputFile(t *testing.T) {
	outputFilename := OutputFile()
	t.Log(outputFilename)
}

func TestFileStorage_Save(t *testing.T) {
	s := New("test")

	wg := sync.WaitGroup{}
	wg.Add(5)

	for i := range [5]int{} {
		index := i
		go func() {
			for ci := range [100]int{} {
				if err := s.Save(fmt.Sprintf("go-%d-w-%d", index, ci)); err != nil {
					log.Println(err.Error())
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()
}
