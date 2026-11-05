package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithCORSOptions(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodOptions, "/api/games", nil)

	WithCORS(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Errorf("OPTIONS /api/games = %d, want %d", recorder.Code, http.StatusNoContent)
	}
	if nextCalled {
		t.Error("OPTIONS request reached the wrapped handler")
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("OPTIONS response body = %q, want empty", recorder.Body.String())
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != developmentOrigin {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, developmentOrigin)
	}
}
