package api

import "net/http"

// OriginGuard rejects browser requests whose Origin is not in allowed. This is
// the MCP spec's one MUST for local HTTP transport and the real defense against
// DNS-rebinding attacks (a page on an attacker origin scripting the loopback
// server) — independent of whether TLS is on.
//
// Requests with no Origin header pass through: that covers native MCP clients
// (Claude Code), curl, healthchecks, and same-origin browser navigations, none
// of which set Origin. Only a present-and-unlisted Origin is blocked (403). An
// empty allow-list therefore blocks every browser origin while still serving
// native clients — a safe closed default.
func OriginGuard(allowed []string) func(http.Handler) http.Handler {
	set := make(map[string]struct{}, len(allowed))
	for _, o := range allowed {
		set[o] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" {
				if _, ok := set[origin]; !ok {
					http.Error(w, "forbidden origin", http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
