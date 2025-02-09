package validator

import (
	"testing"
)

type TestStruct struct {
	Name     string `validate:"required"`
	Age      int    `validate:"gte=0,lte=130"`
	Email    string `validate:"required,email"`
	Optional string
}

func TestValidator_ValidateStruct(t *testing.T) {
	v := New()

	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
		fields  []string
	}{
		{
			name: "Valid struct",
			input: TestStruct{
				Name:  "John Doe",
				Age:   25,
				Email: "john@example.com",
			},
			wantErr: false,
		},
		{
			name: "Missing required fields",
			input: TestStruct{
				Age: 25,
			},
			wantErr: true,
			fields:  []string{"Name", "Email"},
		},
		{
			name: "Invalid email",
			input: TestStruct{
				Name:  "John Doe",
				Age:   25,
				Email: "not-an-email",
			},
			wantErr: true,
			fields:  []string{"Email"},
		},
		{
			name: "Age out of range",
			input: TestStruct{
				Name:  "John Doe",
				Age:   150,
				Email: "john@example.com",
			},
			wantErr: true,
			fields:  []string{"Age"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := v.ValidateStruct(tt.input)

			if tt.wantErr && len(errors) == 0 {
				t.Error("ValidateStruct() expected errors but got none")
				return
			}

			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("ValidateStruct() got unexpected errors: %v", errors)
				return
			}

			if tt.wantErr {
				foundFields := make([]string, 0)
				for _, err := range errors {
					foundFields = append(foundFields, err.Field)
				}
				for _, expectedField := range tt.fields {
					found := false
					for _, foundField := range foundFields {
						if foundField == expectedField {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected validation error for field %s, but got none", expectedField)
					}
				}
			}
		})
	}
}

func TestValidator_Validate(t *testing.T) {
	v := New()

	tests := []struct {
		name    string
		value   interface{}
		tag     string
		wantErr bool
	}{
		{
			name:    "Valid email",
			value:   "test@example.com",
			tag:     "email",
			wantErr: false,
		},
		{
			name:    "Invalid email",
			value:   "not-an-email",
			tag:     "email",
			wantErr: true,
		},
		{
			name:    "Required field present",
			value:   "value",
			tag:     "required",
			wantErr: false,
		},
		{
			name:    "Required field empty",
			value:   "",
			tag:     "required",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := v.Validate(tt.value, tt.tag)

			if tt.wantErr && len(errors) == 0 {
				t.Error("Validate() expected errors but got none")
			}

			if !tt.wantErr && len(errors) > 0 {
				t.Errorf("Validate() got unexpected errors: %v", errors)
			}
		})
	}
}

func TestNew(t *testing.T) {
	v := New()
	if v == nil || v.cli == nil {
		t.Error("New() returned invalid validator")
	}
}
