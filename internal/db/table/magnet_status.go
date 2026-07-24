package table

const (
	// MagnetStatusCollected 已采集，待提交下载。
	MagnetStatusCollected uint8 = 0
	// MagnetStatusSubmitting 正在提交下载，用于防止重复领取。
	MagnetStatusSubmitting uint8 = 1
	// MagnetStatusDownloading 已提交外部下载任务。
	MagnetStatusDownloading uint8 = 2
	// MagnetStatusCompleted 下载完成且后处理完成。
	MagnetStatusCompleted uint8 = 3
	// MagnetStatusFailed 下载提交失败或外部下载失败。
	MagnetStatusFailed uint8 = 4
)

type MagnetStatusOption struct {
	Label string `json:"label"`
	Value uint8  `json:"value"`
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
