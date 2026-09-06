package forwarder

import (
	"github.com/q191201771/lal/pkg/httpflv"
	"testing"
	"time"
)

func frame(kind uint8, ts uint32) httpflv.Tag {
	return httpflv.Tag{Header: httpflv.TagHeader{Type: kind, Timestamp: ts}, Raw: []byte{1}}
}

func TestPacerDoesNotDoubleUpstreamDelay(t *testing.T) {
	p := flvPacer{started: true, mediaStart: 0, wallStart: time.Now().Add(-time.Second)}
	start := time.Now()
	p.write(frame(9, 500), make(chan struct{}), func([]byte) {})
	if time.Since(start) > 100*time.Millisecond {
		t.Fatal("added delay to already paced stream")
	}
}

func TestPacerPreservesInterleavedTracks(t *testing.T) {
	p := flvPacer{started: true, wallStart: time.Now().Add(-time.Second)}
	count := 0
	write := func([]byte) { count++ }
	stop := make(chan struct{})
	p.write(frame(9, 100), stop, write)
	p.write(frame(8, 80), stop, write)
	p.write(frame(9, 90), stop, write)
	if count != 2 {
		t.Fatalf("wanted audio and video, got %d frames", count)
	}
}

func TestPacerCancellation(t *testing.T) {
	p := flvPacer{started: true, wallStart: time.Now()}
	stop := make(chan struct{})
	close(stop)
	if p.write(frame(9, 1000), stop, func([]byte) { t.Fatal("wrote cancelled frame") }) {
		t.Fatal("not cancelled")
	}
}
