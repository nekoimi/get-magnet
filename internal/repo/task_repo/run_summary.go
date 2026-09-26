package task_repo

import "encoding/json"

// RunSummary is rebuilt from durable task attempts when a run finishes.
// A worker never overwrites the whole run with the last page's output.
type RunSummary struct {
	Pages      int `json:"pages"`
	Succeeded  int `json:"succeeded"`
	Failed     int `json:"failed"`
	Cancelled  int `json:"cancelled"`
	Limited    int `json:"limited"`
	Discovered int `json:"discovered"`
	Resources  int `json:"resources"`
	Records    int `json:"records"`
	Created    int `json:"created"`
	Updated    int `json:"updated"`
	Unchanged  int `json:"unchanged"`
}

func (s *RunSummary) Add(status, response string) {
	s.Pages++
	switch status {
	case TaskSucceeded:
		s.Succeeded++
	case TaskFailed, TaskDeadLetter:
		s.Failed++
	case TaskCancelled:
		s.Cancelled++
	}
	if status != TaskSucceeded {
		return
	}
	var output struct {
		DiscoveredCount int   `json:"discovered_count"`
		ResourceID      int64 `json:"resource_id"`
		Limited         bool  `json:"limited"`
		RecordResults   []struct {
			RecordID int64  `json:"record_id"`
			Decision string `json:"decision"`
		} `json:"record_results"`
	}
	if json.Unmarshal([]byte(response), &output) != nil {
		return
	}
	s.Discovered += output.DiscoveredCount
	for _, result := range output.RecordResults {
		if result.RecordID <= 0 {
			continue
		}
		s.Records++
		switch result.Decision {
		case "created":
			s.Created++
		case "updated":
			s.Updated++
		case "unchanged":
			s.Unchanged++
		}
	}
	if output.ResourceID > 0 {
		s.Resources++
	}
	if output.Limited {
		s.Limited++
	}
}

func (s RunSummary) Status() string {
	if s.Cancelled > 0 {
		return RunCancelled
	}
	if s.Resources == 0 && s.Records == 0 {
		return RunFailed
	}
	if s.Limited > 0 {
		return RunLimited
	}
	if s.Failed > 0 {
		return RunPartial
	}
	return RunSucceeded
}

func (s RunSummary) JSON() string {
	b, _ := json.Marshal(s)
	return string(b)
}
