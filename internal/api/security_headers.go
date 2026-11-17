package api

import (
	"net/http"
	"os"
	"strings"
)

// Content Security Policy sources. Fonts and images are served by the
// application itself; the web client only talks to its own API, Firebase
// (Auth and Firestore) and the public GitHub release API.
//
// Every wildcard is a host wildcard scoped to a Google-operated domain, and
// there is no 'unsafe-eval'. 'unsafe-inline' remains in style-src only: React
// inline style attributes and the SVG map need it, and it is tracked in
// specs/architecture.md.
var contentSecurityPolicyDirectives = []string{
	"default-src 'self'",
	"script-src 'self' https://apis.google.com",
	"style-src 'self' 'unsafe-inline'",
	"img-src 'self' data: blob:",
	"font-src 'self' data:",
	"connect-src 'self' https://*.googleapis.com https://*.firebaseapp.com https://api.github.com",
	"frame-src 'self' https://*.firebaseapp.com",
	"object-src 'none'",
	"base-uri 'self'",
	"frame-ancestors 'self'",
}

// loopbackSources are the local Firebase emulators, only allowed outside
// production.
const loopbackSources = "http://localhost:* http://127.0.0.1:*"

// ContentSecurityPolicy returns the policy enforced on every response. Outside
// production it also allows the Firebase emulators on the loopback interface.
func ContentSecurityPolicy(production bool) string {
	directives := make([]string, len(contentSecurityPolicyDirectives))
	copy(directives, contentSecurityPolicyDirectives)
	if !production {
		for i, directive := range directives {
			if strings.HasPrefix(directive, "connect-src ") || strings.HasPrefix(directive, "frame-src ") {
				directives[i] = directive + " " + loopbackSources
			}
		}
	}
	return strings.Join(directives, "; ")
}

// IsProduction reports whether the server runs against the real Firebase
// services: neither the development mode nor a Firebase emulator is in use.
func IsProduction(developmentMode bool) bool {
	return !developmentMode &&
		os.Getenv("FIRESTORE_EMULATOR_HOST") == "" &&
		os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") == ""
}

// WithSecurityHeaders sets the browser hardening headers on every response,
// errors and preflights included. In production (HTTPS) it also sends HSTS and
// keeps the Content Security Policy strict; elsewhere HSTS is left out and the
// policy allows the local Firebase emulators.
func WithSecurityHeaders(next http.Handler, production bool) http.Handler {
	policy := ContentSecurityPolicy(production)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=()")
		h.Set("X-Frame-Options", "SAMEORIGIN")
		h.Set("Content-Security-Policy", policy)
		if production {
			h.Set("Strict-Transport-Security", "max-age=31536000")
		}
		next.ServeHTTP(w, r)
	})
}
