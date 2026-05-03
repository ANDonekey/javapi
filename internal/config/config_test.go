package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTempYAML creates a temporary YAML file for testing and returns its path.
func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte(content), 0o644)
	require.NoError(t, err)
	return path
}

func TestLoad_Defaults(t *testing.T) {
	// Load with a non-existent file: should return defaults + env overrides.
	cfg, err := Load("/tmp/nonexistent-config-xxxx.yaml")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, 10, cfg.Server.ReadTimeoutSec)
	assert.Equal(t, 30, cfg.Server.WriteTimeoutSec)
	assert.Equal(t, 25, cfg.Server.ShutdownTimeoutSec)

	assert.Equal(t, "https://jdforrepam.com", cfg.JavDB.BaseURL)
	assert.Equal(t, "lpw6vgqzsp", cfg.JavDB.Middle)
	assert.NotEmpty(t, cfg.JavDB.Suffix)

	assert.Equal(t, 300, cfg.Cache.MemoryTTLSeconds)
	assert.Empty(t, cfg.Cache.PostgresURL)
	assert.False(t, cfg.Cache.PostgresEnabled, "PostgresEnabled must default to false (db-less)")

	assert.Equal(t, 8, cfg.Scrapers.TimeoutSec)
	assert.Equal(t, 500, cfg.Scrapers.RateLimitDelayMS)
	assert.Contains(t, cfg.Scrapers.UserAgent, "Chrome")
	assert.Equal(t, 6, cfg.Scrapers.MaxConcurrent)

	assert.Empty(t, cfg.Auth.APIKeys)

	assert.Equal(t, "400MiB", cfg.Render.GoMemLimit)
	assert.True(t, cfg.Render.ColdStartTolerant)
}

func TestLoad_YAML(t *testing.T) {
	yamlContent := `
server:
  host: "127.0.0.1"
  port: 9090
javdb:
  middle: "yaml_middle"
  suffix: "yaml_suffix"
cache:
  memory_ttl_seconds: 600
  postgres_enabled: true
  postgres_url: "postgres://test@localhost/test"
scrapers:
  timeout_sec: 15
  max_concurrent: 10
  sites:
    - name: "javdb"
      enabled: true
      proxy_url: "http://proxy:8080"
      proxy_enabled: true
auth:
  api_keys:
    - "key1"
    - "key2"
render:
  gomemlimit: "200MiB"
  cold_start_tolerant: false
`
	path := writeTempYAML(t, yamlContent)
	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "127.0.0.1", cfg.Server.Host)
	assert.Equal(t, 9090, cfg.Server.Port)

	assert.Equal(t, "yaml_middle", cfg.JavDB.Middle)
	assert.Equal(t, "yaml_suffix", cfg.JavDB.Suffix)

	assert.Equal(t, 600, cfg.Cache.MemoryTTLSeconds)
	assert.True(t, cfg.Cache.PostgresEnabled)
	assert.Equal(t, "postgres://test@localhost/test", cfg.Cache.PostgresURL)

	assert.Equal(t, 15, cfg.Scrapers.TimeoutSec)
	assert.Equal(t, 10, cfg.Scrapers.MaxConcurrent)
	require.Len(t, cfg.Scrapers.Sites, 1)
	assert.Equal(t, "javdb", cfg.Scrapers.Sites[0].Name)
	assert.True(t, cfg.Scrapers.Sites[0].Enabled)
	assert.Equal(t, "http://proxy:8080", cfg.Scrapers.Sites[0].ProxyURL)
	assert.True(t, cfg.Scrapers.Sites[0].ProxyEnabled)

	assert.Equal(t, []string{"key1", "key2"}, cfg.Auth.APIKeys)

	assert.Equal(t, "200MiB", cfg.Render.GoMemLimit)
	assert.False(t, cfg.Render.ColdStartTolerant)
}

