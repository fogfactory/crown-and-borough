package main

import (
	"net/http"
	"testing"
	"time"
)

func TestNewHTTPServerTimeouts(t *testing.T) {
	s := newHTTPServer(":0", http.NewServeMux())
	if s.ReadHeaderTimeout <= 0 || s.ReadTimeout <= 0 || s.WriteTimeout <= 0 || s.IdleTimeout <= 0 || s.MaxHeaderBytes <= 0 {
		t.Fatalf("missing limit: %+v", s)
	}
}

func TestRateLimitFromEnv(t *testing.T) {
	t.Setenv("K_SERVICE", "")
	t.Setenv("RATE_LIMIT_REQUESTS", "")
	t.Setenv("RATE_LIMIT_WINDOW", "")
	t.Setenv("TRUSTED_PROXY_HOPS", "")
	cfg := rateLimitFromEnv()
	if cfg.Requests != 120 || cfg.Window != time.Minute || cfg.TrustedProxyHops != 0 {
		t.Fatalf("defaults: %+v", cfg)
	}
	t.Setenv("K_SERVICE", "svc")
	t.Setenv("RATE_LIMIT_REQUESTS", "5")
	t.Setenv("RATE_LIMIT_WINDOW", "10s")
	cfg = rateLimitFromEnv()
	if cfg.Requests != 5 || cfg.Window != 10*time.Second || cfg.TrustedProxyHops != 1 {
		t.Fatalf("env: %+v", cfg)
	}
}
