package v1

import (
	"github.com/iyear/pure-live-core/app/server/internal/config"
	"github.com/iyear/pure-live-core/global"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestM3UTextRemovesRecordDelimitersAndQuotes(t *testing.T) {
	got := m3uText("a\r\nb\"c")
	if got != "a  b'c" {
		t.Fatalf("m3uText() = %q", got)
	}
}

func TestPlayURLEncodesChannelValues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "http://example.test/api/v1/live/m3u", nil)
	got := playURL(c, "bad&plat", "a b")
	if !strings.Contains(got, "plat=bad%26plat") || !strings.Contains(got, "room=a+b") {
		t.Fatalf("playURL() = %q", got)
	}
}

func TestFavoriteListID(t *testing.T) {
	if id, err := favoriteListID("42"); err != nil || id != 42 {
		t.Fatalf("favoriteListID() = %d, %v", id, err)
	}
	for _, raw := range []string{"0", "-1", "not-a-number"} {
		if _, err := favoriteListID(raw); err == nil {
			t.Fatalf("favoriteListID(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestLANHost(t *testing.T) {
	for _, tt := range []struct{ host, ip, want string }{
		{"127.0.0.1:18800", "192.168.1.9", "192.168.1.9:18800"},
		{"localhost:8800", "192.168.1.9", "192.168.1.9:8800"},
		{"[::1]:8800", "10.0.0.2", "10.0.0.2:8800"},
		{"localhost", "192.168.1.9", "192.168.1.9"},
		{"example.com:443", "192.168.1.9", "example.com:443"},
		{"192.168.1.2:8800", "192.168.1.9", "192.168.1.2:8800"},
		{"127.0.0.1:8800", "", "127.0.0.1:8800"},
	} {
		if got := lanHost(tt.host, tt.ip); got != tt.want {
			t.Errorf("lanHost(%q, %q) = %q, want %q", tt.host, tt.ip, got, tt.want)
		}
	}
}

func TestChannelLogoURL(t *testing.T) {
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "http://192.168.1.9:8800/api/v1/live/m3u", nil)
	got := channelLogoURL(c, M3UEntry{Plat: "huya", Room: "12 3"})
	if got != "http://192.168.1.9:8800/api/v1/live/cover?plat=huya&room=12+3" {
		t.Fatalf("unexpected cover URL: %s", got)
	}
	custom := "https://example.com/custom.png"
	if got := channelLogoURL(c, M3UEntry{Plat: "huya", Room: "1", Logo: custom}); got != custom {
		t.Fatalf("custom logo was replaced: %s", got)
	}
}

func TestM3UEmitsAutomaticCoverAndCustomLogo(t *testing.T) {
	previous := config.Channels
	config.Channels = []config.Channel{{Plat: "huya", Room: "cover-fixture", Name: "Fixture"}}
	t.Cleanup(func() { config.Channels = previous })
	key := "m3u_huya_cover-fixture_Fixture\x00"
	globalCacheSet(key, &M3UEntry{Plat: "huya", Room: "cover-fixture", Name: "Fixture", Online: true}, time.Minute)
	t.Cleanup(func() { global.Cache.Delete(key) })
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("GET", "http://192.168.1.9:8800/api/v1/live/m3u", nil)
	GetM3U(c)
	if recorder.Code != 200 || !strings.Contains(recorder.Body.String(), `tvg-logo="http://192.168.1.9:8800/api/v1/live/cover?plat=huya&room=cover-fixture"`) {
		t.Fatalf("M3U missing automatic cover: %d %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "/api/v1/live/play?plat=huya&room=cover-fixture") {
		t.Fatal("play URL lost")
	}
}
