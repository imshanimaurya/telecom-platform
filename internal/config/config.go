package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

/*
Config holds all configuration required by the API process.
All values MUST come from environment variables.
No business logic should depend on raw env vars.
*/
type Config struct {
	App    AppConfig
	DB     DBConfig
	Redis  RedisConfig
	Auth   AuthConfig
	Twilio TwilioConfig
}

/* ===================== APP ===================== */

type AppConfig struct {
	Env           string
	Port          int
	Maintenance   bool // UI read-only / banner
	EmergencyStop bool // HARD STOP all calls
}

/* ===================== DATABASE ===================== */

type DBConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string // disable, require, verify-ca, verify-full
}

/* ===================== REDIS ===================== */

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	UseTLS   bool
}

/* ===================== AUTH ===================== */

type AuthConfig struct {
	JWTSecret        string
	JWTIssuer        string
	JWTAudience      string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

/* ===================== TWILIO ===================== */

type TwilioConfig struct {
	AccountSID    string
	AuthToken     string
	WebhookSecret string
}

/* ===================== LOAD ===================== */

func Load() (Config, error) {
	var parseErrs []error
	var err error

	c := Config{}

	/* ---- APP ---- */
	c.App.Env = strings.TrimSpace(os.Getenv("APP_ENV"))
	c.App.Port, err = mustInt("APP_PORT")
	parseErrs = append(parseErrs, err)

	c.App.Maintenance = strings.ToLower(os.Getenv("APP_MAINTENANCE")) == "true"
	c.App.EmergencyStop = strings.ToLower(os.Getenv("APP_EMERGENCY_STOP")) == "true"

	/* ---- DB ---- */
	c.DB.Host = strings.TrimSpace(os.Getenv("DB_HOST"))
	c.DB.Port, err = mustInt("DB_PORT")
	parseErrs = append(parseErrs, err)

	c.DB.User = strings.TrimSpace(os.Getenv("DB_USER"))
	c.DB.Password = os.Getenv("DB_PASSWORD")
	c.DB.Name = strings.TrimSpace(os.Getenv("DB_NAME"))
	c.DB.SSLMode = strings.TrimSpace(os.Getenv("DB_SSLMODE"))

	/* ---- REDIS ---- */
	c.Redis.Host = strings.TrimSpace(os.Getenv("REDIS_HOST"))
	c.Redis.Port, err = mustInt("REDIS_PORT")
	parseErrs = append(parseErrs, err)

	c.Redis.Password = os.Getenv("REDIS_PASSWORD")
	c.Redis.UseTLS = strings.ToLower(os.Getenv("REDIS_TLS")) == "true"

	/* ---- AUTH ---- */
	c.Auth.JWTSecret = os.Getenv("JWT_SECRET")
	c.Auth.JWTIssuer = strings.TrimSpace(os.Getenv("JWT_ISSUER"))
	c.Auth.JWTAudience = strings.TrimSpace(os.Getenv("JWT_AUDIENCE"))

	c.Auth.AccessTokenTTL, err = mustDuration("JWT_ACCESS_TTL")
	parseErrs = append(parseErrs, err)

	c.Auth.RefreshTokenTTL, err = mustDuration("JWT_REFRESH_TTL")
	parseErrs = append(parseErrs, err)

	/* ---- TWILIO ---- */
	c.Twilio.AccountSID = strings.TrimSpace(os.Getenv("TWILIO_ACCOUNT_SID"))
	c.Twilio.AuthToken = os.Getenv("TWILIO_AUTH_TOKEN")
	c.Twilio.WebhookSecret = os.Getenv("TWILIO_WEBHOOK_SECRET")

	/* ---- APPLY DEFAULTS (NO SIDE EFFECTS IN VALIDATE) ---- */
	if c.Auth.AccessTokenTTL == 0 {
		c.Auth.AccessTokenTTL = 15 * time.Minute
	}
	if c.Auth.RefreshTokenTTL == 0 {
		c.Auth.RefreshTokenTTL = 30 * 24 * time.Hour
	}
	if c.DB.SSLMode == "" && !c.IsProduction() {
		c.DB.SSLMode = "disable"
	}

	if err := joinErrors(parseErrs); err != nil {
		return Config{}, err
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}

	return c, nil
}

/* ===================== VALIDATION ===================== */

func (c Config) Validate() error {
	var errs []error

	/* ---- APP ---- */
	if c.App.Env == "" {
		errs = append(errs, errors.New("APP_ENV is required"))
	}
	if !isValidEnv(c.App.Env) {
		errs = append(errs, fmt.Errorf("APP_ENV must be local, dev, staging, or production"))
	}
	if c.App.Port <= 0 || c.App.Port > 65535 {
		errs = append(errs, fmt.Errorf("APP_PORT must be valid"))
	}

	/* ---- DB ---- */
	if c.DB.Host == "" {
		errs = append(errs, errors.New("DB_HOST is required"))
	}
	if c.DB.Port <= 0 {
		errs = append(errs, errors.New("DB_PORT is required"))
	}
	if c.DB.User == "" {
		errs = append(errs, errors.New("DB_USER is required"))
	}
	if c.DB.Name == "" {
		errs = append(errs, errors.New("DB_NAME is required"))
	}
	if c.IsProduction() && c.DB.SSLMode == "" {
		errs = append(errs, errors.New("DB_SSLMODE required in production"))
	}
	if c.DB.SSLMode != "" && !isValidSSLMode(c.DB.SSLMode) {
		errs = append(errs, fmt.Errorf("invalid DB_SSLMODE"))
	}

	/* ---- REDIS ---- */
	if c.Redis.Host == "" {
		errs = append(errs, errors.New("REDIS_HOST is required"))
	}
	if c.Redis.Port <= 0 {
		errs = append(errs, errors.New("REDIS_PORT is required"))
	}

	/* ---- AUTH ---- */
	if c.Auth.JWTSecret == "" {
		errs = append(errs, errors.New("JWT_SECRET is required"))
	}
	if c.IsProduction() {
		if c.Auth.JWTIssuer == "" {
			errs = append(errs, errors.New("JWT_ISSUER required in production"))
		}
		if c.Auth.JWTAudience == "" {
			errs = append(errs, errors.New("JWT_AUDIENCE required in production"))
		}
	}
	if c.Auth.RefreshTokenTTL <= c.Auth.AccessTokenTTL {
		errs = append(errs, errors.New("JWT_REFRESH_TTL must be greater than JWT_ACCESS_TTL"))
	}

	/* ---- TWILIO ---- */
	if c.Twilio.AccountSID != "" || c.Twilio.AuthToken != "" {
		if c.Twilio.AccountSID == "" || c.Twilio.AuthToken == "" {
			errs = append(errs, errors.New(
				"TWILIO_ACCOUNT_SID and TWILIO_AUTH_TOKEN must both be set",
			))
		}
	}

	return joinErrors(errs)
}

/* ===================== HELPERS ===================== */

func (c Config) IsProduction() bool {
	return c.App.Env == "production"
}

func (c Config) HTTPAddr() string {
	return fmt.Sprintf(":%d", c.App.Port)
}

func (c Config) PostgresDSN() string {
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
	return strconv.Atoi(v)
}

func mustDuration(key string) (time.Duration, error) {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return 0, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be valid duration like 15m", key)
	}
	return d, nil
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
	var b strings.Builder
	b.WriteString("config errors:\n")
	for _, e := range errs {
		b.WriteString("- ")
		b.WriteString(e.Error())
		b.WriteString("\n")
	}
	return errors.New(strings.TrimSpace(b.String()))
}
