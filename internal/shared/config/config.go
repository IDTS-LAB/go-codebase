package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"go.uber.org/fx"
)

var Module = fx.Module("config", fx.Provide(New))

// Config is the root configuration struct, grouped by usage.
type Config struct {
	App         AppConfig         `koanf:"app"`
	Server      ServerConfig      `koanf:"server"`
	Database    DatabaseConfig    `koanf:"database"`
	Redis       RedisConfig       `koanf:"redis"`
	NATS        NATSConfig        `koanf:"nats"`
	Auth        AuthConfig        `koanf:"auth"`
	RateLimit   RateLimitConfig   `koanf:"rate_limit"`
	CORS        CORSConfig        `koanf:"cors"`
	Idempotency IdempotencyConfig `koanf:"idempotency"`
	Log         LogConfig         `koanf:"log"`
	Telemetry   TelemetryConfig   `koanf:"telemetry"`
	Asynq       AsynqConfig       `koanf:"asynq"`
	Email       EmailConfig       `koanf:"email"`
	Tenant      TenantConfig      `koanf:"multitenancy"`
}

// AppConfig holds application-level settings.
type AppConfig struct {
	Env string `koanf:"env"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port               int `koanf:"port"`
	ReadTimeout        int `koanf:"read_timeout"`
	WriteTimeout       int `koanf:"write_timeout"`
	IdleTimeout        int `koanf:"idle_timeout"`
	MaxRequestBodySize int `koanf:"max_request_body_size"`
}

// DatabaseConfig holds PostgreSQL connection settings.
type DatabaseConfig struct {
	Host            string `koanf:"host"`
	Port            int    `koanf:"port"`
	User            string `koanf:"user"`
	Password        string `koanf:"password"`
	Name            string `koanf:"name"`
	SSLMode         string `koanf:"sslmode"`
	MaxOpenConns    int    `koanf:"max_open_conns"`
	MaxIdleConns    int    `koanf:"max_idle_conns"`
	ConnMaxLifetime int    `koanf:"conn_max_lifetime"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Addr     string `koanf:"addr"`
	Password string `koanf:"password"`
	DB       int    `koanf:"db"`
	PoolSize int    `koanf:"pool_size"`
}

// NATSConfig holds NATS connection settings.
type NATSConfig struct {
	URL string `koanf:"url"`
}

// AuthConfig holds authentication and security settings.
type AuthConfig struct {
	JWTSecret        string `koanf:"secret"`
	JWTExpiration    int    `koanf:"expiration"`
	MaxLoginAttempts int    `koanf:"max_login_attempts"`
	LockoutDuration  int    `koanf:"lockout_duration"`
	TokenDenylist    bool   `koanf:"token_denylist"`
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	Requests int `koanf:"requests"`
	Window   int `koanf:"window"`
}

// CORSConfig holds CORS settings.
type CORSConfig struct {
	AllowedOrigins   []string `koanf:"allowed_origins"`
	AllowedMethods   []string `koanf:"allowed_methods"`
	AllowedHeaders   []string `koanf:"allowed_headers"`
	AllowCredentials bool     `koanf:"allow_credentials"`
	MaxAge           int      `koanf:"max_age"`
}

