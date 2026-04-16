package aria2_downloader

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/nekoimi/get-magnet/internal/pkg/files"
)

var singleEditionSuffixPattern = regexp.MustCompile(`^-[A-Z0-9]{1,6}$`)

func buildNormalizedVideoFilename(number, sourceFile string) (string, bool) {
	if !files.IsVideo(sourceFile) {
		return "", false
	}

	normalizedNumber := strings.ToUpper(strings.TrimSpace(number))
	if normalizedNumber == "" {
		return "", false
	}

	ext := strings.ToLower(filepath.Ext(sourceFile))
	if ext == "" {
		return "", false
	}

	fileName := normalizedNumber + ext
	if suffix := extractEditionSuffix(normalizedNumber, sourceFile); suffix != "" {
		fileName = normalizedNumber + suffix + ext
	}

	return fileName, true
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
