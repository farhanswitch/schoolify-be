package users

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

type mockUserService struct {
	GetUserDataByUsernameSrvcFunc func(param models.UserLoginRequest) (models.User, errUtility.CustomError)
	RegisterUserSrvcFunc          func(param models.UserRegisterRequest) (bool, errUtility.CustomError)
}

func (m *mockUserService) GetUserDataByUsernameSrvc(param models.UserLoginRequest) (models.User, errUtility.CustomError) {
	if m.GetUserDataByUsernameSrvcFunc != nil {
		return m.GetUserDataByUsernameSrvcFunc(param)
	}
	return models.User{}, errUtility.CustomError{}
}

func (m *mockUserService) RegisterUserSrvc(param models.UserRegisterRequest) (bool, errUtility.CustomError) {
	if m.RegisterUserSrvcFunc != nil {
		return m.RegisterUserSrvcFunc(param)
	}
	return false, errUtility.CustomError{}
}

func TestPasswordValidation(t *testing.T) {
	type testStruct struct {
		Password string `validate:"password"`
	}

	val := validator.New()
	val.RegisterValidation("password", passwordValidation)

	tests := []struct {
		name     string
		password string
		expected bool
	}{
		{
			name:     "valid with upper, number, and special character",
			password: "Password123!",
			expected: true,
		},
		{
			name:     "valid with multiple symbols and numbers",
			password: "P@ssw0rd#2026",
			expected: true,
		},
		{
			name:     "valid uppercase only with symbol and number",
			password: "PASSWORD123!",
			expected: true,
		},
		{
			name:     "invalid: missing uppercase",
			password: "password123!",
			expected: false,
		},
		{
			name:     "invalid: missing number",
			password: "Password!@#",
			expected: false,
		},
		{
			name:     "invalid: missing special character",
			password: "Password123",
			expected: false,
		},
		{
			name:     "invalid: empty string",
			password: "",
			expected: false,
		},
		{
			name:     "invalid: only letters",
			password: "PasswordOnly",
			expected: false,
		},
		{
			name:     "invalid: only numbers",
			password: "1234567890",
			expected: false,
		},
		{
			name:     "invalid: only special characters",
			password: "!@#$%^&*()",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := val.Struct(testStruct{Password: tc.password})
			isValid := (err == nil)
			if isValid != tc.expected {
				t.Errorf("passwordValidation(%q): expected valid=%v, got valid=%v", tc.password, tc.expected, isValid)
			}
		})
	}
}

