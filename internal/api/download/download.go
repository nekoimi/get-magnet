package download

import (
	"fmt"
	"net/http"

	"github.com/nekoimi/get-magnet/internal/bus"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/downloader"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/request"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
	log "github.com/sirupsen/logrus"
)

type SubmitDownloadRequest struct {
	Id  int64   `json:"id,omitempty"`
	Ids []int64 `json:"ids,omitempty"`
}

type SubmitDownloadItem struct {
	Id     int64  `json:"id"`
	TaskID string `json:"task_id,omitempty"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
}

type SubmitDownloadResponse struct {
	List    []SubmitDownloadItem `json:"list"`
	Total   int                  `json:"total"`
	Success int                  `json:"success"`
	Failed  int                  `json:"failed"`
}

func Submit(downloadService downloader.DownloadService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		submitDownload(w, r, downloadService)
	}
}

func Retry(downloadService downloader.DownloadService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		submitDownload(w, r, downloadService)
	}
}

func submitDownload(w http.ResponseWriter, r *http.Request, downloadService downloader.DownloadService) {
	p := new(SubmitDownloadRequest)
	if err := request.Parse(r, p); err != nil {
		respond.Error(w, err)
		return
	}

	ids := normalizeIDs(p)
	if len(ids) == 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	if downloadService == nil {
		respond.Error(w, fmt.Errorf("下载服务未初始化"))
		return
	}

	resp := SubmitDownloadResponse{
		List:  make([]SubmitDownloadItem, 0, len(ids)),
		Total: len(ids),
	}
	for _, id := range ids {
		item := submitDownloadOne(id, downloadService)
		if item.OK {
			resp.Success++
		} else {
			resp.Failed++
		}
		resp.List = append(resp.List, item)
	}

	respond.Ok(w, resp)
}

func normalizeIDs(p *SubmitDownloadRequest) []int64 {
	seen := make(map[int64]struct{})
	ids := make([]int64, 0, len(p.Ids)+1)
	if p.Id > 0 {
		seen[p.Id] = struct{}{}
		ids = append(ids, p.Id)
	}
	for _, id := range p.Ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func submitDownloadOne(id int64, downloadService downloader.DownloadService) SubmitDownloadItem {
	item := SubmitDownloadItem{Id: id}
	m, exists := magnet_repo.GetById(id)
	if !exists {
		item.Error = error_ext.DataNotFoundError.Error()
		return item
	}
	if m.OptimalLink == "" {
		item.Error = "优选链接为空"
		return item
	}
	if !table.CanSubmitDownloadStatus(m.Status) {
		item.Error = "当前状态不允许提交下载"
		return item
	}
	ok, err := magnet_repo.MarkDownloadSubmittingManual(id)
	if err != nil {
		item.Error = err.Error()
		return item
	}
	if !ok {
		item.Error = "资源已被其他调度领取或状态已变化"
		return item
	}

	taskID, err := downloadService.Download(m.Origin, m.OptimalLink)
	if err != nil {
		item.Error = err.Error()
		_ = magnet_repo.MarkDownloadSubmitFailed(id, err)
		return item
	}
	if err := magnet_repo.MarkDownloadSubmitted(id, taskID); err != nil {
		item.Error = err.Error()
		return item
	}

	item.OK = true
	item.TaskID = taskID
	return item
}

type TaskReq struct {
	Url string `json:"url,omitempty"`
}

func SubmitJavDB(w http.ResponseWriter, r *http.Request) {
	p := new(TaskReq)

	if err := request.Parse(r, &p); err != nil {
		respond.Error(w, err)
		return
	} else {
		if p.Url == "" {
			respond.Error(w, error_ext.ValidateError)
			return
		}
	}

	rawUrl := p.Url
	log.Infof("接收到JavDB链接任务：%s", rawUrl)
	bus.Event().Publish(bus.SubmitJavDB.Topic(), rawUrl)

	respond.Ok(w, nil)
}

func SubmitJavDBPage(w http.ResponseWriter, r *http.Request) {
	p := new(TaskReq)

	if err := request.Parse(r, &p); err != nil {
		respond.Error(w, err)
		return
	} else {
		if p.Url == "" {
			respond.Error(w, error_ext.ValidateError)
			return
		}
	}

	rawUrl := p.Url
	log.Infof("接收到JavDB-Page链接任务：%s", rawUrl)
	bus.Event().Publish(bus.SubmitJavDBPage.Topic(), rawUrl)

	respond.Ok(w, nil)
}

//func SubmitFC2(w http.ResponseWriter, r *http.Request) {
//	p := new(TaskReq)
//
//	if err := request.Parse(r, &p); err != nil {
//		respond.Error(w, err)
//		return
//	} else {
//		if p.Url == "" {
//			respond.Error(w, error_ext.ValidateError)
//			return
//		}
//	}
//
//	rawUrl := p.Url
//	log.Infof("接收到FC2链接任务：%s", rawUrl)
//
//	respond.Ok(w, nil)
//}
