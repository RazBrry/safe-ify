package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

// TestOutputJSON_Success verifies that a successful response has the expected
// {ok: true, data: ..., error: null} structure.
func TestOutputJSON_Success(t *testing.T) {
	type payload struct {
		Value string `json:"value"`
	}
	resp := Response{
		OK:   true,
		Data: payload{Value: "hello"},
	}

	var buf bytes.Buffer
	OutputJSON(&buf, resp)

	var got map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	// ok must be true.
	ok, exists := got["ok"]
	if !exists {
		t.Error("expected 'ok' key in JSON output")
	}
	if ok != true {
		t.Errorf("expected ok=true, got %v", ok)
	}

	// data must be present and non-nil.
	data, exists := got["data"]
	if !exists {
		t.Error("expected 'data' key in JSON output")
	}
	if data == nil {
		t.Error("expected data to be non-nil for success response")
	}

	// error must be present as explicit null (spec 5.1: no omitempty).
	errVal, exists := got["error"]
	if !exists {
		t.Error("expected 'error' key to be present (as null) in success response")
	}
	if errVal != nil {
		t.Errorf("expected error=null in success response, got %v", errVal)
	}
}

// TestOutputJSON_Error verifies that an error response has the expected
// {ok: false, data: null, error: {code, message}} structure.
func TestOutputJSON_Error(t *testing.T) {
	var buf bytes.Buffer
	OutputError(&buf, ErrCodePermissionDenied, "command deploy is not permitted")

	var got map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, buf.String())
	}

	// ok must be false.
	ok, exists := got["ok"]
	if !exists {
		t.Error("expected 'ok' key in JSON output")
	}
	if ok != false {
		t.Errorf("expected ok=false, got %v", ok)
	}

	// data must be present as explicit null (spec 5.1: no omitempty).
	dataVal, exists := got["data"]
	if !exists {
		t.Error("expected 'data' key to be present (as null) in error response")
	}
	if dataVal != nil {
		t.Errorf("expected data=null in error response, got %v", dataVal)
	}

	// error must be present with code and message.
	errVal, exists := got["error"]
	if !exists {
		t.Fatal("expected 'error' key in error response")
	}
	errMap, isMap := errVal.(map[string]interface{})
	if !isMap {
		t.Fatalf("expected error to be an object, got %T", errVal)
	}
	code, exists := errMap["code"]
	if !exists {
		t.Error("expected 'code' in error object")
	}
	if code != ErrCodePermissionDenied {
		t.Errorf("expected code=%s, got %v", ErrCodePermissionDenied, code)
	}
	msg, exists := errMap["message"]
	if !exists {
		t.Error("expected 'message' in error object")
	}
	if msg != "command deploy is not permitted" {
		t.Errorf("unexpected message: %v", msg)
	}
}

// TestOutputJSON_ValidJSON verifies that the output of OutputJSON is always
// parseable as JSON for both success and error variants.
func TestOutputJSON_ValidJSON(t *testing.T) {
	cases := []struct {
		name string
		resp Response
	}{
		{
			name: "success with string data",
			resp: Response{OK: true, Data: "some string"},
		},
		{
			name: "success with nil data",
			resp: Response{OK: true},
		},
		{
			name: "error with code",
			resp: Response{
				OK: false,
				Error: &ErrorInfo{
					Code:    ErrCodeAPIError,
					Message: "something went wrong",
				},
			},
		},
		{
			name: "success with complex data",
			resp: Response{
				OK: true,
				Data: map[string]interface{}{
					"uuid":   "abc-123",
					"status": "running",
					"count":  42,
				},
			},
		},
		{
			name: "network error",
			resp: Response{
				OK: false,
				Error: &ErrorInfo{
					Code:    ErrCodeNetworkError,
					Message: "connection refused",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			OutputJSON(&buf, tc.resp)

			output := buf.Bytes()
			if len(output) == 0 {
				t.Fatal("expected non-empty output")
			}

			var parsed interface{}
			if err := json.Unmarshal(output, &parsed); err != nil {
				t.Errorf("output is not valid JSON: %v\noutput: %s", err, string(output))
			}
		})
	}
}
