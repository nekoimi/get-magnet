package workflows

import (
	"strings"
	"testing"
)

func TestAPITriggerRestrictions(t *testing.T) {
	for _, raw := range []string{
		`{"workflow_id":1,"input":{"url":"https://other.example"}}`,
		`{"workflow_id":1,"input":{"credential_ref":"secret"}}`,
		`{"workflow_id":1,"input":{"page_role":"detail"}}`,
		`{"workflow_id":1,"credentials":{"token":"secret"}}`,
		`{"workflow_id":1,"input":null}`,
		`{"workflow_id":0}`,
		`{"workflow_id":1}{"credentials":{"token":"secret"}}`,
	} {
		if _, err := parseAPITrigger(strings.NewReader(raw)); err == nil {
			t.Fatalf("accepted override: %s", raw)
		}
	}
	for _, raw := range []string{`{"workflow_id":1}`, `{"workflow_id":1,"input":{}}`} {
		if id, err := parseAPITrigger(strings.NewReader(raw)); err != nil || id != 1 {
			t.Fatalf("rejected %s: %v", raw, err)
		}
	}
}
