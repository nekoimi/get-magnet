package aria2_downloader

import (
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
	"strings"
)

const ErrorFileNameTooLong = "File name too long"

func (c *Client) handleFileNameTooLongError(status arigo.Status) {
	if status.Status == arigo.StatusError && status.ErrorCode == arigo.CouldNotOpenExistingFile {
		if strings.Contains(status.ErrorMessage, ErrorFileNameTooLong) {
			log.Infof("处理文件名称过长异常：%s", status.ErrorMessage)
			downloadUrl := util.BuildMagnetLink(status.InfoHash)
			log.Infof("重新提交下载任务：%s", downloadUrl)
			// 删除当前任务
			err := c.client().RemoveDownloadResult(status.GID)
			if err != nil {
				log.Errorf("删除任务失败：%s", err.Error())
				return
			}

			options, err := c.globalOptions()
			if err != nil {
				return
			}

			saveDir := options.Dir + "/error-retry/" + util.NowDate("-")
			if selectIndex, ok, _ := selectDownloadFileBestIndex(status.Files); ok {
				if _, err = c.client().AddURI(arigo.URIs(downloadUrl), &arigo.Options{
					Dir:        saveDir,
					SelectFile: selectIndex,
				}); err != nil {
					log.Errorf("重新添加aria2下载任务异常: [%s] - %s", downloadUrl, err.Error())
				}
			}
		}
	}
}
