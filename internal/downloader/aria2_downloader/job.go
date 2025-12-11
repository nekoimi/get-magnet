package aria2_downloader

import (
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
)

// 扫描已经完成的任务，触发任务完成事件
func (c *Client) triggerDownloadCompleteEventJob() {
	offset := 0
	fetchNum := 20
	for {
		stops := c.FetchStopped(offset, uint(fetchNum))
		if len(stops) == 0 {
			break
		}

		for _, stop := range stops {
			log.Debugf("已完成任务：%s - %s", friendly(stop), stop.Status)
			if stop.Status == arigo.StatusCompleted {
				c.eventCh <- Event{
					Type:       arigo.CompleteEvent,
					taskStatus: stop,
				}
			}
		}

		offset = offset + fetchNum
	}
}
