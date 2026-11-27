package security

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"e-com-rental-cloth/backend/internal/platform/httpx"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type TurnstileVerifier struct {
	secretKey  string
	httpClient *http.Client
}

type siteverifyResponse struct {
	Success bool   `json:"success"`
	Action  string `json:"action"`
}

func NewTurnstileVerifier(secretKey string) *TurnstileVerifier {
	return &TurnstileVerifier{
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// Verify validates a single-use Cloudflare Turnstile token before executing sensitive mutations
// (e.g., booking creation, coupon checks) so scripted bots with stolen cookies cannot replay calls.
func (tv *TurnstileVerifier) Verify(ctx context.Context, token, remoteIP, expectedAction string) error {
	if strings.TrimSpace(token) == "" {
		return httpx.NewError(http.StatusForbidden, "TURNSTILE_REQUIRED", "กรุณายืนยันความปลอดภัยก่อนทำรายการ", nil)
	}

	form := url.Values{}
	form.Set("secret", tv.secretKey)
	form.Set("response", token)
	if remoteIP != "" {
		form.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, turnstileVerifyURL, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := tv.httpClient.Do(req)
	if err != nil {
		return httpx.NewError(http.StatusServiceUnavailable, "TURNSTILE_UNAVAILABLE", "ไม่สามารถตรวจสอบความปลอดภัยได้ชั่วคราว", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var result siteverifyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if !result.Success || (expectedAction != "" && result.Action != "" && result.Action != expectedAction) {
		return httpx.NewError(http.StatusForbidden, "TURNSTILE_FAILED", "การยืนยันความปลอดภัยไม่ผ่านหรือหมดอายุ", nil)
	}
	return nil
}
