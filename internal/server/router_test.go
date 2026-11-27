package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"e-com-rental-cloth/backend/internal/platform/auth"
	"e-com-rental-cloth/backend/internal/platform/httpx"
	"e-com-rental-cloth/backend/internal/platform/id"
	"e-com-rental-cloth/backend/internal/platform/validate"
	"e-com-rental-cloth/backend/internal/server"
)

func TestHealthEndpointAndEnvelope(t *testing.T) {
	t.Parallel()

	router := server.NewRouter(server.Options{AllowedOrigins: []string{"http://localhost:3000"}})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var body httpx.Envelope[server.HealthStatus]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ok", body.Data.Status)
	assert.Equal(t, "e-com-rental-cloth-api", body.Data.Service)
}

func TestPerUserRateLimitBlocksExcessiveCalls(t *testing.T) {
	t.Parallel()

	userID := id.MustNewV7()
	limiter := server.RateLimitByUserOrIP(2, time.Minute)
	handler := limiter(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 1; i <= 3; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/coupons/check", nil)
		// Simulate bot rotating remote IP while reusing the same authenticated user session
		req.RemoteAddr = "203.0.113." + string(rune('0'+i)) + ":12345"
		req = req.WithContext(auth.WithUserID(req.Context(), userID))
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if i <= 2 {
			assert.Equal(t, http.StatusOK, rec.Code)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, rec.Code)
			var errBody httpx.ErrorBody
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &errBody))
			assert.Equal(t, "RATE_LIMIT_EXCEEDED", errBody.Code)
		}
	}
}

func TestAuthAndUUIDv7AndValidationPlatform(t *testing.T) {
	t.Parallel()

	// 1. UUIDv7 time ordering
	first, err := id.NewV7()
	require.NoError(t, err)
	time.Sleep(2 * time.Millisecond)
	second, err := id.NewV7()
	require.NoError(t, err)
	assert.Less(t, first.String(), second.String())

	// 2. Argon2id hash & verify + opaque session token hash
	hash, err := auth.HashPassword("CorrectHorseBatteryStaple!99")
	require.NoError(t, err)
	ok, err := auth.VerifyPassword("CorrectHorseBatteryStaple!99", hash)
	require.NoError(t, err)
	assert.True(t, ok)

	wrong, err := auth.VerifyPassword("wrong-password", hash)
	require.NoError(t, err)
	assert.False(t, wrong)

	rawToken, tokenHash, err := auth.GenerateSessionToken()
	require.NoError(t, err)
	assert.NotEmpty(t, rawToken)
	assert.Len(t, tokenHash, 32)

	// 3. Struct validation maps json tag names into 422 VALIDATION_ERROR fields
	type createBookingInput struct {
		GarmentID string `json:"garmentId" validate:"required,uuid"`
		RentalDays int   `json:"rentalDays" validate:"required,gt=0"`
	}
	v := validate.New()
	req := httptest.NewRequest(http.MethodPost, "/bookings", bytes.NewBufferString(`{"garmentId":"not-a-uuid","rentalDays":0}`))
	_, valErr := validate.DecodeAndValidate[createBookingInput](req, v)
	require.Error(t, valErr)

	var appErr *httpx.AppError
	require.ErrorAs(t, valErr, &appErr)
	assert.Equal(t, http.StatusUnprocessableEntity, appErr.Status)
	assert.Equal(t, "uuid", appErr.Fields["garmentId"])
	assert.Equal(t, "required", appErr.Fields["rentalDays"])
}
