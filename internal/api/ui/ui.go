package ui

import "net/http"

// AdminUI display admin ui
func AdminUI(uiStaticDir string) http.Handler {
	return http.StripPrefix("/", http.FileServer(http.Dir(uiStaticDir)))
}
