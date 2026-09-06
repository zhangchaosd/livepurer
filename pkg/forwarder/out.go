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

// Bound queued tags to absorb short CDN bursts without unbounded buffering.
// The media duration varies with stream frame rate.
const flvTagBufferSize = 64

func Pull(in In, pullURL string, fn func(tag httpflv.Tag)) error {
	return in.Pull(pullURL, fn)
}

// PullURLRefresher resolves a fresh upstream address after a disconnect.
// Correctly signed Huya FLV streams are continuous; reconnect is recovery only.
type PullURLRefresher func() (string, error)

// flvPacer 让上游突发发送的 FLV 标签按媒体时间实时输出，并在续拉时
// 跳过 CDN 缓冲区带来的重叠帧，避免客户端出现快进后重复播放。
type flvPacer struct {
	lastTimestamp [2]uint32
	hasTimestamp  [2]bool
	started       bool
	mediaStart    uint32
	wallStart     time.Time
}

func (p *flvPacer) write(tag httpflv.Tag, stop <-chan struct{}, write func([]byte)) bool {
	if tag.Header.Type != 8 && tag.Header.Type != 9 || isCodecConfig(tag) {
		write(tag.Raw)
		return true
	}
	track := int(tag.Header.Type - 8)
	ts := tag.Header.Timestamp
	// Audio and video clocks are interleaved; compare only within each track.
	if p.hasTimestamp[track] && int32(ts-p.lastTimestamp[track]) < 0 {
		return true
	}
	if !p.started {
		p.started = true
		p.mediaStart = ts
		p.wallStart = time.Now()
	}
	// Use an absolute deadline: upstream I/O and previous writes already consume
	// media time. Sleeping a full frame interval again progressively stalls playback.
	delay := time.Until(p.wallStart.Add(time.Duration(int32(ts-p.mediaStart)) * time.Millisecond))
	if delay > 2*time.Second {
		p.mediaStart, p.wallStart = ts, time.Now()
	} else if delay > 0 {
		timer := time.NewTimer(delay)
		defer timer.Stop()
		select {
		case <-stop:
			return false
		case <-timer.C:
		}
	}
	p.lastTimestamp[track], p.hasTimestamp[track] = ts, true
	write(tag.Raw)
	return true
}

func isCodecConfig(tag httpflv.Tag) bool {
	// AVC/AAC sequence headers，保留它们让下一段的解码器保持可用。
	return (tag.Header.Type == 9 && len(tag.Raw) >= 13 && tag.Raw[11]&15 == 7 && tag.Raw[12] == 0) ||
		(tag.Header.Type == 8 && len(tag.Raw) >= 13 && tag.Raw[11]>>4 == 10 && tag.Raw[12] == 0)
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

	stopPull := make(chan struct{})
	defer close(stopPull)

	// 写入端按直播时间实时输出；与上游拉流分离，避免上游短片段切换时
	// 直接暴露为电视端的卡顿。
	tags := make(chan httpflv.Tag, flvTagBufferSize)
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		pacer := flvPacer{}
		for {
			select {
			case <-stopPull:
				return
			case tag := <-tags:
				if !pacer.write(tag, stopPull, sub.Write) {
					return
				}
			}
		}
	}()

	// 后台拉流并缓冲。直播 CDN 的一次 HTTP-FLV 响应可能只是一个短片段，
	// 因此正常 EOF 也要续拉。
	pullDone := make(chan error, 1)
	go func() {
		currentURL := pullURL
		for {
			err := Pull(in, currentURL, func(tag httpflv.Tag) {
				select {
				case <-stopPull:
				case tags <- tag:
				}
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
		case <-writerDone:
			_ = sub.Dispose()
			_ = in.Shutdown()
			return nil
		}
	}
}
