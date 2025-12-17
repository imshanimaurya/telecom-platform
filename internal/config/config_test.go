package config

import "testing"

func TestLoad_ReportsMissingRequired(t *testing.T) {
	// Ensure a clean env by not setting anything and calling validation directly.
	c := Config{}
	if err := c.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidate_ProductionRequiresSSLMode(t *testing.T) {
	c := Config{
		App: AppConfig{Env: "production", Port: 8080},
		DB: DBConfig{Host: "localhost", Port: 5432, User: "postgres", Password: "x", Name: "telecom", SSLMode: ""},
		Redis: RedisConfig{Host: "localhost", Port: 6379},
		Auth: AuthConfig{JWTSecret: "secret"},
	}
	if err := c.Validate(); err == nil {
		t.Fatalf("expected error for production without DB_SSLMODE")
	}
}

func TestValidate_LocalDefaultsSSLMode(t *testing.T) {
	c := Config{
		App: AppConfig{Env: "local", Port: 8080},
		DB: DBConfig{Host: "localhost", Port: 5432, User: "postgres", Password: "x", Name: "telecom", SSLMode: ""},
		Redis: RedisConfig{Host: "localhost", Port: 6379},
		Auth: AuthConfig{JWTSecret: "secret"},
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if c.DB.SSLMode != "disable" {
		t.Fatalf("expected sslmode disable default, got %q", c.DB.SSLMode)
	}
}
