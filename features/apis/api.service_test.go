package apis

import (
	"context"
	"net/http"
	"testing"

	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"

	"github.com/google/uuid"
)

type mockApiPostgresRepo struct {
	GetAllFunc func(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError)
	CreateFunc func(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError
	UpdateFunc func(ctx context.Context, apiID uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError
	DeleteFunc func(ctx context.Context, apiID uuid.UUID) errUtility.CustomError
}

func (m *mockApiPostgresRepo) GetAll(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(ctx, param)
	}
	return []models.ApiResponse{}, 0, errUtility.CustomError{}
}

func (m *mockApiPostgresRepo) Create(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, param)
	}
	return errUtility.CustomError{}
}

func (m *mockApiPostgresRepo) Update(ctx context.Context, apiID uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, apiID, param)
	}
	return errUtility.CustomError{}
}

func (m *mockApiPostgresRepo) Delete(ctx context.Context, apiID uuid.UUID) errUtility.CustomError {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, apiID)
	}
	return errUtility.CustomError{}
}

func TestDeleteApiSrvc(t *testing.T) {
	ctx := context.Background()
	apiID := uuid.New()

	t.Run("should pass ID to repository", func(t *testing.T) {
		var capturedID uuid.UUID
		srv := apiService{repo: &mockApiPostgresRepo{
			DeleteFunc: func(ctx context.Context, id uuid.UUID) errUtility.CustomError {
				capturedID = id
				return errUtility.CustomError{}
			},
		}}

		errObj := srv.DeleteApiSrvc(ctx, apiID)

		if errObj.Code != 0 {
			t.Fatalf("expected no error, got %+v", errObj)
		}
		if capturedID != apiID {
			t.Errorf("expected apiID %s, got %s", apiID, capturedID)
		}
	})

	t.Run("should return repository not-found error", func(t *testing.T) {
		srv := apiService{repo: &mockApiPostgresRepo{
			DeleteFunc: func(ctx context.Context, id uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{Code: http.StatusNotFound, MessageToSend: "API not found", Property: "apiID"}
			},
		}}

		errObj := srv.DeleteApiSrvc(ctx, apiID)
		if errObj.Code != http.StatusNotFound || errObj.Property != "apiID" {
			t.Errorf("expected not-found error for apiID, got %+v", errObj)
		}
	})

	t.Run("should return repository internal error", func(t *testing.T) {
		srv := apiService{repo: &mockApiPostgresRepo{
			DeleteFunc: func(ctx context.Context, id uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{Code: http.StatusInternalServerError, MessageToSend: "Failed to delete API"}
			},
		}}

		errObj := srv.DeleteApiSrvc(ctx, apiID)
		if errObj.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", errObj.Code)
		}
	})
}

