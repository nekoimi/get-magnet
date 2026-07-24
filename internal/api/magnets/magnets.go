package magnets

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nekoimi/get-magnet/internal/config"
	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/downloader/cloud_downloader"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/request"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_event_repo"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
)

// ListRequest 磁力链接列表查询请求
type ListRequest struct {
	PageNum              int    `json:"page_num,omitempty"`
	PageSize             int    `json:"page_size,omitempty"`
	Keyword              string `json:"keyword,omitempty"`
	Status               *uint8 `json:"status,omitempty"`
	Origin               string `json:"origin,omitempty"`
	HasOptimalLink       *bool  `json:"has_optimal_link,omitempty"`
	CreatedAtStart       string `json:"created_at_start,omitempty"`
	CreatedAtEnd         string `json:"created_at_end,omitempty"`
	LastSubmitAtStart    string `json:"last_submit_at_start,omitempty"`
	LastSubmitAtEnd      string `json:"last_submit_at_end,omitempty"`
	CompletedAtStart     string `json:"completed_at_start,omitempty"`
	CompletedAtEnd       string `json:"completed_at_end,omitempty"`
	DownloadCompletedAt  string `json:"download_completed_at,omitempty"`
	DownloadCompletedEnd string `json:"download_completed_end,omitempty"`
}

// ListResponse 磁力链接列表响应
type ListResponse struct {
	List  []table.Magnets `json:"list,omitempty"`
	Total int64           `json:"total,omitempty"`
}

type DetailResponse struct {
	Magnet        *table.Magnets             `json:"magnet"`
	StatusLabel   string                     `json:"status_label"`
	LinkCount     int                        `json:"link_count"`
	HasPlayInfo   bool                       `json:"has_play_info"`
	PlayURL       string                     `json:"play_url,omitempty"`
	PostProcess   DetailPostProcess          `json:"post_process"`
	Download      DetailDownload             `json:"download"`
	Events        []table.MagnetEvent        `json:"events"`
	StatusOptions []table.MagnetStatusOption `json:"status_options"`
}

type DetailPostProcess struct {
	Done         bool   `json:"done"`
	PlayFileID   string `json:"play_file_id,omitempty"`
	PlayFilePath string `json:"play_file_path,omitempty"`
	PlayFileSize int64  `json:"play_file_size,omitempty"`
	STRMPath     string `json:"strm_path,omitempty"`
}

