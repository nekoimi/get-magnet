package test

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

func TestRange(t *testing.T) {
	var ints []int

	ints = nil

	for i, i2 := range ints {
		t.Log(i, i2)
	}
}
