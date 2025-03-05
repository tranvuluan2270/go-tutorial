package utils

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// FormatValidationError formats validation errors into user-friendly messages
func FormatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return fmt.Sprintf("Minimum length is %s", err.Param())
	case "max":
		return fmt.Sprintf("Maximum length is %s", err.Param())
	case "oneof":
		return fmt.Sprintf("Must be one of: %s", err.Param())
	default:
		return fmt.Sprintf("Validation failed on %s", err.Tag())
	}
}
