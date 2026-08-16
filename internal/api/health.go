package api

import (
	"context"
	"net/http"
	"time"
)

const readinessTimeout = 3 * time.Second

// ReadinessCheck is a dependency check used by /healthz/ready. Checks must
// honor the context so a stalled dependency cannot hold the probe open.
type ReadinessCheck func(context.Context) error

// ReadinessHandler reports whether all configured dependencies are ready. The
// response intentionally contains no dependency error because it is public on
// a Cloud Run service and must not disclose configuration or credentials.
func ReadinessHandler(checks ...ReadinessCheck) http.HandlerFunc {
	return readinessHandler(readinessTimeout, checks...)
}

func readinessHandler(timeout time.Duration, checks ...ReadinessCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		ready := false
		for _, check := range checks {
			if check == nil {
				continue
			}
			ready = true
			if err := check(ctx); err != nil {
				ready = false
				break
			}
		}
		if !ready {
			writeAPIError(w, http.StatusServiceUnavailable, "not_ready", "service is not ready")
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Status string `json:"status"`
		}{Status: "ready"})
	}
}
