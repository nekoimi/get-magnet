package request

import (
	"encoding/json"
	"github.com/nekoimi/get-magnet/pkg/error_ext"
	"net/http"
	"strings"
)

func Parse[T any](r *http.Request, t *T) error {
	contentType := r.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/json") {
		return error_ext.RequestBodyNotSupportedError
	}

	raw := make([]byte, r.ContentLength)
	_, err := r.Body.Read(raw)
	if err != nil {
		return err
	}
	err = json.Unmarshal(raw, t)
	if err != nil {
		return err
	}
	return nil
}
