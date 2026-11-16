package api

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimitConfig configures the in-memory limiter of mutating routes.
type RateLimitConfig struct {
	// Requests is the burst size and the number of requests refilled per
	// Window. Zero or less disables the limiter.
	Requests int
	// Window is the time over which Requests tokens are refilled.
	Window time.Duration
	// TrustedProxyHops is the number of reverse proxies in front of the server
	// that append to X-Forwarded-For (1 behind Cloud Run). With 0 the header is
	// ignored and the peer address is used.
	TrustedProxyHops int
	// Now overrides the clock in tests.
	Now func() time.Time
}

type bucket struct {
	tokens float64
	last   time.Time
}

type rateLimiter struct {
	cfg     RateLimitConfig
	mu      sync.Mutex
	buckets map[string]*bucket
	sweep   time.Time
}

// WithRateLimit answers 429 with a Retry-After header to clients that exceed
// a token bucket per client address on authentication routes and on every
// mutating API request. Reads, health checks and static files are never
// limited.
func WithRateLimit(next http.Handler, cfg RateLimitConfig) http.Handler {
	if cfg.Requests <= 0 || cfg.Window <= 0 {
		return next
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	limiter := &rateLimiter{cfg: cfg, buckets: map[string]*bucket{}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rateLimited(r) {
			next.ServeHTTP(w, r)
			return
		}
		if wait, ok := limiter.allow(ClientIP(r, cfg.TrustedProxyHops)); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
			writeAPIError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func rateLimited(r *http.Request) bool {
	path := r.URL.Path
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return strings.HasPrefix(path, "/api/auth/")
	}
	return strings.HasPrefix(path, "/api/")
}

func (l *rateLimiter) allow(key string) (time.Duration, bool) {
	now := l.cfg.Now()
	capacity := float64(l.cfg.Requests)
	rate := capacity / l.cfg.Window.Seconds()
	l.mu.Lock()
	defer l.mu.Unlock()
	if now.Sub(l.sweep) > l.cfg.Window {
		for k, b := range l.buckets {
			if now.Sub(b.last) > l.cfg.Window {
				delete(l.buckets, k)
			}
		}
		l.sweep = now
	}
	b := l.buckets[key]
	if b == nil {
		b = &bucket{tokens: capacity, last: now}
		l.buckets[key] = b
	}
	b.tokens = math.Min(capacity, b.tokens+now.Sub(b.last).Seconds()*rate)
	b.last = now
	if b.tokens < 1 {
		return time.Duration((1 - b.tokens) / rate * float64(time.Second)), false
	}
	b.tokens--
	return 0, true
}

// ClientIP returns the address of the client. X-Forwarded-For is client
// controlled except for the entries appended by our own proxies, so only the
// entry that many hops from the right is trusted; any shortfall falls back to
// the peer address.
func ClientIP(r *http.Request, trustedHops int) string {
	if trustedHops > 0 {
		var hops []string
		for _, value := range r.Header.Values("X-Forwarded-For") {
			for _, part := range strings.Split(value, ",") {
				hops = append(hops, strings.TrimSpace(part))
			}
		}
		if len(hops) >= trustedHops {
			if ip := net.ParseIP(hops[len(hops)-trustedHops]); ip != nil {
				return ip.String()
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
