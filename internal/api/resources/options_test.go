package resources

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/nekoimi/scrapio/internal/pkg/respond"
)

func TestStatusOptions(t *testing.T) {
	recorder := httptest.NewRecorder()
	StatusOptions(recorder, httptest.NewRequest("GET", "/", nil))
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
}
