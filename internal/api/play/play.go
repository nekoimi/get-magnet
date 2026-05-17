package play

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
)

// Play 通过番号获取视频播放地址，302 重定向
func Play(w http.ResponseWriter, r *http.Request) {
	rawNumber := mux.Vars(r)["number"]
	number := util.ExtractNumber(rawNumber)
	if number == "" {
		respond.Error(w, error_ext.ValidateError)
		return
	}

	m, exists := magnet_repo.GetByNumber(number)
	if !exists {
		respond.Error(w, error_ext.DataNotFoundError)
		return
	}

	// TODO: 对接视频解析服务获取实际播放地址
	playUrl := fmt.Sprintf("https://example.com/play/%s", m.Number)

	http.Redirect(w, r, playUrl, http.StatusFound)
}
