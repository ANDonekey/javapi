package javdb

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/henry/javapi/internal/domain"
)

// Client is an HTTP client for the JavDB API with jdsignature authentication.
type Client struct {
	httpClient *resty.Client
	baseURL    string
	middle     string
	suffix     string
}

// NewClient creates a new JavDB API client.
func NewClient(baseURL, middle, suffix string) *Client {
	rc := resty.New().
		SetBaseURL(baseURL).
		SetTimeout(10*time.Second).
		SetRetryCount(3).
		SetRetryWaitTime(2*time.Second).
		SetHeader("Accept", "application/json").
		SetHeader("User-Agent", "Mozilla/5.0")

	return &Client{
		httpClient: rc,
		baseURL:    baseURL,
		middle:     middle,
		suffix:     suffix,
	}
}

// generateSignature creates the jdsignature header value.
// Format: {timestamp}.{middle}.{md5(timestamp + suffix)}
func (c *Client) generateSignature() string {
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	hash := fmt.Sprintf("%x", md5.Sum([]byte(ts+c.suffix)))
	return ts + "." + c.middle + "." + hash
}

// Search queries the JavDB search endpoint for a JAV code.
// Returns a Movie if found, nil if not found, or an error.
func (c *Client) Search(ctx context.Context, code string) (*domain.Movie, error) {
	var env apiEnvelope

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("jdsignature", c.generateSignature()).
		SetQueryParam("q", code).
		SetQueryParam("page", "1").
		SetResult(&env).
		Get("/api/v2/search")

	if err != nil {
		return nil, fmt.Errorf("javdb search request: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("javdb search HTTP %d", resp.StatusCode())
	}

	if env.Action != "" {
		return nil, apiError(env.Action)
	}

	if len(env.Data.Movies) == 0 {
		return nil, nil
	}

	return env.Data.Movies[0].ToMovie(), nil
}

// GetMovie fetches full movie detail from JavDB by movie ID.
func (c *Client) GetMovie(ctx context.Context, movieID string) (*domain.Movie, error) {
	var env apiEnvelope

	resp, err := c.httpClient.R().
		SetContext(ctx).
		SetHeader("jdsignature", c.generateSignature()).
		SetResult(&env).
		Get(fmt.Sprintf("/api/v4/movies/%s", movieID))

	if err != nil {
		return nil, fmt.Errorf("javdb get movie: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("javdb get movie HTTP %d", resp.StatusCode())
	}

	if env.Action != "" {
		return nil, apiError(env.Action)
	}

	return env.Data.Movie.ToMovie(), nil
}

// apiError converts a JavDB API action string to a descriptive Go error.
func apiError(action string) error {
	switch action {
	case "ParameterInvalid":
		return fmt.Errorf("javdb api: parameter invalid")
	case "InvalidSignature":
		return fmt.Errorf("javdb api: invalid signature — check middle/suffix configuration")
	case "JWTVerificationError":
		return fmt.Errorf("javdb api: authentication required — login not supported")
	default:
		return fmt.Errorf("javdb api: %s", action)
	}
}
