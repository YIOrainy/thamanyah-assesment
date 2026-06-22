package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration parses Go duration strings (e.g. "60s") from YAML.
type Duration time.Duration

func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

func (d Duration) Duration() time.Duration { return time.Duration(d) }

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Store     StoreConfig     `yaml:"store"`
	Cache     CacheConfig     `yaml:"cache"`
	Auth          AuthConfig          `yaml:"auth"`
	Ingestion     IngestionConfig     `yaml:"ingestion"`
	Observability ObservabilityConfig `yaml:"observability"`
	LogLevel      string              `yaml:"log_level"`
}

type ObservabilityConfig struct {
	Profiling ProfilingConfig `yaml:"profiling"` // Pyroscope continuous profiling
	Metrics   MetricsConfig   `yaml:"metrics"`   // Prometheus /metrics
}

type ProfilingConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"` // Pyroscope server address
}

type MetricsConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Endpoint string `yaml:"endpoint"` // informational; Prometheus scrapes GET /metrics
}

// IngestionConfig enables/configures import sources. A disabled source is not
// registered, so importing from it returns "unknown source".
type IngestionConfig struct {
	RSS     SourceToggle  `yaml:"rss"`
	CSV     SourceToggle  `yaml:"csv"`
	YouTube YouTubeSource `yaml:"youtube"`
}

type SourceToggle struct {
	Enabled bool `yaml:"enabled"`
}

type YouTubeSource struct {
	Enabled bool   `yaml:"enabled"`
	APIKey  string `yaml:"api_key"`
}

type AuthConfig struct {
	JWTSecret      string         `yaml:"jwt_secret"`
	TokenTTL       Duration       `yaml:"token_ttl"`
	BootstrapAdmin BootstrapAdmin `yaml:"bootstrap_admin"`
}

// BootstrapAdmin, when set, ensures an admin user exists on CMS startup.
type BootstrapAdmin struct {
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}

type ServerConfig struct {
	Host          string `yaml:"host"`
	CMSPort       string `yaml:"cms_port"`
	DiscoveryPort string `yaml:"discovery_port"`
}

type StoreConfig struct {
	Postgres PostgresConfig `yaml:"postgres"`
	Memory   MemoryConfig   `yaml:"memory"`
}

type CacheConfig struct {
	Redis RedisConfig `yaml:"redis"`
}

type PostgresConfig struct {
	Enabled      bool   `yaml:"enabled"`
	DSN          string `yaml:"dsn"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

type RedisConfig struct {
	Enabled bool     `yaml:"enabled"`
	Addr    string   `yaml:"addr"`
	TTL     Duration `yaml:"ttl"`
}

type MemoryConfig struct {
	Enabled bool `yaml:"enabled"`
	Actors  int  `yaml:"actors"` // shard count for the in-memory store
}

// Load reads and parses a YAML config file, applying defaults.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	c.applyDefaults()
	return &c, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Host == "" {
		c.Server.Host = "0.0.0.0"
	}
	if c.Server.CMSPort == "" {
		c.Server.CMSPort = "8080"
	}
	if c.Server.DiscoveryPort == "" {
		c.Server.DiscoveryPort = "8081"
	}
	if c.Store.Postgres.MaxOpenConns == 0 {
		c.Store.Postgres.MaxOpenConns = 25
	}
	if c.Store.Memory.Actors < 1 {
		c.Store.Memory.Actors = 1
	}
	if c.Auth.JWTSecret == "" {
		c.Auth.JWTSecret = "dev-secret-change-me"
	}
	if c.Auth.TokenTTL <= 0 {
		c.Auth.TokenTTL = Duration(time.Hour)
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
}
