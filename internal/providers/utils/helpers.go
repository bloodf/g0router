package utils

import (
	"encoding/json"
	"fmt"

	"github.com/valyala/fasthttp"
)

// SetJSONBody marshals v as JSON and sets it as the request body with Content-Type.
func SetJSONBody(req *fasthttp.Request, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal request body: %w", err)
	}
	req.SetBodyRaw(b)
	req.Header.SetContentType("application/json")
	return nil
}

// ReadJSONBody unmarshals the response body into v.
func ReadJSONBody(resp *fasthttp.Response, v any) error {
	if err := json.Unmarshal(resp.Body(), v); err != nil {
		return fmt.Errorf("unmarshal response body: %w", err)
	}
	return nil
}

// SetAuthHeader sets the Authorization header with a Bearer token.
func SetAuthHeader(req *fasthttp.Request, token string) {
	req.Header.Set("Authorization", "Bearer "+token)
}

// PtrFloat64 returns a pointer to a float64.
func PtrFloat64(v float64) *float64 {
	return &v
}

// PtrInt returns a pointer to an int.
func PtrInt(v int) *int {
	return &v
}

// PtrString returns a pointer to a string.
func PtrString(v string) *string {
	return &v
}
