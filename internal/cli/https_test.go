package cli

import (
	"strings"
	"testing"
)

func TestHostsHasMnemeEntry(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"canonical entry", "127.0.0.1\tmneme.dev\n", true},
		{"entry with extra hosts", "127.0.0.1 localhost mneme.dev foo\n", true},
		{"spaced entry", "  127.0.0.1   mneme.dev\n", true},
		{"absent", "127.0.0.1 localhost\n::1 localhost\n", false},
		{"substring but wrong ip", "10.0.0.1 mneme.dev\n", false},
		{"commented out", "# 127.0.0.1 mneme.dev\n", false},
		{"different host same prefix", "127.0.0.1 mneme.development\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hostsHasMnemeEntry(tc.content); got != tc.want {
				t.Errorf("hostsHasMnemeEntry(%q) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}

func TestMkcertLeafArgs(t *testing.T) {
	args := mkcertLeafArgs("/c/cert.pem", "/c/key.pem")
	got := strings.Join(args, " ")
	want := "-cert-file /c/cert.pem -key-file /c/key.pem mneme.dev localhost 127.0.0.1"
	if got != want {
		t.Errorf("mkcertLeafArgs:\n got %q\nwant %q", got, want)
	}
}
