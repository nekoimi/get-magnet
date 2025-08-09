package files

import "testing"

func TestTrimUnicodeString(t *testing.T) {
	title := "CAWD-863 「ゴムなくなっちゃった…生でもいいよ」終電なくなり後輩女子社員の部屋に… 無防備すぎる部屋着とナマ脚に興奮した僕は妻ともしたことない人生初中出し 一晩中モウレツにハメ狂った…"
	titleLen := len(title)
	t.Log(titleLen)
	// fix 需要缩短文件名称
	// 缩短标题
	title = TruncateFilename(title, MaxFileNameLength)
	t.Log(title)
}
