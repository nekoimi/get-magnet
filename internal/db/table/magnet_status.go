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

func CanSubmitDownloadStatus(status uint8) bool {
	return status == MagnetStatusCollected || status == MagnetStatusFailed
}
