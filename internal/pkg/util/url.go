package util

import (
	log "github.com/sirupsen/logrus"
	"net/url"
)

const MagnetLinkPrefix = "magnet:?xt=urn:btih:"

func BuildMagnetLink(infoHash string) string {
	return MagnetLinkPrefix + infoHash
}

func JoinUrl(base string, urls ...string) string {
	path, err := url.JoinPath(base, urls...)
	if err != nil {
		panic(err)
	}
	decodeUrl, err := url.QueryUnescape(path)
	if err != nil {
		panic(err)
	}
	return decodeUrl
}

func CleanHost(host string) string {
	u, err := url.Parse(host)
	if err != nil {
		log.Errorf("URL parse (%s) err: %s", host, err.Error())
		return ""
	}

	return u.Host
}
