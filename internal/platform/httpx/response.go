package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

type PageMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"perPage"`
	TotalItems int `json:"totalItems"`
}

type Envelope[T any] struct {
	Data T         `json:"data"`
	Meta *PageMeta `json:"meta,omitempty"`
}

type ErrorBody struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"requestId,omitempty"`
}

type AppError struct {
	Status  int
	Code    string
	Message string
	Fields  map[string]string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return e.Code + ": " + e.Message + " (" + e.Cause.Error() + ")"
	}
	return e.Code + ": " + e.Message
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func NewError(status int, code, message string, cause error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, Cause: cause}
}

func ValidationError(fields map[string]string) *AppError {
	return &AppError{
		Status:  http.StatusUnprocessableEntity,
		Code:    "VALIDATION_ERROR",
		Message: "ข้อมูลที่ส่งมาไม่ถูกต้อง",
		Fields:  fields,
	}
}

func JSON[T any](w http.ResponseWriter, status int, data T, meta *PageMeta) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Envelope[T]{Data: data, Meta: meta})
}

// WriteError writes a standardized non-200 JSON error response so Nuxt's $api onResponseError
// and SWR caching always detect failures reliably without leaking internal SQL/stack details.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	reqID := middleware.GetReqID(r.Context())

	var appErr *AppError
	if errors.As(err, &appErr) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(appErr.Status)
		_ = json.NewEncoder(w).Encode(ErrorBody{
			Code:      appErr.Code,
			Message:   appErr.Message,
			Fields:    appErr.Fields,
			RequestID: reqID,
		})
		return
	}

	slog.ErrorContext(r.Context(), "unhandled server error",
		slog.String("request_id", reqID),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.Any("err", err),
	)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(ErrorBody{
		Code:      "INTERNAL_ERROR",
		Message:   "เกิดข้อผิดพลาดภายในระบบ กรุณาลองใหม่อีกครั้ง",
		RequestID: reqID,
	})
}
