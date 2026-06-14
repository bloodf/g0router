package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bloodf/g0router/internal/store"
)

// deviceGrantType is the RFC 8628 device-code grant type used at the token
// endpoint for the standard device-code poll (qwen/github).
const deviceGrantType = "urn:ietf:params:oauth:grant-type:device_code"

// deviceFlowTimeout bounds the overall device-authorization poll loop.
// Parity: OAUTH_TIMEOUT 5m (oauth.js:261).
const deviceFlowTimeout = 5 * time.Minute

// slowDownStep is the interval increase applied on a "slow_down" poll error.
// Parity: services/qwen.js pollForToken (+5s on slow_down).
const slowDownStep = 5 * time.Second

// DeviceCodeResponse is the result of a device-code request: the codes the user
// must enter at the verification URI plus the poll interval and lifetime.
type DeviceCodeResponse struct {
	DeviceCode      string
	UserCode        string
	VerificationURI string
	Interval        int   // seconds between polls
	ExpiresIn       int64 // seconds until the device code expires
}

// StartDevice requests a device code from the provider's DeviceCodeURL. For the
// PKCE device-code variant (qwen) it generates a verifier and sends the S256
// challenge; the verifier is returned to the caller (kept out of band) for the
// subsequent poll. The kilocode variant POSTs an unauthenticated JSON initiate.
//
// It returns the device-code response and, for the PKCE variant, the code
// verifier the caller must pass to PollDevice. The verifier is empty for
// non-PKCE variants (github/kilocode).
func (f *OAuthFlow) StartDevice() (*DeviceCodeResponse, error) {
	if f.cfg.DeviceVariant == "kilocode" {
		return f.startDeviceKilocode()
	}
	return f.startDeviceStandard()
}

// startDeviceStandard requests a device code via the form POST used by the
// OAuth device-code flow (qwen with PKCE, github). Parity: services/qwen.js
// requestDeviceCode :18, services/github.js requestDeviceCode.
func (f *OAuthFlow) startDeviceStandard() (*DeviceCodeResponse, error) {
	form := url.Values{}
	form.Set("client_id", f.cfg.ClientID)
	if len(f.cfg.Scopes) > 0 {
		form.Set("scope", strings.Join(f.cfg.Scopes, " "))
	}
	// qwen is device-code WITH PKCE: send the S256 challenge.
	verifier, challenge, err := GeneratePKCE()
	if err != nil {
		return nil, fmt.Errorf("generate device pkce: %w", err)
	}
	form.Set("code_challenge", challenge)
	form.Set("code_challenge_method", "S256")

	req, err := http.NewRequest(http.MethodPost, f.cfg.DeviceCodeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build device-code request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device-code request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read device-code response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device-code endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		Interval        int    `json:"interval"`
		ExpiresIn       int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode device-code response: %w", err)
	}
	if parsed.DeviceCode == "" {
		return nil, fmt.Errorf("device-code response missing device_code")
	}

	// Persist the PKCE verifier keyed by the device_code so the caller need not
	// hold it; the verifier is encrypted at rest (verifier_enc).
	if err := f.store.CreateOAuthSession(&store.OAuthSession{
		State:     parsed.DeviceCode,
		Provider:  f.cfg.Provider,
		Verifier:  verifier,
		ExpiresAt: f.nowFunc().Add(oauthStateTTL).Unix(),
	}); err != nil {
		return nil, fmt.Errorf("persist device state: %w", err)
	}

	return &DeviceCodeResponse{
		DeviceCode:      parsed.DeviceCode,
		UserCode:        parsed.UserCode,
		VerificationURI: parsed.VerificationURI,
		Interval:        parsed.Interval,
		ExpiresIn:       parsed.ExpiresIn,
	}, nil
}

