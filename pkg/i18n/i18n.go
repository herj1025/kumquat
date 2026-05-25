package i18n

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/language"

	"github.com/herj1025/kumquat/pkg/constant"
)

var (
	bundle     *i18n.Bundle
	localizers map[string]*i18n.Localizer
	mu         sync.RWMutex
)

const defaultLang = "zh-CN"

func init() {
	bundle = i18n.NewBundle(language.Chinese)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)
}

func Load(path string) error {
	mu.Lock()
	defer mu.Unlock()

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	if localizers == nil {
		localizers = make(map[string]*i18n.Localizer)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}

		lang := strings.TrimSuffix(entry.Name(), ".toml")
		if _, err := bundle.LoadMessageFile(filepath.Join(path, entry.Name())); err != nil {
			return err
		}
		// 允许二次开发覆盖已有语言的 Localizer
		localizers[lang] = i18n.NewLocalizer(bundle, lang)
	}

	return nil
}

func GinGetLang(c *gin.Context) string {
	if v, ok := c.Get(constant.ContextLanguageKey); ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return defaultLang
}

func getLocalizer(lang string) *i18n.Localizer {
	mu.RLock()
	defer mu.RUnlock()

	if l, ok := localizers[lang]; ok {
		return l
	}
	if l, ok := localizers[defaultLang]; ok {
		return l
	}
	return nil
}

func Translate(lang, key string, args ...any) string {
	l := getLocalizer(lang)
	if l == nil {
		return key
	}

	msg, err := l.Localize(&i18n.LocalizeConfig{MessageID: key})
	if err != nil {
		return key
	}

	if len(args) > 0 {
		return fmt.Sprintf(msg, args...)
	}
	return msg
}
