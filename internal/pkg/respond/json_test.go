package respond

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInvalidDefinitionContract(t *testing.T) {
	w := httptest.NewRecorder()
	InvalidDefinition(w, "nodes[0].type", "unsupported executor")
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", w.Code)
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			Issues []struct {
				Path   string `json:"path"`
				Reason string `json:"reason"`
			} `json:"issues"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Code != 422 || len(body.Data.Issues) != 1 || body.Data.Issues[0].Path != "nodes[0].type" || body.Data.Issues[0].Reason != "unsupported executor" {
		t.Fatalf("unexpected response: %s", w.Body.String())
	}
}
