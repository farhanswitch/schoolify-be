package apis

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type mockApiService struct {
	GetAllSrvcFunc    func(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError)
	CreateApiSrvcFunc func(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError
	UpdateApiSrvcFunc func(ctx context.Context, apiID uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError
	DeleteApiSrvcFunc func(ctx context.Context, apiID uuid.UUID) errUtility.CustomError
}

func (m *mockApiService) GetAllSrvc(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
	if m.GetAllSrvcFunc != nil {
		return m.GetAllSrvcFunc(ctx, param)
	}
	return []models.ApiResponse{}, 0, errUtility.CustomError{}
}

func (m *mockApiService) CreateApiSrvc(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError {
	if m.CreateApiSrvcFunc != nil {
		return m.CreateApiSrvcFunc(ctx, param)
	}
	return errUtility.CustomError{}
}

func (m *mockApiService) UpdateApiSrvc(ctx context.Context, apiID uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError {
	if m.UpdateApiSrvcFunc != nil {
		return m.UpdateApiSrvcFunc(ctx, apiID, param)
	}
	return errUtility.CustomError{}
}

func (m *mockApiService) DeleteApiSrvc(ctx context.Context, apiID uuid.UUID) errUtility.CustomError {
	if m.DeleteApiSrvcFunc != nil {
		return m.DeleteApiSrvcFunc(ctx, apiID)
	}
	return errUtility.CustomError{}
}

func updateRequest(apiID, body string) *http.Request {
	request := httptest.NewRequest(http.MethodPut, "/api/endpoint/"+apiID, strings.NewReader(body))
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("apiID", apiID)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func deleteRequest(apiID string) *http.Request {
	request := httptest.NewRequest(http.MethodDelete, "/api/endpoint/"+apiID, nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("apiID", apiID)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func TestDeleteCtrl(t *testing.T) {
	apiID := uuid.New()

	t.Run("should return 200 when API is deleted", func(t *testing.T) {
		var capturedID uuid.UUID
		ctrl := apiController{service: &mockApiService{
			DeleteApiSrvcFunc: func(ctx context.Context, id uuid.UUID) errUtility.CustomError {
				capturedID = id
				return errUtility.CustomError{}
			},
		}}
		writer := httptest.NewRecorder()

		ctrl.deleteCtrl(writer, deleteRequest(apiID.String()))

		if writer.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", writer.Code)
		}
		if capturedID != apiID {
			t.Errorf("expected apiID %s, got %s", apiID, capturedID)
		}
		if !strings.Contains(writer.Body.String(), "API deleted successfully") {
			t.Errorf("expected success message, got %s", writer.Body.String())
		}
	})

	t.Run("should return 400 when apiID is missing", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{}}
		writer := httptest.NewRecorder()

		ctrl.deleteCtrl(writer, deleteRequest(""))

		if writer.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", writer.Code)
		}
	})

	t.Run("should return 400 when apiID is invalid", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{}}
		writer := httptest.NewRecorder()

		ctrl.deleteCtrl(writer, deleteRequest("invalid-id"))

		if writer.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", writer.Code)
		}
		if !strings.Contains(writer.Body.String(), "Invalid apiID") {
			t.Errorf("expected invalid apiID error, got %s", writer.Body.String())
		}
	})

	t.Run("should return 404 when API does not exist", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{
			DeleteApiSrvcFunc: func(ctx context.Context, id uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{Code: http.StatusNotFound, MessageToSend: "API not found", Property: "apiID"}
			},
		}}
		writer := httptest.NewRecorder()

		ctrl.deleteCtrl(writer, deleteRequest(apiID.String()))

		if writer.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", writer.Code)
		}
		if !strings.Contains(writer.Body.String(), "API not found") {
			t.Errorf("expected not-found error, got %s", writer.Body.String())
		}
	})

	t.Run("should return 500 when service fails", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{
			DeleteApiSrvcFunc: func(ctx context.Context, id uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{Code: http.StatusInternalServerError, Message: "database unavailable"}
			},
		}}
		writer := httptest.NewRecorder()

		ctrl.deleteCtrl(writer, deleteRequest(apiID.String()))

		if writer.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", writer.Code)
		}
		if !strings.Contains(writer.Body.String(), "Internal Server Error") {
			t.Errorf("expected generic internal error, got %s", writer.Body.String())
		}
	})
}