type DetailDownload struct {
	TaskID       string     `json:"task_id,omitempty"`
	Error        string     `json:"error,omitempty"`
	RetryCount   int        `json:"retry_count"`
	LastSubmitAt *time.Time `json:"last_submit_at,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
}

// List 获取磁力链接列表
func List(w http.ResponseWriter, r *http.Request) {
	p := new(ListRequest)
	if err := request.Parse(r, &p); err != nil {
		respond.Error(w, err)
		return
	}

	// 设置默认值
	if p.PageNum <= 0 {
		p.PageNum = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 10
	}

	list, total, err := magnet_repo.PageListByFilter(magnet_repo.PageFilter{
		PageNum:           p.PageNum,
		PageSize:          p.PageSize,
		Keyword:           p.Keyword,
		Status:            p.Status,
		Origin:            p.Origin,
		HasOptimalLink:    p.HasOptimalLink,
		CreatedAtStart:    p.CreatedAtStart,
		CreatedAtEnd:      p.CreatedAtEnd,
		LastSubmitAtStart: p.LastSubmitAtStart,
		LastSubmitAtEnd:   p.LastSubmitAtEnd,
		CompletedAtStart:  firstNonEmpty(p.CompletedAtStart, p.DownloadCompletedAt),
		CompletedAtEnd:    firstNonEmpty(p.CompletedAtEnd, p.DownloadCompletedEnd),
	})
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, ListResponse{
		List:  list,
		Total: total,
	})
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// StatusOptions 获取磁力资源状态选项
func StatusOptions(w http.ResponseWriter, r *http.Request) {
	respond.Ok(w, table.MagnetStatusOptions())
}

// Detail 获取磁力链接详情
func Detail(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			respond.Error(w, error_ext.ValidateError)
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			respond.Error(w, error_ext.ValidateError)
			return
		}

		m, exists := magnet_repo.GetById(id)
		if !exists {
			respond.Error(w, error_ext.DataNotFoundError)
			return
		}
		events, err := magnet_event_repo.ListByMagnetID(id, 100)
		if err != nil {
			respond.Error(w, err)
			return
		}

		respond.Ok(w, DetailResponse{
			Magnet:      m,
			StatusLabel: table.MagnetStatusLabel(m.Status),
			LinkCount:   len(m.Links),
			HasPlayInfo: m.PlayFileID != "" || m.PlayFilePath != "",
			PlayURL:     buildPlayURL(cfg, m),
			PostProcess: DetailPostProcess{
				Done:         m.PostProcessDone,
				PlayFileID:   m.PlayFileID,
				PlayFilePath: m.PlayFilePath,
				PlayFileSize: m.PlayFileSize,
				STRMPath:     m.STRMPath,
			},
			Download: DetailDownload{
				TaskID:       m.FollowedBy,
				Error:        m.DownloadError,
				RetryCount:   m.DownloadRetryCount,
				LastSubmitAt: m.LastSubmitAt,
				CompletedAt:  m.DownloadCompletedAt,
			},
			Events:        events,
			StatusOptions: table.MagnetStatusOptions(),
		})
	}
}

func buildPlayURL(cfg *config.Config, m *table.Magnets) string {
	if m == nil || m.Number == "" || (m.PlayFileID == "" && m.PlayFilePath == "") {
		return ""
	}
	base := ""
	if cfg != nil && cfg.App != nil {
		base = strings.TrimRight(cfg.App.ExternalBaseURL, "/")
	}
	playURL := base + "/api/play/" + url.PathEscape(strings.ToUpper(m.Number))
	params := url.Values{}
	if m.PlayFileID != "" {
		params.Set("file_id", m.PlayFileID)
	}
	if m.PlayFilePath != "" {
		params.Set("path", m.PlayFilePath)
	}
	if encoded := params.Encode(); encoded != "" {
		playURL += "?" + encoded
	}
	return playURL
}

// CreateRequest 创建磁力链接请求
type CreateRequest struct {
	Origin              string     `json:"origin,omitempty"`
	Title               string     `json:"title,omitempty"`
	Number              string     `json:"number,omitempty"`
	OptimalLink         string     `json:"optimal_link,omitempty"`
	Links               []string   `json:"links,omitempty"`
	RawURLHost          string     `json:"raw_url_host,omitempty"`
	RawURLPath          string     `json:"raw_url_path,omitempty"`
	Status              uint8      `json:"status,omitempty"`
	Actress0            string     `json:"actress0,omitempty"`
	FollowedBy          string     `json:"followed_by,omitempty"`
	PlayFileID          string     `json:"play_file_id,omitempty"`
	PlayFilePath        string     `json:"play_file_path,omitempty"`
	PlayFileSize        int64      `json:"play_file_size,omitempty"`
	STRMPath            string     `json:"strm_path,omitempty"`
	DownloadError       string     `json:"download_error,omitempty"`
	DownloadRetryCount  int        `json:"download_retry_count,omitempty"`
	LastSubmitAt        *time.Time `json:"last_submit_at,omitempty"`
	DownloadCompletedAt *time.Time `json:"download_completed_at,omitempty"`
}

// Create 创建磁力链接
func Create(w http.ResponseWriter, r *http.Request) {
	p := new(CreateRequest)
	if err := request.Parse(r, &p); err != nil {
		respond.Error(w, err)
		return
	}

	m := &table.Magnets{
		Origin:              p.Origin,
		Title:               p.Title,
		Number:              p.Number,
		OptimalLink:         p.OptimalLink,
		Links:               p.Links,
		RawURLHost:          p.RawURLHost,
		RawURLPath:          p.RawURLPath,
		Status:              p.Status,
		Actress0:            p.Actress0,
		FollowedBy:          p.FollowedBy,
		PlayFileID:          p.PlayFileID,
		PlayFilePath:        p.PlayFilePath,
		PlayFileSize:        p.PlayFileSize,
		STRMPath:            p.STRMPath,
		DownloadError:       p.DownloadError,
		DownloadRetryCount:  p.DownloadRetryCount,
		LastSubmitAt:        p.LastSubmitAt,
		DownloadCompletedAt: p.DownloadCompletedAt,
	}

	magnet_repo.Save(m)

	respond.Ok(w, m)
}

// UpdateRequest 更新磁力链接请求
type UpdateRequest struct {
	Id                  int64      `json:"id,omitempty"`
	Origin              string     `json:"origin,omitempty"`
	Title               string     `json:"title,omitempty"`
	Number              string     `json:"number,omitempty"`
	OptimalLink         string     `json:"optimal_link,omitempty"`
	Links               []string   `json:"links,omitempty"`
	RawURLHost          string     `json:"raw_url_host,omitempty"`
	RawURLPath          string     `json:"raw_url_path,omitempty"`
	Status              uint8      `json:"status,omitempty"`
	Actress0            string     `json:"actress0,omitempty"`
	FollowedBy          string     `json:"followed_by,omitempty"`
	PlayFileID          string     `json:"play_file_id,omitempty"`
	PlayFilePath        string     `json:"play_file_path,omitempty"`
	PlayFileSize        int64      `json:"play_file_size,omitempty"`
	STRMPath            string     `json:"strm_path,omitempty"`
	DownloadError       string     `json:"download_error,omitempty"`
	DownloadRetryCount  int        `json:"download_retry_count,omitempty"`
	LastSubmitAt        *time.Time `json:"last_submit_at,omitempty"`
	DownloadCompletedAt *time.Time `json:"download_completed_at,omitempty"`
}

// Update 更新磁力链接
func Update(w http.ResponseWriter, r *http.Request) {
	p := new(UpdateRequest)
	if err := request.Parse(r, &p); err != nil {
		respond.Error(w, err)
		return
	}

	if p.Id == 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}

	// 检查是否存在
	_, exists := magnet_repo.GetById(p.Id)
	if !exists {
		respond.Error(w, error_ext.DataNotFoundError)
		return
	}

	m := &table.Magnets{
		Id:                  p.Id,
		Origin:              p.Origin,
		Title:               p.Title,
		Number:              p.Number,
		OptimalLink:         p.OptimalLink,
		Links:               p.Links,
		RawURLHost:          p.RawURLHost,
		RawURLPath:          p.RawURLPath,
		Status:              p.Status,
		Actress0:            p.Actress0,
		FollowedBy:          p.FollowedBy,
		PlayFileID:          p.PlayFileID,
		PlayFilePath:        p.PlayFilePath,
		PlayFileSize:        p.PlayFileSize,
		STRMPath:            p.STRMPath,
		DownloadError:       p.DownloadError,
		DownloadRetryCount:  p.DownloadRetryCount,
		LastSubmitAt:        p.LastSubmitAt,
		DownloadCompletedAt: p.DownloadCompletedAt,
	}

	if err := magnet_repo.Update(m); err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, m)
}

// DeleteRequest 删除请求
type DeleteRequest struct {
	Ids []int64 `json:"ids,omitempty"`
}

type MarkStatusRequest struct {
	Id      int64  `json:"id,omitempty"`
	Status  uint8  `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

type RebuildSTRMRequest struct {
	Id  int64   `json:"id,omitempty"`
	Ids []int64 `json:"ids,omitempty"`
}

// Delete 删除磁力链接
func Delete(w http.ResponseWriter, r *http.Request) {
	p := new(DeleteRequest)
	if err := request.Parse(r, &p); err != nil {
		respond.Error(w, err)
		return
	}

	if len(p.Ids) == 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}

	if err := magnet_repo.BatchDelete(p.Ids); err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, nil)
}

