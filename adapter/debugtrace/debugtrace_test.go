package debugtrace

import (
	"net"
	"strings"
	"testing"
)

func TestListenWithFallbackMovesWhenDefaultPortBusy(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen busy port: %v", err)
	}
	defer busy.Close()

	listener, actualAddr, err := listenWithFallback(busy.Addr().String())
	if err != nil {
		t.Fatalf("listenWithFallback: %v", err)
	}
	defer listener.Close()

	if actualAddr == busy.Addr().String() {
		t.Fatalf("fallback should move away from busy addr %s", actualAddr)
	}
	if !strings.HasPrefix(actualAddr, "127.0.0.1:") {
		t.Fatalf("fallback addr should stay local, got %s", actualAddr)
	}
}