func TestRegisterUserCtrl(t *testing.T) {
	defaultPayload := map[string]string{
		"email":           "johndoe@example.com",
		"username":        "johndoe",
		"firstName":       "John",
		"lastName":        "Doe",
		"password":        "Password123!",
		"confirmPassword": "Password123!",
	}

	buildJSONBody := func(payload map[string]string) *bytes.Buffer {
		data, _ := json.Marshal(payload)
		return bytes.NewBuffer(data)
	}

	t.Run("should return 400 Bad Request when JSON is malformed", func(t *testing.T) {
		serviceMock := &mockUserService{}
		ctrl := userController{service: serviceMock}

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader("{invalid-json"))
		w := httptest.NewRecorder()

		ctrl.RegisterUserCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
		if !strings.Contains(w.Body.String(), `"message":"Invalid Data"`) {
			t.Errorf("expected response to contain 'Invalid Data', got %s", w.Body.String())
		}
	})

	t.Run("should return 400 Bad Request when required fields are missing", func(t *testing.T) {
		serviceMock := &mockUserService{}
		ctrl := userController{service: serviceMock}

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader("{}"))
		w := httptest.NewRecorder()

		ctrl.RegisterUserCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}

		var resp map[string]any
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		if err != nil {
			t.Fatalf("failed to parse response JSON: %v", err)
		}

		errorsMap, ok := resp["errors"].(map[string]any)
		if !ok {
			t.Fatalf("expected response to contain 'errors' map, got %v", resp)
		}

		requiredFields := []string{"Email", "Username", "FirstName", "Password", "ConfirmPassword"}
		for _, field := range requiredFields {
			if _, exists := errorsMap[field]; !exists {
				t.Errorf("expected validation error for field %q", field)
			}
		}
	})

	t.Run("should return 400 Bad Request on invalid email", func(t *testing.T) {
		serviceMock := &mockUserService{}
		ctrl := userController{service: serviceMock}

		payload := make(map[string]string)
		for k, v := range defaultPayload {
			payload[k] = v
		}
		payload["email"] = "not-a-valid-email"

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", buildJSONBody(payload))
		w := httptest.NewRecorder()

		ctrl.RegisterUserCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
		if !strings.Contains(w.Body.String(), `"Email"`) {
			t.Errorf("expected error for email field, got: %s", w.Body.String())
		}
	})

	t.Run("should return 400 Bad Request on invalid username (min length, max length, alphanum)", func(t *testing.T) {
		invalidUsernames := []struct {
			name     string
			username string
		}{
			{name: "too short (< 6 chars)", username: "usr"},
			{name: "too long (> 15 chars)", username: "user_is_way_too_long"},
			{name: "non-alphanumeric", username: "user!name"},
		}

		for _, item := range invalidUsernames {
			t.Run(item.name, func(t *testing.T) {
				serviceMock := &mockUserService{}
				ctrl := userController{service: serviceMock}

				payload := make(map[string]string)
				for k, v := range defaultPayload {
					payload[k] = v
				}
				payload["username"] = item.username

				req := httptest.NewRequest(http.MethodPost, "/api/auth/register", buildJSONBody(payload))
				w := httptest.NewRecorder()

				ctrl.RegisterUserCtrl(w, req)

				if w.Code != http.StatusBadRequest {
					t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
				}
				if !strings.Contains(w.Body.String(), `"Username"`) {
					t.Errorf("expected error for username field, got: %s", w.Body.String())
				}
			})
		}
	})

	t.Run("should return 400 Bad Request on invalid firstName (min < 3, max > 30)", func(t *testing.T) {
		invalidFirstNames := []struct {
			name      string
			firstName string
		}{
			{name: "too short (< 3 chars)", firstName: "Jo"},
			{name: "too long (> 30 chars)", firstName: strings.Repeat("A", 31)},
		}

		for _, item := range invalidFirstNames {
			t.Run(item.name, func(t *testing.T) {
				serviceMock := &mockUserService{}
				ctrl := userController{service: serviceMock}

				payload := make(map[string]string)
				for k, v := range defaultPayload {
					payload[k] = v
				}
				payload["firstName"] = item.firstName

				req := httptest.NewRequest(http.MethodPost, "/api/auth/register", buildJSONBody(payload))
				w := httptest.NewRecorder()

				ctrl.RegisterUserCtrl(w, req)

				if w.Code != http.StatusBadRequest {
					t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
				}
				if !strings.Contains(w.Body.String(), `"FirstName"`) {
					t.Errorf("expected error for firstName field, got: %s", w.Body.String())
				}
			})
		}
	})

	t.Run("should return 400 Bad Request on invalid lastName (> 30 chars)", func(t *testing.T) {
		serviceMock := &mockUserService{}
		ctrl := userController{service: serviceMock}

		payload := make(map[string]string)
		for k, v := range defaultPayload {
			payload[k] = v
		}
		payload["lastName"] = strings.Repeat("B", 31)

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", buildJSONBody(payload))
		w := httptest.NewRecorder()

		ctrl.RegisterUserCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
		if !strings.Contains(w.Body.String(), `"LastName"`) {
			t.Errorf("expected error for lastName field, got: %s", w.Body.String())
		}
	})

	t.Run("should return 400 Bad Request on invalid password (complexity failure)", func(t *testing.T) {
		serviceMock := &mockUserService{}
		ctrl := userController{service: serviceMock}

		payload := make(map[string]string)
		for k, v := range defaultPayload {
			payload[k] = v
		}
		payload["password"] = "simplepassword"

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", buildJSONBody(payload))
		w := httptest.NewRecorder()

		ctrl.RegisterUserCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
		}
		if !strings.Contains(w.Body.String(), "Password must contains letters, number, and special characters") {
			t.Errorf("expected password complexity error message, got: %s", w.Body.String())
		}
	})

	t.Run("should return 400 Bad Request when service returns error", func(t *testing.T) {
		serviceErrors := []struct {
			name        string
			messageToSend string
		}{
			{name: "duplicate username", messageToSend: "The username has already been taken."},
			{name: "duplicate email", messageToSend: "This email address is already registered. Please log in."},
			{name: "password mismatch", messageToSend: "Password and Confirm Password must match!"},
			{name: "internal server error", messageToSend: "Internal Server Error"},
		}

		for _, item := range serviceErrors {
			t.Run(item.name, func(t *testing.T) {
				serviceMock := &mockUserService{
					RegisterUserSrvcFunc: func(param models.UserRegisterRequest) (bool, errUtility.CustomError) {
						return true, errUtility.CustomError{
							Code:          http.StatusBadRequest,
							Message:       "error",
							MessageToSend: item.messageToSend,
						}
					},
				}
				ctrl := userController{service: serviceMock}

				req := httptest.NewRequest(http.MethodPost, "/api/auth/register", buildJSONBody(defaultPayload))
				w := httptest.NewRecorder()

				ctrl.RegisterUserCtrl(w, req)

				if w.Code != http.StatusBadRequest {
					t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
				}
				if !strings.Contains(w.Body.String(), item.messageToSend) {
					t.Errorf("expected body to contain '%s', got '%s'", item.messageToSend, w.Body.String())
				}
			})
		}
	})

	t.Run("should return 201 Created when registration succeeds", func(t *testing.T) {
		serviceCalled := false
		serviceMock := &mockUserService{
			RegisterUserSrvcFunc: func(param models.UserRegisterRequest) (bool, errUtility.CustomError) {
				serviceCalled = true
				if param.Username != defaultPayload["username"] {
					t.Errorf("expected username %s, got %s", defaultPayload["username"], param.Username)
				}
				if param.Email != defaultPayload["email"] {
					t.Errorf("expected email %s, got %s", defaultPayload["email"], param.Email)
				}
				return false, errUtility.CustomError{}
			},
		}
		ctrl := userController{service: serviceMock}

		req := httptest.NewRequest(http.MethodPost, "/api/auth/register", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.RegisterUserCtrl(w, req)

		if !serviceCalled {
			t.Errorf("expected service to be called on valid registration")
		}
		if w.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
		}
		if !strings.Contains(w.Body.String(), `"message":"User created successfully"`) {
			t.Errorf("expected 'User created successfully', got %s", w.Body.String())
		}
	})
}