func MarkStatus(w http.ResponseWriter, r *http.Request) {
	p := new(MarkStatusRequest)
	if err := request.Parse(r, p); err != nil {
		respond.Error(w, err)
		return
	}
	if p.Id <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	if _, exists := magnet_repo.GetById(p.Id); !exists {
		respond.Error(w, error_ext.DataNotFoundError)
		return
	}
	if !isKnownStatus(p.Status) {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	if err := magnet_repo.MarkStatus(p.Id, p.Status, p.Message); err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, nil)
}

func RebuildSTRM(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := new(RebuildSTRMRequest)
		if err := request.Parse(r, p); err != nil {
			respond.Error(w, err)
			return
		}
		ids := normalizeIDs(p.Id, p.Ids)
		if len(ids) == 0 {
			respond.Error(w, error_ext.ValidateError)
			return
		}
		results := make([]cloud_downloader.RebuildSTRMResult, 0, len(ids))
		for _, id := range ids {
			m, exists := magnet_repo.GetById(id)
			if !exists {
				respond.Error(w, error_ext.DataNotFoundError)
				return
			}
			result, err := cloud_downloader.RebuildSTRM(context.Background(), cfg.App, cfg.CloudDriver, cfg.STRM, m)
			if err != nil {
				respond.Error(w, err)
				return
			}
			results = append(results, result)
			magnet_event_repo.Record(id, "strm_rebuilt", "STRM 文件已重新生成", strings.Join(result.Paths, "\n"))
		}
		respond.Ok(w, results)
	}
}

func RebuildSTRMBatch(cfg *config.Config) http.HandlerFunc {
	return RebuildSTRM(cfg)
}

func isKnownStatus(status uint8) bool {
	for _, option := range table.MagnetStatusOptions() {
		if option.Value == status {
			return true
		}
	}
	return false
}

func normalizeIDs(id int64, ids []int64) []int64 {
	seen := make(map[int64]struct{})
	result := make([]int64, 0, len(ids)+1)
	if id > 0 {
		seen[id] = struct{}{}
		result = append(result, id)
	}
	for _, item := range ids {
		if item <= 0 {
			continue
		}
		if _, exists := seen[item]; exists {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}
