package v1

import (
	"net/http/httptest"
	"strings"
	"testing"

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
