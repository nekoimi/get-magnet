package aria2

import "github.com/nekoimi/arigo"

func GetEvent(event arigo.EventType) string {
	switch event {
	case arigo.StartEvent:
		return "StartEvent"
	case arigo.PauseEvent:
		return "PauseEvent"
	case arigo.StopEvent:
		return "StopEvent"
	case arigo.CompleteEvent:
		return "CompleteEvent"
	case arigo.BTCompleteEvent:
		return "BTCompleteEvent"
	default:
		return "ErrorEvent"
	}
}
