package mysql

import (
	"context"
	"time"

	"kubeclaw/backend/internal/logger"

	"go.uber.org/zap"
	gormlogger "gorm.io/gorm/logger"
)

const gormSlowSQLThreshold = 500 * time.Millisecond

type zapGormLogger struct {
	slowThreshold time.Duration
	logLevel      gormlogger.LogLevel
	logger        *zap.Logger
}

func newZapGormLogger() gormlogger.Interface {
	return &zapGormLogger{
		slowThreshold: gormSlowSQLThreshold,
		logLevel:      gormlogger.Info,
		logger:        logger.ForScope(logger.ScopeSQL),
	}
}

func (l *zapGormLogger) LogMode(level gormlogger.LogLevel) gormlogger.Interface {
	return &zapGormLogger{
		slowThreshold: l.slowThreshold,
		logLevel:      level,
		logger:        l.logger,
	}
}

func (l *zapGormLogger) Info(_ context.Context, msg string, args ...any) {
	if l.logLevel < gormlogger.Info {
		return
	}

	l.logger.Info("gorm info", zap.String("message", msg), zap.Any("args", args))
}

func (l *zapGormLogger) Warn(_ context.Context, msg string, args ...any) {
	if l.logLevel < gormlogger.Warn {
		return
	}

	l.logger.Warn("gorm warn", zap.String("message", msg), zap.Any("args", args))
}

func (l *zapGormLogger) Error(_ context.Context, msg string, args ...any) {
	if l.logLevel < gormlogger.Error {
		return
	}

	l.logger.Error("gorm error", zap.String("message", msg), zap.Any("args", args))
}

func (l *zapGormLogger) Trace(_ context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.logLevel == gormlogger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	switch {
	case err != nil && l.logLevel >= gormlogger.Error:
		l.logger.Error(
			"gorm trace",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Int64("latency_ms", elapsed.Milliseconds()),
			zap.Error(err),
		)
	case elapsed > l.slowThreshold && l.logLevel >= gormlogger.Warn:
		l.logger.Warn(
			"gorm slow sql",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Int64("latency_ms", elapsed.Milliseconds()),
			zap.Int64("slow_threshold_ms", l.slowThreshold.Milliseconds()),
		)
	case l.logLevel >= gormlogger.Info:
		l.logger.Debug(
			"gorm trace",
			zap.String("sql", sql),
			zap.Int64("rows", rows),
			zap.Int64("latency_ms", elapsed.Milliseconds()),
		)
	}
}
