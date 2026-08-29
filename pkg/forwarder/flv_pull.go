package forwarder

import (
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/q191201771/lal/pkg/httpflv"
)

// FlvPuller 自定义 http-flv 拉流器。
// 原因: lal 的 httpflv.PullSession 在 https 场景使用默认 TLS 配置,
// 而部分 CDN(如斗鱼) 只支持 RSA 密钥交换套件(Go 1.22+ 默认已禁用), 导致握手失败。
// 这里使用带 RSA 套件与 UA/Referer 的 HTTP client 拉流, 并解析为 FLV tag。
type FlvPuller struct {
	client    *http.Client
	resp      *http.Response
	streamURL string
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
	}
	transport := &http.Transport{
		Dial: (&net.Dialer{Timeout: 10 * time.Second}).Dial,
		TLSClientConfig: &tls.Config{
			// 兼容只支持 RSA 密钥交换的 CDN(斗鱼等)
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			},
		},
	}
	return &FlvPuller{
		client: &http.Client{
			Transport: transport,
			// 不设置整体超时, 直播流是长连接
		},
		streamURL: streamURL,
	}
}

// Pull 建立连接并持续读取 FLV 数据, 每读到完整 tag 回调一次。
// 阻塞直到流结束或连接出错。
func (p *FlvPuller) Pull(fn func(tag httpflv.Tag)) error {
	req, err := http.NewRequest(http.MethodGet, p.streamURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://www.douyu.com/")
	req.Header.Set("Connection", "close")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("flv pull: %w", err)
	}
	p.resp = resp
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
	if p.resp != nil && p.resp.Body != nil {
		return p.resp.Body.Close()
	}
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
	_ = binary.BigEndian
	return h
}
