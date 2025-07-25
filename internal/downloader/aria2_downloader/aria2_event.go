package aria2_downloader

import "github.com/siku2/arigo"

type Event struct {
	// 事件类型
	Type arigo.EventType
	// 任务状态信息
	taskStatus arigo.Status
}

func (e *Event) Id() string {
	return e.taskStatus.GID
}

func (e *Event) Name() string {
	return friendly(e.taskStatus)
}

func (e *Event) Files() []string {
	var result []string
	for _, file := range e.taskStatus.Files {
		if file.Selected {
			result = append(result, file.Path)
		}
	}
	return result
}

func (e *Event) FollowedBys() []string {
	return e.taskStatus.FollowedBy
}
