package api

import "net/http"

// contentSecurityPolicy is deployed in report-only mode first: it lists the
// Firebase Auth and Firestore endpoints the web client talks to, and allows
// inline styles because the map SVG is styled inline. Framing is limited to
// the same origin (not denied outright) so that Firebase Auth can embed its
// own handler page when the auth domain is the application domain.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self' https://apis.google.com; " +
	"style-src 'self' 'unsafe-inline'; " +
	"img-src 'self' data: blob:; " +
	"font-src 'self' data:; " +
	"connect-src 'self' https://*.googleapis.com https://*.firebaseapp.com; " +
	"frame-src 'self' https://*.firebaseapp.com; " +
	"object-src 'none'; base-uri 'self'; frame-ancestors 'self'"

// WithSecurityHeaders sets the browser hardening headers on every response.
// HSTS is only sent when hsts is true (production, served over HTTPS).
func WithSecurityHeaders(next http.Handler, hsts bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Content-Security-Policy-Report-Only", contentSecurityPolicy)
		if hsts {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}
