package aria2

type SpeedRecord struct {
	Last []uint
}

// 检查当前任务的下载速度，是否暂停任务
func (a *Aria2) isPauseCheckDownloadSpeed(gid string, speed uint) bool {
	value, found := a.speedCache.Get(gid)
	var rec *SpeedRecord
	if found {
		rec = value.(*SpeedRecord)
	} else {
		rec = &SpeedRecord{}
	}
	// 滑动窗口
	rec.Last = append(rec.Last, speed)
	if len(rec.Last) > LowSpeedMinute {
		rec.Last = rec.Last[1:]
	}
	a.speedCache.Set(gid, rec, LowSpeedTimeout)

	if len(rec.Last) < LowSpeedMinute {
		return false
	}

	// 判断是否平均下载速度
	var total uint
	for _, s := range rec.Last {
		total += s
	}
	avg := total / uint(len(rec.Last))
	// 低速下载过期时间内的平均下载速度没有达到最低下载速度
	return avg < LowSpeedThreshold
}