// startDeviceKilocode initiates the kilocode custom device-auth flow: an
// unauthenticated POST to initiateUrl returning {code, verificationUrl,
// expiresIn}. Parity: providers.js kilocode requestDeviceCode :1063.
func (f *OAuthFlow) startDeviceKilocode() (*DeviceCodeResponse, error) {
	req, err := http.NewRequest(http.MethodPost, f.cfg.DeviceCodeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build device-auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("device-auth request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read device-auth response: %w", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, fmt.Errorf("too many pending authorization requests; try again later")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("device-auth endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Code            string `json:"code"`
		VerificationURL string `json:"verificationUrl"`
		ExpiresIn       int64  `json:"expiresIn"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode device-auth response: %w", err)
	}
	if parsed.Code == "" {
		return nil, fmt.Errorf("device-auth response missing code")
	}
	expiresIn := parsed.ExpiresIn
	if expiresIn == 0 {
		expiresIn = 300
	}
	return &DeviceCodeResponse{
		DeviceCode:      parsed.Code,
		UserCode:        parsed.Code,
		VerificationURI: parsed.VerificationURL,
		Interval:        3,
		ExpiresIn:       expiresIn,
	}, nil
}

// PollDevice polls until the user authorizes, the device code expires, the user
// denies, or the overall deadline passes. interval is the seconds between polls
// (from the device-code response); verifier is the PKCE code verifier for the
// qwen variant ("" for github/kilocode). The clock is injectable so tests run
// with no real sleep.
func (f *OAuthFlow) PollDevice(ctx context.Context, deviceCode, verifier string, interval int) (*OAuthToken, error) {
	if interval <= 0 {
		interval = 5
	}
	wait := time.Duration(interval) * time.Second
	deadline := f.nowFunc().Add(deviceFlowTimeout)

	for {
		if f.nowFunc().After(deadline) {
			return nil, fmt.Errorf("device authorization timeout")
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-f.afterFunc(wait):
		}
		if f.nowFunc().After(deadline) {
			return nil, fmt.Errorf("device authorization timeout")
		}

		var tok *OAuthToken
		var pending, slowDown bool
		var err error
		if f.cfg.DeviceVariant == "kilocode" {
			tok, pending, err = f.pollKilocode(deviceCode)
		} else {
			tok, pending, slowDown, err = f.pollStandard(deviceCode, verifier)
		}
		if err != nil {
			return nil, err
		}
		if tok != nil {
			return tok, nil
		}
		if slowDown {
			wait += slowDownStep
		}
		_ = pending // pending → continue looping
	}
}

// pollStandard performs one OAuth device-code token poll. It returns a non-nil
// token on success, pending=true while the user has not yet authorized,
// slowDown=true to bump the interval, or a terminal error.
// Parity: services/qwen.js pollForToken :41.
func (f *OAuthFlow) pollStandard(deviceCode, verifier string) (tok *OAuthToken, pending, slowDown bool, err error) {
	form := url.Values{}
	form.Set("grant_type", deviceGrantType)
	form.Set("client_id", f.cfg.ClientID)
	form.Set("device_code", deviceCode)
	if verifier != "" {
		form.Set("code_verifier", verifier)
	}

	resp, perr := f.client.PostForm(f.cfg.TokenURL, form)
	if perr != nil {
		return nil, false, false, fmt.Errorf("device poll request: %w", perr)
	}
	defer resp.Body.Close()
	body, rerr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if rerr != nil {
		return nil, false, false, fmt.Errorf("read device poll response: %w", rerr)
	}

	if resp.StatusCode == http.StatusOK {
		parsed, derr := decodeStandardToken(body)
		if derr != nil {
			return nil, false, false, derr
		}
		return parsed, false, false, nil
	}

	var errResp struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	_ = json.Unmarshal(body, &errResp)
	switch errResp.Error {
	case "authorization_pending":
		return nil, true, false, nil
	case "slow_down":
		return nil, true, true, nil
	case "expired_token":
		return nil, false, false, fmt.Errorf("device code expired")
	case "access_denied":
		return nil, false, false, fmt.Errorf("device authorization denied")
	default:
		msg := errResp.ErrorDescription
		if msg == "" {
			msg = errResp.Error
		}
		if msg == "" {
			msg = strings.TrimSpace(string(body))
		}
		return nil, false, false, fmt.Errorf("device poll failed (%d): %s", resp.StatusCode, msg)
	}
}

// pollKilocode performs one kilocode poll: GET pollUrlBase/{code} with
// status-coded responses. Parity: providers.js kilocode pollToken :1086.
func (f *OAuthFlow) pollKilocode(deviceCode string) (tok *OAuthToken, pending bool, err error) {
	pollURL := strings.TrimRight(f.cfg.DeviceCodeURL, "/") + "/" + deviceCode
	resp, gerr := f.client.Get(pollURL)
	if gerr != nil {
		return nil, false, fmt.Errorf("device poll request: %w", gerr)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	switch resp.StatusCode {
	case http.StatusAccepted: // 202 pending
		return nil, true, nil
	case http.StatusForbidden: // 403 denied
		return nil, false, fmt.Errorf("device authorization denied")
	case http.StatusGone: // 410 expired
		return nil, false, fmt.Errorf("device code expired")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false, fmt.Errorf("device poll failed: %d", resp.StatusCode)
	}

	var parsed struct {
		Status string `json:"status"`
		Token  string `json:"token"`
	}
	if jerr := json.Unmarshal(body, &parsed); jerr != nil {
		return nil, false, fmt.Errorf("decode device poll response: %w", jerr)
	}
	if parsed.Status == "approved" && parsed.Token != "" {
		// kilocode device tokens have no expiry/refresh.
		return &OAuthToken{AccessToken: parsed.Token}, false, nil
	}
	return nil, true, nil
}

func decodeStandardToken(body []byte) (*OAuthToken, error) {
	var parsed struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}
	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("token response missing access_token")
	}
	tok := &OAuthToken{AccessToken: parsed.AccessToken, RefreshToken: parsed.RefreshToken}
	if parsed.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second).Unix()
	}
	return tok, nil
}
