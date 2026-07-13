package logger

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/herj1025/kumquat/config"
	"github.com/herj1025/kumquat/pkg/constant"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger = zap.NewNop()

func InitLogger(cfg *config.LogConfig) error {
	logDir := filepath.Dir(cfg.Filename)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	writeSyncer := getLogWriter(cfg)
	encoder := getEncoder()

	var l = new(zapcore.Level)
	err := l.UnmarshalText([]byte(cfg.Level))
	if err != nil {
		return err
	}

	core := zapcore.NewCore(encoder, writeSyncer, l)

	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	if logger == nil {
		return errors.New("failed to init logger")
	}

	return nil
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewJSONEncoder(encoderConfig)
}

func getLogWriter(cfg *config.LogConfig) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}

	return zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(lumberJackLogger))
}

func Info(msg string, fields ...zap.Field) {
	logger.Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	logger.Error(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	logger.Debug(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	logger.Warn(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	logger.Fatal(msg, fields...)
}

// watchedContextKeys 定义了需要从 context 中自动提取并打印的键
var watchedContextKeys = []string{
	constant.RequestTraceIDKey,
	constant.AcceptLanguageKey,
}

func withContextFields(ctx context.Context, fields []zap.Field) []zap.Field {
	if ctx == nil {
		return fields
	}
	for _, key := range watchedContextKeys {
		if v := ctx.Value(key); v != nil {
			if s, ok := v.(string); ok && s != "" {
				fields = append(fields, zap.String(key, s))
			}
		}
	}
	return fields
}

// C 返回一个携带 context 的 Entry，用于链式调用
func C(ctx context.Context) *Entry {
	return &Entry{ctx: ctx}
}

type Entry struct {
	ctx context.Context
}

func (e *Entry) Info(msg string, fields ...zap.Field) {
	logger.Info(msg, withContextFields(e.ctx, fields)...)
}

func (e *Entry) Error(msg string, fields ...zap.Field) {
	logger.Error(msg, withContextFields(e.ctx, fields)...)
}

func (e *Entry) Debug(msg string, fields ...zap.Field) {
	logger.Debug(msg, withContextFields(e.ctx, fields)...)
}

func (e *Entry) Warn(msg string, fields ...zap.Field) {
	logger.Warn(msg, withContextFields(e.ctx, fields)...)
}

func (e *Entry) Fatal(msg string, fields ...zap.Field) {
	logger.Fatal(msg, withContextFields(e.ctx, fields)...)
}

func Sync() error {
	if logger != nil {
		return logger.Sync()
	}
	return nil
}
