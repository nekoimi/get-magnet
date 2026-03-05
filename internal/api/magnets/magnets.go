package magnets

import (
	"net/http"
	"strconv"

	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/request"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"github.com/nekoimi/get-magnet/internal/repo/magnet_repo"
)

// ListRequest 磁力链接列表查询请求
type ListRequest struct {
	PageNum   int    `json:"page_num,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
	Keyword   string `json:"keyword,omitempty"`
	Status    *uint8 `json:"status,omitempty"`
}

// ListResponse 磁力链接列表响应
type ListResponse struct {
	List  []table.Magnets `json:"list,omitempty"`
	Total int64           `json:"total,omitempty"`
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

	list, total, err := magnet_repo.PageList(p.PageNum, p.PageSize, p.Keyword, p.Status)
	if err != nil {
		respond.Error(w, err)
		return
	}

	respond.Ok(w, ListResponse{
		List:  list,
		Total: total,
	})
}

// Detail 获取磁力链接详情
func Detail(w http.ResponseWriter, r *http.Request) {
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

	respond.Ok(w, m)
}

// CreateRequest 创建磁力链接请求
type CreateRequest struct {
	Origin      string   `json:"origin,omitempty"`
	Title       string   `json:"title,omitempty"`
	Number      string   `json:"number,omitempty"`
	OptimalLink string   `json:"optimal_link,omitempty"`
	Links       []string `json:"links,omitempty"`
	RawURLHost  string   `json:"raw_url_host,omitempty"`
	RawURLPath  string   `json:"raw_url_path,omitempty"`
	Status      uint8    `json:"status,omitempty"`
	Actress0    string   `json:"actress0,omitempty"`
	FollowedBy  string   `json:"followed_by,omitempty"`
}

// Create 创建磁力链接
func Create(w http.ResponseWriter, r *http.Request) {
	p := new(CreateRequest)
	if err := request.Parse(r, &p); err != nil {
		respond.Error(w, err)
		return
	}

	m := &table.Magnets{
		Origin:      p.Origin,
		Title:       p.Title,
		Number:      p.Number,
		OptimalLink: p.OptimalLink,
		Links:       p.Links,
		RawURLHost:  p.RawURLHost,
		RawURLPath:  p.RawURLPath,
		Status:      p.Status,
		Actress0:    p.Actress0,
		FollowedBy:  p.FollowedBy,
	}

	magnet_repo.Save(m)

	respond.Ok(w, m)
}

// UpdateRequest 更新磁力链接请求
type UpdateRequest struct {
	Id          int64    `json:"id,omitempty"`
	Origin      string   `json:"origin,omitempty"`
	Title       string   `json:"title,omitempty"`
	Number      string   `json:"number,omitempty"`
	OptimalLink string   `json:"optimal_link,omitempty"`
	Links       []string `json:"links,omitempty"`
	RawURLHost  string   `json:"raw_url_host,omitempty"`
	RawURLPath  string   `json:"raw_url_path,omitempty"`
	Status      uint8    `json:"status,omitempty"`
	Actress0    string   `json:"actress0,omitempty"`
	FollowedBy  string   `json:"followed_by,omitempty"`
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
		Id:          p.Id,
		Origin:      p.Origin,
		Title:       p.Title,
		Number:      p.Number,
		OptimalLink: p.OptimalLink,
		Links:       p.Links,
		RawURLHost:  p.RawURLHost,
		RawURLPath:  p.RawURLPath,
		Status:      p.Status,
		Actress0:    p.Actress0,
		FollowedBy:  p.FollowedBy,
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