func TestUpdateCtrl(t *testing.T) {
	apiID := uuid.New()
	validBody := `{"name":"Update user","description":"Updates an existing user account","path":"/api/users/{id}","method":"PUT"}`

	t.Run("should return 200 when API is updated", func(t *testing.T) {
		var capturedID uuid.UUID
		var capturedRequest models.UpdateApiRequest
		ctrl := apiController{service: &mockApiService{
			UpdateApiSrvcFunc: func(ctx context.Context, id uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError {
				capturedID = id
				capturedRequest = param
				return errUtility.CustomError{}
			},
		}}
		writer := httptest.NewRecorder()

		ctrl.updateCtrl(writer, updateRequest(apiID.String(), validBody))

		if writer.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", writer.Code)
		}
		if capturedID != apiID || capturedRequest.Name != "Update user" || capturedRequest.Method != "PUT" {
			t.Errorf("unexpected service parameters: id=%s request=%+v", capturedID, capturedRequest)
		}
		if !strings.Contains(writer.Body.String(), "API updated successfully") {
			t.Errorf("expected success message, got %s", writer.Body.String())
		}
	})

	t.Run("should return 400 when apiID is missing", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{}}
		writer := httptest.NewRecorder()

		ctrl.updateCtrl(writer, updateRequest("", validBody))

		if writer.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", writer.Code)
		}
	})

	t.Run("should return 400 when apiID is invalid", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{}}
		writer := httptest.NewRecorder()

		ctrl.updateCtrl(writer, updateRequest("invalid-id", validBody))

		if writer.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", writer.Code)
		}
		if !strings.Contains(writer.Body.String(), "Invalid apiID") {
			t.Errorf("expected invalid apiID error, got %s", writer.Body.String())
		}
	})

	t.Run("should return 400 for invalid JSON", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{}}
		writer := httptest.NewRecorder()

		ctrl.updateCtrl(writer, updateRequest(apiID.String(), `{"name":`))

		if writer.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", writer.Code)
		}
	})

	t.Run("should return 400 for an invalid request body", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{}}
		writer := httptest.NewRecorder()

		ctrl.updateCtrl(writer, updateRequest(apiID.String(), `{"name":"x","description":"test","path":"","method":"TRACE"}`))

		if writer.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", writer.Code)
		}
	})

	t.Run("should return 404 when API does not exist", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{
			UpdateApiSrvcFunc: func(ctx context.Context, id uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError {
				return errUtility.CustomError{Code: http.StatusNotFound, MessageToSend: "API not found", Property: "apiID"}
			},
		}}
		writer := httptest.NewRecorder()

		ctrl.updateCtrl(writer, updateRequest(apiID.String(), validBody))

		if writer.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", writer.Code)
		}
		if !strings.Contains(writer.Body.String(), "API not found") {
			t.Errorf("expected not-found error, got %s", writer.Body.String())
		}
	})

	t.Run("should return 500 when service fails", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{
			UpdateApiSrvcFunc: func(ctx context.Context, id uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError {
				return errUtility.CustomError{Code: http.StatusInternalServerError, Message: "database unavailable"}
			},
		}}
		writer := httptest.NewRecorder()

		ctrl.updateCtrl(writer, updateRequest(apiID.String(), validBody))

		if writer.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", writer.Code)
		}
		if !strings.Contains(writer.Body.String(), "Internal Server Error") {
			t.Errorf("expected generic internal error, got %s", writer.Body.String())
		}
	})
}

func TestCreateCtrl(t *testing.T) {
	validBody := `{"name":"Create user","description":"Creates a new user account","path":"/api/users","method":"POST"}`

	t.Run("should return 201 when API is created", func(t *testing.T) {
		var captured models.CreateApiRequest
		ctrl := apiController{service: &mockApiService{
			CreateApiSrvcFunc: func(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError {
				captured = param
				return errUtility.CustomError{}
			},
		}}
		request := httptest.NewRequest(http.MethodPost, "/api/endpoint", strings.NewReader(validBody))
		writer := httptest.NewRecorder()

		ctrl.createCtrl(writer, request)

		if writer.Code != http.StatusCreated {
			t.Fatalf("expected status 201, got %d", writer.Code)
		}
		if captured.Name != "Create user" || captured.Path != "/api/users" || captured.Method != "POST" {
			t.Errorf("unexpected request passed to service: %+v", captured)
		}
		if !strings.Contains(writer.Body.String(), "API created successfully") {
			t.Errorf("expected success message, got %s", writer.Body.String())
		}
	})

	t.Run("should return 400 for invalid JSON", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{}}
		request := httptest.NewRequest(http.MethodPost, "/api/endpoint", strings.NewReader(`{"name":`))
		writer := httptest.NewRecorder()

		ctrl.createCtrl(writer, request)

		if writer.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", writer.Code)
		}
	})

	t.Run("should return 422 for invalid request", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{}}
		request := httptest.NewRequest(http.MethodPost, "/api/endpoint", strings.NewReader(`{"name":"x","description":"test","path":"/api/users","method":"TRACE"}`))
		writer := httptest.NewRecorder()

		ctrl.createCtrl(writer, request)

		if writer.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status 422, got %d", writer.Code)
		}
	})

	t.Run("should return service client error", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{
			CreateApiSrvcFunc: func(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError {
				return errUtility.CustomError{Code: http.StatusConflict, MessageToSend: "API already exists", Property: "id"}
			},
		}}
		request := httptest.NewRequest(http.MethodPost, "/api/endpoint", strings.NewReader(validBody))
		writer := httptest.NewRecorder()

		ctrl.createCtrl(writer, request)

		if writer.Code != http.StatusConflict {
			t.Errorf("expected status 409, got %d", writer.Code)
		}
		if !strings.Contains(writer.Body.String(), "API already exists") {
			t.Errorf("expected conflict response, got %s", writer.Body.String())
		}
	})
}

