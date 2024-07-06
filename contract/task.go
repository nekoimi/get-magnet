package contract

type Task interface {
	Url() string
	Category() string
}

type DownloadTask interface {
	Task
}

type WorkerTask interface {
	Task
	GetHandler() TaskHandler
	GetDownloader() Downloader
}
