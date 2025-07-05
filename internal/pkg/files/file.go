package files

import (
	"errors"
	"github.com/google/uuid"
	"io"
	"os"
	"strings"
)

var videoSuffixArr = []string{".avi", ".flv", ".m4v", ".mkv", ".mov", ".mp4", ".mpeg", ".mpg", ".wmv"}

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

// IsVideo check file is video
// *.avi;*.flv;*.m4v;*.mkv;*.mov;*.mp4;*.mpeg;*.mpg;*.wmv
func IsVideo(filename string) bool {
	for _, suffix := range videoSuffixArr {
		if strings.HasSuffix(filename, suffix) {
			return true
		}
	}
	return false
}

// TempFile 封装创建一个带扩展名的临时文件，自动在系统临时目录下创建
func TempFile(ext string) (file *os.File, path string, cleanup func(), err error) {
	prefix := uuid.NewString()

	// 确保扩展名带点
	if !strings.HasPrefix(ext, ".") {
		ext = "." + ext
	}

	// 临时目录
	dir := os.TempDir()

	// 创建一个临时文件名（无扩展名）
	tmpFile, err := os.CreateTemp(dir, prefix+"-*")
	if err != nil {
		return nil, "", nil, err
	}

	// 重命名为带扩展名的文件
	tmpPathWithExt := tmpFile.Name() + ext
	tmpFile.Close() // 先关闭临时文件
	err = os.Rename(tmpFile.Name(), tmpPathWithExt)
	if err != nil {
		return nil, "", nil, err
	}

	// 重新打开为写入模式
	f, err := os.OpenFile(tmpPathWithExt, os.O_RDWR, 0600)
	if err != nil {
		return nil, "", nil, err
	}

	// 返回清理函数
	cleanup = func() {
		f.Close()
		os.Remove(tmpPathWithExt)
	}

	return f, tmpPathWithExt, cleanup, nil
}
