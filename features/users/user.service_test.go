package users

import (
	"net/http"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type mockUserRepo struct {
	GetUserByUsernameFunc func(param models.UserLoginRequest) (models.User, errUtility.CustomError)
	RegisterUserFunc      func(user models.UserRegisterRequest) (bool, errUtility.CustomError)
}

func (m *mockUserRepo) GetUserByUsername(param models.UserLoginRequest) (models.User, errUtility.CustomError) {
	if m.GetUserByUsernameFunc != nil {
		return m.GetUserByUsernameFunc(param)
	}
	return models.User{}, errUtility.CustomError{}
}

func (m *mockUserRepo) RegisterUser(user models.UserRegisterRequest) (bool, errUtility.CustomError) {
	if m.RegisterUserFunc != nil {
		return m.RegisterUserFunc(user)
	}
	return false, errUtility.CustomError{}
}

func TestRegisterUserSrvc(t *testing.T) {
	validRequest := models.UserRegisterRequest{
		Email:           "johndoe@example.com",
		Username:        "johndoe",
		FirstName:       "John",
		LastName:        "Doe",
		Password:        "Password123!",
		ConfirmPassword: "Password123!",
	}

	t.Run("should succeed when all inputs are valid", func(t *testing.T) {
		var capturedUser models.UserRegisterRequest
		repoMock := &mockUserRepo{
			RegisterUserFunc: func(user models.UserRegisterRequest) (bool, errUtility.CustomError) {
				capturedUser = user
				return false, errUtility.CustomError{}
			},
		}

		service := userService{repo: repoMock}
		isErr, errObj := service.RegisterUserSrvc(validRequest)

		if isErr {
			t.Fatalf("expected isErr to be false, got true with message: %s", errObj.Message)
		}
		if errObj.Code != 0 {
			t.Errorf("expected errObj.Code to be 0, got %d", errObj.Code)
		}

		// Verify ID generation (UUIDv7)
		if capturedUser.ID == uuid.Nil {
			t.Errorf("expected generated ID to not be nil UUID")
		}

		// Verify RoleID generation (UUIDv7)
		if capturedUser.RoleID == uuid.Nil {
			t.Errorf("expected generated RoleID to not be nil UUID")
		}

		// Verify Password was hashed
		if capturedUser.Password == validRequest.Password {
			t.Errorf("expected password to be hashed, but was still plaintext")
		}
		err := bcrypt.CompareHashAndPassword([]byte(capturedUser.Password), []byte(validRequest.Password))
		if err != nil {
			t.Errorf("hashed password failed bcrypt verification: %v", err)
		}

		// Verify rest of fields were forwarded correctly
		if capturedUser.Email != validRequest.Email {
			t.Errorf("expected email %s, got %s", validRequest.Email, capturedUser.Email)
		}
		if capturedUser.Username != validRequest.Username {
			t.Errorf("expected username %s, got %s", validRequest.Username, capturedUser.Username)
		}
		if capturedUser.FirstName != validRequest.FirstName {
			t.Errorf("expected firstName %s, got %s", validRequest.FirstName, capturedUser.FirstName)
		}
		if capturedUser.LastName != validRequest.LastName {
			t.Errorf("expected lastName %s, got %s", validRequest.LastName, capturedUser.LastName)
		}
	})

	t.Run("should return error when password and confirm password do not match", func(t *testing.T) {
		repoCalled := false
		repoMock := &mockUserRepo{
			RegisterUserFunc: func(user models.UserRegisterRequest) (bool, errUtility.CustomError) {
				repoCalled = true
				return false, errUtility.CustomError{}
			},
		}

		req := validRequest
		req.ConfirmPassword = "DifferentPassword123!"

		service := userService{repo: repoMock}
		isErr, errObj := service.RegisterUserSrvc(req)

		if !isErr {
			t.Fatalf("expected isErr to be true, got false")
		}
		if errObj.Code != 400 {
			t.Errorf("expected error code 400, got %d", errObj.Code)
		}
		if errObj.MessageToSend != "Password and Confirm Password must match!" {
			t.Errorf("expected 'Password and Confirm Password must match!', got '%s'", errObj.MessageToSend)
		}
		if repoCalled {
			t.Errorf("expected repo not to be called when password mismatch occurs")
		}
	})

	t.Run("should return error when password exceeds bcrypt max length (72 bytes)", func(t *testing.T) {
		repoCalled := false
		repoMock := &mockUserRepo{
			RegisterUserFunc: func(user models.UserRegisterRequest) (bool, errUtility.CustomError) {
				repoCalled = true
				return false, errUtility.CustomError{}
			},
		}

		longPassword := strings.Repeat("A", 73)
		req := validRequest
		req.Password = longPassword
		req.ConfirmPassword = longPassword

		service := userService{repo: repoMock}
		isErr, errObj := service.RegisterUserSrvc(req)

		if !isErr {
			t.Fatalf("expected isErr to be true on bcrypt failure, got false")
		}
		if errObj.Code != 500 {
			t.Errorf("expected error code 500, got %d", errObj.Code)
		}
		if errObj.MessageToSend != "Internal Server Error" {
			t.Errorf("expected 'Internal Server Error', got '%s'", errObj.MessageToSend)
		}
		if repoCalled {
			t.Errorf("expected repo not to be called when hashing fails")
		}
	})

	t.Run("should return error when repo returns duplicate username error", func(t *testing.T) {
		repoMock := &mockUserRepo{
			RegisterUserFunc: func(user models.UserRegisterRequest) (bool, errUtility.CustomError) {
				return true, errUtility.CustomError{
					Code:          http.StatusUnprocessableEntity,
					Message:       "pq: duplicate key value violates unique constraint \"users_username_key\"",
					MessageToSend: "The username has already been taken.",
				}
			},
		}

		service := userService{repo: repoMock}
		isErr, errObj := service.RegisterUserSrvc(validRequest)

		if !isErr {
			t.Fatalf("expected isErr to be true, got false")
		}
		if errObj.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected error code %d, got %d", http.StatusUnprocessableEntity, errObj.Code)
		}
		if errObj.MessageToSend != "The username has already been taken." {
			t.Errorf("expected message 'The username has already been taken.', got '%s'", errObj.MessageToSend)
		}
	})

	t.Run("should return error when repo returns duplicate email error", func(t *testing.T) {
		repoMock := &mockUserRepo{
			RegisterUserFunc: func(user models.UserRegisterRequest) (bool, errUtility.CustomError) {
				return true, errUtility.CustomError{
					Code:          http.StatusUnprocessableEntity,
					Message:       "pq: duplicate key value violates unique constraint \"users_email_key\"",
					MessageToSend: "This email address is already registered. Please log in.",
				}
			},
		}

		service := userService{repo: repoMock}
		isErr, errObj := service.RegisterUserSrvc(validRequest)

		if !isErr {
			t.Fatalf("expected isErr to be true, got false")
		}
		if errObj.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected error code %d, got %d", http.StatusUnprocessableEntity, errObj.Code)
		}
		if errObj.MessageToSend != "This email address is already registered. Please log in." {
			t.Errorf("expected message 'This email address is already registered. Please log in.', got '%s'", errObj.MessageToSend)
		}
	})

	t.Run("should return error when repo returns internal server error", func(t *testing.T) {
		repoMock := &mockUserRepo{
			RegisterUserFunc: func(user models.UserRegisterRequest) (bool, errUtility.CustomError) {
				return true, errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "pq: connection refused",
					MessageToSend: "Internal server error.",
				}
			},
		}

		service := userService{repo: repoMock}
		isErr, errObj := service.RegisterUserSrvc(validRequest)

		if !isErr {
			t.Fatalf("expected isErr to be true, got false")
		}
		if errObj.Code != http.StatusInternalServerError {
			t.Errorf("expected error code %d, got %d", http.StatusInternalServerError, errObj.Code)
		}
		if errObj.MessageToSend != "Internal server error." {
			t.Errorf("expected message 'Internal server error.', got '%s'", errObj.MessageToSend)
		}
	})
}
