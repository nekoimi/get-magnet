package table

import "testing"

func TestMagnetStatusLabelsAndSubmitTransitions(t *testing.T) {
	tests := []struct {
		status    uint8
		label     string
		canSubmit bool
	}{
		{MagnetStatusCollected, "已采集", true},
		{MagnetStatusSubmitting, "提交中", false},
		{MagnetStatusDownloading, "下载中", false},
		{MagnetStatusCompleted, "已完成", false},
		{MagnetStatusFailed, "失败", true},
	}
	for _, tt := range tests {
		if got := MagnetStatusLabel(tt.status); got != tt.label {
			t.Errorf("MagnetStatusLabel(%d) = %q; want %q", tt.status, got, tt.label)
		}
		if got := CanSubmitDownloadStatus(tt.status); got != tt.canSubmit {
			t.Errorf("CanSubmitDownloadStatus(%d) = %v; want %v", tt.status, got, tt.canSubmit)
		}
	}
	if got := MagnetStatusLabel(255); got != "未知" {
		t.Errorf("unknown status label = %q; want 未知", got)
	}
}

func TestMagnetOptionsHaveUniqueValues(t *testing.T) {
	statuses := map[uint8]bool{}
	for _, option := range MagnetStatusOptions() {
		if statuses[option.Value] {
			t.Fatalf("duplicate status value: %d", option.Value)
		}
		statuses[option.Value] = true
	}
	sources := map[string]bool{}
	for _, option := range MagnetSourceOptions() {
		if option.Value == "" || sources[option.Value] {
			t.Fatalf("invalid or duplicate source value: %q", option.Value)
		}
		sources[option.Value] = true
	}
}
