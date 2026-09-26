package resources

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/nekoimi/scrapio/internal/api/middleware"
	"github.com/nekoimi/scrapio/internal/db/table"
	"github.com/nekoimi/scrapio/internal/pkg/error_ext"
	"github.com/nekoimi/scrapio/internal/pkg/request"
	"github.com/nekoimi/scrapio/internal/pkg/respond"
	"github.com/nekoimi/scrapio/internal/repo/audit_repo"
	"github.com/nekoimi/scrapio/internal/repo/record_repo"
	"github.com/nekoimi/scrapio/internal/repo/resource_repo"
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
	List  []ResourceView `json:"list"`
	Total int64          `json:"total"`
}

type ResourceView struct {
	table.Resource
	SourceName string               `json:"source_name,omitempty"`
	Links      []table.ResourceLink `json:"links,omitempty"`
}

type DetailResponse struct {
	Resource table.Resource        `json:"resource"`
	Links    []table.ResourceLink  `json:"links"`
	Events   []table.ResourceEvent `json:"events"`
}

type ResourceRequest struct {
	Id           int64                     `json:"id,omitempty"`
	ResourceType string                    `json:"resource_type,omitempty"`
	SourceID     *int64                    `json:"source_id,omitempty"`
	Source       string                    `json:"source,omitempty"`
	SourceName   string                    `json:"source_name,omitempty"`
	SourceURL    string                    `json:"source_url,omitempty"`
	CanonicalKey string                    `json:"canonical_key,omitempty"`
	Title        string                    `json:"title,omitempty"`
	Status       string                    `json:"status,omitempty"`
	Attributes   json.RawMessage           `json:"attributes,omitempty"`
	Links        []resource_repo.LinkInput `json:"links,omitempty"`
}

type DeleteRequest struct {
	Id  int64   `json:"id,omitempty"`
	Ids []int64 `json:"ids,omitempty"`
}

