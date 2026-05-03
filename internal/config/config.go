// Package config provides configuration loading for the JAV Search API.
// Configuration is loaded from a YAML file with environment variable overrides.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration struct.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	JavDB    JavDBConfig    `yaml:"javdb"`
	Cache    CacheConfig    `yaml:"cache"`
	Scrapers ScrapersConfig `yaml:"scrapers"`
	Auth     AuthConfig     `yaml:"auth"`
	Render   RenderConfig   `yaml:"render"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host               string `yaml:"host"`
	Port               int    `yaml:"port"`
	ReadTimeoutSec     int    `yaml:"read_timeout_sec"`
	WriteTimeoutSec    int    `yaml:"write_timeout_sec"`
	ShutdownTimeoutSec int    `yaml:"shutdown_timeout_sec"`
}

// JavDBConfig holds JavDB signature parameters.
type JavDBConfig struct {
	BaseURL string `yaml:"base_url"`
	Middle  string `yaml:"middle"`
	Suffix  string `yaml:"suffix"`
}

// CacheConfig holds cache settings. Postgres is disabled by default (db-less mode).
type CacheConfig struct {
	MemoryTTLSeconds int    `yaml:"memory_ttl_seconds"`
	PostgresURL      string `yaml:"postgres_url"`
	PostgresEnabled  bool   `yaml:"postgres_enabled"`
}

// ScraperSiteConfig holds per-site scraper settings.
type ScraperSiteConfig struct {
	Name         string `yaml:"name"`
	Enabled      bool   `yaml:"enabled"`
	ProxyURL     string `yaml:"proxy_url"`
	ProxyEnabled bool   `yaml:"proxy_enabled"`
	TimeoutSec   int    `yaml:"timeout_sec"`
}

// ScrapersConfig holds global scraper settings and per-site overrides.
type ScrapersConfig struct {
	TimeoutSec       int                `yaml:"timeout_sec"`
	RateLimitDelayMS int                `yaml:"rate_limit_delay_ms"`
	UserAgent        string             `yaml:"user_agent"`
	MaxConcurrent    int                `yaml:"max_concurrent"`
	Sites            []ScraperSiteConfig `yaml:"sites"`
}

// AuthConfig holds API key authentication settings.
type AuthConfig struct {
	APIKeys []string `yaml:"api_keys"`
}

// RenderConfig holds Render deployment-specific settings.
type RenderConfig struct {
	GoMemLimit        string `yaml:"gomemlimit"`
	ColdStartTolerant bool   `yaml:"cold_start_tolerant"`
}

// Load reads configuration from a YAML file, applies defaults,
// then overrides with environment variables.
func Load(configPath string) (*Config, error) {
	cfg := defaultConfig()

	data, err := os.ReadFile(configPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config file: %w", err)
		}
		// File not found: proceed with defaults + env overrides only
	} else {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config file: %w", err)
		}
	}

	applyEnvOverrides(cfg)

	return cfg, nil
}

// defaultConfig returns a Config with all defaults populated.
func defaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:               "0.0.0.0",
			Port:               8080,
			ReadTimeoutSec:     10,
			WriteTimeoutSec:    30,
			ShutdownTimeoutSec: 25,
		},
		JavDB: JavDBConfig{
			BaseURL: "https://jdforrepam.com",
			Middle:  "lpw6vgqzsp",
			Suffix:  "71cf27bb3c0bcdf207b64abecddc970098c7421ee7203b9cdae54478478a199e7d5a6e1a57691123c1a931c057842fb73ba3b3c83bcd69c17ccf174081e3d8aa",
		},
		Cache: CacheConfig{
			MemoryTTLSeconds: 300,
			PostgresEnabled:  false,
		},
		Scrapers: ScrapersConfig{
			TimeoutSec:       8,
			RateLimitDelayMS: 500,
			UserAgent:        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
			MaxConcurrent:    6,
		},
		Auth: AuthConfig{
			APIKeys: []string{},
		},
		Render: RenderConfig{
			GoMemLimit:        "400MiB",
			ColdStartTolerant: true,
		},
	}
}

// applyEnvOverrides reads environment variables and overrides matching config fields.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("JAVDB_MIDDLE"); v != "" {
		cfg.JavDB.Middle = v
	}
	if v := os.Getenv("JAVDB_SUFFIX"); v != "" {
		cfg.JavDB.Suffix = v
	}
	if v := os.Getenv("CACHE_POSTGRES_URL"); v != "" {
		cfg.Cache.PostgresURL = v
	}
	if v := os.Getenv("AUTH_API_KEYS"); v != "" {
		cfg.Auth.APIKeys = splitComma(v)
	}
}

// splitComma splits a comma-separated string, trimming whitespace.
func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
