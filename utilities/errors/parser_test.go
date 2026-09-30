package custom_errors

import (
	"encoding/json"
	"testing"

	"github.com/go-playground/validator/v10"
)

type dummyRequest struct {
	Password string `validate:"password"`
	Email    string `validate:"email"`
	Username string `validate:"min=6"`
}

func dummyPasswordValidation(fl validator.FieldLevel) bool {
	return false // Always fail to trigger tag
}

func TestParseError_PasswordValidation(t *testing.T) {
	val := validator.New()
	val.RegisterValidation("password", dummyPasswordValidation)

	req := dummyRequest{
		Password: "plain",
		Email:    "invalid-email",
		Username: "usr",
	}

	err := val.Struct(req)
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}

	result := ParseError(err)

	if passErrors, ok := result["Password"]; ok {
		found := false
		for _, msg := range passErrors {
			if msg == "Password must contains letters, number, and special characters" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected error message 'Password must contains letters, number, and special characters', got %v", passErrors)
		}
	} else {
		t.Errorf("expected error for field 'Password' in result map, got %v", result)
	}
}

func TestParseError_UnmarshalTypeError(t *testing.T) {
	var target struct {
		Age int `json:"age"`
	}

	err := json.Unmarshal([]byte(`{"age": "not-a-number"}`), &target)
	if err == nil {
		t.Fatalf("expected unmarshal error, got nil")
	}

	result := ParseError(err)
	if _, ok := result["age"]; !ok {
		t.Errorf("expected error for field 'age', got %v", result)
	}
}
