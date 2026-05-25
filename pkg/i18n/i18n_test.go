package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTranslateParams(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "zh-CN.toml"), []byte("\"common.greet\" = \"你好 %s\"\n"), 0644); err != nil {
		t.Fatalf("write zh-CN.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "en-US.toml"), []byte("\"common.greet\" = \"Hello %s\"\n"), 0644); err != nil {
		t.Fatalf("write en-US.toml: %v", err)
	}

	if err := Load(dir); err != nil {
		t.Fatalf("Load: %v", err)
	}

	got := Translate("en-US", "common.greet", "Bob")
	if got != "Hello Bob" {
		t.Fatalf("expected %q, got %q", "Hello Bob", got)
	}

	got = Translate("zh-CN", "common.greet", "Bob")
	if got != "你好 Bob" {
		t.Fatalf("expected %q, got %q", "你好 Bob", got)
	}

	got = Translate("fr-FR", "common.greet", "Bob")
	if got != "你好 Bob" {
		t.Fatalf("expected %q, got %q", "你好 Bob", got)
	}
}
