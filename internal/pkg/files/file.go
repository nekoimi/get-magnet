package files

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

// MaxFileNameLength 最大文件名长度 255
const MaxFileNameLength = 255

var (
	moveMu         = new(sync.Mutex)
	lockMap        = make(map[string]*sync.Mutex)
	videoSuffixArr = []string{".avi", ".flv", ".m4v", ".mkv", ".mov", ".mp4", ".mpeg", ".mpg", ".wmv"}
)

func getMoveLock(path string) *sync.Mutex {
	moveMu.Lock()
	defer moveMu.Unlock()
	if _, ok := lockMap[path]; !ok {
		lockMap[path] = &sync.Mutex{}
	}
	return lockMap[path]
}

func releaseMoveLock(path string) {
	moveMu.Lock()
	defer moveMu.Unlock()
	if _, ok := lockMap[path]; ok {
		delete(lockMap, path)
	}
}

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

func IsTorrentFile(filename string) bool {
	return strings.HasSuffix(filename, ".torrent")
}

// IsValidFileName 检查文件名是否合法（长度与非法字符）
func IsValidFileName(path string) error {
	base := filepath.Base(path)

	// 检查是否为空
	if strings.TrimSpace(base) == "" {
		return errors.New("文件名为空")
	}

	// 检查是否包含非法字符（可扩展）
	illegalChars := []string{"/", "\\", "\x00"} // 你可以根据需求增加
	for _, ch := range illegalChars {
		if strings.Contains(base, ch) {
			return errors.New("文件名包含非法字符: " + ch)
		}
	}

	// 检查文件名长度（字节数，非字符数）
	if len(base) > MaxFileNameLength {
		return errors.New("文件名过长（字节数超过 255）")
	}

	// 可选：检查字符数量（非必要）
	if utf8.RuneCountInString(base) == 0 {
		return errors.New("文件名无效")
	}

	return nil
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

func Delete(filepath string) {
	if exists, err := Exists(filepath); err == nil && exists {
		if err = os.Remove(filepath); err != nil {
			log.Errorf("删除文件异常：%s - %s", filepath, err.Error())
		} else {
			log.Debugf("删除文件OK：%s", filepath)
		}
	} else if !exists {
		log.Debugf("删除文件 - 文件不存在：%s", filepath)
	} else if err != nil {
		log.Errorf("删除文件 - 检查异常：%s - %s", filepath, err.Error())
	}
}

func MoveOnce(srcPath, dstPath string) error {
	if srcPath == "" || dstPath == "" {
		return fmt.Errorf("path is empty")
	}

	lock := getMoveLock(srcPath)
	lock.Lock()
	defer func() {
		releaseMoveLock(srcPath)
		lock.Unlock()
	}()

	if exists, err := Exists(srcPath); err != nil {
		return err
	} else if !exists {
		// 文件不存在，ignore
		log.Debugf("[移动文件] 文件 %s 不存在，ignore", srcPath)
		return nil
	}

	targetDir := filepath.Dir(dstPath)
	err := os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("[移动文件] 创建目标文件夹: %s，异常：%s", targetDir, err.Error())
	}

	// 尝试直接 rename（同设备最快）
	if err = os.Rename(srcPath, dstPath); err != nil {
		// 移动失败，记录日志，尝试复制文件
		log.Warnf("[移动文件] 移动文件异常，将尝试复制模式: %s -> %s，异常：%s", srcPath, dstPath, err.Error())
	}

	// 跨设备处理：手动 copy + remove
	return copyThenDelete(srcPath, dstPath)
}

func copyThenDelete(srcPath, dstPath string) error {
	// 打开源文件
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("[移动文件] 打开源文件失败: %w", err)
	}
	defer src.Close()

	// 创建目标文件（临时）
	dst, err := os.Create(dstPath + ".tmp")
	if err != nil {
		return fmt.Errorf("[移动文件] 创建目标文件失败: %w", err)
	}
	defer dst.Close()

	// 使用 32MB 缓冲区（你可以根据系统内存调整）
	buf := make([]byte, 32*1024*1024)
	if _, err = io.CopyBuffer(dst, src, buf); err != nil {
		return fmt.Errorf("[移动文件] 复制失败: %w", err)
	}

	// 确保写入磁盘
	if err = dst.Sync(); err != nil {
		return fmt.Errorf("[移动文件] 写入磁盘失败: %w", err)
	}

	// 关闭目标文件
	if err = dst.Close(); err != nil {
		return fmt.Errorf("[移动文件] 关闭目标文件失败: %w", err)
	}

	// 原子重命名临时文件
	if err = os.Rename(dstPath+".tmp", dstPath); err != nil {
		return fmt.Errorf("[移动文件] 重命名目标文件失败: %w", err)
	}

	// 删除源文件
	_ = src.Close()
	if err = os.Remove(srcPath); err != nil {
		return fmt.Errorf("[移动文件] 删除源文件失败: %w", err)
	}

	log.Infof("[移动文件] 移动文件成功: %s -> %s", srcPath, dstPath)
	return nil
}
