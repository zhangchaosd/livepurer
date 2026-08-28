package request

import (
	"github.com/guonaihong/gout"
	"github.com/guonaihong/gout/dataflow"
	"github.com/iyear/pure-live-core/pkg/util"
	"net"
	"net/http"
	"sync"
	"time"
)

var (
	mu      sync.RWMutex
	client  *http.Client
	enabled = false // 是否已启用 socks5
	socks5  = socks5Config{}
)

type socks5Config struct {
	host     string
	port     int
	user     string
	password string
}

// initDefaultClient 构建默认 HTTP client(复用 Transport 连接池)
func initDefaultClient() {
	transport := &http.Transport{
		Dial:                net.Dial,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 16,
		IdleConnTimeout:     90 * time.Second,
	}
	client = &http.Client{Transport: transport}
}

func init() {
	initDefaultClient()
}

// SetSocks5 set socks5 proxy, 切换代理时会重建 HTTP client(连接池随旧 Transport 一并释放)
func SetSocks5(host string, port int, user, password string) {
	mu.Lock()
	defer mu.Unlock()
	socks5 = socks5Config{host: host, port: port, user: user, password: password}
	enabled = host != "" && port != 0

	if !enabled {
		initDefaultClient()
		return
	}
	transport := &http.Transport{
		Dial:                util.MustGetSocks5(host, port, user, password).Dial,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 16,
		IdleConnTimeout:     90 * time.Second,
	}
	client = &http.Client{Transport: transport}
}

// getClient 返回当前 HTTP client, 保证并发读取安全
func getClient() *http.Client {
	mu.RLock()
	defer mu.RUnlock()
	return client
}

// HTTP http request
func HTTP() *dataflow.DataFlow {
	return gout.New(getClient()).Debug(false)
}
