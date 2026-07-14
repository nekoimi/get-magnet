package play

import (
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/downloader/cloud_downloader"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
)

func Play(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		if m.FollowedBy == "" || m.FollowedBy == "unknow" {
			respond.Error(w, error_ext.DataNotFoundError)
			return
		}

		var cloudCfg *config.CloudDriverConfig
		if cfg != nil {
			cloudCfg = cfg.CloudDriver
		}
		fileID := strings.TrimSpace(r.URL.Query().Get("file_id"))
		filePath := strings.TrimSpace(r.URL.Query().Get("path"))
		if fileID == "" && filePath == "" {
			fileID = m.PlayFileID
			filePath = m.PlayFilePath
		}
		playURL, err := cloud_downloader.ResolveMediaURLWithFile(r.Context(), cloudCfg, m.FollowedBy, fileID, filePath)
		if err != nil {
			respond.Error(w, err)
			return
		}

		http.Redirect(w, r, playURL, http.StatusFound)
	}
}
