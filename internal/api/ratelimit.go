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

// RateLimitConfig configures the in-memory limiter of costly and mutating
// routes.
type RateLimitConfig struct {
	// Requests is the burst size and the number of requests refilled per
	// Window, for each client and route class. Zero or less disables the
	// limiter.
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
// a token bucket on authentication routes, on the order preview and victory
// simulation routes, and on every other mutating API request. Each of these
// three classes has its own bucket per client, so previews never starve the
// submission of orders. Reads, health checks and static files are never
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
		class := rateClass(r)
		if class == "" {
			next.ServeHTTP(w, r)
			return
		}
		if wait, ok := limiter.allow(class + "|" + clientIP(r, cfg.TrustedProxyHops)); !ok {
			w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
			writeAPIError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// rateClass names the bucket a request draws from, or "" when it is not
// limited.
func rateClass(r *http.Request) string {
	path := r.URL.Path
	switch {
	case strings.HasPrefix(path, "/api/auth/"):
		return "auth"
	case !strings.HasPrefix(path, "/api/"):
		return ""
	case r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions:
		return ""
	case strings.HasSuffix(path, "/orders/preview") || strings.HasSuffix(path, "/victory/simulate"):
		return "compute"
	}
	return "write"
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

// clientIP returns the address that identifies the client. X-Forwarded-For is
// client controlled except for the entries appended by our own proxies, so
// only the entry that many hops from the right is trusted; any shortfall falls
// back to the peer address. IPv6 clients are keyed by their /64 network: a
// single subscriber owns that whole range and could otherwise mint unlimited
// identities.
func clientIP(r *http.Request, trustedHops int) string {
	if trustedHops > 0 {
		hops := strings.Split(strings.Join(r.Header.Values("X-Forwarded-For"), ","), ",")
		if len(hops) >= trustedHops {
			if ip := net.ParseIP(strings.TrimSpace(hops[len(hops)-trustedHops])); ip != nil {
				return networkKey(ip)
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	if ip := net.ParseIP(host); ip != nil {
		return networkKey(ip)
	}
	return host
}

func networkKey(ip net.IP) string {
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	return ip.Mask(net.CIDRMask(64, 128)).String()
}
