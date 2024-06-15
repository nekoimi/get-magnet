package main

import (
	"fmt"
	"testing"
)

func TestRun(t *testing.T) {
	for i := range [10]int{} {
		visitUrl := fmt.Sprintf("https://javdb.com/censored?page=%d&vft=2&vst=2", i+1)
		t.Log(visitUrl)
	}
}
