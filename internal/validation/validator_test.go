package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	ID    string `validate:"required,custom_id"`
	Name  string `validate:"required"`
	Email string `validate:"required,email"`
}

func TestValidateStruct(t *testing.T) {
	testCases := []struct {
		name             string
		input            TestStruct
		expectError      bool
		expectedErrorMsg string
	}{
		{
			name: "Success: All fields are valid",
			input: TestStruct{
				ID:    "valid-id_123-",
				Name:  "John Doe",
				Email: "test@example.com",
			},
			expectError: false,
		},
		{
			name: "Failure: Invalid custom_id with spaces",
			input: TestStruct{
				ID:    "invalid id",
				Name:  "John Doe",
				Email: "test@example.com",
			},
			expectError:      true,
			expectedErrorMsg: "field 'ID' must contain only letters, numbers, hyphens, and underscores",
		},
		{
			name: "Failure: Invalid custom_id with special characters",
			input: TestStruct{
				ID:    "invalid-id-!",
				Name:  "John Doe",
				Email: "test@example.com",
			},
			expectError:      true,
			expectedErrorMsg: "field 'ID' must contain only letters, numbers, hyphens, and underscores",
		},
		{
			name: "Failure: Missing required field (Name)",
			input: TestStruct{
				ID:    "valid-id",
				Name:  "",
				Email: "test@example.com",
			},
			expectError:      true,
			expectedErrorMsg: "field 'Name' failed on the 'required' tag",
		},
		{
			name: "Failure: Invalid email format",
			input: TestStruct{
				ID:    "valid-id",
				Name:  "Jane Doe",
				Email: "not-an-email",
			},
			expectError:      true,
			expectedErrorMsg: "field 'Email' failed on the 'email' tag",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStruct(tc.input)

			if tc.expectError {
				assert.Error(t, err)
				require.IsType(t, &ValidationError{}, err, "error should be of type ValidationError")
				verr := err.(*ValidationError)
				assert.Contains(t, verr.Error(), tc.expectedErrorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	err := &ValidationError{
		Errors: []string{"error 1", "error 2"},
	}
	assert.Equal(t, "error 1, error 2", err.Error())
}
