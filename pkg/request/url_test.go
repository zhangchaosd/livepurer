package request

import "testing"

func TestValidatePublicURLRejectsUnsafeLiterals(t *testing.T) {
	tests := []string{
		"http://127.0.0.1/", "http://[::1]/", "http://10.0.0.1/", "http://169.254.1.1/", "http://localhost/",
	}
	for _, raw := range tests {
		if _, err := ValidatePublicURL(raw, "http", "https"); err == nil {
			t.Errorf("ValidatePublicURL(%q) accepted an unsafe target", raw)
		}
	}
}

func TestValidatePublicURLRejectsInvalidSchemeAndUserInfo(t *testing.T) {
	for _, raw := range []string{"file:///etc/passwd", "http://user@example.com/"} {
		if _, err := ValidatePublicURL(raw, "http", "https"); err == nil {
			t.Errorf("ValidatePublicURL(%q) unexpectedly succeeded", raw)
		}
	}
}
