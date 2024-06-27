package util

import (
	"log"
	"net/url"
)

func CleanHost(host string) string {
	u, err := url.Parse(host)
	if err != nil {
		log.Printf("URL parse (%s) err: %s \n", host, err.Error())
		return ""
	}

	return u.Host
}
