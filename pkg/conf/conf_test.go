package conf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAccountSettingsAreMaskedAndSecretsAreRetained(t *testing.T) {
	path := filepath.Join(t.TempDir(), "account.yaml")
	input := "bilibili:\n  enable: true\n  SESSDATA: secret\nhuya:\n  enable: false\ndouyu:\n  enable: false\n"
	if err := os.WriteFile(path, []byte(input), 0600); err != nil {
		t.Fatal(err)
	}
	if err := InitAccount(path); err != nil {
		t.Fatal(err)
	}
	if got := GetAccountMasked().BiliBili.SESSDATA; got != "********" {
		t.Fatalf("masked value = %q", got)
	}
	if err := SaveAccount(AccountConfig{BiliBili: BiliBiliConfig{Enable: true, SESSDATA: "********"}}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "SESSDATA: secret") {
		t.Fatalf("secret was not retained: %s", content)
	}
}
