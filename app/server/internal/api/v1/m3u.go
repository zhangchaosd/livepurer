package v1

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/iyear/pure-live-core/app/server/internal/config"
	"github.com/iyear/pure-live-core/global"
	"github.com/iyear/pure-live-core/pkg/ecode"
	"github.com/iyear/pure-live-core/pkg/format"
	"github.com/iyear/pure-live-core/service/svc_fav"
	"github.com/iyear/pure-live-core/service/svc_live"
	"github.com/iyear/pure-live-core/service/svc_os"
	"go.uber.org/zap"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func globalCacheGet(key string) (interface{}, bool) {
	return global.Cache.Get(key)
}

func globalCacheSet(key string, v interface{}, d time.Duration) {
	global.Cache.Set(key, v, d)
}

// M3UEntry 播放列表条目
type M3UEntry struct {
	Name   string
	Logo   string
	Plat   string
	Room   string
	Online bool // 是否在线(离线频道也会输出, 由播放器处理)
}

// GetM3U 生成局域网 IPTV 播放列表(m3u), 供 APTV 等播放器加载
// 每个频道指向稳定的转发地址: /api/v1/live/play?plat=&room=
// 服务端在每次播放时动态解析最新流地址, 无需担心签名过期
//
// GET /api/v1/live/m3u
// 可选参数: online=1 仅输出在线频道; refresh=1 跳过缓存强制刷新
func GetM3U(c *gin.Context) {
	channels := config.GetChannels()
	favoriteListID, err := favoriteListID(c.Query("fav_list_id"))
	if err != nil {
		format.HTTP(c, ecode.InvalidParams, err, nil)
		return
	}
	if favoriteListID != 0 {
		_, favorites, err := svc_fav.GetFavList(favoriteListID)
		if err != nil {
			format.HTTP(c, ecode.InvalidParams, err, nil)
			return
		}
		channels = make([]config.Channel, 0, len(favorites))
		for _, favorite := range favorites {
			channels = append(channels, config.Channel{Plat: favorite.Plat, Room: favorite.Room, Name: favorite.Upper})
		}
	}
	if len(channels) == 0 && favoriteListID == 0 {
		format.HTTP(c, ecode.InvalidParams, fmt.Errorf("no channels configured"), nil)
		return
	}

	onlyOnline := c.Query("online") == "1"
	forceRefresh := c.Query("refresh") == "1"

	entries := make([]M3UEntry, 0, len(channels))
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	sem := make(chan struct{}, 10)
	for _, ch := range channels {
		wg.Add(1)
		go func(ch config.Channel) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			entry := M3UEntry{
				Name: ch.Name,
				Logo: ch.Logo,
				Plat: ch.Plat,
				Room: ch.Room,
			}

			key := fmt.Sprintf("m3u_%s_%s_%s", ch.Plat, ch.Room, ch.Name+"\x00"+ch.Logo)
			var cached *M3UEntry
			if !forceRefresh {
				if v, ok := globalCacheGet(key); ok {
					cached, _ = v.(*M3UEntry)
				}
			}
			if cached != nil {
				mu.Lock()
				entries = append(entries, *cached)
				mu.Unlock()
				return
			}

			// 动态获取直播间信息(名称/在线状态)
			info, infoErr := svc_live.GetRoomInfo(ch.Plat, ch.Room)
			if infoErr != nil {
				zap.S().Debugw("m3u: failed to get room info", "plat", ch.Plat, "room", ch.Room, "err", infoErr)
				entry.Online = false
			} else {
				entry.Online = info.Status == 1
				if entry.Name == "" {
					entry.Name = info.Upper + " - " + info.Title
					if entry.Name == " - " {
						entry.Name = fmt.Sprintf("%s %s", ch.Plat, ch.Room)
					}
				}
			}
			globalCacheSet(key, &entry, 1*time.Minute)

			mu.Lock()
			entries = append(entries, entry)
			mu.Unlock()
		}(ch)
	}
	wg.Wait()

	// 在线频道在前
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Online && !entries[j].Online
	})

	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	for _, e := range entries {
		if onlyOnline && !e.Online {
			continue
		}
		name := e.Name
		if name == "" {
			name = fmt.Sprintf("%s %s", e.Plat, e.Room)
		}
		attrs := fmt.Sprintf(` tvg-logo="%s"`, m3uText(channelLogoURL(c, e)))
		if !e.Online {
			attrs += " tvg-chnum-live=\"false\""
		}
		b.WriteString(fmt.Sprintf("#EXTINF:-1%s,%s\n", attrs, m3uText(name)))
		b.WriteString(playURL(c, e.Plat, e.Room) + "\n")
	}

	filename := "live.m3u"
	if favoriteListID != 0 {
		filename = fmt.Sprintf("favorites-%d.m3u", favoriteListID)
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/vnd.apple.mpegurl", []byte(b.String()))
}

func favoriteListID(raw string) (uint64, error) {
	if raw == "" {
		return 0, nil
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("fav_list_id must be a positive integer")
	}
	return id, nil
}

func m3uText(s string) string {
	return strings.NewReplacer("\r", " ", "\n", " ", `"`, "'").Replace(s)
}

func playURL(c *gin.Context, plat, room string) string {
	query := url.Values{"plat": {plat}, "room": {room}}
	return fmt.Sprintf("%s/api/v1/live/play?%s", schemeHost(c), query.Encode())
}

// schemeHost 返回当前请求的协议与主机, 用于拼接频道流地址
func schemeHost(c *gin.Context) string {
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, lanHost(c.Request.Host, svc_os.LANIPv4()))
}

// Rewrite local-only hosts in downloaded playlists while preserving the port.
func lanHost(host, lanIP string) string {
	if lanIP == "" {
		return host
	}
	hostname, port, err := net.SplitHostPort(host)
	if err != nil {
		hostname = strings.Trim(host, "[]")
	}
	ip := net.ParseIP(hostname)
	if !strings.EqualFold(hostname, "localhost") && (ip == nil || !ip.IsLoopback()) {
		return host
	}
	if port != "" {
		return net.JoinHostPort(lanIP, port)
	}
	return lanIP
}

func channelLogoURL(c *gin.Context, entry M3UEntry) string {
	if strings.TrimSpace(entry.Logo) != "" {
		return entry.Logo
	}
	query := url.Values{"plat": {entry.Plat}, "room": {entry.Room}}
	return fmt.Sprintf("%s/api/v1/live/cover?%s", schemeHost(c), query.Encode())
}
