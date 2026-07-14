package cloud_downloader

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/pkg/files"
)

var singleEditionSuffixPattern = regexp.MustCompile(`^-[A-Z0-9]{1,6}$`)

func buildPlayURL(cfg *config.AppConfig, number string) (string, error) {
	if cfg == nil || strings.TrimSpace(cfg.ExternalBaseURL) == "" {
		return "", fmt.Errorf("app.external_base_url 未配置")
	}
	normalizedNumber := strings.ToUpper(strings.TrimSpace(number))
	if normalizedNumber == "" {
		return "", fmt.Errorf("番号为空，无法生成播放地址")
	}
	return strings.TrimRight(cfg.ExternalBaseURL, "/") + "/api/play/" + url.PathEscape(normalizedNumber), nil
}

func buildSTRMTargetPath(rootDir string, magnet *table.Magnets, sourceFile string) string {
	sourceBase := filepath.Base(sourceFile)
	targetFile := strings.TrimSuffix(sourceBase, filepath.Ext(sourceBase)) + ".strm"
	if normalizedFile, ok := buildNormalizedSTRMFilename(magnet.Number, sourceBase); ok {
		targetFile = normalizedFile
	}

	actress := getActressName(magnet.Actress0)
	title := files.TruncateFilename(magnet.Title, files.MaxFileNameLength)
	return buildTargetPath(rootDir, actress, magnet.CreatedAt, title, targetFile)
}

func buildNormalizedSTRMFilename(number, sourceFile string) (string, bool) {
	normalizedNumber := strings.ToUpper(strings.TrimSpace(number))
	if normalizedNumber == "" || !files.IsVideo(sourceFile) {
		return "", false
	}
	if suffix := extractEditionSuffix(normalizedNumber, sourceFile); suffix != "" {
		return normalizedNumber + suffix + ".strm", true
	}
	return normalizedNumber + ".strm", true
}

func extractEditionSuffix(number, sourceFile string) string {
	base := strings.ToUpper(strings.TrimSuffix(filepath.Base(sourceFile), filepath.Ext(sourceFile)))
	idx := strings.Index(base, number)
	if idx < 0 {
		return ""
	}

	suffix := strings.TrimSpace(base[idx+len(number):])
	if singleEditionSuffixPattern.MatchString(suffix) {
		return suffix
	}

	return ""
}

func writeSTRMFile(targetPath string, playURL string) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
		return fmt.Errorf("创建strm目标文件夹失败: %w", err)
	}
	if err := os.WriteFile(targetPath, []byte(playURL+"\n"), 0644); err != nil {
		return fmt.Errorf("写入strm文件失败: %w", err)
	}
	return nil
}

func getActressName(actress0 string) string {
	if len(actress0) == 0 {
		return "0未知"
	}
	parts := strings.Split(actress0, ",")
	return parts[0]
}

func buildTargetPath(rootDir, actress string, createdAt time.Time, title, sourceFile string) string {
	targetPrefix := filepath.Join(rootDir, actress, createdAt.Format("2006-01-02"))
	return filepath.Join(targetPrefix, title, sourceFile)
}
