package server

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Browser requests must originate from this server. Reject foreign Host headers
// to prevent DNS rebinding, even when the socket is loopback-only.
func localBoundary(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			writeError(w, 403, "Player2 accepts loopback hosts only")
			return
		}
		// A browser must not embed the local app or silently call its API from another site.
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self' ws://"+r.Host+"; object-src 'none'; base-uri 'none'; frame-ancestors 'none'; form-action 'self'")
		site := r.Header.Get("Sec-Fetch-Site")
		if strings.HasPrefix(r.URL.Path, "/api/") && site != "" && site != "none" && site != "same-origin" {
			writeError(w, 403, "Open Player2 directly to use its local API")
			return
		}
		if !sameOrigin(r) {
			writeError(w, 403, "cross-origin requests are not allowed")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			limit := int64(1 << 20)
			if r.Method == "POST" && r.URL.Path == "/api/recovery/preview" {
				limit = 32 << 20
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return err == nil && u.Host == r.Host && u.Scheme == scheme && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == ""
}
