package respond

import (
	"encoding/json"
	"github.com/nekoimi/get-magnet/internal/pkg/error_ext"
	"log"
	"net/http"
)

type JsonResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg,omitempty"`
	Data any    `json:"data,omitempty"`
}

func sendJsonResponse(w http.ResponseWriter, response JsonResponse) {
	bs, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Marshal json error: %s\n", err.Error())
		return
	}

	_, err = w.Write(bs)
	if err != nil {
		log.Printf("Send json response error: %s\n", err.Error())
	}
}

func Ok(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	sendJsonResponse(w, JsonResponse{
		Code: 0,
		Msg:  "OK",
		Data: data,
	})
}

func Error(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp := JsonResponse{
		Code: 500,
		Msg:  err.Error(),
	}
	switch err.(type) {
	case error_ext.CodeError:
		ext := err.(error_ext.CodeError)
		w.WriteHeader(ext.GetHttpStatus())
		resp.Code = ext.GetCode()
		resp.Msg = ext.Error()
	case error:
		w.WriteHeader(http.StatusInternalServerError)
		resp.Code = http.StatusInternalServerError
		resp.Msg = err.Error()
	}
	sendJsonResponse(w, resp)
}
