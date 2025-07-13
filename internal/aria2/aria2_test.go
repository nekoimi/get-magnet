package aria2

import (
	"github.com/nekoimi/get-magnet/internal/pkg/util"
	"github.com/siku2/arigo"
	"runtime/debug"
	"strings"
	"testing"
)

func TestAria2_Start(t *testing.T) {
	// Failed to open the file xxxxx cause: File name too long
	t.Log(len("hello"))
	t.Log(len("你好"))
}

func TestAria2_Client(t *testing.T) {
	client, err := arigo.Dial("wss://aria2.sakuraio.com/jsonrpc", "nekoimi")
	if err != nil {
		t.Fatal(err.Error())
	}

	// 获取全局属性
	options, err := client.GetGlobalOptions()
	if err != nil {
		t.Fatal(err.Error())
	}

	t.Log(options)
	t.Log(options.MaxConcurrentDownloads)

	//// 获取下载任务状态
	//tasks, err := client.TellActive("gid", "status", "downloadSpeed")
	//if err != nil {
	//	t.Log(err)
	//	t.Log(string(debug.Stack()))
	//}
	//
	//for _, task := range tasks {
	//	t.Log(task)
	//}

	// 获取下载任务状态
	tasks, err := client.TellStopped(0, 100, "gid", "status", "infoHash", "files", "bittorrent", "errorCode", "errorMessage")
	if err != nil {
		t.Log(err)
		t.Log(string(debug.Stack()))
	}

	t.Log(len(tasks))

	for _, task := range tasks {
		if task.Status == arigo.StatusError {
			if task.ErrorCode == arigo.CouldNotOpenExistingFile {
				if strings.Contains(task.ErrorMessage, ErrorFileNameTooLong) {
					downloadUrl := util.BuildMagnetLink(task.InfoHash)
					t.Logf("重新提交下载任务：%s", downloadUrl)

					err = client.RemoveDownloadResult(task.GID)
					if err != nil {
						t.Logf("删除任务失败1：%s", err.Error())
						return
					}

					//err := client.Remove(task.GID)
					//if err != nil {
					//	t.Logf("删除任务失败2：%s", err.Error())
					//}

					options, err := client.GetGlobalOptions()
					if err != nil {
						t.Logf("查询全局异常：%s", err.Error())
						return
					}

					saveDir := options.Dir + "/error-retry/" + util.NowDate("-")
					if selectIndex, ok, _ := downloadFileBestSelect(task.Files); ok {
						t.Logf("优选文件：%s", selectIndex)
						if _, err = client.AddURI(arigo.URIs(downloadUrl), &arigo.Options{
							Dir:        saveDir,
							SelectFile: selectIndex,
						}); err != nil {
							t.Logf("重新添加aria2下载任务异常: [%s] - %s \n", downloadUrl, err.Error())
						}
					}

					break

					//// 先删除任务
					//err = client.RemoveDownloadResult(task.GID)
					//if err != nil {
					//	t.Log(err)
					//	continue
					//}
					//
					//downloadUrl := util.BuildMagnetLink(task.InfoHash)
					//t.Log(downloadUrl)
					//saveDir := options.Dir + "/test/"
					//if _, err = client.AddURI(arigo.URIs(downloadUrl), &arigo.Options{
					//	Dir:            saveDir,
					//	BTMetadataOnly: true,
					//	FollowTorrent:  true,
					//}); err != nil {
					//	t.Logf("添加aria2下载任务异常: %s \n", err.Error())
					//}
				}
			}
		}
	}

	//
	//var testTask arigo.Status
	//
	//for _, task := range tasks {
	//	if task.Status == arigo.StatusError {
	//		if "3cd7c67e5256eb5a" == task.GID {
	//			testTask = task
	//			break
	//		}
	//	}
	//}

	// 20
	// 333
	// 244
	// GID: 3cd7c67e5256eb5a
	// infoHash: c68a431a59c63ea2332fa24c1f0cd219f685a336
	//t.Log(testTask.GID, testTask.Status, testTask.ErrorCode, testTask.InfoHash)
}
