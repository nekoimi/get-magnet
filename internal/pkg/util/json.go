package util

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
)

func ToJson(v any) string {
	bs, err := json.Marshal(v)
	if err != nil {
		log.Errorf("tojson err: %s", err.Error())
		return ""
	}

	return string(bs)
}
