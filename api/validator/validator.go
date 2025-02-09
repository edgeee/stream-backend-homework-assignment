package validator

import (
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

func (v *Validator) formatError(err error) []ValidationError {
	errors := make([]ValidationError, 0)
	for _, err := range err.(validator.ValidationErrors) {
		msg := err.Error()
		errors = append(errors, ValidationError{
			Field:   err.StructField(),
			Message: msg,
		})
	}

	return errors
}

// ValidateStruct validates the provided struct using the underlying validator and returns a slice of validation errors.
func (v *Validator) ValidateStruct(s interface{}) []ValidationError {
	err := v.cli.Struct(s)
	if err != nil {
		return v.formatError(err)
	}
	return nil
}

// Validate checks the provided value against the specified validation tags and returns a slice of validation errors.
func (v *Validator) Validate(value interface{}, tag string) []ValidationError {
	err := v.cli.Var(value, tag)
	if err != nil {
		return v.formatError(err)
	}
	return nil
}

// New initializes and returns a new instance of the Validator
func New() *Validator {
	return &Validator{
		cli: validator.New(validator.WithRequiredStructEnabled()),
	}
}
