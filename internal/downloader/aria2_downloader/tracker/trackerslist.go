package tracker

import (
	"bufio"
	"errors"
	"fmt"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strings"
	"time"
)

const DownloadUrl = "https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_all.txt"

var httpClient = &http.Client{
	Timeout: 10 * time.Second,
}

func FetchTrackers() string {
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

	if trackers == nil || len(trackers) == 0 {
		// ignore
		return ""
	}

	btTracker := strings.Join(trackers, ",")
	log.Debugf("最新的tracker服务器信息：%s", btTracker)
	return btTracker
}

func downloadLatestTrackers() ([]string, error) {
	log.Debugln("下载最新的tracker服务器信息...")

	req, err := http.NewRequest("GET", DownloadUrl, nil)
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
