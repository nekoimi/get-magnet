package sehuatang

import (
	"net/url"
	"testing"
)

func TestDetails_Handle(t *testing.T) {
	find := fc2NumberRe.FindString("[FC2PPV] FC2PPV-4715094 [即ハメ中出し素人妻］無修正≪Gcupグラマー系❤️デカぃおっぱいＪＤ")
	t.Log(find)

	u, err := url.Parse("https://www.sehuatang.net/forum.php?mod=viewthread&tid=2860787&extra=page=1&filter=typeid&typeid=368")
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Log(u.Path)
	t.Log(u.RawQuery)
	t.Log(u.RequestURI())
}
