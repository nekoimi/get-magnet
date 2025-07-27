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
		case arigo.ErrorEvent:
			c.eventCh <- Event{
				Type:       evtType,
				taskStatus: s,
			}
		}
	}
}