func TestGetAllCtrl(t *testing.T) {
	t.Run("should return 200 OK with listData and count on success", func(t *testing.T) {
		id1 := uuid.New()
		mockList := []models.ApiResponse{
			{
				Id:          id1,
				Name:        "Get Users",
				Description: "Retrieve list of users",
				Path:        "/users",
				Method:      "GET",
				CreatedAt:   "2026-10-06 12:00:00",
				UpdatedAt:   "",
			},
		}

		mockSrv := &mockApiService{
			GetAllSrvcFunc: func(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
				return mockList, 1, errUtility.CustomError{}
			},
		}

		ctrl := apiController{service: mockSrv}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/apis?paginationPage=1&paginationRows=10&sortField=name&sortOrder=ASC&search=Users", nil)
		w := httptest.NewRecorder()

		ctrl.getAllCtrl(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		respBody := w.Body.String()
		if !strings.Contains(respBody, "Success get list api") {
			t.Errorf("expected response to contain 'Success get list api', got %s", respBody)
		}
		if !strings.Contains(respBody, "Get Users") {
			t.Errorf("expected response to contain 'Get Users', got %s", respBody)
		}
		if !strings.Contains(respBody, `"count":1`) {
			t.Errorf("expected response to contain '\"count\":1', got %s", respBody)
		}
	})

	t.Run("should return 400 Bad Request when validation fails", func(t *testing.T) {
		ctrl := apiController{service: &mockApiService{}}
		// Invalid paginationPage=0 (fails min=1 validator)
		req := httptest.NewRequest(http.MethodGet, "/api/v1/apis?paginationPage=0&paginationRows=10", nil)
		w := httptest.NewRecorder()

		ctrl.getAllCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		respBody := w.Body.String()
		if !strings.Contains(respBody, "Validation failed") {
			t.Errorf("expected response to contain 'Validation failed', got %s", respBody)
		}
	})

	t.Run("should return 500 when service fails with internal error", func(t *testing.T) {
		mockSrv := &mockApiService{
			GetAllSrvcFunc: func(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
				return nil, 0, errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "database connection lost",
					MessageToSend: "Failed to get list api",
				}
			},
		}

		ctrl := apiController{service: mockSrv}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/apis?paginationPage=1&paginationRows=10", nil)
		w := httptest.NewRecorder()

		ctrl.getAllCtrl(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		respBody := w.Body.String()
		if !strings.Contains(respBody, "Internal Server Error") {
			t.Errorf("expected response to contain 'Internal Server Error', got %s", respBody)
		}
	})

	t.Run("should return 400 when service returns client error", func(t *testing.T) {
		mockSrv := &mockApiService{
			GetAllSrvcFunc: func(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
				return nil, 0, errUtility.CustomError{
					Code:          http.StatusBadRequest,
					Message:       "invalid parameter",
					MessageToSend: "Invalid request parameter",
					Property:      "param",
				}
			},
		}

		ctrl := apiController{service: mockSrv}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/apis?paginationPage=1&paginationRows=10", nil)
		w := httptest.NewRecorder()

		ctrl.getAllCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		respBody := w.Body.String()
		if !strings.Contains(respBody, "Invalid request parameter") {
			t.Errorf("expected response to contain 'Invalid request parameter', got %s", respBody)
		}
	})

	t.Run("should use default values for sort and pagination when omitted", func(t *testing.T) {
		var capturedParam models.GetAllApiRequest
		mockSrv := &mockApiService{
			GetAllSrvcFunc: func(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
				capturedParam = param
				return []models.ApiResponse{}, 0, errUtility.CustomError{}
			},
		}

		ctrl := apiController{service: mockSrv}
		req := httptest.NewRequest(http.MethodGet, "/api/v1/apis", nil)
		w := httptest.NewRecorder()

		ctrl.getAllCtrl(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if capturedParam.Page != 1 {
			t.Errorf("expected default Page=1, got %d", capturedParam.Page)
		}
		if capturedParam.Rows != 10 {
			t.Errorf("expected default Rows=10, got %d", capturedParam.Rows)
		}
		if capturedParam.SortField != "created_at" {
			t.Errorf("expected default SortField='created_at', got %s", capturedParam.SortField)
		}
		if capturedParam.SortOrder != "DESC" {
			t.Errorf("expected default SortOrder='DESC', got %s", capturedParam.SortOrder)
		}
	})
}
