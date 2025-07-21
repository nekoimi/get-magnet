package aria2

import (
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
)

func (a *Aria2) detailsEventHandle(eventName string, event *arigo.DownloadEvent) (arigo.Status, error) {
	gid := event.GID

	// 清除下载速度缓存
	a.speedCache.Delete(gid)

	// 获取下载任务信息
	task, err := a.client().TellStatus(gid, "gid", "status", "files", "downloadSpeed", "followedBy")
	if err != nil {
		log.Errorf("查询当前(%s - %s)下载任务信息异常: %s", gid, display(task), err.Error())
		return arigo.Status{}, err
	}

	log.Debugf("GID#%s %s：%s", gid, eventName, display(task))

	return task, nil
}

func (a *Aria2) startEventHandle(event *arigo.DownloadEvent) {
	var (
		err    error
		status arigo.Status
	)
	if status, err = a.detailsEventHandle("startEventHandle", event); err != nil {
		return
	}

	log.Debugf("detailsEventHandle: %s - %s", status.GID, status.Status)

	// 下载文件优选
	a.handleFileBestSelect(status)
}

func (a *Aria2) pauseEventHandle(event *arigo.DownloadEvent) {
	var (
		err    error
		status arigo.Status
	)
	if status, err = a.detailsEventHandle("pauseEventHandle", event); err != nil {
		return
	}

	log.Debugf("detailsEventHandle: %s - %s", status.GID, status.Status)
}

func (a *Aria2) stopEventHandle(event *arigo.DownloadEvent) {
	var (
		err    error
		status arigo.Status
	)
	if status, err = a.detailsEventHandle("stopEventHandle", event); err != nil {
		return
	}

	log.Debugf("detailsEventHandle: %s - %s", status.GID, status.Status)
}

func (a *Aria2) completeEventHandle(event *arigo.DownloadEvent) {
	var (
		err    error
		status arigo.Status
	)
	if status, err = a.detailsEventHandle("completeEventHandle", event); err != nil {
		return
	}

	log.Debugf("completeEventHandle - FollowedBy: %s - %s - %s", status.GID, status.FollowedBy, display(status))
}

func (a *Aria2) btCompleteEventHandle(event *arigo.DownloadEvent) {
	var (
		err    error
		status arigo.Status
	)
	if status, err = a.detailsEventHandle("btCompleteEventHandle", event); err != nil {
		return
	}

	log.Debugf("btCompleteEventHandle - FollowedBy: %s - %s - %s", status.GID, status.FollowedBy, display(status))
}

func (a *Aria2) errorEventHandle(event *arigo.DownloadEvent) {
	// 清除下载速度缓存
	a.speedCache.Delete(event.GID)

	status, err := a.client().TellStatus(event.GID, "gid", "status", "infoHash", "files", "bittorrent", "errorCode", "errorMessage")
	if err != nil {
		log.Errorf("查询下载任务GID#%s状态信息异常: %s", event.GID, err.Error())
		return
	}
	log.Errorf("下载任务(%s)出错：[%s] %s - %s", display(status), status.Status, status.ErrorCode, status.ErrorMessage)

	// 处理文件出错的情况
	a.onErrorFileNameTooLong(status)
}
