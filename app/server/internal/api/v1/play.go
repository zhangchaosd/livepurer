package v1

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/iyear/pure-live-core/pkg/forwarder"
	"github.com/iyear/pure-live-core/service/svc_live"
	"go.uber.org/zap"
	"net/http"
)

func Play(c *gin.Context) {
	conn, bio, err := c.Writer.Hijack()
	if err != nil {
		zap.S().Warnw("failed to hijack conn", "error", err)
		return
	}

	if bio.Reader.Buffered() != 0 || bio.Writer.Buffered() != 0 {
		zap.S().Warnw("failed to get buffer", "error", err)
		return
	}

	rawUrl := fmt.Sprintf("http://%s%s", c.Request.Host, c.Request.RequestURI)

	// 方式一(推荐): 传入 plat + room, 服务端动态获取最新流地址并转发,
	// 解决原始流地址签名过期问题, 可作为 IPTV/APTV 等播放器的稳定源:
	//   GET /api/v1/live/play?plat=douyu&room=6556593
	plat := c.Query("plat")
	room := c.Query("room")
	if plat != "" && room != "" {
		url, err := svc_live.GetPlayURL(plat, room)
		if err != nil {
			zap.S().Warnw("failed to get play url", "error", err, "plat", plat, "room", room)
			return
		}
		in := forwarder.GetIn(url.Type)
		if in == nil {
			c.Status(http.StatusForbidden)
			return
		}
		if err = forwarder.OutLoop(conn, url.Origin, rawUrl, in); err != nil {
			zap.S().Warnw("play loop failed", "error", err)
			return
		}
		return
	}

	// 方式二(旧): 直接传入已解析好的流地址与类型
	//   GET /api/v1/live/play?type=flv&url=<urlencoded>
	in := forwarder.GetIn(c.Query("type"))
	if in == nil {
		c.Status(http.StatusForbidden)
		return
	}

	if err = forwarder.OutLoop(conn, c.Query("url"), rawUrl, in); err != nil {
		return
	}
}
