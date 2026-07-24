package magnets

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nekoimi/get-magnet/internal/pkg/respond"
)

func TestOptionsHandlers(t *testing.T) {
	tests := []struct {
		name    string
		handler func(http.ResponseWriter, *http.Request)
	}{
		{name: "status", handler: StatusOptions},
		{name: "source", handler: SourceOptions},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tt.handler(recorder, httptest.NewRequest("GET", "/", nil))
			if recorder.Code != 200 {
				t.Fatalf("status = %d", recorder.Code)
			}
			var response respond.JsonResponse
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if response.Code != 0 || response.Data == nil {
				t.Fatalf("unexpected response: %+v", response)
			}
		})
	}
}
