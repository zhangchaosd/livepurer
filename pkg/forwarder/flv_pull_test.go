package forwarder

import (
	"github.com/q191201771/lal/pkg/httpflv"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestShutdownCancelsPendingHeaders(t *testing.T) {
	arrived := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(arrived); <-r.Context().Done() }))
	defer server.Close()
	in := &Flv{}
	done := make(chan error, 1)
	go func() { done <- in.Pull(server.URL, func(httpflv.Tag) {}) }()
	select {
	case <-arrived:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	in.Shutdown()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not cancel pending request")
	}
}
