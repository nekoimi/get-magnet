package aria2

import (
	log "github.com/sirupsen/logrus"
	"time"
)

// LowSpeedNum 低速下载区间测速检查数量
const LowSpeedNum = 10

// LowSpeedTimeout 低速下载多长时间超时
const LowSpeedTimeout = (LowSpeedNum + 5) * time.Minute

// LowSpeedInterval 低速下载检查周期
const LowSpeedInterval = 1 * time.Minute

// LowSpeedCleanupInterval 低速下载记录清除检查周期
const LowSpeedCleanupInterval = 5 * time.Minute

// LowSpeedThreshold 速度小于100KB，speed 单位是 Bytes/s
const LowSpeedThreshold = 102400

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
	log.Debugf("下载任务(%s)存在下载速度记录：size-%d", gid, len(rec.Last))
	// 滑动窗口
	rec.Last = append(rec.Last, speed)
	if len(rec.Last) > LowSpeedNum {
		rec.Last = rec.Last[1:]
	}
	a.speedCache.Set(gid, rec, LowSpeedTimeout)

	if len(rec.Last) < LowSpeedNum {
		log.Debugf("下载任务(%s)存在下载速度数量少于临界值: size-%d，临界值数量：size-%d", gid, len(rec.Last), LowSpeedNum)
		return false
	}

	// 判断是否平均下载速度
	var total uint
	for _, s := range rec.Last {
		total += s
	}
	avg := total / uint(len(rec.Last))
	// 低速下载过期时间内的平均下载速度没有达到最低下载速度
	log.Debugf("下载任务(%s)存在下载速度平均值：value-%d，低速下载临界值：value-%d", gid, avg, LowSpeedThreshold)
	return avg < LowSpeedThreshold
}
