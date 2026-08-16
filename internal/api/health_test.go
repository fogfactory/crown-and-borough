package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReadinessHandlerReturnsReadyWhenChecksPass(t *testing.T) {
	called := false
	server := ReadinessHandler(func(ctx context.Context) error {
		called = true
		if _, ok := ctx.Deadline(); !ok {
			t.Error("readiness check has no deadline")
		}
		return nil
	})

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz/ready", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /healthz/ready = %d, want %d", recorder.Code, http.StatusOK)
	}
	if !called {
		t.Fatal("readiness check was not called")
	}
	if got := recorder.Body.String(); !strings.Contains(got, `"status":"ready"`) {
		t.Errorf("readiness response = %q, want ready status", got)
	}
}

func TestReadinessHandlerHidesDependencyError(t *testing.T) {
	secret := errors.New("service-account private key: do-not-return")
	server := ReadinessHandler(func(context.Context) error { return secret })

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz/ready", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /healthz/ready = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"error":"not_ready"`) {
		t.Errorf("readiness response = %q, want not_ready", body)
	}
	if strings.Contains(body, secret.Error()) {
		t.Errorf("readiness response leaked dependency error: %q", body)
	}
}

func TestReadinessHandlerTimesOutChecks(t *testing.T) {
	server := readinessHandler(10*time.Millisecond, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz/ready", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /healthz/ready timeout = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}

func TestReadinessHandlerRequiresAConfiguredCheck(t *testing.T) {
	server := ReadinessHandler()

	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/healthz/ready", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /healthz/ready without checks = %d, want %d", recorder.Code, http.StatusServiceUnavailable)
	}
}
