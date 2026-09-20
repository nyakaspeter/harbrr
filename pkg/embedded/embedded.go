// Package embedded exposes harbrr's daemon lifecycle to another Go application.
package embedded

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/autobrr/harbrr/internal/app"
	"github.com/autobrr/harbrr/internal/config"
	"github.com/autobrr/harbrr/internal/logger"
)

// Options configures an embedded harbrr server.
type Options struct {
	Host     string
	Port     int
	DataDir  string
	LogLevel string
	Log      io.Writer
}

// Server is a running embedded harbrr instance.
type Server struct {
	address string
	cancel  context.CancelFunc
	done    chan struct{}

	mu  sync.Mutex
	err error
}

// Start constructs harbrr, reserves its listener, and starts serving. Binding
// happens synchronously, so a successful return means the advertised address
// belongs to this instance.
func Start(parent context.Context, options Options) (*Server, error) {
	cfg, output, err := runtimeConfig(options)
	if err != nil {
		return nil, err
	}

	var listenConfig net.ListenConfig
	address := net.JoinHostPort(cfg.Server.Host, strconv.Itoa(cfg.Server.Port))
	listener, err := listenConfig.Listen(parent, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("embedded harbrr: listen %s: %w", address, err)
	}

	log := logger.New(cfg.Log, output)
	if err := logger.SetLevel(cfg.Log.Level); err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("embedded harbrr: %w", err)
	}

	application, err := app.New(parent, app.Deps{Config: cfg, Logger: log})
	if err != nil {
		_ = listener.Close()
		return nil, fmt.Errorf("embedded harbrr: construct: %w", err)
	}

	runContext, cancel := context.WithCancel(parent)
	server := &Server{
		address: "http://" + address,
		cancel:  cancel,
		done:    make(chan struct{}),
	}
	go func() {
		defer close(server.done)
		err := application.RunListener(runContext, listener)
		server.mu.Lock()
		server.err = err
		server.mu.Unlock()
	}()
	return server, nil
}

// Address returns the HTTP origin advertised for this instance.
func (server *Server) Address() string { return server.address }

// Done is closed after the daemon and its background workers stop.
func (server *Server) Done() <-chan struct{} { return server.done }

// Wait blocks until shutdown completes and returns the daemon's terminal error.
func (server *Server) Wait() error {
	<-server.done
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.err
}

// Stop gracefully stops the daemon and waits for all background work to finish.
func (server *Server) Stop() error {
	server.cancel()
	return server.Wait()
}

func runtimeConfig(options Options) (*config.Config, io.Writer, error) {
	cfg := config.Defaults()
	cfg.Server.Host = options.Host
	cfg.Server.Port = options.Port
	cfg.DataDir = options.DataDir
	cfg.Log.Format = "console"
	if options.LogLevel != "" {
		cfg.Log.Level = options.LogLevel
	}
	if err := cfg.Validate(); err != nil {
		return nil, nil, fmt.Errorf("embedded harbrr: %w", err)
	}
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, nil, fmt.Errorf("embedded harbrr: create data directory: %w", err)
	}
	output := options.Log
	if output == nil {
		output = io.Discard
	}
	return &cfg, output, nil
}

// IsStopped reports whether err only reflects cancellation of the parent
// context. It is useful to callers that monitor unexpected daemon exits.
func IsStopped(err error) bool {
	return err == nil || errors.Is(err, context.Canceled)
}
