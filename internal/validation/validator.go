package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func init() {
	err := validate.RegisterValidation("custom_id", func(fl validator.FieldLevel) bool {
		if fl.Field().String() == "" {
			return true
		}

		re := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

		return re.MatchString(fl.Field().String())
	})
	if err != nil {
		panic(fmt.Sprintf("failed to register custom validation: %v", err))
	}
}

type ValidationError struct {
	Errors []string
}

func (v *ValidationError) Error() string {
	return strings.Join(v.Errors, ", ")
}

func ValidateStruct(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		var validationErrors []string

		for _, err := range err.(validator.ValidationErrors) {
			var message string

			switch err.Tag() {
			case "custom_id":
				message = fmt.Sprintf(
					"field '%s' must contain only letters, numbers, hyphens, and underscores",
					err.Field(),
				)
			default:
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
