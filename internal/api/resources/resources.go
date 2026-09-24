package resources

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/nekoimi/get-magnet/internal/db/table"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"github.com/nekoimi/get-magnet/internal/pkg/request"
	"github.com/nekoimi/get-magnet/internal/pkg/respond"
	"github.com/nekoimi/get-magnet/internal/repo/resource_repo"
)

type ListRequest struct {
	PageNum      int    `json:"page_num,omitempty"`
	PageSize     int    `json:"page_size,omitempty"`
	Keyword      string `json:"keyword,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
	SourceID     *int64 `json:"source_id,omitempty"`
	Status       string `json:"status,omitempty"`
	CreatedStart string `json:"created_start,omitempty"`
	CreatedEnd   string `json:"created_end,omitempty"`
}

type ListResponse struct {
	List  []table.Resource `json:"list"`
	Total int64            `json:"total"`
}

type DetailResponse struct {
	Resource table.Resource        `json:"resource"`
	Links    []table.ResourceLink  `json:"links"`
	Events   []table.ResourceEvent `json:"events"`
}

func List(w http.ResponseWriter, r *http.Request) {
	input := new(ListRequest)
	if err := request.Parse(r, input); err != nil {
		respond.Error(w, err)
		return
	}
	list, total, err := resource_repo.PageList(resource_repo.PageFilter{
		PageNum: input.PageNum, PageSize: input.PageSize, Keyword: input.Keyword,
		ResourceType: input.ResourceType, SourceID: input.SourceID, Status: input.Status,
		CreatedStart: input.CreatedStart, CreatedEnd: input.CreatedEnd,
	})
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, ListResponse{List: list, Total: total})
}

func Detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(strings.TrimSpace(r.URL.Query().Get("id")), 10, 64)
	if err != nil || id <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	resource, exists := resource_repo.GetByID(id)
	if !exists {
		respond.Error(w, error_ext.DataNotFoundError)
		return
	}
	links, err := resource_repo.ListLinks(id)
	if err != nil {
		respond.Error(w, err)
		return
	}
	events, err := resource_repo.ListEvents(id, 100)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, DetailResponse{Resource: *resource, Links: links, Events: events})
}

func StatusOptions(w http.ResponseWriter, _ *http.Request) {
	respond.Ok(w, table.ResourceStatusOptions())
}

func SourceOptions(w http.ResponseWriter, _ *http.Request) {
	sources, err := resource_repo.ListSources()
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, sources)
}
