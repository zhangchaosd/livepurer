package conf

import (
	"fmt"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
	"os"
	"path/filepath"
	"sync"
)

var (
	Account     AccountConfig
	accountPath string
	accountMu   sync.RWMutex
)

// InitAccount init account config
func InitAccount(path string) error {
	c := viper.New()
	c.SetConfigFile(path)
	if err := c.ReadInConfig(); err != nil {
		return err
	}
	var next AccountConfig
	if err := c.Unmarshal(&next); err != nil {
		return err
	}
	accountMu.Lock()
	Account = next
	accountPath = path
	accountMu.Unlock()

	return nil
}

func GetAccountMasked() AccountConfig {
	accountMu.RLock()
	defer accountMu.RUnlock()
	masked := Account
	masked.BiliBili.DedeUserID = mask(masked.BiliBili.DedeUserID)
	masked.BiliBili.DedeUserIDCkMd5 = mask(masked.BiliBili.DedeUserIDCkMd5)
	masked.BiliBili.SESSDATA = mask(masked.BiliBili.SESSDATA)
	masked.BiliBili.BiliJCT = mask(masked.BiliBili.BiliJCT)
	masked.Huya.Cookies = mask(masked.Huya.Cookies)
	return masked
}

// SaveAccount updates account settings. Empty credential fields retain their stored values.
func SaveAccount(next AccountConfig) error {
	accountMu.Lock()
	defer accountMu.Unlock()
	if accountPath == "" {
		return fmt.Errorf("account configuration path is not initialized")
	}
	mergeCredentials(&next, Account)
	if err := writeAccountYAML(accountPath, next); err != nil {
		return err
	}
	Account = next
	return nil
}

func mergeCredentials(next *AccountConfig, current AccountConfig) {
	if isUnchangedSecret(next.BiliBili.DedeUserID) {
		next.BiliBili.DedeUserID = current.BiliBili.DedeUserID
	}
	if isUnchangedSecret(next.BiliBili.DedeUserIDCkMd5) {
		next.BiliBili.DedeUserIDCkMd5 = current.BiliBili.DedeUserIDCkMd5
	}
	if isUnchangedSecret(next.BiliBili.SESSDATA) {
		next.BiliBili.SESSDATA = current.BiliBili.SESSDATA
	}
	if isUnchangedSecret(next.BiliBili.BiliJCT) {
		next.BiliBili.BiliJCT = current.BiliBili.BiliJCT
	}
	if isUnchangedSecret(next.Huya.Cookies) {
		next.Huya.Cookies = current.Huya.Cookies
	}
}

func isUnchangedSecret(value string) bool { return value == "" || value == "********" }

func mask(value string) string {
	if value == "" {
		return ""
	}
	return "********"
}

func writeAccountYAML(path string, value AccountConfig) error {
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
