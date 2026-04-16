package aria2_downloader

import (
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
)

func (c *Client) handleEvent(evtType arigo.EventType, event *arigo.DownloadEvent) {
	c.downloadSpeedManager.Clean(event.GID)
	s, ok := c.GetStatus(event.GID)
	if !ok && (evtType == arigo.CompleteEvent || evtType == arigo.BTCompleteEvent || evtType == arigo.ErrorEvent) {
		s, ok = c.FindStoppedByGID(event.GID)
	}
	if !ok {
		log.Warnf("aria2事件状态恢复失败，等待补偿任务兜底: %s - %s", evtType, event.GID)
		return
	}

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
	case arigo.ErrorEvent:
		c.eventCh <- Event{
			Type:       evtType,
			taskStatus: s,
		}
	}
}
