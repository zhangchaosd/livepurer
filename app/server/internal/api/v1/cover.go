package v1

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iyear/pure-live-core/model"
	"github.com/iyear/pure-live-core/pkg/request"
	"github.com/iyear/pure-live-core/service/svc_live"
	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

var coverReferers = map[string]string{
	"bilibili": "https://live.bilibili.com/", "douyu": "https://www.douyu.com/",
	"huya": "https://www.huya.com/", "inke": "https://www.inke.cn/",
}
var coverRoom = regexp.MustCompile(`^[A-Za-z0-9_-]{1,80}$`)
var serveCover = newCoverHandler(svc_live.GetRoomInfo, fetchCover)

// GetCover is a stable TV-compatible image URL. It never starts a live stream.
func GetCover(c *gin.Context) { serveCover(c) }

type coverImage struct {
	data    []byte
	source  string
	expires time.Time
}

// Keep image memory and concurrent upstream requests bounded, independently of
// the metadata cache. Short lifetimes allow platform previews to refresh.
func newCoverHandler(resolve func(string, string) (*model.RoomInfo, error), fetch func(context.Context, string, string) ([]byte, error)) gin.HandlerFunc {
	var mu sync.Mutex
	cache := map[string]coverImage{}
	cacheBytes := 0
	slots := make(chan struct{}, 4)
	lookup := func(key string) (coverImage, bool) {
		mu.Lock()
		defer mu.Unlock()
		value, ok := cache[key]
		return value, ok && time.Now().Before(value.expires)
	}
	return func(c *gin.Context) {
		plat, room := c.Query("plat"), strings.TrimSpace(c.Query("room"))
		referer, supported := coverReferers[plat]
		if !supported || !coverRoom.MatchString(room) {
			c.String(http.StatusBadRequest, "supported plat and valid room are required")
			return
		}
		key := plat + ":" + room
		if cached, ok := lookup(key); ok {
			writeCover(c, cached)
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
		defer cancel()
		select {
		case slots <- struct{}{}:
			defer func() { <-slots }()
		case <-ctx.Done():
			c.Status(http.StatusGatewayTimeout)
			return
		}
		if cached, ok := lookup(key); ok {
			writeCover(c, cached)
			return
		}
		result := coverImage{data: defaultCover, source: "placeholder"}
		if info, err := resolve(plat, room); err == nil && info != nil {
			for _, candidate := range []struct{ url, kind string }{{info.Cover, "cover"}, {info.Avatar, "avatar"}} {
				if candidate.url == "" {
					continue
				}
				if data, err := fetch(ctx, candidate.url, referer); err == nil {
					result.data, result.source = data, candidate.kind
					break
				}
			}
		}
		lifetime := time.Minute
		if result.source == "placeholder" {
			lifetime = 15 * time.Second
		}
		result.expires = time.Now().Add(lifetime)
		mu.Lock()
		if previous, ok := cache[key]; ok {
			cacheBytes -= len(previous.data)
			delete(cache, key)
		}
		if len(cache) >= 64 || cacheBytes+len(result.data) > 16<<20 {
			cache = map[string]coverImage{}
			cacheBytes = 0
		}
		cache[key] = result
		cacheBytes += len(result.data)
		mu.Unlock()
		writeCover(c, result)
	}
}

func writeCover(c *gin.Context, img coverImage) {
	etag := fmt.Sprintf(`"%x"`, sha256.Sum256(img.data))
	seconds := 60
	if img.source == "placeholder" {
		seconds = 15
	}
	c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", seconds))
	c.Header("ETag", etag)
	c.Header("X-Cover-Source", img.source)
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Type", "image/jpeg")
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	if c.Request.Method == http.MethodHead {
		c.Header("Content-Length", fmt.Sprint(len(img.data)))
		c.Status(http.StatusOK)
		return
	}
	c.Data(http.StatusOK, "image/jpeg", img.data)
}

func fetchCover(ctx context.Context, raw, referer string) ([]byte, error) {
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	target, err := request.ValidatePublicURL(raw, "http", "https")
	if err != nil {
		return nil, err
	}
	// Huya's optional format negotiation may force WebP on otherwise JPEG URLs.
	if strings.HasSuffix(target.Hostname(), ".msstatic.com") {
		q := target.Query()
		q.Del("spformat")
		target.RawQuery = q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", referer)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "image/jpeg,image/png,image/webp,image/gif")
	client := *request.HTTP().Client()
	client.Timeout = 8 * time.Second
	client.CheckRedirect = func(next *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return fmt.Errorf("too many image redirects")
		}
		_, err := request.ValidatePublicURL(next.URL.String(), "http", "https")
		return err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image status %d", response.StatusCode)
	}
	return jpegThumbnail(response.Body)
}

func jpegThumbnail(reader io.Reader) ([]byte, error) {
	const maxBytes = 4 << 20
	raw, err := io.ReadAll(io.LimitReader(reader, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > maxBytes {
		return nil, fmt.Errorf("image is too large")
	}
	dimensions, _, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	if dimensions.Width <= 0 || dimensions.Height <= 0 || int64(dimensions.Width)*int64(dimensions.Height) > 8_000_000 {
		return nil, fmt.Errorf("image dimensions are too large")
	}
	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	width, height := dimensions.Width, dimensions.Height
	if width > 640 {
		height = max(1, height*640/width)
		width = 640
	}
	if height > 360 {
		width = max(1, width*360/height)
		height = 360
	}
	target := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(target, target.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	xdraw.CatmullRom.Scale(target, target.Bounds(), source, source.Bounds(), draw.Over, nil)
	var out bytes.Buffer
	if err = jpeg.Encode(&out, target, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

var defaultCover = func() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 320, 180))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.RGBA{36, 62, 49, 255}), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(99, 49, 221, 131), image.NewUniform(color.RGBA{109, 150, 126, 255}), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(104, 54, 216, 126), image.NewUniform(color.RGBA{36, 62, 49, 255}), image.Point{}, draw.Src)
	for x := 145; x < 181; x++ {
		for y := 70 + (x-145)/2; y < 110-(x-145)/2; y++ {
			img.Set(x, y, color.White)
		}
	}
	var out bytes.Buffer
	_ = jpeg.Encode(&out, img, &jpeg.Options{Quality: 85})
	return out.Bytes()
}()
