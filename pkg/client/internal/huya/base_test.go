package huya

import (
	"github.com/tidwall/gjson"
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
	if err := refreshAntiCodeWithUID(u, time.Unix(1_700_000_000, 0), 12345678); err != nil {
		t.Fatal(err)
	}
	query := u.Query()
	for key, want := range map[string]string{"u": "3160493568", "ctype": "huya_live", "t": "100", "sdk_sid": "1700000000000"} {
		if got := query.Get(key); got != want {
			t.Fatalf("%s = %s, want %s", key, got, want)
		}
	}
	if got, want := query.Get("seqid"), "1700012345678"; got != want {
		t.Fatalf("seqid = %q, want %q", got, want)
	}
	if got := query.Get("wsSecret"); got != "0c2f9f77cbdcfd6ceca6d20907747d92" {
		t.Fatalf("wsSecret = %q", got)
	}
	if got := query.Get("fm"); got != "" {
		t.Fatalf("fm = %q, want empty", got)
	}
}

func TestNativeFLVURLPrefersContinuousLine(t *testing.T) {
	room := gjson.Parse(`{"roomInfo":{"tLiveInfo":{"tLiveStreamInfo":{"vStreamInfo":{"value":[
 {"sCdnType":"TX","sFlvUrl":"http://tx.flv.huya.com/src","sStreamName":"live","sFlvAntiCode":"fm=test"},
 {"sCdnType":"HS","sFlvUrl":"http://hs.flv.huya.com/src","sStreamName":"live","sFlvAntiCode":"fm=test"}
 ]}}}}}`)
	u, err := nativeFLVURL(room)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "https" || u.Host != "hs.flv.huya.com" || u.Path != "/src/live.flv" {
		t.Fatalf("unexpected native URL: %s", u)
	}
}

func TestNativeFLVURLRejectsMissingStream(t *testing.T) {
	if _, err := nativeFLVURL(gjson.Parse(`{"roomInfo":{}}`)); err == nil {
		t.Fatal("missing stream accepted")
	}
}

func TestRefreshAntiCodeRejectsInvalidInput(t *testing.T) {
	for _, query := range []string{"", "fm=bad&wsTime=123", "fm=RFdxOEJjSjNoNkRKdDZUWV8kMF8kMV8kMl8kMw%3D%3D"} {
		u, _ := url.Parse("https://hs.flv.huya.com/src/live.flv?" + query)
		if err := refreshAntiCodeWithUID(u, time.Unix(1700000000, 0), 12345678); err == nil {
			t.Fatalf("invalid query accepted: %s", query)
		}
	}
}
