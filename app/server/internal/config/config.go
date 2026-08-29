package config

import (
	"fmt"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
	"os"
	"path/filepath"
	"sync"
)

var (
	Server     ServerConfig
	serverPath string
	serverMu   sync.RWMutex
)

func InitServer(path string) error {
	c := viper.New()
	c.SetConfigFile(path)
	if err := c.ReadInConfig(); err != nil {
		return err
	}
	var next ServerConfig
	if err := c.Unmarshal(&next); err != nil {
		return err
	}
	if err := validateServer(next); err != nil {
		return err
	}
	serverMu.Lock()
	Server = next
	serverPath = path
	serverMu.Unlock()

	return nil
}

type ServerConfig struct {
	Port   int          `mapstructure:"port" json:"port" yaml:"port"`
	Debug  bool         `mapstructure:"debug" json:"debug" yaml:"debug"`
	Path   string       `mapstructure:"path" json:"path" yaml:"path"`
	Socks5 Socks5Config `mapstructure:"socks5" json:"socks5" yaml:"socks5"`
}

type Socks5Config struct {
	Enable   bool   `mapstructure:"enable" json:"enable" yaml:"enable"`
	Host     string `mapstructure:"host" json:"host" yaml:"host"`
	Port     int    `mapstructure:"port" json:"port" yaml:"port"`
	User     string `mapstructure:"user" json:"user" yaml:"user"`
	Password string `mapstructure:"password" json:"password" yaml:"password"`
}

func GetServer() ServerConfig {
	serverMu.RLock()
	defer serverMu.RUnlock()
	return Server
}

func SaveServer(next ServerConfig) error {
	if err := validateServer(next); err != nil {
		return err
	}
	serverMu.Lock()
	defer serverMu.Unlock()
	if serverPath == "" {
		return fmt.Errorf("server configuration path is not initialized")
	}
	if err := writeYAML(serverPath, next); err != nil {
		return err
	}
	Server = next
	return nil
}

func validateServer(next ServerConfig) error {
	if next.Port < 1 || next.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if next.Path == "" {
		return fmt.Errorf("data path is required")
	}
	if next.Socks5.Enable && (next.Socks5.Host == "" || next.Socks5.Port < 1 || next.Socks5.Port > 65535) {
		return fmt.Errorf("a SOCKS5 host and valid port are required when enabled")
	}
	return nil
}

func writeYAML(path string, value any) error {
	b, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pure-live-*.yaml")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err = tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
