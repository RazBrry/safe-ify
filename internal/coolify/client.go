package coolify

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// Client is an HTTP client for the Coolify v4 API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new Coolify API client with a 30-second timeout
// and the safe-ify/1.0 User-Agent.
func NewClient(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest builds and executes an HTTP request. query may be nil.
// Non-2xx responses are returned as *CoolifyError.
func (c *Client) doRequest(ctx context.Context, method, path string, query url.Values) (*http.Response, error) {
	u, err := url.Parse(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	if query != nil {
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("User-Agent", "safe-ify/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		// All errors from httpClient.Do are transport/network failures —
		// classify them as NetworkError, distinct from API-layer errors.
		return nil, &NetworkError{Cause: err}
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		return nil, c.handleError(resp)
	}

	return resp, nil
}

// handleError reads the response body and maps HTTP status codes to
// user-friendly CoolifyError messages.
func (c *Client) handleError(resp *http.Response) error {
	raw, _ := io.ReadAll(resp.Body)
	rawStr := string(raw)

	var message string
	switch resp.StatusCode {
	case http.StatusUnauthorized: // 401
		message = "authentication failed — please run `safe-ify auth add` to reconfigure your token"
	case http.StatusForbidden: // 403
		message = "insufficient token permissions — check that your Coolify token has the required abilities (read + deploy)"
	case http.StatusNotFound: // 404
		message = "resource not found — check that the app UUID is correct"
	case http.StatusTooManyRequests: // 429
		retryAfter := resp.Header.Get("Retry-After")
		if retryAfter != "" {
			message = fmt.Sprintf("rate limit exceeded — retry after %s seconds", retryAfter)
		} else {
			message = "rate limit exceeded — please retry later"
		}
	case http.StatusBadRequest: // 400
		message = fmt.Sprintf("bad request: %s", rawStr)
	case http.StatusConflict: // 409
		message = fmt.Sprintf("conflict: %s", rawStr)
	case http.StatusUnprocessableEntity: // 422
		message = fmt.Sprintf("validation error: %s", rawStr)
	default:
		message = fmt.Sprintf("unexpected status %d: %s", resp.StatusCode, rawStr)
	}

	return &CoolifyError{
		StatusCode: resp.StatusCode,
		Message:    message,
		Raw:        rawStr,
	}
}
