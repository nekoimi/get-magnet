package table

const (
	// MagnetStatusCollected is the legacy magnets status retained until the v2
	// resource migration is complete.
	MagnetStatusCollected uint8 = 0
	// Legacy download statuses remain for isolated compatibility packages.
	MagnetStatusSubmitting  uint8 = 1
	MagnetStatusDownloading uint8 = 2
	MagnetStatusCompleted   uint8 = 3
	MagnetStatusFailed      uint8 = 4
)

// ResourceStatus is the v2 resource lifecycle. It is intentionally separate
// from the legacy uint8 status on table.Magnets until database migration.
type ResourceStatus string

const (
	ResourceStatusDiscovered ResourceStatus = "discovered"
	ResourceStatusCollected  ResourceStatus = "collected"
	ResourceStatusValidated  ResourceStatus = "validated"
	ResourceStatusInvalid    ResourceStatus = "invalid"
	ResourceStatusDuplicate  ResourceStatus = "duplicate"
	ResourceStatusArchived   ResourceStatus = "archived"
)

func ResourceStatusOptions() []map[string]string {
	return []map[string]string{
		{"label": "已发现", "value": string(ResourceStatusDiscovered)},
		{"label": "已采集", "value": string(ResourceStatusCollected)},
		{"label": "已校验", "value": string(ResourceStatusValidated)},
		{"label": "无效", "value": string(ResourceStatusInvalid)},
		{"label": "重复", "value": string(ResourceStatusDuplicate)},
		{"label": "已归档", "value": string(ResourceStatusArchived)},
	}
}

func IsValidResourceStatus(status ResourceStatus) bool {
	switch status {
	case ResourceStatusDiscovered, ResourceStatusCollected, ResourceStatusValidated,
		ResourceStatusInvalid, ResourceStatusDuplicate, ResourceStatusArchived:
		return true
	default:
		return false
	}
}

func CanTransitionResourceStatus(from, to ResourceStatus) bool {
	if !IsValidResourceStatus(from) || !IsValidResourceStatus(to) {
		return false
	}
	if from == to || to == ResourceStatusArchived {
		return true
	}
	switch from {
	case ResourceStatusDiscovered:
		return to == ResourceStatusCollected || to == ResourceStatusInvalid || to == ResourceStatusDuplicate
	case ResourceStatusCollected:
		return to == ResourceStatusValidated || to == ResourceStatusInvalid || to == ResourceStatusDuplicate
	case ResourceStatusValidated:
		return to == ResourceStatusInvalid || to == ResourceStatusDuplicate
	case ResourceStatusInvalid, ResourceStatusDuplicate:
		return to == ResourceStatusCollected || to == ResourceStatusValidated
	case ResourceStatusArchived:
		return false
	default:
		return false
	}
}

type MagnetStatusOption struct {
	Label string `json:"label"`
	Value uint8  `json:"value"`
}

type MagnetSourceOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func MagnetSourceOptions() []MagnetSourceOption {
	return []MagnetSourceOption{
		{Label: "JavDB", Value: "JavDB"},
		{Label: "SeHuaTang", Value: "SeHuaTang"},
		{Label: "手动", Value: "Manual"},
	}
}

func MagnetStatusOptions() []MagnetStatusOption {
	return []MagnetStatusOption{
		{Label: "已采集", Value: MagnetStatusCollected},
		{Label: "提交中", Value: MagnetStatusSubmitting},
		{Label: "下载中", Value: MagnetStatusDownloading},
		{Label: "已完成", Value: MagnetStatusCompleted},
		{Label: "失败", Value: MagnetStatusFailed},
	}
}

func MagnetStatusLabel(status uint8) string {
	for _, option := range MagnetStatusOptions() {
		if option.Value == status {
			return option.Label
		}
	}
	return "未知"
}

func CanSubmitDownloadStatus(status uint8) bool {
	return status == MagnetStatusCollected || status == MagnetStatusFailed
}
