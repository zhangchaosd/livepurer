package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveServerWritesValidConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "server.yaml")
	if err := os.WriteFile(path, []byte("port: 8800\ndebug: false\npath: ./data\nsocks5:\n  enable: false\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := InitServer(path); err != nil {
		t.Fatal(err)
	}
	if err := SaveServer(ServerConfig{Port: 9900, Debug: true, Path: "./new-data"}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "port: 9900") || !strings.Contains(string(content), "path: ./new-data") {
		t.Fatalf("unexpected saved content: %s", content)
	}
}

func TestSaveChannelsCreatesOptionalConfiguration(t *testing.T) {
	path := filepath.Join(t.TempDir(), "channels.yaml")
	_ = InitChannels(path) // A missing file is a supported first-run state.
	channels := []Channel{{Plat: "bilibili", Room: "6", Name: "官方"}}
	if err := SaveChannels(channels); err != nil {
		t.Fatal(err)
	}
	if got := GetChannels(); len(got) != 1 || got[0].Room != "6" {
		t.Fatalf("channels = %#v", got)
	}
}
