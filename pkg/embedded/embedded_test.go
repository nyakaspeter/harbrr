package embedded_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/autobrr/harbrr/pkg/embedded"
)

func TestStartServeAndStop(t *testing.T) {
	port := freePort(t)
	server, err := embedded.Start(context.Background(), embedded.Options{
		Host: "127.0.0.1", Port: port, DataDir: t.TempDir(), Log: io.Discard,
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	client := &http.Client{Timeout: time.Second}
	var response *http.Response
	for range 50 {
		request, requestErr := http.NewRequestWithContext(t.Context(), http.MethodGet, server.Address()+"/healthz", nil)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		response, err = client.Do(request)
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want %d", response.StatusCode, http.StatusOK)
	}

	if err := server.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestStartReportsOccupiedPort(t *testing.T) {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	port := listener.Addr().(*net.TCPAddr).Port

	_, err = embedded.Start(context.Background(), embedded.Options{
		Host: "127.0.0.1", Port: port, DataDir: t.TempDir(), Log: io.Discard,
	})
	if err == nil {
		t.Fatal("Start() error = nil, want occupied-port error")
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}
