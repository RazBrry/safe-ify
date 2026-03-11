package cli

import (
	"encoding/json"
	"io"
)

// Response is the standard JSON envelope for all agent-facing commands.
type Response struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data"`
	Error *ErrorInfo  `json:"error"`
}

// ErrorInfo holds a structured error code and message.
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error code constants used in agent command responses.
const (
	ErrCodePermissionDenied  = "PERMISSION_DENIED"
	ErrCodeConfigNotFound    = "CONFIG_NOT_FOUND"
	ErrCodeInstanceNotFound  = "INSTANCE_NOT_FOUND"
	ErrCodeAPIError          = "API_ERROR"
	ErrCodeNetworkError      = "NETWORK_ERROR"
	ErrCodeConfigInsecure    = "CONFIG_INSECURE"
	ErrCodeAppNotFound       = "APP_NOT_FOUND"
	ErrCodeAppAmbiguous      = "APP_AMBIGUOUS"
)

// OutputJSON marshals resp as indented JSON and writes it to w.
func OutputJSON(w io.Writer, resp Response) {
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		// Fallback: write a minimal error JSON.
		_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"API_ERROR","message":"failed to marshal response"}}`))
		return
	}
	_, _ = w.Write(data)
	_, _ = w.Write([]byte("\n"))
}

// OutputError creates an error Response and writes it as JSON to w.
func OutputError(w io.Writer, code, message string) {
	OutputJSON(w, Response{
		OK: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	})
}
