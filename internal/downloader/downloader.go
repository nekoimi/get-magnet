package downloader

type DownloadCallback func(task DownloadTask)

type DownloadTask struct {
	// ID
	Id string
	// 名称
	Name string
	// 文件信息
	Files []string
}

type DownloadService interface {
	// Download 发起下载
	Download(category string, url string) (string, error)

	// OnComplete 下载完成回调
	OnComplete(callback DownloadCallback)

	// OnError 下载异常回调
	OnError(callback DownloadCallback)
}