type MarkStatusRequest struct {
	Id      int64  `json:"id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
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
	views := make([]ResourceView, 0, len(list))
	for _, resource := range list {
		view := ResourceView{Resource: resource}
		if source, ok := resource_repo.GetSource(resource.SourceId); ok {
			view.SourceName = source.Name
		}
		links, linkErr := resource_repo.ListLinks(resource.Id)
		if linkErr != nil {
			respond.Error(w, linkErr)
			return
		}
		view.Links = links
		views = append(views, view)
	}
	respond.Ok(w, ListResponse{List: views, Total: total})
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
	options := make([]map[string]any, 0, len(sources))
	for _, source := range sources {
		options = append(options, map[string]any{"label": source.Name, "value": source.Id, "code": source.Code})
	}
	respond.Ok(w, options)
}

func Create(w http.ResponseWriter, r *http.Request) {
	input := new(ResourceRequest)
	if err := request.Parse(r, input); err != nil {
		respond.Error(w, err)
		return
	}
	resource, err := buildResource(input, nil)
	if err != nil {
		respond.Error(w, err)
		return
	}
	if err := resource_repo.Save(resource); err != nil {
		respond.Error(w, err)
		return
	}
	if input.Links != nil {
		if err := resource_repo.ReplaceLinks(resource.Id, input.Links); err != nil {
			respond.Error(w, err)
			return
		}
	}
	if resource.ResourceType == "magnet" {
		if _, err := record_repo.ImportLegacyResource(resource.Id); err != nil {
			respond.Error(w, err)
			return
		}
	}
	_ = audit_repo.Record(middleware.RequestID(r.Context()), "resource.created", "resource", &resource.Id, map[string]any{"resource_type": resource.ResourceType})
	respond.Ok(w, resource)
}

func Update(w http.ResponseWriter, r *http.Request) {
	input := new(ResourceRequest)
	if err := request.Parse(r, input); err != nil || input.Id <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	current, exists := resource_repo.GetByID(input.Id)
	if !exists {
		respond.Error(w, error_ext.DataNotFoundError)
		return
	}
	resource, err := buildResource(input, current)
	if err != nil {
		respond.Error(w, err)
		return
	}
	if err := resource_repo.Update(resource, input.Links); err != nil {
		respond.Error(w, err)
		return
	}
	if resource.ResourceType == "magnet" {
		if _, err := record_repo.ImportLegacyResource(resource.Id); err != nil {
			respond.Error(w, err)
			return
		}
	}
	_ = audit_repo.Record(middleware.RequestID(r.Context()), "resource.updated", "resource", &resource.Id, map[string]any{"status": resource.Status, "title_changed": current.Title != resource.Title, "attributes_changed": current.Attributes != resource.Attributes, "links_changed": input.Links != nil})
	respond.Ok(w, resource)
}

func Delete(w http.ResponseWriter, r *http.Request) {
	input := new(DeleteRequest)
	if err := request.Parse(r, input); err != nil {
		respond.Error(w, err)
		return
	}
	ids := input.Ids
	if input.Id > 0 {
		ids = append(ids, input.Id)
	}
	if err := resource_repo.BatchDelete(ids); err != nil {
		respond.Error(w, err)
		return
	}
	for _, id := range ids {
		if id > 0 {
			_ = audit_repo.Record(middleware.RequestID(r.Context()), "resource.deleted", "resource", &id, nil)
		}
	}
	respond.Ok(w, nil)
}

func MarkStatus(w http.ResponseWriter, r *http.Request) {
	input := new(MarkStatusRequest)
	if err := request.Parse(r, input); err != nil || input.Id <= 0 || !table.IsValidResourceStatus(table.ResourceStatus(input.Status)) {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	if err := resource_repo.MarkStatus(input.Id, input.Status, input.Message); err != nil {
		respond.Error(w, err)
		return
	}
	_ = audit_repo.Record(middleware.RequestID(r.Context()), "resource.status_changed", "resource", &input.Id, map[string]any{"status": input.Status, "message": input.Message})
	respond.Ok(w, nil)
}

func buildResource(input *ResourceRequest, current *table.Resource) (*table.Resource, error) {
	resource := &table.Resource{ResourceType: input.ResourceType, SourceId: valueOrZero(input.SourceID), SourceURL: strings.TrimSpace(input.SourceURL), CanonicalKey: strings.TrimSpace(input.CanonicalKey), Title: strings.TrimSpace(input.Title), Status: strings.TrimSpace(input.Status), Attributes: strings.TrimSpace(string(input.Attributes)), CreatedAt: time.Now(), UpdatedAt: time.Now(), FirstSeenAt: time.Now(), LastSeenAt: time.Now()}
	if current != nil {
		*resource = *current
		resource.UpdatedAt = time.Now()
		if input.ResourceType != "" {
			resource.ResourceType = input.ResourceType
		}
		if input.SourceURL != "" {
			resource.SourceURL = strings.TrimSpace(input.SourceURL)
		}
		if input.CanonicalKey != "" {
			resource.CanonicalKey = strings.TrimSpace(input.CanonicalKey)
		}
		if input.Title != "" {
			resource.Title = strings.TrimSpace(input.Title)
		}
		if input.Status != "" {
			resource.Status = strings.TrimSpace(input.Status)
		}
		if len(input.Attributes) > 0 {
			resource.Attributes = strings.TrimSpace(string(input.Attributes))
		}
	}
	if resource.ResourceType == "" {
		resource.ResourceType = "magnet"
	}
	if resource.Status == "" {
		resource.Status = string(table.ResourceStatusDiscovered)
	}
	if resource.Attributes == "" {
		resource.Attributes = "{}"
	}
	if input.SourceID != nil || strings.TrimSpace(input.Source) != "" || strings.TrimSpace(input.SourceName) != "" || current == nil {
		sourceID, err := resource_repo.ResolveSourceID(input.SourceID, input.Source, input.SourceName)
		if err != nil {
			return nil, err
		}
		resource.SourceId = sourceID
	}
	if resource.CanonicalKey == "" {
		return nil, error_ext.ValidateError
	}
	if !table.IsValidResourceStatus(table.ResourceStatus(resource.Status)) {
		return nil, error_ext.ValidateError
	}
	return resource, nil
}

func valueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
