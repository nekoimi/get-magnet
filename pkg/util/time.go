package util

import (
	"fmt"
	"time"
)

// NowDate default with time.DateOnly -> 2006-01-02
func NowDate(split string) string {
	if len(split) == 0 {
		return time.Now().Format(time.DateOnly)
	}
	return time.Now().Format(fmt.Sprintf("2006%s01%s02", split, split))
}
