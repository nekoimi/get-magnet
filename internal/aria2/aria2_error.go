package aria2

import (
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
	"strings"
)

const ErrorFileNameTooLong = "File name too long"

func (a *Aria2) onErrorFileNameTooLong(task arigo.Status) {
	if task.Status == arigo.StatusError && task.ErrorCode == arigo.CouldNotOpenExistingFile {
		if strings.Contains(task.ErrorMessage, ErrorFileNameTooLong) {
			log.Infof("处理文件名称过长异常：%s", task.ErrorMessage)
			downloadUrl := util.BuildMagnetLink(task.InfoHash)
			log.Infof("重新提交下载任务：%s", downloadUrl)
			// 删除当前任务
			err := a.client().RemoveDownloadResult(task.GID)
			if err != nil {
				log.Errorf("删除任务失败：%s", err.Error())
				return
			}

			options, err := a.globalOptions()
			if err != nil {
				return
			}

			saveDir := options.Dir + "/error-retry/" + util.NowDate("-")
			if selectIndex, ok, _ := downloadFileBestSelect(task.Files); ok {
				if _, err = a.client().AddURI(arigo.URIs(downloadUrl), &arigo.Options{
					Dir:        saveDir,
					SelectFile: selectIndex,
				}); err != nil {
					log.Errorf("重新添加aria2下载任务异常: [%s] - %s", downloadUrl, err.Error())
				}
			}
		}
	}
}
