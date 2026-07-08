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

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	NATS      NATSConfig
	JWT       JWTConfig
	Log       LogConfig
	Telemetry TelemetryConfig
	Asynq     AsynqConfig
}

type ServerConfig struct {
	Port         int
	ReadTimeout  int
	WriteTimeout int
	IdleTimeout  int
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime int
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	PoolSize int
}

type NATSConfig struct {
	URL string
}

type JWTConfig struct {
	Secret     string
	Expiration int
}

type LogConfig struct {
	Level  string
	Format string
}

type TelemetryConfig struct {
	ServiceName      string
	ExporterEndpoint string
	SampleRate       float64
}

type AsynqConfig struct {
	RedisAddr string
}

func New() (*Config, error) {
	k := koanf.New(".")

	if err := k.Load(file.Provider("configs/config.yaml"), yaml.Parser()); err != nil {
		fmt.Printf("warning: could not load config.yaml: %v\n", err)
	}

	k.Load(env.ProviderWithValue("", ".", func(s string, v string) (string, interface{}) {
		return strings.Replace(strings.ToLower(s), "_", ".", -1), v
	}), nil)

	cfg := &Config{}
	if err := k.Unmarshal("", cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	applyEnvOverrides(cfg)

	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
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
	if cfg.JWT.Expiration == 0 {
		cfg.JWT.Expiration = 3600
	}
	if cfg.Telemetry.SampleRate == 0 {
		cfg.Telemetry.SampleRate = 1.0
	}

	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = port
		}
	}
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
	if v := os.Getenv("REDIS_ADDR"); v != "" {
		cfg.Redis.Addr = v
	}
	if v := os.Getenv("NATS_URL"); v != "" {
		cfg.NATS.URL = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
	if v := os.Getenv("OTEL_EXPORTER_ENDPOINT"); v != "" {
		cfg.Telemetry.ExporterEndpoint = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Log.Level = v
	}
}
