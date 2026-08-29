package forwarder

import (
	"github.com/q191201771/lal/pkg/base"
	"github.com/q191201771/lal/pkg/httpflv"
	"net"
	"time"
)

func Pull(in In, pullURL string, fn func(tag httpflv.Tag)) error {
	return in.Pull(pullURL, fn)
}

func OutLoop(conn net.Conn, pullURL string, rawURL string, in In) error {
	urlCtx, err := base.ParseHttpUrl(rawURL, "")
	if err != nil {
		return err
	}
	sub := httpflv.NewSubSession(conn, urlCtx, false, "")

	sub.WriteHttpResponseHeader()
	sub.WriteFlvHeader()

	// RunLoop 处理与客户端的读写
	go func() {
		_ = sub.RunLoop()
	}()

	// 后台拉流并转发, 避免阻塞主循环的存活检测
	pullDone := make(chan error, 1)
	go func() {
		pullDone <- Pull(in, pullURL, func(tag httpflv.Tag) {
			sub.Write(tag.Raw)
		})
	}()

	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case err := <-pullDone:
			// 拉流结束(流结束或出错)
			_ = sub.Dispose()
			return err
		case <-tick.C:
			_, write := sub.IsAlive()
			if !write {
				// 客户端断开
				_ = sub.Dispose()
				_ = in.Shutdown()
				return nil
			}
		}
	}
}
