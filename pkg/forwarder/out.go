package forwarder

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/httpflv"
	"net"
	"time"
)

const flvResponseHeader = "HTTP/1.1 200 OK\r\n" +
	"Server: pure-live-core\r\n" +
	"Cache-Control: no-cache\r\n" +
	"Content-Type: video/x-flv\r\n" +
	"Connection: keep-alive\r\n" +
	"Access-Control-Allow-Credentials: true\r\n" +
	"Access-Control-Allow-Origin: *\r\n\r\n"

func Pull(in In, pullURL string, fn func(tag httpflv.Tag)) error {
	return in.Pull(pullURL, fn)
}

// PullURLRefresher 返回一个新的上游地址。部分 CDN（例如虎牙）会在数秒后
// 主动结束 HTTP-FLV 片段；此时需要重新解析地址后续拉，而不是关闭播放端连接。
type PullURLRefresher func() (string, error)

// flvPacer 让上游突发发送的 FLV 标签按媒体时间实时输出，并在续拉时
// 跳过 CDN 缓冲区带来的重叠帧，避免客户端出现快进后重复播放。
type flvPacer struct {
	lastTimestamp uint32
	hasTimestamp  bool
}

func (p *flvPacer) write(tag httpflv.Tag, stop <-chan struct{}, write func([]byte)) bool {
	if tag.Header.Type != 8 && tag.Header.Type != 9 { // 仅以音视频帧作为时间基准
		write(tag.Raw)
		return true
	}

	if p.hasTimestamp {
		if tag.Header.Timestamp < p.lastTimestamp {
			// 新片段回退到已播放的 CDN 缓冲区。音视频配置帧仍要保留，
			// 其余旧帧跳过，直到时间戳追上当前播放位置。
			if isCodecConfig(tag) {
				write(tag.Raw)
			}
			return true
		}
		if tag.Header.Timestamp == p.lastTimestamp {
			write(tag.Raw)
			return true
		}
		delta := tag.Header.Timestamp - p.lastTimestamp
		// 不因异常时间戳或大跳变长时间阻塞；正常直播帧间隔远小于此值。
		if delta <= 2_000 {
			timer := time.NewTimer(time.Duration(delta) * time.Millisecond)
			select {
			case <-stop:
				timer.Stop()
				return false
			case <-timer.C:
			}
		}
	}

	p.lastTimestamp = tag.Header.Timestamp
	p.hasTimestamp = true
	write(tag.Raw)
	return true
}

func isCodecConfig(tag httpflv.Tag) bool {
	// AVC/AAC sequence headers，保留它们让下一段的解码器保持可用。
	return (tag.Header.Type == 9 && len(tag.Raw) >= 13 && tag.Raw[12] == 0) ||
		(tag.Header.Type == 8 && len(tag.Raw) >= 13 && tag.Raw[12] == 0)
}

func OutLoop(conn net.Conn, pullURL string, rawURL string, in In, refresh PullURLRefresher) error {
	urlCtx, err := base.ParseHttpUrl(rawURL, "")
	if err != nil {
		return err
	}
	sub := httpflv.NewSubSession(conn, urlCtx, false, "")

	if _, err = conn.Write([]byte(flvResponseHeader)); err != nil {
		return err
	}
	sub.WriteFlvHeader()

	// RunLoop 在客户端实际关闭连接时返回。不要用短时间内没有写数据作为
	// 客户端断开的判断：虎牙切片续拉之间会有正常的短暂空档。
	clientDone := make(chan struct{})
	go func() {
		_ = sub.RunLoop()
		close(clientDone)
	}()

	// 后台拉流并转发, 避免阻塞主循环的存活检测。
	// 直播 CDN 的一次 HTTP-FLV 响应可能只是一个短片段，因此正常 EOF 也要续拉。
	pullDone := make(chan error, 1)
	stopPull := make(chan struct{})
	defer close(stopPull)
	go func() {
		currentURL := pullURL
		pacer := flvPacer{}
		for {
			err := Pull(in, currentURL, func(tag httpflv.Tag) {
				pacer.write(tag, stopPull, sub.Write)
			})
			select {
			case <-stopPull:
				return
			default:
			}

			if refresh != nil {
				for {
					var refreshErr error
					currentURL, refreshErr = refresh()
					if refreshErr == nil {
						break
					}
					// 虎牙页面偶尔会返回不完整的房间数据。客户端仍连接时，
					// 等待后重试解析，避免一次瞬时错误中断电视端播放。
					select {
					case <-stopPull:
						return
					case <-time.After(time.Second):
					}
				}
			} else if err != nil {
				pullDone <- err
				return
			}

			select {
			case <-stopPull:
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	}()

	for {
		select {
		case err := <-pullDone:
			// 拉流结束(流结束或出错)
			_ = sub.Dispose()
			return err
		case <-clientDone:
			// 客户端断开
			_ = sub.Dispose()
			_ = in.Shutdown()
			return nil
		}
	}
}
