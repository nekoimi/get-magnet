package workflows

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/nekoimi/scrapio/internal/db"
	"github.com/nekoimi/scrapio/internal/db/table"
	"github.com/nekoimi/scrapio/internal/drission_rod"
	"github.com/nekoimi/scrapio/internal/pkg/error_ext"
	"github.com/nekoimi/scrapio/internal/pkg/request"
	"github.com/nekoimi/scrapio/internal/pkg/respond"
	"github.com/nekoimi/scrapio/internal/repo/workflow_repo"
	"github.com/nekoimi/scrapio/internal/workflow"
)

type previewRequest struct {
	VersionID       int64    `json:"version_id"`
	SampleID        int64    `json:"sample_id,omitempty"`
	ContentType     string   `json:"content_type,omitempty"`
	Content         string   `json:"content,omitempty"`
	PageURL         string   `json:"page_url,omitempty"`
	IdempotencyKeys []string `json:"idempotency_keys,omitempty"`
}

func SaveSample(browser *drission_rod.DrissionRod) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body=http.MaxBytesReader(w,r.Body,11<<20)
		input := new(workflow_repo.SampleInput)
		if err := request.Parse(r, input); err != nil || input.VersionID <= 0 {
			respond.Error(w, error_ext.ValidateError)
			return
		}
		version, has, err := workflow_repo.GetVersion(input.VersionID)
		if err != nil || !has {
			respond.Error(w, error_ext.DataNotFoundError)
			return
		}
		if version.Status != workflow_repo.VersionDraft {
			respond.Error(w, errors.New("samples can only be added to a draft version"))
			return
		}
		definition, err := workflow.ParseExecutableDefinition(version.Definition)
		if err != nil || definition.Persistence != "records" {
			respond.Error(w, errors.New("sample requires executable records workflow"))
			return
		}
		switch input.Source {
		case "live":
			if definition.Trigger.Fetch.Mode == "browser" && browser == nil {
				respond.Error(w, errors.New("browser worker is unavailable"))
				return
			}
			fetched, err := (workflow.Fetcher{Browser: browser}).Fetch(r.Context(), definition.Trigger.URL, definition.Trigger.Fetch)
			if err != nil {
				respond.Error(w, err)
				return
			}
			input.PageURL = fetched.FinalURL
			if fetched.JSON != "" {
				input.ContentType = "json"
				input.Content = fetched.JSON
			} else {
				input.ContentType = "html"
				input.Content = fetched.HTML
			}
		case "document":
			if input.DocumentID == nil {
				respond.Error(w, error_ext.ValidateError)
				return
			}
			doc := new(table.Document)
			has, err := db.Instance().ID(*input.DocumentID).Get(doc)
			if err != nil || !has {
				respond.Error(w, error_ext.DataNotFoundError)
				return
			}
			input.ContentType = doc.DocumentType
			input.Content = doc.Content
			var metadata struct { FinalURL string `json:"final_url"`; URL string `json:"url"` }
			_ = json.Unmarshal([]byte(doc.Metadata),&metadata)
			input.PageURL=metadata.FinalURL
			if input.PageURL=="" { input.PageURL=metadata.URL }
			if input.PageURL=="" { input.PageURL=definition.Trigger.URL }
		case "paste":
			if input.PageURL == "" {
				input.PageURL = definition.Trigger.URL
			}
		default:
			respond.Error(w, error_ext.ValidateError)
			return
		}
		row, err := workflow_repo.SaveSample(*input)
		if err != nil {
			respond.Error(w, err)
			return
		}
		row.Content = ""
		respond.Ok(w, row)
	}
}

func ListSamples(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("version_id"), 10, 64)
	if err != nil || id <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	rows, err := workflow_repo.ListSamples(id)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, map[string]any{"list": rows})
}

func DeleteSample(w http.ResponseWriter, r *http.Request) {
	id, err := idFromRequest(r)
	if err != nil || id <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	if err := workflow_repo.DeleteSample(id); err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, nil)
}

func PreviewSample(w http.ResponseWriter, r *http.Request) {
	r.Body=http.MaxBytesReader(w,r.Body,11<<20)
	input := new(previewRequest)
	if err := request.Parse(r, input); err != nil || input.VersionID <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	var sample *table.WorkflowSample
	if input.SampleID > 0 {
		var err error
		sample, err = workflow_repo.GetSample(input.SampleID)
		if err != nil {
			respond.Error(w, err)
			return
		}
	} else {
		if input.Content == "" || len(input.Content) > 10<<20 || (input.ContentType != "html" && input.ContentType != "json") {
			respond.Error(w, error_ext.ValidateError)
			return
		}
		sample = &table.WorkflowSample{WorkflowVersionId: input.VersionID, Source: "paste", ContentType: input.ContentType, Content: input.Content, PageURL: input.PageURL}
	}
	preview, err := workflow_repo.PreviewSampleWithKeys(input.VersionID, sample, input.IdempotencyKeys)
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, preview)
}

func CheckSamples(w http.ResponseWriter, r *http.Request) {
	id, err := idFromRequest(r)
	if err != nil || id <= 0 {
		respond.Error(w, error_ext.ValidateError)
		return
	}
	if err := workflow_repo.ValidateVersion(id); err != nil {
		respond.Error(w, err)
		return
	}
	checks, err := workflow_repo.CheckSamples(id)
	if err != nil {
		respond.Ok(w, map[string]any{"passed": false, "checks": checks, "error": err.Error()})
		return
	}
	respond.Ok(w, map[string]any{"passed": true, "checks": checks})
}
