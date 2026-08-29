package v1

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestProxyRejectsPrivateTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Any("/proxy", Proxy)
	req := httptest.NewRequest(http.MethodGet, "/proxy", nil)
	req.Header.Set("PL-URL", "http://127.0.0.1:8080/")
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestPlayRejectsUnsafeLegacyTargetBeforeHijack(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/play", Play)
	req := httptest.NewRequest(http.MethodGet, "/play?type=flv&url=http://127.0.0.1/live.flv", nil)
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}
