package v1

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/iyear/pure-live-core/pkg/forwarder"
	"github.com/iyear/pure-live-core/pkg/request"
	"github.com/iyear/pure-live-core/service/svc_live"
	"go.uber.org/zap"
	"net/http"
	"time"
)

func Play(c *gin.Context) {
	// 方式一(推荐): 传入 plat + room, 服务端动态获取最新流地址并转发,
	// 解决原始流地址签名过期问题, 可作为 IPTV/APTV 等播放器的稳定源:
	//   GET /api/v1/live/play?plat=douyu&room=6556593
	plat := c.Query("plat")
	room := c.Query("room")
	var pullURL string
	var refresh forwarder.PullURLRefresher
	var in forwarder.In
	if plat != "" && room != "" {
		url, err := svc_live.GetPlayURL(plat, room)
		if err != nil {
			zap.S().Warnw("failed to get play url", "error", err, "plat", plat, "room", room)
			c.String(http.StatusBadGateway, "failed to resolve stream")
			return
		}
		in = forwarder.GetIn(url.Type)
		if in == nil {
			c.String(http.StatusBadGateway, "unsupported stream type")
			return
		}
		pullURL = url.Origin
		refresh = func() (string, error) {
			latest, err := svc_live.GetPlayURL(plat, room)
			if err != nil {
				return "", err
			}
			if latest.Type != url.Type {
				return "", fmt.Errorf("stream type changed from %s to %s", url.Type, latest.Type)
			}
			return latest.Origin, nil
		}
	} else {
		// 方式二(旧): 直接传入已解析好的流地址与类型。
		// 仅允许公网 HTTP(S)/RTMP(S) 地址，避免该兼容接口被用作 SSRF。
		streamType := c.Query("type")
		in = forwarder.GetIn(streamType)
		if in == nil {
			c.String(http.StatusBadRequest, "unsupported stream type")
			return
		}
		allowedSchemes := []string{"http", "https"}
		if streamType == "rtmp" {
			allowedSchemes = []string{"rtmp", "rtmps"}
		}
		streamURL, err := request.ValidatePublicURL(c.Query("url"), allowedSchemes...)
		if err != nil {
			c.String(http.StatusBadRequest, "invalid stream URL: %v", err)
			return
		}
		pullURL = streamURL.String()
	}

	conn, bio, err := c.Writer.Hijack()
	if err != nil {
		zap.S().Warnw("failed to hijack conn", "error", err)
		return
	}
	if bio.Reader.Buffered() != 0 || bio.Writer.Buffered() != 0 {
		zap.S().Warn("cannot start stream with buffered connection data")
		_ = conn.Close()
		return
	}
	defer conn.Close()
	// Hijacked live streams must not inherit the HTTP request deadlines.
	if err := conn.SetDeadline(time.Time{}); err != nil {
		return
	}

	rawURL := fmt.Sprintf("http://localhost%s", c.Request.URL.RequestURI())
	if err = forwarder.OutLoop(conn, pullURL, rawURL, in, refresh); err != nil {
		zap.S().Warnw("play loop failed", "error", err)
	}
}
