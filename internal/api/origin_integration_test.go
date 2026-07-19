package api_test

import (
	"net/http"
	"testing"
)

// The Router actually enforces OriginGuard: a foreign Origin is refused before
// reaching any handler, while an allowed Origin and no-Origin (native clients,
// healthchecks) reach /health normally. newServer wires CORSOrigins to
// http://localhost:5173.
func TestRouterEnforcesOriginGuard(t *testing.T) {
	srv, _ := newServer(t)

	cases := []struct {
		name   string
		origin string
		want   int
	}{
		{"no origin passes", "", http.StatusOK},
		{"allowed origin passes", "http://localhost:5173", http.StatusOK},
		{"foreign origin blocked", "http://attacker.example", http.StatusForbidden},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, srv.URL+"/health", nil)
			if err != nil {
				t.Fatal(err)
			}
			if tc.origin != "" {
				req.Header.Set("Origin", tc.origin)
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("GET /health: %v", err)
			}
			resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Errorf("origin %q: got %d, want %d", tc.origin, resp.StatusCode, tc.want)
			}
		})
	}
}