func TestUpdateApiSrvc(t *testing.T) {
	ctx := context.Background()
	apiID := uuid.New()
	req := models.UpdateApiRequest{
		Name:        "Update user",
		Description: "Updates an existing user account",
		Path:        "/api/users/{id}",
		Method:      "PUT",
	}

	t.Run("should pass ID and request to repository", func(t *testing.T) {
		var capturedID uuid.UUID
		var capturedRequest models.UpdateApiRequest
		srv := apiService{repo: &mockApiPostgresRepo{
			UpdateFunc: func(ctx context.Context, id uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError {
				capturedID = id
				capturedRequest = param
				return errUtility.CustomError{}
			},
		}}

		errObj := srv.UpdateApiSrvc(ctx, apiID, req)

		if errObj.Code != 0 {
			t.Fatalf("expected no error, got %+v", errObj)
		}
		if capturedID != apiID || capturedRequest != req {
			t.Errorf("unexpected repository parameters: id=%s request=%+v", capturedID, capturedRequest)
		}
	})

	t.Run("should return repository not-found error", func(t *testing.T) {
		srv := apiService{repo: &mockApiPostgresRepo{
			UpdateFunc: func(ctx context.Context, id uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError {
				return errUtility.CustomError{Code: http.StatusNotFound, MessageToSend: "API not found", Property: "apiID"}
			},
		}}

		errObj := srv.UpdateApiSrvc(ctx, apiID, req)
		if errObj.Code != http.StatusNotFound || errObj.Property != "apiID" {
			t.Errorf("expected not-found error for apiID, got %+v", errObj)
		}
	})

	t.Run("should return repository internal error", func(t *testing.T) {
		srv := apiService{repo: &mockApiPostgresRepo{
			UpdateFunc: func(ctx context.Context, id uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError {
				return errUtility.CustomError{Code: http.StatusInternalServerError, MessageToSend: "Failed to update API"}
			},
		}}

		errObj := srv.UpdateApiSrvc(ctx, apiID, req)
		if errObj.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", errObj.Code)
		}
	})
}

func TestCreateApiSrvc(t *testing.T) {
	ctx := context.Background()
	req := models.CreateApiRequest{
		Name:        "Create user",
		Description: "Creates a new user account",
		Path:        "/api/users",
		Method:      "POST",
	}

	t.Run("should generate UUIDv7 and pass request to repository", func(t *testing.T) {
		var captured models.CreateApiRequest
		srv := apiService{repo: &mockApiPostgresRepo{
			CreateFunc: func(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError {
				captured = param
				return errUtility.CustomError{}
			},
		}}

		errObj := srv.CreateApiSrvc(ctx, req)

		if errObj.Code != 0 {
			t.Fatalf("expected no error, got %+v", errObj)
		}
		if captured.Id == uuid.Nil || captured.Id.Version() != 7 {
			t.Errorf("expected generated UUIDv7, got %s", captured.Id)
		}
		if captured.Name != req.Name || captured.Path != req.Path || captured.Method != req.Method {
			t.Errorf("unexpected request passed to repository: %+v", captured)
		}
	})

	t.Run("should preserve a supplied ID", func(t *testing.T) {
		suppliedID := uuid.New()
		var captured models.CreateApiRequest
		srv := apiService{repo: &mockApiPostgresRepo{
			CreateFunc: func(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError {
				captured = param
				return errUtility.CustomError{}
			},
		}}

		errObj := srv.CreateApiSrvc(ctx, models.CreateApiRequest{Id: suppliedID, Name: req.Name, Description: req.Description, Path: req.Path, Method: req.Method})

		if errObj.Code != 0 || captured.Id != suppliedID {
			t.Errorf("expected supplied ID %s to be preserved, got %s", suppliedID, captured.Id)
		}
	})

	t.Run("should return repository error", func(t *testing.T) {
		srv := apiService{repo: &mockApiPostgresRepo{
			CreateFunc: func(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError {
				return errUtility.CustomError{Code: http.StatusConflict, MessageToSend: "API already exists"}
			},
		}}

		errObj := srv.CreateApiSrvc(ctx, req)
		if errObj.Code != http.StatusConflict {
			t.Errorf("expected status 409, got %d", errObj.Code)
		}
	})
}

func TestGetAllSrvc(t *testing.T) {
	ctx := context.Background()

	t.Run("should return api list and count successfully when repo succeeds", func(t *testing.T) {
		id1 := uuid.New()
		id2 := uuid.New()

		mockApis := []models.ApiResponse{
			{
				Id:          id1,
				Name:        "Get User List",
				Description: "API for retrieving user list",
				Path:        "/users",
				Method:      "GET",
				CreatedAt:   "2026-10-06 10:00:00",
			},
			{
				Id:          id2,
				Name:        "Create User",
				Description: "API for creating user",
				Path:        "/users",
				Method:      "POST",
				CreatedAt:   "2026-10-06 10:05:00",
			},
		}

		mockRepo := &mockApiPostgresRepo{
			GetAllFunc: func(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
				return mockApis, len(mockApis), errUtility.CustomError{}
			},
		}

		srv := apiService{repo: mockRepo}
		param := models.GetAllApiRequest{
			Page:      1,
			Rows:      10,
			SortField: "created_at",
			SortOrder: "DESC",
			Search:    "",
		}

		list, count, customErr := srv.GetAllSrvc(ctx, param)

		if customErr.Code != 0 {
			t.Fatalf("expected no error, got code %d, message: %s", customErr.Code, customErr.Message)
		}
		if count != 2 {
			t.Errorf("expected count 2, got %d", count)
		}
		if len(list) != 2 {
			t.Fatalf("expected 2 items, got %d", len(list))
		}
		if list[0].Name != "Get User List" {
			t.Errorf("expected list[0].Name to be 'Get User List', got '%s'", list[0].Name)
		}
		if list[1].Name != "Create User" {
			t.Errorf("expected list[1].Name to be 'Create User', got '%s'", list[1].Name)
		}
	})

	t.Run("should pass request parameters accurately to repository", func(t *testing.T) {
		var capturedParam models.GetAllApiRequest

		mockRepo := &mockApiPostgresRepo{
			GetAllFunc: func(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
				capturedParam = param
				return []models.ApiResponse{}, 0, errUtility.CustomError{}
			},
		}

		srv := apiService{repo: mockRepo}
		req := models.GetAllApiRequest{
			Page:      3,
			Rows:      25,
			SortField: "name",
			SortOrder: "ASC",
			Search:    "profile",
		}

		_, _, _ = srv.GetAllSrvc(ctx, req)

		if capturedParam.Page != req.Page {
			t.Errorf("expected Page %d, got %d", req.Page, capturedParam.Page)
		}
		if capturedParam.Rows != req.Rows {
			t.Errorf("expected Rows %d, got %d", req.Rows, capturedParam.Rows)
		}
		if capturedParam.SortField != req.SortField {
			t.Errorf("expected SortField %s, got %s", req.SortField, capturedParam.SortField)
		}
		if capturedParam.SortOrder != req.SortOrder {
			t.Errorf("expected SortOrder %s, got %s", req.SortOrder, capturedParam.SortOrder)
		}
		if capturedParam.Search != req.Search {
			t.Errorf("expected Search %s, got %s", req.Search, capturedParam.Search)
		}
	})

	t.Run("should return error when repository fails", func(t *testing.T) {
		mockRepo := &mockApiPostgresRepo{
			GetAllFunc: func(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
				return nil, 0, errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "connection refused",
					MessageToSend: "Failed get list api",
					Function:      "ApiPostgresRepo.GetAll",
				}
			},
		}

		srv := apiService{repo: mockRepo}
		param := models.GetAllApiRequest{
			Page: 1,
			Rows: 10,
		}

		list, count, customErr := srv.GetAllSrvc(ctx, param)

		if customErr.Code != http.StatusInternalServerError {
			t.Errorf("expected error code %d, got %d", http.StatusInternalServerError, customErr.Code)
		}
		if customErr.Message != "connection refused" {
			t.Errorf("expected message 'connection refused', got '%s'", customErr.Message)
		}
		if list != nil {
			t.Errorf("expected nil list on error, got %v", list)
		}
		if count != 0 {
			t.Errorf("expected count 0 on error, got %d", count)
		}
	})

	t.Run("should return empty list and zero count when repo has no data", func(t *testing.T) {
		mockRepo := &mockApiPostgresRepo{
			GetAllFunc: func(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
				return []models.ApiResponse{}, 0, errUtility.CustomError{}
			},
		}

		srv := apiService{repo: mockRepo}
		param := models.GetAllApiRequest{
			Page:   1,
			Rows:   10,
			Search: "nonexistent",
		}

		list, count, customErr := srv.GetAllSrvc(ctx, param)

		if customErr.Code != 0 {
			t.Fatalf("expected no error, got code %d", customErr.Code)
		}
		if count != 0 {
			t.Errorf("expected count 0, got %d", count)
		}
		if len(list) != 0 {
			t.Errorf("expected empty list, got length %d", len(list))
		}
	})
}

func TestFactoryApiService(t *testing.T) {
	t.Run("should create and return apiService instance with repo", func(t *testing.T) {
		mockRepo := &mockApiPostgresRepo{}
		srv := factoryApiService(mockRepo)

		if srv.repo == nil {
			t.Fatal("expected srv.repo to not be nil")
		}
	})
}
