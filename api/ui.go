package api

import "net/http"

// AdminUI display admin ui
func AdminUI(uiStaticDir string) http.Handler {
	return http.StripPrefix("/", http.FileServer(http.Dir(uiStaticDir)))
}

// Aria2WebUI display aria2 web ui
func Aria2WebUI(uiAriaNgDir string) http.Handler {
	return http.StripPrefix("/ui/aria-ng/", http.FileServer(http.Dir(uiAriaNgDir)))
}
