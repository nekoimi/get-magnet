package util

import (
	"encoding/json"
	"errors"
	"log"
)

func ToJson(v any) string {
	bs, err := json.Marshal(v)
	if err != nil {
		log.Println(errors.New("ToJson err: " + err.Error()))
		return ""
	}

	return string(bs)
}
