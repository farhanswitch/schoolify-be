package users

import (
	"context"
	"errors"
	"net/http"
	"os"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	os.Setenv("DB_PORT2", "5432")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("APP_PORT", "8080")
	os.Setenv("SESSION_TIME", "3600")
	os.Setenv("REFRESH_TIME", "7200")
	os.Setenv("ENCRYPT_KEY", "12345678901234567890123456789012") // 32 bytes required for AES-256
}

type mockUserRepo struct {
	GetUserByUsernameFunc func(param models.UserLoginRequest) (models.User, errUtility.CustomError)
	RegisterUserFunc      func(user models.UserRegisterRequest) (bool, errUtility.CustomError)
	UserLoginFunc         func(user models.UserLoginRequest) (models.UserLoginData, errUtility.CustomError)
	GetAccessMenusFunc    func(ctx context.Context, userID string) ([]models.UserAccessMenuResponse, errUtility.CustomError)
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

func (m *mockUserRepo) UserLogin(user models.UserLoginRequest) (models.UserLoginData, errUtility.CustomError) {
	if m.UserLoginFunc != nil {
		return m.UserLoginFunc(user)
	}
	return models.UserLoginData{}, errUtility.CustomError{}
}

func (m *mockUserRepo) GetAccessMenus(ctx context.Context, userID string) ([]models.UserAccessMenuResponse, errUtility.CustomError) {
	if m.GetAccessMenusFunc != nil {
		return m.GetAccessMenusFunc(ctx, userID)
	}
	return []models.UserAccessMenuResponse{}, errUtility.CustomError{}
}

type mockRedisUtility struct {
	SaveValueFunc   func(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	DeleteValueFunc func(ctx context.Context, key string) error
	GetValueFunc    func(ctx context.Context, key string) (string, error)
}

func TestGetAccessMenusSrvc(t *testing.T) {
	endpointRoot := uuid.New()
	endpointChild := uuid.New()
	endpointLeaf := uuid.New()
	childID := "child"
	rootID := "root"

	service := userService{repo: &mockUserRepo{GetAccessMenusFunc: func(_ context.Context, userID string) ([]models.UserAccessMenuResponse, errUtility.CustomError) {
		if userID != "user-1" {
			t.Fatalf("expected user ID user-1, got %q", userID)
		}
		return []models.UserAccessMenuResponse{
			{ID: "leaf", ParentID: &childID, Name: "Leaf", SortOrder: 1, Endpoints: []models.MenuApiResponse{{Id: endpointLeaf, Name: "Leaf endpoint", Status: 1}}},
			{ID: "child", ParentID: &rootID, Name: "Child", SortOrder: 1, Endpoints: []models.MenuApiResponse{{Id: endpointChild, Name: "Child endpoint", Status: 1}}},
			{ID: "root", Name: "Root", SortOrder: 1, Endpoints: []models.MenuApiResponse{{Id: endpointRoot, Name: "Root endpoint", Status: 1}}},
		}, errUtility.CustomError{}
	}}}

	menus, errObj := service.GetAccessMenusSrvc(context.Background(), "user-1")
	if errObj.Code != 0 {
		t.Fatalf("expected no error, got %d", errObj.Code)
	}
	if len(menus) != 1 || menus[0].ID != "root" {
		t.Fatalf("expected one root menu, got %#v", menus)
	}
	if len(menus[0].Children) != 1 || menus[0].Children[0].ID != "child" {
		t.Fatalf("expected child hierarchy, got %#v", menus[0].Children)
	}
	child := menus[0].Children[0]
	if len(child.Children) != 1 || child.Children[0].ID != "leaf" {
		t.Fatalf("expected leaf hierarchy, got %#v", child.Children)
	}
	if len(child.Endpoints) != 2 || child.Endpoints[0].Id != endpointChild || child.Endpoints[1].Id != endpointLeaf {
		t.Fatalf("expected child endpoints plus leaf endpoint, got %#v", child.Endpoints)
	}
}

func (m *mockRedisUtility) SaveValue(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m.SaveValueFunc != nil {
		return m.SaveValueFunc(ctx, key, value, ttl)
	}
	return nil
}

func (m *mockRedisUtility) DeleteValue(ctx context.Context, key string) error {
	if m.DeleteValueFunc != nil {
		return m.DeleteValueFunc(ctx, key)
	}
	return nil
}

func (m *mockRedisUtility) GetValue(ctx context.Context, key string) (string, error) {
	if m.GetValueFunc != nil {
		return m.GetValueFunc(ctx, key)
	}
	return "", nil
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

func TestUserLoginSrvc(t *testing.T) {
	rawPassword := "Password123!"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(rawPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password for test: %v", err)
	}

	mockID := uuid.New()
	mockRoleID := uuid.New()

	validLoginData := models.UserLoginData{
		ID:        mockID,
		Username:  "johndoe",
		Password:  string(hashedPassword),
		RoleID:    mockRoleID,
		RoleName:  "Student",
		FirstName: "John",
		LastName:  "Doe",
	}

	validRequest := models.UserLoginRequest{
		Username: "johndoe",
		Password: rawPassword,
	}

	t.Run("should succeed and return token claims when credentials and redis are valid", func(t *testing.T) {
		repoMock := &mockUserRepo{
			UserLoginFunc: func(user models.UserLoginRequest) (models.UserLoginData, errUtility.CustomError) {
				return validLoginData, errUtility.CustomError{}
			},
		}

		deletedKeys := []string{}
		savedKeys := make(map[string]any)

		redisMock := &mockRedisUtility{
			DeleteValueFunc: func(ctx context.Context, key string) error {
				deletedKeys = append(deletedKeys, key)
				return nil
			},
			SaveValueFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
				savedKeys[key] = value
				return nil
			},
		}

		service := userService{repo: repoMock, redis: redisMock}
		claims, errObj := service.UserLogin(validRequest)

		if errObj.Code > 0 {
			t.Fatalf("expected errObj.Code to be 0, got %d with message: %s", errObj.Code, errObj.Message)
		}
		if claims["username"] != "johndoe" {
			t.Errorf("expected username 'johndoe', got %v", claims["username"])
		}
		if claims["roleName"] != "Student" {
			t.Errorf("expected roleName 'Student', got %v", claims["roleName"])
		}
		if claims["firstName"] != "John" || claims["lastName"] != "Doe" {
			t.Errorf("expected firstName and lastName in login response, got %v and %v", claims["firstName"], claims["lastName"])
		}
		if claims["token"] == nil || claims["token"] == "" {
			t.Errorf("expected token in claims, got nil/empty")
		}

		// Verify redis deletions and save
		if len(deletedKeys) != 2 {
			t.Errorf("expected 2 keys deleted from redis, got %d", len(deletedKeys))
		}
		if _, ok := savedKeys["TOKEN_johndoe_1"]; !ok {
			t.Errorf("expected TOKEN_johndoe_1 to be saved in redis")
		}
	})

	t.Run("should return error when repo UserLogin fails (user not found)", func(t *testing.T) {
		repoMock := &mockUserRepo{
			UserLoginFunc: func(user models.UserLoginRequest) (models.UserLoginData, errUtility.CustomError) {
				return models.UserLoginData{}, errUtility.CustomError{
					Code:          http.StatusBadRequest,
					Message:       "User not found",
					MessageToSend: "Invalid username or password",
				}
			},
		}
		redisMock := &mockRedisUtility{}

		service := userService{repo: repoMock, redis: redisMock}
		claims, errObj := service.UserLogin(validRequest)

		if errObj.Code != http.StatusBadRequest {
			t.Errorf("expected code 400, got %d", errObj.Code)
		}
		if errObj.MessageToSend != "Invalid username or password" {
			t.Errorf("expected 'Invalid username or password', got '%s'", errObj.MessageToSend)
		}
		if len(claims) != 0 {
			t.Errorf("expected empty claims on failure, got %v", claims)
		}
	})

	t.Run("should return error when password does not match", func(t *testing.T) {
		repoMock := &mockUserRepo{
			UserLoginFunc: func(user models.UserLoginRequest) (models.UserLoginData, errUtility.CustomError) {
				return validLoginData, errUtility.CustomError{}
			},
		}
		redisMock := &mockRedisUtility{}

		wrongPassRequest := models.UserLoginRequest{
			Username: "johndoe",
			Password: "WrongPassword!",
		}

		service := userService{repo: repoMock, redis: redisMock}
		claims, errObj := service.UserLogin(wrongPassRequest)

		if errObj.Code != 500 {
			t.Errorf("expected error code 500 on password mismatch, got %d", errObj.Code)
		}
		if errObj.MessageToSend != "Invalid username or password" {
			t.Errorf("expected 'Invalid username or password', got '%s'", errObj.MessageToSend)
		}
		if len(claims) != 0 {
			t.Errorf("expected empty claims on failure, got %v", claims)
		}
	})

	t.Run("should return error when redis DeleteValue slot 1 fails", func(t *testing.T) {
		repoMock := &mockUserRepo{
			UserLoginFunc: func(user models.UserLoginRequest) (models.UserLoginData, errUtility.CustomError) {
				return validLoginData, errUtility.CustomError{}
			},
		}
		redisMock := &mockRedisUtility{
			DeleteValueFunc: func(ctx context.Context, key string) error {
				if key == "TOKEN_johndoe_1" {
					return errors.New("redis connection timeout")
				}
				return nil
			},
		}

		service := userService{repo: repoMock, redis: redisMock}
		_, errObj := service.UserLogin(validRequest)

		if errObj.Code != 500 {
			t.Errorf("expected code 500 on redis delete error, got %d", errObj.Code)
		}
		if errObj.MessageToSend != "Invalid username or password" {
			t.Errorf("expected 'Invalid username or password', got '%s'", errObj.MessageToSend)
		}
	})

	t.Run("should return error when redis DeleteValue slot 2 fails", func(t *testing.T) {
		repoMock := &mockUserRepo{
			UserLoginFunc: func(user models.UserLoginRequest) (models.UserLoginData, errUtility.CustomError) {
				return validLoginData, errUtility.CustomError{}
			},
		}
		redisMock := &mockRedisUtility{
			DeleteValueFunc: func(ctx context.Context, key string) error {
				if key == "TOKEN_johndoe_2" {
					return errors.New("redis error slot 2")
				}
				return nil
			},
		}

		service := userService{repo: repoMock, redis: redisMock}
		_, errObj := service.UserLogin(validRequest)

		if errObj.Code != 500 {
			t.Errorf("expected code 500 on redis delete error, got %d", errObj.Code)
		}
		if errObj.MessageToSend != "Invalid username or password" {
			t.Errorf("expected 'Invalid username or password', got '%s'", errObj.MessageToSend)
		}
	})

	t.Run("should return error when redis SaveValue fails", func(t *testing.T) {
		repoMock := &mockUserRepo{
			UserLoginFunc: func(user models.UserLoginRequest) (models.UserLoginData, errUtility.CustomError) {
				return validLoginData, errUtility.CustomError{}
			},
		}
		redisMock := &mockRedisUtility{
			DeleteValueFunc: func(ctx context.Context, key string) error {
				return nil
			},
			SaveValueFunc: func(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
				return errors.New("redis OOM")
			},
		}

		service := userService{repo: repoMock, redis: redisMock}
		_, errObj := service.UserLogin(validRequest)

		if errObj.Code != 500 {
			t.Errorf("expected code 500 on redis save error, got %d", errObj.Code)
		}
		if errObj.MessageToSend != "Invalid username or password" {
			t.Errorf("expected 'Invalid username or password', got '%s'", errObj.MessageToSend)
		}
	})
}

func TestLogoutSrvc(t *testing.T) {
	ctx := context.Background()
	t.Run("should delete both active session keys", func(t *testing.T) {
		var deletedKeys []string
		service := userService{redis: &mockRedisUtility{DeleteValueFunc: func(ctx context.Context, key string) error {
			deletedKeys = append(deletedKeys, key)
			return nil
		}}}

		errObj := service.LogoutSrvc(ctx, "johndoe")

		if errObj.Code != 0 {
			t.Fatalf("expected no error, got %+v", errObj)
		}
		if len(deletedKeys) != 2 || deletedKeys[0] != "TOKEN_johndoe_1" || deletedKeys[1] != "TOKEN_johndoe_2" {
			t.Errorf("unexpected deleted keys: %v", deletedKeys)
		}
	})

	t.Run("should return an error when Redis deletion fails", func(t *testing.T) {
		service := userService{redis: &mockRedisUtility{DeleteValueFunc: func(ctx context.Context, key string) error {
			return errors.New("redis unavailable")
		}}}

		errObj := service.LogoutSrvc(ctx, "johndoe")

		if errObj.Code != http.StatusInternalServerError || errObj.MessageToSend != "Failed to logout" {
			t.Errorf("unexpected error: %+v", errObj)
		}
	})
}
