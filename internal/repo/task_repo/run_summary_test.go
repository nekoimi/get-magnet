package task_repo

import "testing"

func TestRunSummaryAggregatesAllPages(t *testing.T) {
	s := RunSummary{}
	s.Add(TaskSucceeded, `{"resource_id":41,"discovered_count":2}`)
	s.Add(TaskSucceeded, `{"resource_id":42}`)
	s.Add(TaskDeadLetter, `{}`)
	if s.Pages != 3 || s.Resources != 2 || s.Discovered != 2 || s.Status() != RunPartial {
		t.Fatalf("unexpected partial summary: %+v", s)
	}
	if s.JSON() == "" {
		t.Fatal("summary must be serializable")
	}
	noResult := RunSummary{}
	noResult.Add(TaskSucceeded, `{"discovered_count":0}`)
	if noResult.Status() != RunFailed {
		t.Fatalf("empty run must fail: %+v", noResult)
	}
	limited := RunSummary{}
	limited.Add(TaskSucceeded, `{"resource_id":41,"limited":true}`)
	if limited.Status() != RunLimited {
		t.Fatalf("budget limited run: %+v", limited)
	}
	emptyLimited := RunSummary{}
	emptyLimited.Add(TaskSucceeded, `{"limited":true}`)
	if emptyLimited.Status() != RunFailed {
		t.Fatalf("limited run without a resource must fail: %+v", emptyLimited)
	}
}
