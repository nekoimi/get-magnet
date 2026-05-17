package util

import "regexp"

var numberRegexp = regexp.MustCompile(`(?i)([A-Za-z]+-\d+)`)

// ExtractNumber 从输入字符串中提取标准番号格式（如 SONE-566）。
// 支持带后缀的番号，如 SONE-566-UC -> SONE-566，ABW-301_4K -> ABW-301。
// 无法匹配时返回空串。
func ExtractNumber(s string) string {
	m := numberRegexp.FindString(s)
	if m == "" {
		return ""
	}
	return m
}
