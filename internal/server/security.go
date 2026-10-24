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
		if !sameOrigin(r) {
			writeError(w, 403, "cross-origin requests are not allowed")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
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
	return err == nil && u.Host == r.Host && (u.Scheme == "http" || u.Scheme == "https") && u.User == nil && u.Path == ""
}
