package audit

import (
	"net/http"
	"strconv"

	"github.com/nekoimi/scrapio/internal/pkg/respond"
	"github.com/nekoimi/scrapio/internal/repo/audit_repo"
)

func List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	rows, total, err := audit_repo.List(audit_repo.Filter{ResourceType: r.URL.Query().Get("resource_type"), Action: r.URL.Query().Get("action"), Page: page, Size: size})
	if err != nil {
		respond.Error(w, err)
		return
	}
	respond.Ok(w, map[string]any{"list": rows, "total": total})
}
