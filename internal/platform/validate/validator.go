package validate

import (
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"

	"e-com-rental-cloth/backend/internal/platform/httpx"
)

type Validator struct {
	v *validator.Validate
}

func New() *Validator {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})
	return &Validator{v: v}
}

func (val *Validator) Struct(s any) error {
	err := val.v.Struct(s)
	if err == nil {
		return nil
	}
	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		fields := make(map[string]string, len(verrs))
		for _, fe := range verrs {
			fields[fe.Field()] = fe.Tag()
		}
		return httpx.ValidationError(fields)
	}
	return httpx.NewError(http.StatusBadRequest, "INVALID_INPUT", "รูปแบบข้อมูลไม่ถูกต้อง", err)
}

func DecodeAndValidate[T any](r *http.Request, val *Validator) (T, error) {
	var dst T
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&dst); err != nil {
		return dst, httpx.NewError(http.StatusBadRequest, "MALFORMED_JSON", "รูปแบบ JSON ไม่ถูกต้องหรือมีฟิลด์ที่ไม่อนุญาต", err)
	}
	if err := val.Struct(dst); err != nil {
		return dst, err
	}
	return dst, nil
}
