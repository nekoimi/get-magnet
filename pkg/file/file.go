package file

import (
	"errors"
	"io"
	"os"
)

// Exists 判断文件是否存在
func Exists(f string) (bool, error) {
	if _, err := os.Stat(f); err == nil {
		// f exists
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		// f does not exists
		return false, nil
	} else {
		// f stat err, return false and err
		return false, err
	}
}

// WriteLine 写入一行文本
func WriteLine(w io.Writer, content string) error {
	_, err := io.WriteString(w, content)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, "\n")
	if err != nil {
		return err
	}
	return nil
}
