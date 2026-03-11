package coolify

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
)

// uuidPattern matches a standard UUID: 8-4-4-4-12 hexadecimal characters with dashes.
var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// validateUUID returns an error if uuid does not match the standard UUID format.
// This prevents path-manipulation attacks (e.g., "../admin", "../../etc/passwd").
func validateUUID(uuid string) error {
	if !uuidPattern.MatchString(uuid) {
		return fmt.Errorf("invalid UUID format: %q — expected xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx", uuid)
	}
	return nil
}

// Healthcheck calls GET /api/v1/healthcheck. Returns nil on 200.
func (c *Client) Healthcheck(ctx context.Context) error {
	resp, err := c.doRequest(ctx, "GET", "/api/v1/healthcheck", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Version calls GET /api/v1/version and returns the version string.
func (c *Client) Version(ctx context.Context) (string, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v1/version", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading version response: %w", err)
	}

	// The version endpoint returns a plain string (possibly quoted JSON string).
	version := string(body)
	// Try to unmarshal as JSON string first.
	var v string
	if err := json.Unmarshal(body, &v); err == nil {
		return v, nil
	}
	return version, nil
}

// ListApplications calls GET /api/v1/applications and returns all applications.
func (c *Client) ListApplications(ctx context.Context) ([]Application, error) {
	resp, err := c.doRequest(ctx, "GET", "/api/v1/applications", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apps []Application
	if err := json.NewDecoder(resp.Body).Decode(&apps); err != nil {
		return nil, fmt.Errorf("decoding applications response: %w", err)
	}
	return apps, nil
}

// GetApplication calls GET /api/v1/applications/{uuid} and returns the application.
func (c *Client) GetApplication(ctx context.Context, uuid string) (*Application, error) {
	if err := validateUUID(uuid); err != nil {
		return nil, err
	}
	resp, err := c.doRequest(ctx, "GET", "/api/v1/applications/"+uuid, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var app Application
	if err := json.NewDecoder(resp.Body).Decode(&app); err != nil {
		return nil, fmt.Errorf("decoding application response: %w", err)
	}
	return &app, nil
}

// GetLogs calls GET /api/v1/applications/{uuid}/logs?tail={tail}
// and returns the log lines. If tail is 0, no tail parameter is sent.
func (c *Client) GetLogs(ctx context.Context, uuid string, tail int) ([]string, error) {
	if err := validateUUID(uuid); err != nil {
		return nil, err
	}

	var query url.Values
	if tail > 0 {
		query = url.Values{"tail": {strconv.Itoa(tail)}}
	}

	resp, err := c.doRequest(ctx, "GET", "/api/v1/applications/"+uuid+"/logs", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// The logs endpoint returns either a JSON array of strings or plain text lines.
	// Try JSON first; fall back to line-by-line scanning.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading logs response: %w", err)
	}

	var lines []string
	if err := json.Unmarshal(body, &lines); err == nil {
		return lines, nil
	}

	// Fall back: split by newlines.
	lineScanner := bufio.NewScanner(bytes.NewReader(body))
	for lineScanner.Scan() {
		line := lineScanner.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, lineScanner.Err()
}
