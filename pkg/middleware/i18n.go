package middleware

import (
	"strings"

	"github.com/herj1025/kumquat/pkg/constant"

	"github.com/gin-gonic/gin"
)

func I18n() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.GetHeader(constant.AcceptLanguageKey)
		finalLang := "zh-CN"

		if lang != "" {
			parts := strings.Split(lang, ",")
			if len(parts) > 0 {
				primary := strings.TrimSpace(parts[0])
				subParts := strings.Split(primary, ";")
				if len(subParts) > 0 {
					code := strings.TrimSpace(subParts[0])
					if strings.HasPrefix(code, "en") {
						finalLang = "en-US"
					} else if strings.HasPrefix(code, "zh") {
						finalLang = "zh-CN"
					}
				}
			}
		}

		c.Set(constant.ContextLanguageKey, finalLang)
		c.Next()
	}
}
