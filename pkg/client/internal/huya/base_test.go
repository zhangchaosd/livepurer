package huya

import (
	"net/url"
	"testing"
	"time"
)

func TestHuyaPlayURLRequestsH264(t *testing.T) {
	u, err := url.Parse("https://tx.flv.huya.com/src/example.flv?ratio=4000")
	if err != nil {
		t.Fatal(err)
	}
	configurePlayURL(u)

	if got := u.Query().Get("codec"); got != "264" {
		t.Fatalf("codec = %q, want 264", got)
	}
	if got := u.Query().Get("ratio"); got != "0" {
		t.Fatalf("ratio = %q, want 0", got)
	}
}

func TestRefreshAntiCode(t *testing.T) {
	u, err := url.Parse("https://tx.flv.huya.com/src/123-456.flv?fm=RFdxOEJjSjNoNkRKdDZUWV8kMF8kMV8kMl8kMw%3D%3D&wsTime=6a9431da")
	if err != nil {
		t.Fatal(err)
	}
	if err := refreshAntiCode(u, time.Unix(1_700_000_000, 0)); err != nil {
		t.Fatal(err)
	}
	query := u.Query()
	if got, want := query.Get("seqid"), "17000000000000000"; got != want {
		t.Fatalf("seqid = %q, want %q", got, want)
	}
	if got := query.Get("wsSecret"); got != "7473a9e4d5ddeed3e62ca37963365c2d" {
		t.Fatalf("wsSecret = %q", got)
	}
	if got := query.Get("fm"); got != "" {
		t.Fatalf("fm = %q, want empty", got)
	}
}
