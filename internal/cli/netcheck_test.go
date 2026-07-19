package cli

import (
	"net"
	"strconv"
	"testing"
)

// A free preferred port is returned as-is and is bindable.
func TestAvailablePortReturnsFree(t *testing.T) {
	// Grab a free port from the OS, release it, then ask for it.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())
	ln.Close()

	got := availablePort("127.0.0.1", portStr)
	// The exact port may race with other processes, but the result must bind.
	l2, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", got))
	if err != nil {
		t.Fatalf("returned port %q is not bindable: %v", got, err)
	}
	l2.Close()
}

// A busy preferred port is skipped for a different, bindable one.
func TestAvailablePortSkipsBusy(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	_, portStr, _ := net.SplitHostPort(ln.Addr().String())

	got := availablePort("127.0.0.1", portStr)
	if got == portStr {
		t.Fatalf("busy port %q should have been skipped", portStr)
	}
	if n, _ := strconv.Atoi(got); n <= 0 {
		t.Fatalf("unexpected port %q", got)
	}
	// And the chosen one is actually free.
	l2, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", got))
	if err != nil {
		t.Fatalf("chosen port %q not bindable: %v", got, err)
	}
	l2.Close()
}

// A non-numeric port is passed through untouched (the server surfaces the error).
func TestAvailablePortNonNumericPassthrough(t *testing.T) {
	if got := availablePort("127.0.0.1", "not-a-port"); got != "not-a-port" {
		t.Errorf("non-numeric port should pass through: got %q", got)
	}
}
