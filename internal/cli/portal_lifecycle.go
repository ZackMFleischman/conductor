package cli

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

func runPortalServer(ctx context.Context, server *http.Server, listener net.Listener) error {
	done := make(chan struct{})
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		select {
		case <-ctx.Done():
			shutdown, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if server.Shutdown(shutdown) != nil {
				server.Close()
			}
		case <-done:
		}
	}()
	err := server.Serve(listener)
	close(done)
	if !errors.Is(err, http.ErrServerClosed) {
		server.Close()
	}
	<-drained
	return err
}
