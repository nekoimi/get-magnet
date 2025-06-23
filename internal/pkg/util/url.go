package util

import (
	log "github.com/sirupsen/logrus"
	"net/url"
)

func CleanHost(host string) string {
	u, err := url.Parse(host)
	if err != nil {
		log.Errorf("URL parse (%s) err: %s \n", host, err.Error())
		return ""
	}

	return u.Host
}
