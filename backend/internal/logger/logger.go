package logger

import (
	"fmt"

	"kubeclaw/backend/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var global *zap.Logger
var bus = NewBus(2000)

func Init(cfg config.Config) (*zap.Logger, error) {
	zapCfg := zap.NewProductionConfig()
	if cfg.LogDevelopment {
		zapCfg = zap.NewDevelopmentConfig()
	}

	zapCfg.Encoding = cfg.LogEncoding
	zapCfg.OutputPaths = []string{"stdout"}
	zapCfg.ErrorOutputPaths = []string{"stderr"}

	level := new(zapcore.Level)
	if err := level.UnmarshalText([]byte(cfg.LogLevel)); err != nil {
		return nil, fmt.Errorf("parse zap log level: %w", err)
	}
	zapCfg.Level = zap.NewAtomicLevelAt(*level)

	logger, err := zapCfg.Build(zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	if err != nil {
		return nil, fmt.Errorf("build zap logger: %w", err)
	}

	logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return zapcore.NewTee(core, newBusCore(bus))
	}))

	global = logger
	zap.ReplaceGlobals(logger)
	return logger, nil
}

func L() *zap.Logger {
	if global != nil {
		return global
	}
	return zap.L()
}

func S() *zap.SugaredLogger {
	return L().Sugar()
}

func ForScope(scope string) *zap.Logger {
	return L().With(zap.String("scope", scope))
}

func GlobalBus() *Bus {
	return bus
}

type busCore struct {
	bus    *Bus
	fields []zap.Field
}

func newBusCore(bus *Bus) zapcore.Core {
	return &busCore{bus: bus}
}

func (c *busCore) Enabled(level zapcore.Level) bool {
	return level >= zapcore.DebugLevel
}

func (c *busCore) With(fields []zap.Field) zapcore.Core {
	return &busCore{
		bus:    c.bus,
		fields: append(append([]zap.Field{}, c.fields...), fields...),
	}
}

func (c *busCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return checked.AddCore(entry, c)
	}
	return checked
}

func (c *busCore) Write(entry zapcore.Entry, fields []zap.Field) error {
	allFields := append(append([]zap.Field{}, c.fields...), fields...)
	encoder := zapcore.NewMapObjectEncoder()
	for _, field := range allFields {
		field.AddTo(encoder)
	}

	scope := ScopeRuntime
	if value, ok := encoder.Fields["scope"].(string); ok && value != "" {
		scope = value
		delete(encoder.Fields, "scope")
	}

	c.bus.Publish(scope, entry.Level.String(), entry.Message, encoder.Fields)
	return nil
}

func (c *busCore) Sync() error {
	return nil
}
