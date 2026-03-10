package coolify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Deploy triggers a deployment via POST /api/v1/deploy?uuid={uuid}&force={force}.
// POST is used because deploy is a side-effecting operation (D11).
func (c *Client) Deploy(ctx context.Context, uuid string, force bool) (*DeployResponse, error) {
	if err := validateUUID(uuid); err != nil {
		return nil, err
	}
	query := url.Values{
		"uuid":  {uuid},
		"force": {boolStr(force)},
	}

	resp, err := c.doRequest(ctx, "POST", "/api/v1/deploy", query)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var deployResp DeployResponse
	if err := json.NewDecoder(resp.Body).Decode(&deployResp); err != nil {
		return nil, fmt.Errorf("decoding deploy response: %w", err)
	}
	return &deployResp, nil
}

// Restart triggers an application restart via POST /api/v1/applications/{uuid}/restart.
// POST is used because restart is a side-effecting operation (D11).
func (c *Client) Restart(ctx context.Context, uuid string) error {
	if err := validateUUID(uuid); err != nil {
		return err
	}
	resp, err := c.doRequest(ctx, "POST", "/api/v1/applications/"+uuid+"/restart", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// boolStr converts a bool to its string representation.
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
