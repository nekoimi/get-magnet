package response

import (
	"encoding/json"
	"github.com/nekoimi/get-magnet/pkg/error_ext"
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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err = w.Write(bs)
	if err != nil {
		log.Printf("Send json response error: %s\n", err.Error())
	}
}

func Ok(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	sendJsonResponse(w, JsonResponse{
		Code: 0,
		Msg:  "OK",
	})
}

func OkData(w http.ResponseWriter, data any) {
	w.WriteHeader(http.StatusOK)
	sendJsonResponse(w, JsonResponse{
		Code: 0,
		Msg:  "OK",
		Data: data,
	})
}

func ValidateError(w http.ResponseWriter, err error_ext.Error) {
	w.WriteHeader(http.StatusBadRequest)
	sendJsonResponse(w, JsonResponse{
		Code: err.Code(),
		Msg:  err.Msg(),
	})
}

func Error(w http.ResponseWriter, err error_ext.Error) {
	w.WriteHeader(http.StatusInternalServerError)
	sendJsonResponse(w, JsonResponse{
		Code: err.Code(),
		Msg:  err.Msg(),
	})
}
