package validator

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

// Validator is a struct that provides methods for struct validation using the underlying validator library.
type Validator struct {
	cli *validator.Validate
}

// ValidationError represents an error encountered during validation of a struct field.
type ValidationError struct {
	Field   string
	Message interface{}
}

// ValidateStruct validates the provided struct using the underlying validator and returns a slice of validation errors.
func (v *Validator) ValidateStruct(s interface{}) []ValidationError {
	errors := make([]ValidationError, 0)

	err := v.cli.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			var msg string
			switch err.Tag() {
			case "required":
				msg = fmt.Sprintf("%s is required", err.Namespace())
			default:
				msg = err.Error()
			}

			errors = append(errors, ValidationError{
				Field:   err.StructField(),
				Message: msg,
			})
		}
	}

	return errors
}

// New creates a new validator
func New() *Validator {
	return &Validator{
		cli: validator.New(validator.WithRequiredStructEnabled()),
	}
}
