package cloud_downloader

type cloudResponse[T any] struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg,omitempty"`
	Message string `json:"message,omitempty"`
	Data    T      `json:"data,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func (r cloudResponse[T]) message() string {
	if r.Message != "" {
		return r.Message
	}
	return r.Msg
}

type addOfflineTaskRequest struct {
	URL          string            `json:"url"`
	Category     string            `json:"category,omitempty"`
	SavePath     string            `json:"save_path,omitempty"`
	ClientTaskID string            `json:"client_task_id,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

type addOfflineTaskResponse struct {
	TaskID         string `json:"task_id"`
	ProviderTaskID string `json:"provider_task_id,omitempty"`
	Status         string `json:"status,omitempty"`
}

type offlineTask struct {
	TaskID         string      `json:"task_id"`
	ProviderTaskID string      `json:"provider_task_id,omitempty"`
	Status         string      `json:"status"`
	Name           string      `json:"name,omitempty"`
	Progress       float64     `json:"progress,omitempty"`
	SavePath       string      `json:"save_path,omitempty"`
	ErrorCode      string      `json:"error_code,omitempty"`
	ErrorMessage   string      `json:"error_message,omitempty"`
	Files          []cloudFile `json:"files,omitempty"`
}

type cloudFile struct {
	FileID string `json:"file_id,omitempty"`
	Name   string `json:"name,omitempty"`
	Path   string `json:"path,omitempty"`
	Size   int64  `json:"size,omitempty"`
}

type removeCloudFileRequest struct {
	FileID string `json:"file_id,omitempty"`
	Path   string `json:"path,omitempty"`
}
