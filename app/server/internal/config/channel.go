package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
	"sync"
)

// Channel 直播频道配置, 用于生成局域网 IPTV 播放列表(m3u)
type Channel struct {
	Plat string `mapstructure:"plat" json:"plat" yaml:"plat"` // 平台名: bilibili/huya/douyu/inke
	Room string `mapstructure:"room" json:"room" yaml:"room"` // 房间号
	Name string `mapstructure:"name" json:"name" yaml:"name"` // 可选, 频道显示名, 为空则使用直播间标题
	Logo string `mapstructure:"logo" json:"logo" yaml:"logo"` // 可选, 频道图标 URL
}

var (
	Channels     []Channel
	channelsPath string
	channelsMu   sync.RWMutex
)

// InitChannels 加载频道配置文件
func InitChannels(path string) error {
	channelsMu.Lock()
	channelsPath = path
	channelsMu.Unlock()
	c := viper.New()
	c.SetConfigFile(path)
	if err := c.ReadInConfig(); err != nil {
		return err
	}
	var next []Channel
	if err := c.UnmarshalKey("channels", &next); err != nil {
		return err
	}
	if err := validateChannels(next); err != nil {
		return err
	}
	channelsMu.Lock()
	Channels = next
	channelsPath = path
	channelsMu.Unlock()
	return nil
}

func GetChannels() []Channel {
	channelsMu.RLock()
	defer channelsMu.RUnlock()
	return append([]Channel(nil), Channels...)
}

func SaveChannels(next []Channel) error {
	if err := validateChannels(next); err != nil {
		return err
	}
	channelsMu.Lock()
	defer channelsMu.Unlock()
	if channelsPath == "" {
		return fmt.Errorf("channels configuration path is not initialized")
	}
	if err := writeYAML(channelsPath, struct {
		Channels []Channel `yaml:"channels"`
	}{Channels: next}); err != nil {
		return err
	}
	Channels = append([]Channel(nil), next...)
	return nil
}

func validateChannels(channels []Channel) error {
	if len(channels) > 100 {
		return fmt.Errorf("at most 100 channels are allowed")
	}
	for i, channel := range channels {
		if !isPlatform(channel.Plat) || strings.TrimSpace(channel.Room) == "" {
			return fmt.Errorf("channel %d must contain a supported platform and room", i+1)
		}
	}
	return nil
}

func isPlatform(plat string) bool {
	switch plat {
	case "bilibili", "huya", "douyu", "inke":
		return true
	default:
		return false
	}
}
