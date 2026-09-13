package v1

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/jpeg"
	"image/png"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/iyear/pure-live-core/model"
)

func TestCoverFallbackCacheAndConditionalRequests(t *testing.T) {
	resolves, fetches := 0, 0
	handler := newCoverHandler(func(plat, room string) (*model.RoomInfo, error) {
		resolves++
		return &model.RoomInfo{Cover: "unsupported.avif", Avatar: "avatar.jpg"}, nil
	}, func(_ context.Context, raw, referer string) ([]byte, error) {
		fetches++
		if referer != "https://www.douyu.com/" {
			t.Fatalf("unexpected referer %q", referer)
		}
		if raw == "unsupported.avif" {
			return nil, fmt.Errorf("unsupported image")
		}
		return defaultCover, nil
	})
	r := gin.New()
	r.GET("/cover", handler)
	r.HEAD("/cover", handler)
	call := func(method, etag string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, "/cover?plat=douyu&room=9999", nil)
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		res := httptest.NewRecorder()
		r.ServeHTTP(res, req)
		return res
	}
	first := call("GET", "")
	if first.Code != 200 || first.Header().Get("Content-Type") != "image/jpeg" || first.Header().Get("X-Cover-Source") != "avatar" {
		t.Fatalf("unexpected response: %d %v", first.Code, first.Header())
	}
	if _, err := jpeg.Decode(first.Body); err != nil {
		t.Fatal(err)
	}
	if next := call("GET", first.Header().Get("ETag")); next.Code != 304 || next.Body.Len() != 0 {
		t.Fatal("conditional request did not return empty 304")
	}
	if head := call("HEAD", ""); head.Code != 200 || head.Body.Len() != 0 || head.Header().Get("Content-Length") == "" {
		t.Fatal("HEAD response is invalid")
	}
	if resolves != 1 || fetches != 2 {
		t.Fatalf("cache missed: resolves=%d fetches=%d", resolves, fetches)
	}
}

func TestCoverPlaceholderAndValidation(t *testing.T) {
	calls := 0
	handler := newCoverHandler(func(string, string) (*model.RoomInfo, error) { calls++; return nil, fmt.Errorf("offline") }, nil)
	r := gin.New()
	r.GET("/cover", handler)
	for _, query := range []string{"plat=invalid&room=1", "plat=huya&room=..%2Fsecret", "plat=huya", "plat=huya&room=" + strings.Repeat("1", 81)} {
		res := httptest.NewRecorder()
		r.ServeHTTP(res, httptest.NewRequest("GET", "/cover?"+query, nil))
		if res.Code != 400 {
			t.Fatalf("invalid query accepted: %s", query)
		}
	}
	if calls != 0 {
		t.Fatal("invalid query reached resolver")
	}
	res := httptest.NewRecorder()
	r.ServeHTTP(res, httptest.NewRequest("GET", "/cover?plat=huya&room=1", nil))
	if res.Code != 200 || res.Header().Get("X-Cover-Source") != "placeholder" || res.Header().Get("Cache-Control") != "public, max-age=15" {
		t.Fatalf("invalid fallback: %v", res.Header())
	}
	if _, err := jpeg.Decode(res.Body); err != nil {
		t.Fatal(err)
	}
}

func TestJPEGThumbnailNormalizesAndBoundsImages(t *testing.T) {
	var source bytes.Buffer
	if err := png.Encode(&source, image.NewRGBA(image.Rect(0, 0, 1280, 720))); err != nil {
		t.Fatal(err)
	}
	out, err := jpegThumbnail(bytes.NewReader(source.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(out))
	if err != nil || cfg.Width != 640 || cfg.Height != 360 {
		t.Fatalf("unexpected thumbnail: %+v %v", cfg, err)
	}
	for _, bad := range [][]byte{[]byte("<html>Access denied</html>"), bytes.Repeat([]byte{0}, (4<<20)+1)} {
		if _, err := jpegThumbnail(bytes.NewReader(bad)); err == nil {
			t.Fatal("invalid or oversized image accepted")
		}
	}
	oversized := append([]byte(nil), source.Bytes()...)
	binary.BigEndian.PutUint32(oversized[16:20], 100000)
	binary.BigEndian.PutUint32(oversized[29:33], crc32.ChecksumIEEE(oversized[12:29]))
	if _, err := jpegThumbnail(bytes.NewReader(oversized)); err == nil {
		t.Fatal("oversized dimensions accepted")
	}
}

func TestCoverRejectsPrivateImageSource(t *testing.T) {
	for _, raw := range []string{"http://127.0.0.1/private", "http://192.168.1.1/admin", "file:///etc/passwd"} {
		if _, err := fetchCover(context.Background(), raw, "https://www.huya.com/"); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}
