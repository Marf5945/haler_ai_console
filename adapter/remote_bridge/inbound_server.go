package remote_bridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	"ui_console/shared/eventbus"
)

// InboundServer accepts local webhook callbacks and emits reduced command
// events. Public internet exposure is intentionally left to a user-chosen
// tunnel/relay, so the app remains portable across machines.
type InboundServer struct {
	mu       sync.Mutex
	service  *Service
	eventBus *eventbus.Bus
	server   *http.Server
	addr     string
}

func NewInboundServer(service *Service, bus *eventbus.Bus) *InboundServer {
	return &InboundServer{service: service, eventBus: bus}
}

func (s *InboundServer) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.addr != "" {
		return nil
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/remote-bridge/inbound/", s.handleInbound)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	s.addr = listener.Addr().String()
	s.server = &http.Server{Handler: mux}
	go func() {
		if err := s.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("remote bridge inbound server stopped: %v\n", err)
		}
	}()
	return nil
}

func (s *InboundServer) URLForChannel(channelID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.addr == "" || strings.TrimSpace(channelID) == "" {
		return ""
	}
	return fmt.Sprintf("http://%s/remote-bridge/inbound/%s", s.addr, channelID)
}

func (s *InboundServer) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addr
}

func (s *InboundServer) handleInbound(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	channelID := strings.TrimPrefix(r.URL.Path, "/remote-bridge/inbound/")
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		http.Error(w, "missing channel id", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "read body failed", http.StatusBadRequest)
		return
	}
	result, err := s.service.HandleInboundHTTPRequest(channelID, r.Header, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, event := range result.Events {
		s.eventBus.Emit(EventInboundCommand, event)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":     true,
		"events": len(result.Events),
	})
}
