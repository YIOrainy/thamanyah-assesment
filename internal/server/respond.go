package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validate is the shared validator. Field names in errors use the json tag.
var validate = newValidator()

func newValidator() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})
	return v
}

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

type Problem struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail,omitempty"`
}

// FieldError is a single field-level validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func Error(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Problem{Title: http.StatusText(status), Status: status, Detail: detail})
}

func validationError(w http.ResponseWriter, fields []FieldError) {
	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(struct {
		Problem
		Errors []FieldError `json:"errors"`
	}{
		Problem: Problem{Title: http.StatusText(http.StatusBadRequest), Status: http.StatusBadRequest, Detail: "validation failed"},
		Errors:  fields,
	})
}

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		slog.WarnContext(r.Context(), "decode json request failed", "method", r.Method, "path", r.URL.Path, "error", err)
		Error(w, http.StatusBadRequest, "invalid JSON body")
		return false
	}

	if err := validate.Struct(dst); err != nil {
		var verrs validator.ValidationErrors
		if errors.As(err, &verrs) {
			fields := make([]FieldError, 0, len(verrs))
			for _, fe := range verrs {
				fields = append(fields, FieldError{Field: fe.Field(), Message: validationMessage(fe)})
			}
			slog.WarnContext(r.Context(), "request validation failed", "method", r.Method, "path", r.URL.Path, "fields", len(fields))
			validationError(w, fields)
			return false
		}
		Error(w, http.StatusBadRequest, "validation failed")
		return false
	}
	return true
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "oneof":
		return "must be one of: " + fe.Param()
	case "min":
		return "must be at least " + fe.Param() + " characters"
	case "gte":
		return "must be >= " + fe.Param()
	case "gt":
		return "must be > " + fe.Param()
	default:
		return "is invalid"
	}
}
