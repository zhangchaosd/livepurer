package forwarder

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/q191201771/lal/pkg/httpflv"
)

// FlvPuller 自定义 http-flv 拉流器。
// 原因: lal 的 httpflv.PullSession 在 https 场景使用默认 TLS 配置,
// 而部分 CDN(如斗鱼) 只支持 RSA 密钥交换套件(Go 1.22+ 默认已禁用), 导致握手失败。
// 这里使用带 RSA 套件与 UA/Referer 的 HTTP client 拉流, 并解析为 FLV tag。
// A read deadline detects a silent CDN without limiting total stream duration.
type idleReadConn struct{ net.Conn }

func (c idleReadConn) Read(b []byte) (int, error) {
	if err := c.SetReadDeadline(time.Now().Add(15 * time.Second)); err != nil {
		return 0, err
	}
	return c.Conn.Read(b)
}

type FlvPuller struct {
	client    *http.Client
	ctx       context.Context
	cancel    context.CancelFunc
	streamURL string
	ua        string
	referer   string
}

const (
	flvHeaderSize       = 9
	flvTagHeaderSize    = 11
	flvPrevTagSizeField = 4
)

// NewFlvPuller 创建拉流器。ua/referer 为空时使用默认值
func NewFlvPuller(streamURL string, ua, referer string) *FlvPuller {
	if ua == "" {
		ua = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	}
	if referer == "" {
		referer = "https://www.douyu.com/"
		if u, err := url.Parse(streamURL); err == nil {
			host := strings.ToLower(u.Hostname())
			if strings.Contains(host, "huya") || strings.Contains(host, "huyaliv") {
				referer = "https://www.huya.com/"
			}
			if strings.Contains(host, "bilivideo") {
				referer = "https://live.bilibili.com/"
			}
		}
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			conn, err := (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			return idleReadConn{conn}, nil
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		TLSClientConfig: &tls.Config{
			// 兼容只支持 RSA 密钥交换的 CDN(斗鱼等)
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			},
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &FlvPuller{ctx: ctx, cancel: cancel,
		client: &http.Client{
			Transport: transport,
			// 不设置整体超时, 直播流是长连接
		},
		streamURL: streamURL,
		ua:        ua,
		referer:   referer,
	}
}

// Pull 建立连接并持续读取 FLV 数据, 每读到完整 tag 回调一次。
// 阻塞直到流结束或连接出错。
func (p *FlvPuller) Pull(fn func(tag httpflv.Tag)) error {
	defer p.cancel()
	req, err := http.NewRequestWithContext(p.ctx, http.MethodGet, p.streamURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", p.ua)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", p.referer)
	req.Header.Set("Connection", "close")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("flv pull: %w", err)
	}
	defer resp.Body.Close()
	defer p.client.CloseIdleConnections()
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return fmt.Errorf("flv pull: http status %d", resp.StatusCode)
	}

	// 跳过 FLV 文件头(9字节)
	flvHeader := make([]byte, flvHeaderSize)
	if _, err := io.ReadFull(resp.Body, flvHeader); err != nil {
		_ = resp.Body.Close()
		return fmt.Errorf("flv pull: read flv header: %w", err)
	}
	// 验证 FLV 魔数
	if string(flvHeader[:3]) != "FLV" {
		_ = resp.Body.Close()
		return fmt.Errorf("flv pull: invalid flv header")
	}
	offset := binary.BigEndian.Uint32(flvHeader[5:9])
	if offset < 9 || offset > 1024*1024 {
		return fmt.Errorf("flv pull: invalid header offset")
	}
	if _, err := io.CopyN(io.Discard, resp.Body, int64(offset-9)); err != nil {
		return err
	}
	// 跳过 PreviousTagSize0(4字节)
	if _, err := io.CopyN(io.Discard, resp.Body, flvPrevTagSizeField); err != nil {
		_ = resp.Body.Close()
		return fmt.Errorf("flv pull: skip prev tag size: %w", err)
	}

	for {
		tag, err := readFlvTag(resp.Body)
		if err != nil {
			_ = resp.Body.Close()
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("flv pull: read tag: %w", err)
		}
		fn(tag)
	}
}

// Shutdown 关闭拉流连接
func (p *FlvPuller) Shutdown() error {
	p.cancel()
	return nil
}

// readFlvTag 从 reader 读取一个完整的 FLV tag
// FLV tag 结构: [11字节 tag header][body][4字节 prev tag size]
func readFlvTag(rd io.Reader) (tag httpflv.Tag, err error) {
	rawHeader := make([]byte, flvTagHeaderSize)
	if _, err = io.ReadFull(rd, rawHeader); err != nil {
		return
	}
	tag.Header = parseTagHeader(rawHeader)
	// body + prev tag size
	needed := int(tag.Header.DataSize) + flvPrevTagSizeField
	tag.Raw = make([]byte, flvTagHeaderSize+needed)
	copy(tag.Raw, rawHeader)
	if _, err = io.ReadFull(rd, tag.Raw[flvTagHeaderSize:]); err != nil {
		return
	}
	return
}

func parseTagHeader(raw []byte) httpflv.TagHeader {
	var h httpflv.TagHeader
	h.Type = raw[0]
	// 3字节大端 DataSize
	h.DataSize = uint32(raw[1])<<16 | uint32(raw[2])<<8 | uint32(raw[3])
	// 4字节时间戳(低3字节 + 高1字节扩展)
	h.Timestamp = uint32(raw[7])<<24 | uint32(raw[4])<<16 | uint32(raw[5])<<8 | uint32(raw[6])
	// StreamID 3字节, 恒为0
	return h
}
