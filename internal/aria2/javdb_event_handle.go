package aria2

import (
	"github.com/nekoimi/get-magnet/internal/db/repository"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
)

func downloadCompleteEventHandle(status arigo.Status, followedBys []string) {
	if len(followedBys) >= 1 {
		// 不是最终的下载任务，尝试更新数据表中关联的id
		followedBy := followedBys[0]
		log.Debugf("任务[%s]下载完成 - 尝试更新 FollowedBy: [%s -> %s]", display(status), status.GID, followedBy)
		if err := repository.UpdateFollowedBy(status.GID, followedBy); err != nil {
			log.Errorf("任务[%s]下载完成 - 尝试更新 FollowedBy: [%s -> %s]，异常：%s", display(status), status.GID, followedBy, err.Error())
			return
		}
	} else {
		log.Debugf("bt任务下载完成 - FollowedBy: %s - %s - %s", status.GID, status.FollowedBy, display(status))

		// 最终完成，需要移动位置
	}
}
