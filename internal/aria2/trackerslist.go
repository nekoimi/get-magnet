package aria2

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/siku2/arigo"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const TrackerDownloadUrl = "https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all.txt"

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func upgradeTrackers(a *Aria2) {
	var (
		err      error
		trackers []string
		retryNum = 0
	)

	for {
		if retryNum > 3 {
			log.Errorln("too many downloads latest trackers error")
			break
		}
		trackers, err = downloadLatestTrackers()
		if err != nil {
			log.Errorln(err.Error())
			time.Sleep(1 * time.Second)
			retryNum++
			continue
		}

		break
	}

	if len(trackers) == 0 {
		// ignore
		return
	}

	btTracker := strings.Join(trackers, ",")
	if err = a.client().ChangeGlobalOptions(arigo.Options{
		BTTracker: btTracker,
	}); err != nil {
		log.Errorf("更新aria2最新tracker服务器信息异常：%s", err.Error())
	}
}

func downloadLatestTrackers() ([]string, error) {
	log.Debugln("下载最新的tracker服务器信息...")

	req, err := http.NewRequest("GET", TrackerDownloadUrl, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	buf := bufio.NewReader(resp.Body)
	var res []string
	for {
		line, _, err := buf.ReadLine()
		if err != nil {
			break
		}

		if len(line) == 0 {
			continue
		}

		res = append(res, string(line))
	}

	if resp.StatusCode == 200 {
		return res, nil
	}

	return nil, errors.New(fmt.Sprintf("download latest trackers error, status code - %d, resp: %s", resp.StatusCode, strings.Join(res, "\n")))
}
