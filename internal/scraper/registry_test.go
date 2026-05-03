package scraper

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/henry/javapi/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock scrapers
// ---------------------------------------------------------------------------

type mockScraper struct {
	name       string
	enabled    bool
	panicOn    bool
	searchErr  error
	searchRes  []domain.VideoResult
}

func (m *mockScraper) Name() string                          { return m.name }
func (m *mockScraper) IsEnabled() bool                       { return m.enabled }
func (m *mockScraper) RequiresCFBypass() bool                { return false }
func (m *mockScraper) GetProxyConfig() domain.ProxyConfig    { return domain.ProxyConfig{} }
func (m *mockScraper) FormatCode(code string) string         { return code }

func (m *mockScraper) Search(ctx context.Context, code string) ([]domain.VideoResult, error) {
	if m.panicOn {
		panic("test panic")
	}
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return m.searchRes, nil
}

// ---------------------------------------------------------------------------
// Registry tests
// ---------------------------------------------------------------------------

func TestRegisterAndGetAll(t *testing.T) {
	mu.Lock()
	scrapers = make(map[string]domain.Scraper)
	mu.Unlock()

	s1 := &mockScraper{name: "test-a", enabled: true}
	s2 := &mockScraper{name: "test-b", enabled: false}

	Register(s1)
	Register(s2)

	all := GetAll()
	assert.Len(t, all, 2)

	names := make(map[string]bool)
	for _, s := range all {
		names[s.Name()] = true
	}
	assert.True(t, names["test-a"])
	assert.True(t, names["test-b"])
}

func TestGetEnabled(t *testing.T) {
	mu.Lock()
	scrapers = make(map[string]domain.Scraper)
	mu.Unlock()

	s1 := &mockScraper{name: "enabled", enabled: true}
	s2 := &mockScraper{name: "disabled", enabled: false}

	Register(s1)
	Register(s2)

	enabled := GetEnabled()
	assert.Len(t, enabled, 1)
	assert.Equal(t, "enabled", enabled[0].Name())
}

func TestRegisterReplacesExisting(t *testing.T) {
	mu.Lock()
	scrapers = make(map[string]domain.Scraper)
	mu.Unlock()

	s1 := &mockScraper{name: "dup", enabled: false}
	s2 := &mockScraper{name: "dup", enabled: true}

	Register(s1)
	Register(s2)

	all := GetAll()
	assert.Len(t, all, 1, "same name should replace")
	assert.True(t, all[0].IsEnabled(), "last registered wins")
}

func TestGetAllReturnsCopy(t *testing.T) {
	mu.Lock()
	scrapers = make(map[string]domain.Scraper)
	mu.Unlock()

	Register(&mockScraper{name: "a", enabled: true})

	all := GetAll()
	// Modify returned slice (should not affect registry)
	all[0] = nil

	// Re-fetch — should still have the original
	all2 := GetAll()
	assert.Len(t, all2, 1)
	assert.NotNil(t, all2[0])
}

func TestGetEnabledEmptyRegistry(t *testing.T) {
	mu.Lock()
	scrapers = make(map[string]domain.Scraper)
	mu.Unlock()

	assert.Empty(t, GetAll())
	assert.Empty(t, GetEnabled())
}

// ---------------------------------------------------------------------------
// Code normalization tests
// ---------------------------------------------------------------------------

func TestNormalizeCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ABC-123", "abc123"},
		{"ABC_123", "abc123"},
		{"ABC 123", "abc123"},
		{"abc123", "abc123"},
		{"ABC-1234", "abc1234"},
		{"", ""},
		{"ABCDE-99999", "abcde99999"},
		{"aBc-DeF-456", "abcdef456"},
		{"  ABC  -  123  ", "abc123"},
		{"MIXED_Spaces-And_Case", "mixedspacesandcase"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeCode(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestExtractCode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ABC-123", "ABC-123"},
		{"abc123", "abc123"},
		{"some text ABC-123 here", "ABC-123"},
		{"ABC_123 extra", "ABC_123"},
		{"ABC 123", "ABC 123"},
		{"ABCDE-99999 trailing", "ABCDE-99999"},
		{"no digits here", "no digits here"},
		{"", ""},
		{"12-AB", "12-AB"}, // digits first, still matches 2+ lett then 2+ dig
		{"a-1", "a-1"},     // 1 letter + 1 digit — too short, matches nothing → returns raw
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ExtractCode(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// ---------------------------------------------------------------------------
// SafeSearch tests
// ---------------------------------------------------------------------------

func TestSafeSearchSuccess(t *testing.T) {
	expected := []domain.VideoResult{
		{SiteName: "test", Status: domain.StatusSuccess},
	}
	s := &mockScraper{name: "good", searchRes: expected}

	results, err := SafeSearch(context.Background(), s, "ABC-123")
	require.NoError(t, err)
	assert.Equal(t, expected, results)
}

func TestSafeSearchPanic(t *testing.T) {
	s := &mockScraper{name: "panicker", panicOn: true}

	results, err := SafeSearch(context.Background(), s, "ABC-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panicker")
	assert.Contains(t, err.Error(), "test panic")
	assert.Nil(t, results)
}

func TestSafeSearchError(t *testing.T) {
	s := &mockScraper{name: "errored", searchErr: assert.AnError}

	results, err := SafeSearch(context.Background(), s, "ABC-123")
	assert.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
	assert.Nil(t, results)
}

func TestSafeSearchPanicPreservesNilResults(t *testing.T) {
	// Ensure that when a scraper both panics AND would have returned nil,
	// the recovered error still has the right information.
	s := &mockScraper{name: "nilPanic", panicOn: true}

	results, err := SafeSearch(context.Background(), s, "ABC-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nilPanic")
	assert.Nil(t, results)
}

// ---------------------------------------------------------------------------
// CFBypass tests
// ---------------------------------------------------------------------------

func TestCFBypassPassed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>Welcome to our site</body></html>`))
	}))
	defer srv.Close()

	result, err := CFBypassTest(context.Background(), srv.URL, "")
	require.NoError(t, err)
	assert.True(t, result.Passed)
	assert.Empty(t, result.Error)
}

func TestCFBypassBlocked(t *testing.T) {
	blockedBodies := []string{
		"Just a moment...",
		"Checking your browser before accessing the site",
		`id="cf-browser-verification"`,
		"Attention Required! | Cloudflare",
	}

	for _, body := range blockedBodies {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(body))
		}))

		result, err := CFBypassTest(context.Background(), srv.URL, "")
		srv.Close()

		require.NoError(t, err)
		assert.False(t, result.Passed)
		assert.Equal(t, "Cloudflare challenge detected", result.Error)
	}
}

func TestCFBypassPassedWithCFScriptRefs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><head><title>Real Page</title>
<script src="/cdn-cgi/challenge-platform/scripts/jsd/main.js"></script>
</head><body>Welcome to our site - Cloudflare</body></html>`))
	}))
	defer srv.Close()

	result, err := CFBypassTest(context.Background(), srv.URL, "")
	require.NoError(t, err)
	assert.True(t, result.Passed, "page with CF script refs and HTML structure should pass")
	assert.Empty(t, result.Error)
}

func TestCFBypassHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Internal Server Error`))
	}))
	defer srv.Close()

	result, err := CFBypassTest(context.Background(), srv.URL, "")
	require.NoError(t, err)
	assert.False(t, result.Passed)
	assert.Equal(t, "HTTP 500", result.Error)
}

func TestCFBypassBadURL(t *testing.T) {
	result, err := CFBypassTest(context.Background(), "http://127.0.0.1:1", "")
	require.NoError(t, err) // CFBypassTest never returns non-nil error for connection failures
	assert.False(t, result.Passed)
	assert.NotEmpty(t, result.Error)
}

func TestCFBypassContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := CFBypassTest(ctx, "http://example.com", "")
	require.NoError(t, err)
	assert.False(t, result.Passed)
	assert.NotEmpty(t, result.Error)
}
