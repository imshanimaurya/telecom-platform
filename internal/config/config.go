package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration required by the API process.
// All values must come from env (or env-file loaded by the process runner).
// No business logic should depend on raw environment variables.
type Config struct {
	App   AppConfig
	DB    DBConfig
	Redis RedisConfig
	Auth  AuthConfig
	Twilio TwilioConfig
}

type AppConfig struct {
	Env  string
	Port int
}

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string

	// SSLMode is kept explicit for AWS-ready posture.
	// Accepts: disable, require, verify-ca, verify-full
	SSLMode string
}

type RedisConfig struct {
	Host string
	Port int
}

type AuthConfig struct {
	JWTSecret string
	JWTIssuer string
	JWTAudience string
	AccessTokenTTL time.Duration
	RefreshTokenTTL time.Duration
}

type TwilioConfig struct {
	AccountSID    string
	AuthToken     string
	WebhookSecret string
}

func Load() (Config, error) {
	c := Config{}
	var parseErrs []error

	c.App.Env = strings.TrimSpace(os.Getenv("APP_ENV"))
	{
		n, err := mustInt("APP_PORT")
		n, parseErrs = appendParseErr(parseErrs, n, err)
		c.App.Port = n
	}

	c.DB.Host = strings.TrimSpace(os.Getenv("DB_HOST"))
	{
		n, err := mustInt("DB_PORT")
		n, parseErrs = appendParseErr(parseErrs, n, err)
		c.DB.Port = n
	}
	c.DB.User = strings.TrimSpace(os.Getenv("DB_USER"))
	c.DB.Password = os.Getenv("DB_PASSWORD")
	c.DB.Name = strings.TrimSpace(os.Getenv("DB_NAME"))
	c.DB.SSLMode = strings.TrimSpace(os.Getenv("DB_SSLMODE"))

	c.Redis.Host = strings.TrimSpace(os.Getenv("REDIS_HOST"))
	{
		n, err := mustInt("REDIS_PORT")
		n, parseErrs = appendParseErr(parseErrs, n, err)
		c.Redis.Port = n
	}

	c.Auth.JWTSecret = os.Getenv("JWT_SECRET")
	c.Auth.JWTIssuer = strings.TrimSpace(os.Getenv("JWT_ISSUER"))
	c.Auth.JWTAudience = strings.TrimSpace(os.Getenv("JWT_AUDIENCE"))
	// Duration env vars are optional; defaults applied in Validate() based on env.
	c.Auth.AccessTokenTTL = mustDuration("JWT_ACCESS_TTL")
	c.Auth.RefreshTokenTTL = mustDuration("JWT_REFRESH_TTL")

	c.Twilio.AccountSID = strings.TrimSpace(os.Getenv("TWILIO_ACCOUNT_SID"))
	c.Twilio.AuthToken = os.Getenv("TWILIO_AUTH_TOKEN")
	c.Twilio.WebhookSecret = os.Getenv("TWILIO_WEBHOOK_SECRET")

	if err := joinErrors(parseErrs); err != nil {
		return Config{}, err
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func (c Config) Validate() error {
	var errs []error

	if c.App.Env == "" {
		errs = append(errs, errors.New("APP_ENV is required"))
	} else if !isValidEnv(c.App.Env) {
		errs = append(errs, fmt.Errorf("APP_ENV must be one of local, dev, staging, production, got %q", c.App.Env))
	}
	if c.App.Port <= 0 || c.App.Port > 65535 {
		errs = append(errs, fmt.Errorf("APP_PORT must be a valid port, got %d", c.App.Port))
	}

	if c.DB.Host == "" {
		errs = append(errs, errors.New("DB_HOST is required"))
	}
	if c.DB.Port <= 0 || c.DB.Port > 65535 {
		errs = append(errs, fmt.Errorf("DB_PORT must be a valid port, got %d", c.DB.Port))
	}
	if c.DB.User == "" {
		errs = append(errs, errors.New("DB_USER is required"))
	}
	if c.DB.Name == "" {
		errs = append(errs, errors.New("DB_NAME is required"))
	}
	if strings.TrimSpace(c.DB.SSLMode) == "" {
		if c.IsProduction() {
			errs = append(errs, errors.New("DB_SSLMODE is required in production"))
		} else {
			// Local-friendly default; production must be explicit.
			// Allowed values are enforced below.
			c.DB.SSLMode = "disable"
		}
	}
	if c.DB.SSLMode != "" && !isValidSSLMode(c.DB.SSLMode) {
		errs = append(errs, fmt.Errorf("DB_SSLMODE must be one of disable, require, verify-ca, verify-full, got %q", c.DB.SSLMode))
	}

	if c.Redis.Host == "" {
		errs = append(errs, errors.New("REDIS_HOST is required"))
	}
	if c.Redis.Port <= 0 || c.Redis.Port > 65535 {
		errs = append(errs, fmt.Errorf("REDIS_PORT must be a valid port, got %d", c.Redis.Port))
	}

	if c.Auth.JWTSecret == "" {
		errs = append(errs, errors.New("JWT_SECRET is required"))
	}
	if c.IsProduction() {
		if c.Auth.JWTIssuer == "" {
			errs = append(errs, errors.New("JWT_ISSUER is required in production"))
		}
		if c.Auth.JWTAudience == "" {
			errs = append(errs, errors.New("JWT_AUDIENCE is required in production"))
		}
	}

	if c.Auth.AccessTokenTTL <= 0 {
		// Default: short-lived access tokens.
		c.Auth.AccessTokenTTL = 15 * time.Minute
	}
	if c.Auth.RefreshTokenTTL <= 0 {
		// Default: longer-lived refresh tokens.
		c.Auth.RefreshTokenTTL = 30 * 24 * time.Hour
	}
	if c.Auth.RefreshTokenTTL <= c.Auth.AccessTokenTTL {
		errs = append(errs, errors.New("JWT_REFRESH_TTL must be greater than JWT_ACCESS_TTL"))
	}

	return joinErrors(errs)
}

func (c Config) IsProduction() bool {
	return c.App.Env == "production"
}

func (c Config) HTTPAddr() string {
	return fmt.Sprintf(":%d", c.App.Port)
}

func (c Config) PostgresDSN() string {
	// Avoid logging this string; it contains secrets.
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.DB.Host,
		c.DB.Port,
		c.DB.User,
		c.DB.Password,
		c.DB.Name,
		c.DB.SSLMode,
	)
}

func (c Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

func mustInt(key string) (int, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0, fmt.Errorf("%s is required", key)
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", key, v)
	}
	return n, nil
}

func mustDuration(key string) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0
	}
	return d
}

func appendParseErr(errs []error, n int, err error) (int, []error) {
	if err != nil {
		errs = append(errs, err)
	}
	return n, errs
}

func isValidEnv(v string) bool {
	switch v {
	case "local", "dev", "staging", "production":
		return true
	default:
		return false
	}
}

func isValidSSLMode(v string) bool {
	switch v {
	case "disable", "require", "verify-ca", "verify-full":
		return true
	default:
		return false
	}
}

func joinErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	var b strings.Builder
	b.WriteString("config errors:\n")
	for _, e := range errs {
		b.WriteString("- ")
		b.WriteString(e.Error())
		b.WriteString("\n")
	}
	return errors.New(strings.TrimSpace(b.String()))
}
