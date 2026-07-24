package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nekoimi/get-magnet/internal/config"
)

func TestQuickAPIOptionalToken(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := AuthMiddleware(&config.Config{QuickAPI: &config.QuickAPIConfig{Token: "expected"}})(next)

	withoutToken := httptest.NewRecorder()
	handler.ServeHTTP(withoutToken, httptest.NewRequest(http.MethodPost, "/quick-api/download", nil))
	if withoutToken.Code != http.StatusUnauthorized {
		t.Fatalf("without token status = %d; want %d", withoutToken.Code, http.StatusUnauthorized)
	}

	withToken := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/quick-api/download", nil)
	req.Header.Set("X-Quick-API-Token", "expected")
	handler.ServeHTTP(withToken, req)
	if withToken.Code != http.StatusNoContent {
		t.Fatalf("with token status = %d; want %d", withToken.Code, http.StatusNoContent)
	}
}

func TestQuickAPIRemainsOpenWhenTokenIsNotConfigured(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	handler := AuthMiddleware(&config.Config{})(next)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/quick-api/download", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d; want %d", recorder.Code, http.StatusNoContent)
	}
}
