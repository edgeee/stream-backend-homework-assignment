package validator

import (
	"fmt"
	"github.com/go-playground/validator/v10"
)

type Validator struct {
	cli *validator.Validate
}

type ValidationError struct {
	Field   string
	Message interface{}
}

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
