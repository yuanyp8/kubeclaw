package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadReadsYamlAndAppliesEnvironmentOverrides(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "test.yaml")
	content := []byte(`
app_name: kubeclaw-test
env: dev
http_addr: ":8080"
read_timeout: 10s
write_timeout: 15s
shutdown_timeout: 20s
jwt_secret: yaml-secret
mysql_port: 3306
mysql_auto_migrate: false
`)
	if err := os.WriteFile(configPath, content, 0o600); err != nil {
		t.Fatalf("write config yaml: %v", err)
	}

	t.Setenv("CONFIG_FILE", configPath)
	t.Setenv("HTTP_ADDR", ":18080")
	t.Setenv("JWT_SECRET", "env-secret")
	t.Setenv("MYSQL_PORT", "4406")
	t.Setenv("MYSQL_AUTO_MIGRATE", "true")
	t.Setenv("HTTP_READ_TIMEOUT", "45s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.HTTPAddr != ":18080" {
		t.Fatalf("expected env override for HTTP_ADDR, got %s", cfg.HTTPAddr)
	}
	if cfg.JWTSecret != "env-secret" {
		t.Fatalf("expected env override for JWT_SECRET, got %s", cfg.JWTSecret)
	}
	if cfg.MySQLPort != 4406 {
		t.Fatalf("expected env override for MYSQL_PORT, got %d", cfg.MySQLPort)
	}
	if !cfg.MySQLAutoMigrate {
		t.Fatal("expected env override for MYSQL_AUTO_MIGRATE")
	}
	if cfg.ReadTimeout != 45*time.Second {
		t.Fatalf("expected overridden read timeout, got %s", cfg.ReadTimeout)
	}
}

func TestOverrideWithInvalidEnvFallsBackToYamlValue(t *testing.T) {
	cfg := Config{
		MySQLPort:            3306,
		MySQLAutoMigrate:     true,
		MySQLConnMaxLifetime: 30 * time.Minute,
	}

	t.Setenv("MYSQL_PORT", "not-a-number")
	t.Setenv("MYSQL_AUTO_MIGRATE", "not-a-bool")
	t.Setenv("MYSQL_CONN_MAX_LIFETIME", "not-a-duration")

	overrideWithEnv(&cfg)

	if cfg.MySQLPort != 3306 {
		t.Fatalf("expected fallback mysql port 3306, got %d", cfg.MySQLPort)
	}
	if !cfg.MySQLAutoMigrate {
		t.Fatal("expected fallback mysql auto migrate to stay true")
	}
	if cfg.MySQLConnMaxLifetime != 30*time.Minute {
		t.Fatalf("expected fallback conn max lifetime 30m, got %s", cfg.MySQLConnMaxLifetime)
	}
}
