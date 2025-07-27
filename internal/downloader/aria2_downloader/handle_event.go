package aria2_downloader

import (
	"github.com/siku2/arigo"
)

func (c *Client) handleEvent(evtType arigo.EventType, event *arigo.DownloadEvent) {
	c.downloadSpeedManager.Clean(event.GID)

	if s, ok := c.GetStatus(event.GID); ok {
		switch evtType {
		case arigo.StartEvent:
			c.fileSelectCh <- s
		//case arigo.PauseEvent:
		//case arigo.StopEvent:
		case arigo.CompleteEvent, arigo.BTCompleteEvent:
			c.eventCh <- Event{
				Type:       evtType,
				taskStatus: s,
			}
			// 文件下载完成移动文件
			handleDownloadCompleteMoveFile(s, "JavDB", c.cfg.MoveTo.JavDBDir)
		case arigo.ErrorEvent:
			c.eventCh <- Event{
				Type:       evtType,
				taskStatus: s,
			}
			// 处理文件出错的情况
			c.handleFileNameTooLongError(s)
		}
	}
}
