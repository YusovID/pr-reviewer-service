// package validation provides helper functions for request data validation.
// It uses the go-playground/validator library and includes custom validation rules.
package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// init registers custom validation rules with the validator instance.
// This function runs automatically when the package is imported.
func init() {
	// RegisterValidation registers the "custom_id" tag.
	// This validator ensures that fields like user_id and pull_request_id
	// only contain alphanumeric characters, hyphens, and underscores.
	err := validate.RegisterValidation("custom_id", func(fl validator.FieldLevel) bool {
		if fl.Field().String() == "" {
			// Allow empty strings to be handled by the 'required' tag.
			return true
		}

		re := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

		return re.MatchString(fl.Field().String())
	})
	if err != nil {
		// Panic on initialization if a custom validator fails to register,
		// as it indicates a critical startup failure.
		panic(fmt.Sprintf("failed to register custom validation: %v", err))
	}
}

// ValidationError is a custom error type that holds a slice of validation error messages.
type ValidationError struct {
	Errors []string
}

// Error returns a single string concatenating all validation error messages.
func (v *ValidationError) Error() string {
	return strings.Join(v.Errors, ", ")
}

// ValidateStruct performs validation on a given struct based on its validation tags.
// If validation fails, it returns a *ValidationError with user-friendly messages.
func ValidateStruct(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		var validationErrors []string

		// Cast the error to validator.ValidationErrors to iterate over individual field errors.
		for _, err := range err.(validator.ValidationErrors) {
			var message string

			switch err.Tag() {
			case "custom_id":
				message = fmt.Sprintf(
					"field '%s' must contain only letters, numbers, hyphens, and underscores",
					err.Field(),
				)
			default:
				// Default message for other standard validation tags like 'required', 'min', 'max', etc.
				message = fmt.Sprintf(
					"field '%s' failed on the '%s' tag",
					err.Field(),
					err.Tag(),
				)
			}
			validationErrors = append(validationErrors, message)
		}

		return &ValidationError{Errors: validationErrors}
	}

	return nil
}
