package svc_live

import (
	"context"
	"crypto/tls"
	"github.com/gorilla/websocket"
	"github.com/iyear/pure-live-core/model"
	"github.com/iyear/pure-live-core/pkg/conf"
	"go.uber.org/zap"
	"time"
)

const (
	// revBufSize 转发通道缓冲大小, 高热度直播间弹幕量大, 缓冲避免接收循环阻塞
	revBufSize = 128
	// readTimeout 单次读超时, 用于检测连接假死; 大于平台心跳间隔即可
	readTimeout = 90 * time.Second
)

func Serve(ctx context.Context, dialer *websocket.Dialer, id string, client model.Client, room string) (chan *model.Transport, error) {
	// 过程
	// conn -> enter(on entered) -> go receive()   -> for{send msg to local}
	//  						 -> go heartbeat() -

	// 斗鱼弹幕服务器只支持 RSA 密钥交换的 TLS 套件(Go 1.22+ 默认已禁用)，需显式启用
	if client.Plat() == conf.PlatDouyu && dialer.TLSClientConfig == nil {
		d := *dialer
		d.TLSClientConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			},
		}
		dialer = &d
	}

	live, _, err := dialer.DialContext(ctx, client.Host(room), nil)
	if err != nil {
		return nil, err
	}

	zap.S().Infow("connected to live danmaku server", "id", id)

	rev := make(chan *model.Transport, revBufSize)

	tp, data, err := client.Enter(room)

	if err != nil && err != conf.ErrSkip {
		_ = live.Close()
		return nil, err
	}
	for _, d := range data {
		if err = live.WriteMessage(tp, d); err != nil {
			_ = live.Close()
			return nil, err
		}
	}

	zap.S().Infow("entered the room", "id", id)

	// ctx 取消时主动关闭连接, 让阻塞中的 ReadMessage 立即返回
	go func() {
		<-ctx.Done()
		_ = live.Close()
	}()

	go receive(ctx, id, client, live, rev)
	go heartbeat(ctx, id, client, live)

	return rev, nil
}

func heartbeat(ctx context.Context, id string, client model.Client, live *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	hb := func() {
		tp, data, err := client.HeartBeat()
		if err == conf.ErrSkip {
			return
		}
		if err != nil {
			zap.S().Warnw("failed to get heartbeat data", "id", id, "error", err)
			return
		}
		if err = live.WriteMessage(tp, data); err != nil {
			zap.S().Warnw("failed to send heartbeat", "id", id, "error", err)
			// 写失败说明连接已不可用, 关闭连接让 receive 退出
			_ = live.Close()
			return
		}
	}
	// 开头先执行一次
	hb()
	for {
		select {
		case <-ctx.Done():
			zap.S().Infow("heartbeat stopped",
				"id", id)
			return
		case <-ticker.C:
			hb()
		}
	}

}

// receive 读取弹幕消息并转发, 读错误时推送错误后退出(不再空转)
func receive(ctx context.Context, id string, client model.Client, live *websocket.Conn, rev chan *model.Transport) {
	for {
		if ctx.Err() != nil {
			zap.S().Infow("receive stopped", "id", id)
			return
		}
		// 每次读前设置 deadline, 检测连接假死
		if err := live.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			zap.S().Warnw("failed to set read deadline", "id", id, "error", err)
			return
		}
		t, data, err := live.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				// 正常退出
				return
			}
			zap.S().Warnw("danmaku connection error", "id", id, "error", err)
			send(ctx, rev, &model.Transport{Msg: nil, Error: err})
			return
		}
		msg, ok, err := client.Handle(t, data)

		// 错误判断
		if err != nil {
			send(ctx, rev, &model.Transport{
				Msg:   nil,
				Error: err,
			})
			continue
		}
		// 跳过判断
		if !ok {
			continue
		}

		for _, m := range msg {
			send(ctx, rev, &model.Transport{
				Msg:   m,
				Error: nil,
			})
		}
	}
}

// send 非阻塞转发消息, 通道满或上下文取消时丢弃, 避免阻塞接收循环和高热度房间的 goroutine 风暴
func send(ctx context.Context, rev chan *model.Transport, tp *model.Transport) {
	select {
	case rev <- tp:
	case <-ctx.Done():
	default:
		// 通道已满, 丢弃当前消息(弹幕场景可容忍)
	}
}