// IdempotencyConfig holds idempotency settings.
type IdempotencyConfig struct {
	Enabled bool `koanf:"enabled"`
	TTL     int  `koanf:"ttl"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

// TelemetryConfig holds OpenTelemetry settings.
type TelemetryConfig struct {
	ServiceName      string  `koanf:"service_name"`
	ExporterEndpoint string  `koanf:"exporter_endpoint"`
	SampleRate       float64 `koanf:"sample_rate"`
}

// AsynqConfig holds background job settings.
type AsynqConfig struct {
	RedisAddr string `koanf:"redis_addr"`
}

// TenantConfig holds multi-tenancy settings.
type TenantConfig struct {
	Enabled        bool   `koanf:"enabled"`
	TenantHeader   string `koanf:"tenant_header"`
	TenantJWTClaim string `koanf:"tenant_jwt_claim"`
	Domain         string `koanf:"domain"`
}

// EmailConfig holds email service settings.
type EmailConfig struct {
	Provider    string         `koanf:"provider"`
	From        string         `koanf:"from"`
	FromName    string         `koanf:"from_name"`
	FrontendURL string         `koanf:"frontend_url"`
	SMTP        SMTPConfig     `koanf:"smtp"`
	SendGrid    SendGridConfig `koanf:"sendgrid"`
}

// SMTPConfig holds SMTP server settings.
type SMTPConfig struct {
	Host     string `koanf:"host"`
	Port     int    `koanf:"port"`
	Username string `koanf:"username"`
	Password string `koanf:"password"`
	UseTLS   bool   `koanf:"use_tls"`
}

// SendGridConfig holds SendGrid API settings.
type SendGridConfig struct {
	APIKey string `koanf:"api_key"`
}

func New() (*Config, error) {
	k := koanf.New(".")

	if err := k.Load(file.Provider("configs/config.yaml"), yaml.Parser()); err != nil {
		fmt.Printf("warning: could not load config.yaml: %v\n", err)
	}

	if err := k.Load(env.ProviderWithValue("", ".", func(s string, v string) (string, interface{}) {
		return strings.ReplaceAll(strings.ToLower(s), "_", "."), v
	}), nil); err != nil {
		fmt.Printf("warning: could not load env config: %v\n", err)
	}

	cfg := &Config{}
	if err := k.Unmarshal("", cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	applyEnvOverrides(cfg)
	setDefaults(cfg)

	if cfg.App.Env == "production" && cfg.Auth.JWTSecret == "your-secret-key-change-in-production" {
		return nil, fmt.Errorf("JWT_SECRET must be changed in production")
	}

	return cfg, nil
}

func setDefaults(cfg *Config) {
	// App
	if cfg.App.Env == "" {
		cfg.App.Env = "development"
	}

	// Server
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30
	}
	if cfg.Server.IdleTimeout == 0 {
		cfg.Server.IdleTimeout = 120
	}
	if cfg.Server.MaxRequestBodySize == 0 {
		cfg.Server.MaxRequestBodySize = 10 * 1024 * 1024
	}

	// Database
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 25
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 5
	}
	if cfg.Database.ConnMaxLifetime == 0 {
		cfg.Database.ConnMaxLifetime = 300
	}

	// Auth
	if cfg.Auth.JWTExpiration == 0 {
		cfg.Auth.JWTExpiration = 3600
	}
	if cfg.Auth.MaxLoginAttempts == 0 {
		cfg.Auth.MaxLoginAttempts = 5
	}
	if cfg.Auth.LockoutDuration == 0 {
		cfg.Auth.LockoutDuration = 900
	}
	cfg.Auth.TokenDenylist = true

	// Rate Limit
	if cfg.RateLimit.Requests == 0 {
		cfg.RateLimit.Requests = 100
	}
	if cfg.RateLimit.Window == 0 {
		cfg.RateLimit.Window = 60
	}

	// CORS
	if cfg.CORS.AllowedOrigins == nil {
		cfg.CORS.AllowedOrigins = []string{"*"}
	}
	if cfg.CORS.AllowedMethods == nil {
		cfg.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	}
	if cfg.CORS.AllowedHeaders == nil {
		cfg.CORS.AllowedHeaders = []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"}
	}
	if cfg.CORS.MaxAge == 0 {
		cfg.CORS.MaxAge = 300
	}

	// Idempotency
	if cfg.Idempotency.TTL == 0 {
		cfg.Idempotency.TTL = 86400
	}

	// Telemetry
	if cfg.Telemetry.SampleRate == 0 {
		cfg.Telemetry.SampleRate = 1.0
	}

	// Email
	if cfg.Email.Provider == "" {
		cfg.Email.Provider = "console"
	}
	if cfg.Email.From == "" {
		cfg.Email.From = "no-reply@example.com"
	}
	if cfg.Email.FromName == "" {
		cfg.Email.FromName = "App"
	}
	if cfg.Email.FrontendURL == "" {
		cfg.Email.FrontendURL = "http://localhost:3000"
	}
	if cfg.Email.SMTP.Host == "" {
		cfg.Email.SMTP.Host = "localhost"
	}
	if cfg.Email.SMTP.Port == 0 {
		cfg.Email.SMTP.Port = 587
	}
	if !cfg.Email.SMTP.UseTLS && cfg.Email.SMTP.Host == "localhost" {
		cfg.Email.SMTP.UseTLS = true
	}

	// Tenant
	if cfg.Tenant.TenantHeader == "" {
		cfg.Tenant.TenantHeader = "X-Tenant-ID"
	}
	if cfg.Tenant.TenantJWTClaim == "" {
		cfg.Tenant.TenantJWTClaim = "tenant_id"
	}
	if cfg.Tenant.Domain == "" {
		cfg.Tenant.Domain = "app.com"
	}
}

func applyEnvOverrides(cfg *Config) {
	// App
	if v := os.Getenv("APP_ENV"); v != "" {
		cfg.App.Env = v
	}

	// Server
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("SERVER_READ_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Server.ReadTimeout = n
		}
	}
	if v := os.Getenv("SERVER_WRITE_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Server.WriteTimeout = n
		}
	}
	if v := os.Getenv("SERVER_IDLE_TIMEOUT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Server.IdleTimeout = n
		}
	}
	if v := os.Getenv("MAX_REQUEST_BODY_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			cfg.Server.MaxRequestBodySize = int(n)
		}
	}

	// Database
	if v := os.Getenv("DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("DB_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		}
	}
	if v := os.Getenv("DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v := os.Getenv("DB_PASSWORD"); v != "" {
		cfg.Database.Password = v
	}
	if v := os.Getenv("DB_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		cfg.Database.SSLMode = v
	}
	if v := os.Getenv("DB_MAX_OPEN_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxOpenConns = n
		}
	}
	if v := os.Getenv("DB_MAX_IDLE_CONNS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.MaxIdleConns = n
		}
	}
	if v := os.Getenv("DB_CONN_MAX_LIFETIME"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.ConnMaxLifetime = n
		}
	}

	// Redis
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("REDIS_PASSWORD"); v != "" {
		cfg.Redis.Password = v
	}
	if v := os.Getenv("REDIS_DB"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Redis.DB = n
		}
	}
	if v := os.Getenv("REDIS_POOL_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Redis.PoolSize = n
		}
	}

	// NATS
	if v := os.Getenv("NATS_URL"); v != "" {
		cfg.NATS.URL = v
	}

	// Auth
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("JWT_EXPIRATION"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Auth.JWTExpiration = n
		}
	}
	if v := os.Getenv("MAX_LOGIN_ATTEMPTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Auth.MaxLoginAttempts = n
		}
	}
	if v := os.Getenv("LOCKOUT_DURATION"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Auth.LockoutDuration = n
		}
	}
	if v := os.Getenv("TOKEN_DENYLIST"); v != "" {
		cfg.Auth.TokenDenylist = v == "true" || v == "1"
	}

	// CORS
	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		cfg.CORS.AllowedOrigins = strings.Split(v, ",")
	}
	if v := os.Getenv("CORS_ALLOW_CREDENTIALS"); v != "" {
		cfg.CORS.AllowCredentials = v == "true" || v == "1"
	}
	if v := os.Getenv("CORS_MAX_AGE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.CORS.MaxAge = n
		}
	}

	// Rate Limit
	if v := os.Getenv("RATE_LIMIT_REQUESTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimit.Requests = n
		}
	}
	if v := os.Getenv("RATE_LIMIT_WINDOW"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.RateLimit.Window = n
		}
	}

	// Idempotency
	if v := os.Getenv("IDEMPOTENCY_ENABLED"); v != "" {
		cfg.Idempotency.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("IDEMPOTENCY_TTL"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Idempotency.TTL = n
		}
	}

	// Logging
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
	if v := os.Getenv("LOG_FORMAT"); v != "" {
		cfg.Log.Format = v
	}

	// Telemetry
	if v := os.Getenv("OTEL_SERVICE_NAME"); v != "" {
		cfg.Telemetry.ServiceName = v
	}
	if v := os.Getenv("OTEL_EXPORTER_ENDPOINT"); v != "" {
		cfg.Telemetry.ExporterEndpoint = v
	}
	if v := os.Getenv("OTEL_SAMPLE_RATE"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			cfg.Telemetry.SampleRate = f
		}
	}

	// Email
	if v := os.Getenv("EMAIL_PROVIDER"); v != "" {
		cfg.Email.Provider = v
	}
	if v := os.Getenv("EMAIL_FROM"); v != "" {
		cfg.Email.From = v
	}
	if v := os.Getenv("EMAIL_FROM_NAME"); v != "" {
		cfg.Email.FromName = v
	}
	if v := os.Getenv("FRONTEND_URL"); v != "" {
		cfg.Email.FrontendURL = v
	}
	if v := os.Getenv("SMTP_HOST"); v != "" {
		cfg.Email.SMTP.Host = v
	}
	if v := os.Getenv("SMTP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Email.SMTP.Port = port
		}
	}
	if v := os.Getenv("SMTP_USERNAME"); v != "" {
		cfg.Email.SMTP.Username = v
	}
	if v := os.Getenv("SMTP_PASSWORD"); v != "" {
		cfg.Email.SMTP.Password = v
	}
	if v := os.Getenv("SMTP_USE_TLS"); v != "" {
		cfg.Email.SMTP.UseTLS = v == "true" || v == "1"
	}
	if v := os.Getenv("SENDGRID_API_KEY"); v != "" {
		cfg.Email.SendGrid.APIKey = v
	}

	// Tenant
	if v := os.Getenv("MULTITENANCY_ENABLED"); v != "" {
		cfg.Tenant.Enabled = v == "true" || v == "1"
	}
	if v := os.Getenv("MULTITENANCY_TENANT_HEADER"); v != "" {
		cfg.Tenant.TenantHeader = v
	}
	if v := os.Getenv("MULTITENANCY_DOMAIN"); v != "" {
		cfg.Tenant.Domain = v
	}
}
