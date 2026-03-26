package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 汇总后端服务当前阶段需要的基础配置。
type Config struct {
	AppName         string        `yaml:"app_name"`
	Env             string        `yaml:"env"`
	AppVersion      string        `yaml:"app_version"`
	LogLevel        string        `yaml:"log_level"`
	LogEncoding     string        `yaml:"log_encoding"`
	LogDevelopment  bool          `yaml:"log_development"`
	HTTPAddr        string        `yaml:"http_addr"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`

	JWTSecret       string        `yaml:"jwt_secret"`
	DataSecret      string        `yaml:"data_secret"`
	AccessTokenTTL  time.Duration `yaml:"access_token_ttl"`
	RefreshTokenTTL time.Duration `yaml:"refresh_token_ttl"`

	BootstrapAdminUsername string `yaml:"bootstrap_admin_username"`
	BootstrapAdminEmail    string `yaml:"bootstrap_admin_email"`
	BootstrapAdminPassword string `yaml:"bootstrap_admin_password"`
	SystemTenantName       string `yaml:"system_tenant_name"`
	SystemTenantSlug       string `yaml:"system_tenant_slug"`

	MySQLHost            string        `yaml:"mysql_host"`
	MySQLPort            int           `yaml:"mysql_port"`
	MySQLUser            string        `yaml:"mysql_user"`
	MySQLPassword        string        `yaml:"mysql_password"`
	MySQLDatabase        string        `yaml:"mysql_database"`
	MySQLCharset         string        `yaml:"mysql_charset"`
	MySQLParseTime       bool          `yaml:"mysql_parse_time"`
	MySQLMaxOpenConns    int           `yaml:"mysql_max_open_conns"`
	MySQLMaxIdleConns    int           `yaml:"mysql_max_idle_conns"`
	MySQLConnMaxLifetime time.Duration `yaml:"mysql_conn_max_lifetime"`
	MySQLAutoMigrate     bool          `yaml:"mysql_auto_migrate"`
}

// Load 优先读取默认 YAML 配置，再用环境变量覆盖。
func Load() (Config, error) {
	cfg, err := loadFromYAML(resolveConfigFile())
	if err != nil {
		return Config{}, err
	}

	overrideWithEnv(&cfg)
	return cfg, nil
}

func resolveConfigFile() string {
	if value := os.Getenv("CONFIG_FILE"); value != "" {
		return value
	}

	return filepath.Join("configs", "default.yaml")
}

func loadFromYAML(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config yaml %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("unmarshal config yaml %s: %w", path, err)
	}

	return cfg, nil
}

func overrideWithEnv(cfg *Config) {
	cfg.AppName = envOrDefault("APP_NAME", cfg.AppName)
	cfg.Env = envOrDefault("APP_ENV", cfg.Env)
	cfg.AppVersion = envOrDefault("APP_VERSION", cfg.AppVersion)
	cfg.LogLevel = envOrDefault("LOG_LEVEL", cfg.LogLevel)
	cfg.LogEncoding = envOrDefault("LOG_ENCODING", cfg.LogEncoding)
	cfg.LogDevelopment = boolOrDefault("LOG_DEVELOPMENT", cfg.LogDevelopment)
	cfg.HTTPAddr = envOrDefault("HTTP_ADDR", cfg.HTTPAddr)
	cfg.ReadTimeout = durationOrDefault("HTTP_READ_TIMEOUT", cfg.ReadTimeout)
	cfg.WriteTimeout = durationOrDefault("HTTP_WRITE_TIMEOUT", cfg.WriteTimeout)
	cfg.ShutdownTimeout = durationOrDefault("HTTP_SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout)

	cfg.JWTSecret = envOrDefault("JWT_SECRET", cfg.JWTSecret)
	cfg.DataSecret = envOrDefault("DATA_SECRET", cfg.DataSecret)
	cfg.AccessTokenTTL = durationOrDefault("ACCESS_TOKEN_TTL", cfg.AccessTokenTTL)
	cfg.RefreshTokenTTL = durationOrDefault("REFRESH_TOKEN_TTL", cfg.RefreshTokenTTL)

	cfg.BootstrapAdminUsername = envOrDefault("BOOTSTRAP_ADMIN_USERNAME", cfg.BootstrapAdminUsername)
	cfg.BootstrapAdminEmail = envOrDefault("BOOTSTRAP_ADMIN_EMAIL", cfg.BootstrapAdminEmail)
	cfg.BootstrapAdminPassword = envOrDefault("BOOTSTRAP_ADMIN_PASSWORD", cfg.BootstrapAdminPassword)
	cfg.SystemTenantName = envOrDefault("SYSTEM_TENANT_NAME", cfg.SystemTenantName)
	cfg.SystemTenantSlug = envOrDefault("SYSTEM_TENANT_SLUG", cfg.SystemTenantSlug)

	cfg.MySQLHost = envOrDefault("MYSQL_HOST", cfg.MySQLHost)
	cfg.MySQLPort = intOrDefault("MYSQL_PORT", cfg.MySQLPort)
	cfg.MySQLUser = envOrDefault("MYSQL_USER", cfg.MySQLUser)
	cfg.MySQLPassword = envOrDefault("MYSQL_PASSWORD", cfg.MySQLPassword)
	cfg.MySQLDatabase = envOrDefault("MYSQL_DATABASE", cfg.MySQLDatabase)
	cfg.MySQLCharset = envOrDefault("MYSQL_CHARSET", cfg.MySQLCharset)
	cfg.MySQLParseTime = boolOrDefault("MYSQL_PARSE_TIME", cfg.MySQLParseTime)
	cfg.MySQLMaxOpenConns = intOrDefault("MYSQL_MAX_OPEN_CONNS", cfg.MySQLMaxOpenConns)
	cfg.MySQLMaxIdleConns = intOrDefault("MYSQL_MAX_IDLE_CONNS", cfg.MySQLMaxIdleConns)
	cfg.MySQLConnMaxLifetime = durationOrDefault("MYSQL_CONN_MAX_LIFETIME", cfg.MySQLConnMaxLifetime)
	cfg.MySQLAutoMigrate = boolOrDefault("MYSQL_AUTO_MIGRATE", cfg.MySQLAutoMigrate)
}

// envOrDefault 用于读取字符串环境变量。
func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}

// durationOrDefault 用于读取 duration 类型配置，例如 15s、2h。
func durationOrDefault(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}

// intOrDefault 用于读取整数配置。
func intOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

// boolOrDefault 用于读取布尔配置。
func boolOrDefault(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}
