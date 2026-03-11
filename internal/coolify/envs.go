package coolify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

// ListEnvs calls GET /api/v1/applications/{uuid}/envs and returns all env vars.
func (c *Client) ListEnvs(ctx context.Context, appUUID string) ([]EnvVar, error) {
	if err := validateUUID(appUUID); err != nil {
		return nil, err
	}
	resp, err := c.doRequest(ctx, "GET", "/api/v1/applications/"+appUUID+"/envs", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var envs []EnvVar
	if err := json.NewDecoder(resp.Body).Decode(&envs); err != nil {
		return nil, fmt.Errorf("decoding envs response: %w", err)
	}
	return envs, nil
}

// CreateEnv calls POST /api/v1/applications/{uuid}/envs to create a new env var.
// Returns the UUID of the created env var.
func (c *Client) CreateEnv(ctx context.Context, appUUID string, req CreateEnvRequest) (string, error) {
	if err := validateUUID(appUUID); err != nil {
		return "", err
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshalling create env request: %w", err)
	}

	resp, err := c.doRequestWithBody(ctx, "POST", "/api/v1/applications/"+appUUID+"/envs", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		UUID string `json:"uuid"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decoding create env response: %w", err)
	}
	return result.UUID, nil
}

// UpdateEnv calls PATCH /api/v1/applications/{uuid}/envs to update an env var by key.
func (c *Client) UpdateEnv(ctx context.Context, appUUID string, req UpdateEnvRequest) error {
	if err := validateUUID(appUUID); err != nil {
		return err
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshalling update env request: %w", err)
	}

	resp, err := c.doRequestWithBody(ctx, "PATCH", "/api/v1/applications/"+appUUID+"/envs", bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// DeleteEnv calls DELETE /api/v1/applications/{uuid}/envs/{envUUID}.
func (c *Client) DeleteEnv(ctx context.Context, appUUID string, envUUID string) error {
	if err := validateUUID(appUUID); err != nil {
		return err
	}
	if err := validateUUID(envUUID); err != nil {
		return err
	}

	resp, err := c.doRequest(ctx, "DELETE", "/api/v1/applications/"+appUUID+"/envs/"+envUUID, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