func TestLoad_EnvOverrides(t *testing.T) {
	// Set env vars
	t.Setenv("JAVDB_MIDDLE", "env_middle")
	t.Setenv("JAVDB_SUFFIX", "env_suffix")
	t.Setenv("CACHE_POSTGRES_URL", "postgres://env@localhost/envdb")
	t.Setenv("AUTH_API_KEYS", "envkey1, envkey2 ,envkey3")

	// Write a YAML with different values to verify env takes precedence
	yamlContent := `
javdb:
  middle: "yaml_middle"
  suffix: "yaml_suffix"
cache:
  postgres_url: "postgres://yaml@localhost/yamldb"
auth:
  api_keys:
    - "yamlkey"
`
	path := writeTempYAML(t, yamlContent)
	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Env overrides YAML
	assert.Equal(t, "env_middle", cfg.JavDB.Middle)
	assert.Equal(t, "env_suffix", cfg.JavDB.Suffix)
	assert.Equal(t, "postgres://env@localhost/envdb", cfg.Cache.PostgresURL)
	assert.Equal(t, []string{"envkey1", "envkey2", "envkey3"}, cfg.Auth.APIKeys)

	// Non-overridden fields still come from YAML defaults
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
}

func TestLoad_PostgresDisabledByDefault(t *testing.T) {
	// Minimal YAML without cache section: PostgresEnabled should remain false
	yamlContent := `server:
  host: "0.0.0.0"
`
	path := writeTempYAML(t, yamlContent)
	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.False(t, cfg.Cache.PostgresEnabled, "PostgresEnabled must be false by default")
	assert.Equal(t, 300, cfg.Cache.MemoryTTLSeconds)
	assert.Empty(t, cfg.Cache.PostgresURL)
}

func TestLoad_ProxyPerSite(t *testing.T) {
	yamlContent := `
scrapers:
  sites:
    - name: "javdb"
      enabled: true
      proxy_url: "http://user:pass@brd.superproxy.io:22225"
      proxy_enabled: true
      timeout_sec: 15
    - name: "javlibrary"
      enabled: true
      proxy_enabled: false
    - name: "arzon"
      enabled: false
`
	path := writeTempYAML(t, yamlContent)
	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Len(t, cfg.Scrapers.Sites, 3)

	// Site 0: full proxy config
	assert.Equal(t, "javdb", cfg.Scrapers.Sites[0].Name)
	assert.True(t, cfg.Scrapers.Sites[0].Enabled)
	assert.Equal(t, "http://user:pass@brd.superproxy.io:22225", cfg.Scrapers.Sites[0].ProxyURL)
	assert.True(t, cfg.Scrapers.Sites[0].ProxyEnabled)
	assert.Equal(t, 15, cfg.Scrapers.Sites[0].TimeoutSec)

	// Site 1: proxy disabled, no URL
	assert.Equal(t, "javlibrary", cfg.Scrapers.Sites[1].Name)
	assert.True(t, cfg.Scrapers.Sites[1].Enabled)
	assert.Empty(t, cfg.Scrapers.Sites[1].ProxyURL)
	assert.False(t, cfg.Scrapers.Sites[1].ProxyEnabled)

	// Site 2: disabled site
	assert.Equal(t, "arzon", cfg.Scrapers.Sites[2].Name)
	assert.False(t, cfg.Scrapers.Sites[2].Enabled)
}

func TestLoad_RenderDefaults(t *testing.T) {
	cfg, err := Load("/tmp/nonexistent-render-test.yaml")
	require.NoError(t, err)

	assert.Equal(t, "400MiB", cfg.Render.GoMemLimit)
	assert.True(t, cfg.Render.ColdStartTolerant)
}

func TestLoad_EmptyEnvVarsDontOverride(t *testing.T) {
	// Set empty env vars — should NOT override
	t.Setenv("JAVDB_MIDDLE", "")
	t.Setenv("CACHE_POSTGRES_URL", "")

	yamlContent := `
javdb:
  middle: "yaml_middle"
cache:
  postgres_url: "postgres://yaml@localhost/yamldb"
`
	path := writeTempYAML(t, yamlContent)
	cfg, err := Load(path)
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "yaml_middle", cfg.JavDB.Middle)
	assert.Equal(t, "postgres://yaml@localhost/yamldb", cfg.Cache.PostgresURL)
}

func TestLoad_InvalidYAML(t *testing.T) {
	path := writeTempYAML(t, `invalid: yaml: {bad`)
	_, err := Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse config file")
}
