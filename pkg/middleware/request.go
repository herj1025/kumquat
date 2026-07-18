package middleware

import (
	"time"

	"github.com/herj1025/kumquat/pkg/constant"
	"github.com/herj1025/kumquat/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func RequestTrace() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(constant.RequestTraceIDKey)
		if traceID == "" {
			traceID = uuid.New().String()
		}
		c.Set(constant.RequestTraceIDKey, traceID)
		c.Writer.Header().Set(constant.RequestTraceIDKey, traceID)

		start := time.Now()

		c.Next()

		duration := time.Since(start)

		logger.Info("http access",
			zap.String("trace_id", traceID),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("duration_ms", duration.Milliseconds()),
		)
	}
}
