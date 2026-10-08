package menus

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type mockMenuService struct {
	GetAllMenusSrvcFunc          func(ctx context.Context) ([]*models.MenuResponse, errUtility.CustomError)
	CreateMenuSrvcFunc           func(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError
	CreateMenuWithOrderSrvcFunc  func(ctx context.Context, param models.CreateMenuRequest) (models.MenuSortOrderInfo, errUtility.CustomError)
	GetByIDSrvcFunc              func(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError)
	UpdateMenuSrvcFunc           func(ctx context.Context, param models.UpdateMenuRequest) errUtility.CustomError
	UpdateMenuWithOrderSrvcFunc  func(ctx context.Context, param models.UpdateMenuRequest) (models.MenuSortOrderInfo, errUtility.CustomError)
	AssignApiSrvcFunc            func(ctx context.Context, menuID uuid.UUID, param models.AssignApiRequest) errUtility.CustomError
	GetAssignedEndpointsSrvcFunc func(ctx context.Context, menuID uuid.UUID) ([]models.MenuApiResponse, errUtility.CustomError)
	DeleteSrvcFunc               func(ctx context.Context, menuID uuid.UUID) errUtility.CustomError
}

func (m *mockMenuService) GetAllMenusSrvc(ctx context.Context) ([]*models.MenuResponse, errUtility.CustomError) {
	if m.GetAllMenusSrvcFunc != nil {
		return m.GetAllMenusSrvcFunc(ctx)
	}
	return []*models.MenuResponse{}, errUtility.CustomError{}
}

func (m *mockMenuService) CreateMenuSrvc(ctx context.Context, param models.CreateMenuRequest) (models.MenuSortOrderInfo, errUtility.CustomError) {
	if m.CreateMenuWithOrderSrvcFunc != nil {
		return m.CreateMenuWithOrderSrvcFunc(ctx, param)
	}
	if m.CreateMenuSrvcFunc != nil {
		return models.MenuSortOrderInfo{}, m.CreateMenuSrvcFunc(ctx, param)
	}
	return models.MenuSortOrderInfo{}, errUtility.CustomError{}
}

func (m *mockMenuService) GetByIDSrvc(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError) {
	if m.GetByIDSrvcFunc != nil {
		return m.GetByIDSrvcFunc(ctx, menuID)
	}
	return models.MenuResponse{}, errUtility.CustomError{}
}

func (m *mockMenuService) UpdateMenuSrvc(ctx context.Context, param models.UpdateMenuRequest) (models.MenuSortOrderInfo, errUtility.CustomError) {
	if m.UpdateMenuWithOrderSrvcFunc != nil {
		return m.UpdateMenuWithOrderSrvcFunc(ctx, param)
	}
	if m.UpdateMenuSrvcFunc != nil {
		return models.MenuSortOrderInfo{}, m.UpdateMenuSrvcFunc(ctx, param)
	}
	return models.MenuSortOrderInfo{}, errUtility.CustomError{}
}

func (m *mockMenuService) AssignApiSrvc(ctx context.Context, menuID uuid.UUID, param models.AssignApiRequest) errUtility.CustomError {
	if m.AssignApiSrvcFunc != nil {
		return m.AssignApiSrvcFunc(ctx, menuID, param)
	}
	return errUtility.CustomError{}
}

func (m *mockMenuService) GetAssignedEndpointsSrvc(ctx context.Context, menuID uuid.UUID) ([]models.MenuApiResponse, errUtility.CustomError) {
	if m.GetAssignedEndpointsSrvcFunc != nil {
		return m.GetAssignedEndpointsSrvcFunc(ctx, menuID)
	}
	return []models.MenuApiResponse{}, errUtility.CustomError{}
}

func (m *mockMenuService) DeleteSrvc(ctx context.Context, menuID uuid.UUID) errUtility.CustomError {
	if m.DeleteSrvcFunc != nil {
		return m.DeleteSrvcFunc(ctx, menuID)
	}
	return errUtility.CustomError{}
}

func TestAssignApiCtrl(t *testing.T) {
	menuID := uuid.New()
	apiID := uuid.New()
	newRequest := func(menuIDParam, body string) *http.Request {
		req := httptest.NewRequest(http.MethodPut, "/api/menu/assign-api/"+menuIDParam, strings.NewReader(body))
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("menuID", menuIDParam)
		return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	t.Run("should assign multiple APIs successfully", func(t *testing.T) {
		secondAPIID := uuid.New()
		var capturedMenuID uuid.UUID
		var capturedParam models.AssignApiRequest
		ctrl := menuController{service: &mockMenuService{AssignApiSrvcFunc: func(ctx context.Context, id uuid.UUID, param models.AssignApiRequest) errUtility.CustomError {
			capturedMenuID, capturedParam = id, param
			return errUtility.CustomError{}
		}}}
		w := httptest.NewRecorder()
		ctrl.assignApiCtrl(w, newRequest(menuID.String(), `{"listID":["`+apiID.String()+`","`+secondAPIID.String()+`"]}`))
		if w.Code != http.StatusCreated || capturedMenuID != menuID || len(capturedParam.ApiIDs) != 2 {
			t.Errorf("unexpected result: status=%d menuID=%s param=%+v", w.Code, capturedMenuID, capturedParam)
		}
	})

	t.Run("should reject missing or invalid menuID", func(t *testing.T) {
		for _, id := range []string{"", "invalid-id"} {
			ctrl := menuController{service: &mockMenuService{}}
			w := httptest.NewRecorder()
			ctrl.assignApiCtrl(w, newRequest(id, `{"listID":["`+apiID.String()+`"]}`))
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400 for %q, got %d", id, w.Code)
			}
		}
	})

	t.Run("should reject invalid JSON and invalid payload", func(t *testing.T) {
		for _, body := range []string{`{"listID":`, `{}`} {
			ctrl := menuController{service: &mockMenuService{}}
			w := httptest.NewRecorder()
			ctrl.assignApiCtrl(w, newRequest(menuID.String(), body))
			if body == `{}` && w.Code != http.StatusUnprocessableEntity || body != `{}` && w.Code != http.StatusBadRequest {
				t.Errorf("unexpected status %d for body %s", w.Code, body)
			}
		}
	})

	t.Run("should return service not-found and internal errors", func(t *testing.T) {
		for _, errObj := range []errUtility.CustomError{
			{Code: http.StatusNotFound, MessageToSend: "Menu not found", Property: "menuID"},
			{Code: http.StatusInternalServerError, Message: "database unavailable"},
		} {
			ctrl := menuController{service: &mockMenuService{AssignApiSrvcFunc: func(ctx context.Context, id uuid.UUID, param models.AssignApiRequest) errUtility.CustomError {
				return errObj
			}}}
			w := httptest.NewRecorder()
			ctrl.assignApiCtrl(w, newRequest(menuID.String(), `{"listID":["`+apiID.String()+`"]}`))
			if w.Code != int(errObj.Code) {
				t.Errorf("expected status %d, got %d", errObj.Code, w.Code)
			}
		}
	})
}

func TestListAssignedEndpointCtrl(t *testing.T) {
	menuID := uuid.New()
	newRequest := func(value string) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/api/menu/list-assigned-endpoint/"+value, nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("menuID", value)
		return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}
	t.Run("should return assigned endpoint statuses", func(t *testing.T) {
		ctrl := menuController{service: &mockMenuService{GetAssignedEndpointsSrvcFunc: func(ctx context.Context, id uuid.UUID) ([]models.MenuApiResponse, errUtility.CustomError) {
			return []models.MenuApiResponse{{Id: uuid.New(), Name: "List users", Path: "/api/users", Method: "GET", Status: 1}}, errUtility.CustomError{}
		}}}
		w := httptest.NewRecorder()
		ctrl.listAssignedEndpointCtrl(w, newRequest(menuID.String()))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"status":1`) {
			t.Errorf("unexpected response: status=%d body=%s", w.Code, w.Body.String())
		}
	})
	t.Run("should reject missing and invalid menuID", func(t *testing.T) {
		for _, id := range []string{"", "invalid-id"} {
			w := httptest.NewRecorder()
			menuController{service: &mockMenuService{}}.listAssignedEndpointCtrl(w, newRequest(id))
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400, got %d", w.Code)
			}
		}
	})
	t.Run("should return service errors", func(t *testing.T) {
		for _, errObj := range []errUtility.CustomError{{Code: http.StatusNotFound, MessageToSend: "Menu not found", Property: "menuID"}, {Code: http.StatusInternalServerError, Message: "db failure"}} {
			ctrl := menuController{service: &mockMenuService{GetAssignedEndpointsSrvcFunc: func(ctx context.Context, id uuid.UUID) ([]models.MenuApiResponse, errUtility.CustomError) {
				return nil, errObj
			}}}
			w := httptest.NewRecorder()
			ctrl.listAssignedEndpointCtrl(w, newRequest(menuID.String()))
			if w.Code != int(errObj.Code) {
				t.Errorf("expected %d, got %d", errObj.Code, w.Code)
			}
		}
	})
}

func TestGetAllMenusCtrl(t *testing.T) {
	t.Run("should return 500 when service fails", func(t *testing.T) {
		mockSrv := &mockMenuService{
			GetAllMenusSrvcFunc: func(ctx context.Context) ([]*models.MenuResponse, errUtility.CustomError) {
				return nil, errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "db failure",
					MessageToSend: "Failed get menus",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := httptest.NewRequest(http.MethodGet, "/api/menu/list", nil)
		w := httptest.NewRecorder()

		ctrl.getAllMenusCtrl(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Failed get menus") {
			t.Errorf("expected response to contain 'Failed get menus', got %s", w.Body.String())
		}
	})

	t.Run("should return 200 with menu tree on success", func(t *testing.T) {
		mockTree := []*models.MenuResponse{
			{
				Id:          uuid.New().String(),
				Name:        "Dashboard",
				Description: "Dashboard menu",
				Children:    []*models.MenuResponse{},
			},
		}
		mockSrv := &mockMenuService{
			GetAllMenusSrvcFunc: func(ctx context.Context) ([]*models.MenuResponse, errUtility.CustomError) {
				return mockTree, errUtility.CustomError{}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := httptest.NewRequest(http.MethodGet, "/api/menu/list", nil)
		w := httptest.NewRecorder()

		ctrl.getAllMenusCtrl(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Menu list") {
			t.Errorf("expected response to contain 'Menu list', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "Dashboard") {
			t.Errorf("expected response to contain 'Dashboard', got %s", w.Body.String())
		}
	})
}

func TestCreateCtrl(t *testing.T) {
	defaultPayload := map[string]interface{}{
		"name":        "Dashboard",
		"description": "Main dashboard menu",
	}

	buildJSONBody := func(payload interface{}) *bytes.Buffer {
		data, _ := json.Marshal(payload)
		return bytes.NewBuffer(data)
	}

	t.Run("should return 400 when JSON body is invalid", func(t *testing.T) {
		ctrl := menuController{service: &mockMenuService{}}
		req := httptest.NewRequest(http.MethodPost, "/api/menu", strings.NewReader("{invalid-json"))
		w := httptest.NewRecorder()

		ctrl.createCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid data") {
			t.Errorf("expected response to contain 'Invalid data', got %s", w.Body.String())
		}
	})

	t.Run("should return 422 when validation fails (missing required name)", func(t *testing.T) {
		ctrl := menuController{service: &mockMenuService{}}
		req := httptest.NewRequest(http.MethodPost, "/api/menu", buildJSONBody(map[string]interface{}{
			"description": "missing name",
		}))
		w := httptest.NewRecorder()

		ctrl.createCtrl(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status 422, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Validation failed") {
			t.Errorf("expected response to contain 'Validation failed', got %s", w.Body.String())
		}
	})

	t.Run("should reject non-positive sort order", func(t *testing.T) {
		for _, sortOrder := range []int{0, -1} {
			ctrl := menuController{service: &mockMenuService{}}
			req := httptest.NewRequest(http.MethodPost, "/api/menu", buildJSONBody(map[string]interface{}{
				"name":        "Dashboard",
				"description": "Main dashboard menu",
				"sortOrder":   sortOrder,
			}))
			w := httptest.NewRecorder()

			ctrl.createCtrl(w, req)

			if w.Code != http.StatusUnprocessableEntity {
				t.Errorf("expected status 422 for sortOrder %d, got %d", sortOrder, w.Code)
			}
			if !strings.Contains(w.Body.String(), "SortOrder") {
				t.Errorf("expected sortOrder validation error, got %s", w.Body.String())
			}
		}
	})

	t.Run("should return 400 when parentID does not exist in database", func(t *testing.T) {
		parentID := uuid.New().String()
		payload := map[string]interface{}{
			"parentID":    parentID,
			"name":        "Sub Dashboard",
			"description": "Sub dashboard menu",
		}
		mockSrv := &mockMenuService{
			CreateMenuSrvcFunc: func(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					Message:       "pq: insert or update on table \"menus\" violates foreign key constraint \"menus_parent_id_fkey\"",
					MessageToSend: "Parent menu with the specified ID does not exist",
					Property:      "parentID",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPost, "/api/menu", buildJSONBody(payload))
		w := httptest.NewRecorder()

		ctrl.createCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Parent menu with the specified ID does not exist") {
			t.Errorf("expected response to contain 'Parent menu with the specified ID does not exist', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"parentID"`) {
			t.Errorf("expected response errors to specify 'parentID' field, got %s", w.Body.String())
		}
	})

	t.Run("should return 409 when menu name already exists", func(t *testing.T) {
		mockSrv := &mockMenuService{
			CreateMenuSrvcFunc: func(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusConflict,
					Message:       "unique constraint",
					MessageToSend: "Menu name already exists",
					Property:      "name",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPost, "/api/menu", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.createCtrl(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected status 409, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Menu name already exists") {
			t.Errorf("expected response to contain 'Menu name already exists', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"name"`) {
			t.Errorf("expected response errors to specify 'name' field, got %s", w.Body.String())
		}
	})

	t.Run("should return 500 when service fails unexpectedly", func(t *testing.T) {
		mockSrv := &mockMenuService{
			CreateMenuSrvcFunc: func(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "database error",
					MessageToSend: "Failed create menu",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPost, "/api/menu", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.createCtrl(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Internal Server Error") {
			t.Errorf("expected response to contain 'Internal Server Error', got %s", w.Body.String())
		}
	})

	t.Run("should return 201 Created on successful menu creation", func(t *testing.T) {
		mockSrv := &mockMenuService{
			CreateMenuWithOrderSrvcFunc: func(ctx context.Context, param models.CreateMenuRequest) (models.MenuSortOrderInfo, errUtility.CustomError) {
				return models.MenuSortOrderInfo{SortOrder: 2, SuggestedSortOrder: 3}, errUtility.CustomError{}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPost, "/api/menu", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.createCtrl(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Menu created successfully") {
			t.Errorf("expected response to contain 'Menu created successfully', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"sortOrder":2`) || !strings.Contains(w.Body.String(), `"suggestedSortOrder":3`) {
			t.Errorf("expected response to include chosen and suggested sort orders, got %s", w.Body.String())
		}
	})

	t.Run("should return a suggested order when the requested order is unavailable", func(t *testing.T) {
		ctrl := menuController{service: &mockMenuService{CreateMenuWithOrderSrvcFunc: func(ctx context.Context, param models.CreateMenuRequest) (models.MenuSortOrderInfo, errUtility.CustomError) {
			return models.MenuSortOrderInfo{SuggestedSortOrder: 3}, errUtility.CustomError{
				Code:          http.StatusConflict,
				MessageToSend: "Sort order is already in use",
				Property:      "sortOrder",
			}
		}}}
		req := httptest.NewRequest(http.MethodPost, "/api/menu", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.createCtrl(w, req)

		if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"suggestedSortOrder":3`) {
			t.Errorf("expected 409 with suggestedSortOrder 3, got status=%d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestGetByIDCtrl(t *testing.T) {
	testUUID := uuid.New()

	newRequestWithParam := func(method, target, paramKey, paramVal string) *http.Request {
		req := httptest.NewRequest(method, target, nil)
		rctx := chi.NewRouteContext()
		if paramKey != "" {
			rctx.URLParams.Add(paramKey, paramVal)
		}
		return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	t.Run("should return 400 when menuID is missing from path parameter", func(t *testing.T) {
		ctrl := menuController{service: &mockMenuService{}}
		req := newRequestWithParam(http.MethodGet, "/api/menu/", "", "")
		w := httptest.NewRecorder()

		ctrl.GetByIDCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "menuID path parameter is required") {
			t.Errorf("expected response to contain 'menuID path parameter is required', got %s", w.Body.String())
		}
	})

	t.Run("should return 400 when menuID is not a valid UUID", func(t *testing.T) {
		ctrl := menuController{service: &mockMenuService{}}
		req := newRequestWithParam(http.MethodGet, "/api/menu/not-a-uuid", "menuID", "not-a-uuid")
		w := httptest.NewRecorder()

		ctrl.GetByIDCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid menuID") {
			t.Errorf("expected response to contain 'Invalid menuID', got %s", w.Body.String())
		}
	})

	t.Run("should return 404 when menu is not found", func(t *testing.T) {
		mockSrv := &mockMenuService{
			GetByIDSrvcFunc: func(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError) {
				return models.MenuResponse{}, errUtility.CustomError{
					Code:          http.StatusNotFound,
					MessageToSend: "Menu not found",
					Property:      "id",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := newRequestWithParam(http.MethodGet, "/api/menu/"+testUUID.String(), "menuID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.GetByIDCtrl(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Menu not found") {
			t.Errorf("expected response to contain 'Menu not found', got %s", w.Body.String())
		}
	})

	t.Run("should return 500 when service fails unexpectedly", func(t *testing.T) {
		mockSrv := &mockMenuService{
			GetByIDSrvcFunc: func(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError) {
				return models.MenuResponse{}, errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "internal database failure",
					MessageToSend: "Failed get menu",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := newRequestWithParam(http.MethodGet, "/api/menu/"+testUUID.String(), "menuID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.GetByIDCtrl(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Internal Server Error") {
			t.Errorf("expected response to contain 'Internal Server Error', got %s", w.Body.String())
		}
	})

	t.Run("should return 200 with menu data on success", func(t *testing.T) {
		expectedMenu := models.MenuResponse{
			Id:          testUUID.String(),
			Name:        "Dashboard",
			Description: "Dashboard menu",
			CreatedAt:   "2026-10-04 13:00:00",
		}
		mockSrv := &mockMenuService{
			GetByIDSrvcFunc: func(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError) {
				return expectedMenu, errUtility.CustomError{}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := newRequestWithParam(http.MethodGet, "/api/menu/"+testUUID.String(), "menuID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.GetByIDCtrl(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Menu retrieved successfully") {
			t.Errorf("expected response to contain 'Menu retrieved successfully', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "Dashboard") {
			t.Errorf("expected response to contain menu name 'Dashboard', got %s", w.Body.String())
		}
	})
}

func TestUpdateCtrl(t *testing.T) {
	defaultMenuID := uuid.New()
	defaultPayload := map[string]interface{}{
		"id":          defaultMenuID.String(),
		"name":        "Dashboard",
		"description": "Main dashboard menu",
	}

	buildJSONBody := func(payload interface{}) *bytes.Buffer {
		data, _ := json.Marshal(payload)
		return bytes.NewBuffer(data)
	}

	t.Run("should return 400 when JSON body is invalid", func(t *testing.T) {
		ctrl := menuController{service: &mockMenuService{}}
		req := httptest.NewRequest(http.MethodPut, "/api/menu", strings.NewReader("{invalid-json"))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid data") {
			t.Errorf("expected response to contain 'Invalid data', got %s", w.Body.String())
		}
	})

	t.Run("should return 422 when validation fails (missing required id)", func(t *testing.T) {
		ctrl := menuController{service: &mockMenuService{}}
		req := httptest.NewRequest(http.MethodPut, "/api/menu", buildJSONBody(map[string]interface{}{
			"name":        "Dashboard",
			"description": "missing id",
		}))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status 422, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Validation failed") {
			t.Errorf("expected response to contain 'Validation failed', got %s", w.Body.String())
		}
	})

	t.Run("should reject non-positive sort order", func(t *testing.T) {
		for _, sortOrder := range []int{0, -1} {
			ctrl := menuController{service: &mockMenuService{}}
			req := httptest.NewRequest(http.MethodPut, "/api/menu", buildJSONBody(map[string]interface{}{
				"id":          uuid.New().String(),
				"name":        "Dashboard",
				"description": "Main dashboard menu",
				"sortOrder":   sortOrder,
			}))
			w := httptest.NewRecorder()

			ctrl.updateCtrl(w, req)

			if w.Code != http.StatusUnprocessableEntity {
				t.Errorf("expected status 422 for sortOrder %d, got %d", sortOrder, w.Code)
			}
			if !strings.Contains(w.Body.String(), "SortOrder") {
				t.Errorf("expected sortOrder validation error, got %s", w.Body.String())
			}
		}
	})

	t.Run("should return 400 when parentID does not exist in database", func(t *testing.T) {
		parentID := uuid.New().String()
		payload := map[string]interface{}{
			"id":          defaultMenuID.String(),
			"parentID":    parentID,
			"name":        "Sub Dashboard",
			"description": "Sub dashboard menu",
		}
		mockSrv := &mockMenuService{
			UpdateMenuSrvcFunc: func(ctx context.Context, param models.UpdateMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					Message:       "pq: insert or update on table \"menus\" violates foreign key constraint \"menus_parent_id_fkey\"",
					MessageToSend: "Parent menu with the specified ID does not exist",
					Property:      "parentID",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPut, "/api/menu", buildJSONBody(payload))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Parent menu with the specified ID does not exist") {
			t.Errorf("expected response to contain 'Parent menu with the specified ID does not exist', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"parentID"`) {
			t.Errorf("expected response errors to specify 'parentID' field, got %s", w.Body.String())
		}
	})

	t.Run("should return 409 when menu name already exists", func(t *testing.T) {
		mockSrv := &mockMenuService{
			UpdateMenuSrvcFunc: func(ctx context.Context, param models.UpdateMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusConflict,
					Message:       "unique constraint",
					MessageToSend: "Menu name already exists",
					Property:      "name",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPut, "/api/menu", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected status 409, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Menu name already exists") {
			t.Errorf("expected response to contain 'Menu name already exists', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"name"`) {
			t.Errorf("expected response errors to specify 'name' field, got %s", w.Body.String())
		}
	})

	t.Run("should return 404 when menu to update is not found", func(t *testing.T) {
		mockSrv := &mockMenuService{
			UpdateMenuSrvcFunc: func(ctx context.Context, param models.UpdateMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusNotFound,
					MessageToSend: "Menu not found",
					Property:      "id",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPut, "/api/menu", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Menu not found") {
			t.Errorf("expected response to contain 'Menu not found', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"id"`) {
			t.Errorf("expected response errors to specify 'id' field, got %s", w.Body.String())
		}
	})

	t.Run("should return 500 when service fails unexpectedly", func(t *testing.T) {
		mockSrv := &mockMenuService{
			UpdateMenuSrvcFunc: func(ctx context.Context, param models.UpdateMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "internal database failure",
					MessageToSend: "Failed update menu",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPut, "/api/menu", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Internal Server Error") {
			t.Errorf("expected response to contain 'Internal Server Error', got %s", w.Body.String())
		}
	})

	t.Run("should return 201 Created on successful menu update", func(t *testing.T) {
		mockSrv := &mockMenuService{
			UpdateMenuWithOrderSrvcFunc: func(ctx context.Context, param models.UpdateMenuRequest) (models.MenuSortOrderInfo, errUtility.CustomError) {
				return models.MenuSortOrderInfo{SortOrder: 4, SuggestedSortOrder: 5}, errUtility.CustomError{}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPut, "/api/menu", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Menu updated successfully") {
			t.Errorf("expected response to contain 'Menu updated successfully', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"sortOrder":4`) || !strings.Contains(w.Body.String(), `"suggestedSortOrder":5`) {
			t.Errorf("expected response to include chosen and suggested sort orders, got %s", w.Body.String())
		}
	})

	t.Run("should return a suggested order when the requested order is unavailable", func(t *testing.T) {
		ctrl := menuController{service: &mockMenuService{UpdateMenuWithOrderSrvcFunc: func(ctx context.Context, param models.UpdateMenuRequest) (models.MenuSortOrderInfo, errUtility.CustomError) {
			return models.MenuSortOrderInfo{SuggestedSortOrder: 5}, errUtility.CustomError{
				Code:          http.StatusConflict,
				MessageToSend: "Sort order is already in use",
				Property:      "sortOrder",
			}
		}}}
		req := httptest.NewRequest(http.MethodPut, "/api/menu", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

		if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"suggestedSortOrder":5`) {
			t.Errorf("expected 409 with suggestedSortOrder 5, got status=%d body=%s", w.Code, w.Body.String())
		}
	})
}

func TestDeleteCtrl(t *testing.T) {
	testUUID := uuid.New()

	newRequestWithParam := func(method, target, paramKey, paramVal string) *http.Request {
		req := httptest.NewRequest(method, target, nil)
		rctx := chi.NewRouteContext()
		if paramKey != "" {
			rctx.URLParams.Add(paramKey, paramVal)
		}
		return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	t.Run("should return 400 when menuID is missing from path parameter", func(t *testing.T) {
		ctrl := menuController{service: &mockMenuService{}}
		req := newRequestWithParam(http.MethodDelete, "/api/menu/", "", "")
		w := httptest.NewRecorder()

		ctrl.DeleteCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "menuID path parameter is required") {
			t.Errorf("expected response to contain 'menuID path parameter is required', got %s", w.Body.String())
		}
	})

	t.Run("should return 400 when menuID is not a valid UUID", func(t *testing.T) {
		ctrl := menuController{service: &mockMenuService{}}
		req := newRequestWithParam(http.MethodDelete, "/api/menu/not-a-uuid", "menuID", "not-a-uuid")
		w := httptest.NewRecorder()

		ctrl.DeleteCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid menuID") {
			t.Errorf("expected response to contain 'Invalid menuID', got %s", w.Body.String())
		}
	})

	t.Run("should return 400 when menu has child menus and cannot be deleted", func(t *testing.T) {
		mockSrv := &mockMenuService{
			DeleteSrvcFunc: func(ctx context.Context, menuID uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					MessageToSend: "Cannot delete menu because it has child menus",
					Property:      "id",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := newRequestWithParam(http.MethodDelete, "/api/menu/"+testUUID.String(), "menuID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.DeleteCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Cannot delete menu because it has child menus") {
			t.Errorf("expected response to contain 'Cannot delete menu because it has child menus', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"id"`) {
			t.Errorf("expected response errors to specify 'id' field, got %s", w.Body.String())
		}
	})

	t.Run("should return 404 when menu to delete is not found", func(t *testing.T) {
		mockSrv := &mockMenuService{
			DeleteSrvcFunc: func(ctx context.Context, menuID uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusNotFound,
					MessageToSend: "Menu not found",
					Property:      "id",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := newRequestWithParam(http.MethodDelete, "/api/menu/"+testUUID.String(), "menuID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.DeleteCtrl(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Menu not found") {
			t.Errorf("expected response to contain 'Menu not found', got %s", w.Body.String())
		}
	})

	t.Run("should return 500 when service fails unexpectedly", func(t *testing.T) {
		mockSrv := &mockMenuService{
			DeleteSrvcFunc: func(ctx context.Context, menuID uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "internal database failure",
					MessageToSend: "Failed delete menu",
				}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := newRequestWithParam(http.MethodDelete, "/api/menu/"+testUUID.String(), "menuID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.DeleteCtrl(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Internal Server Error") {
			t.Errorf("expected response to contain 'Internal Server Error', got %s", w.Body.String())
		}
	})

	t.Run("should return 200 on successful menu deletion", func(t *testing.T) {
		mockSrv := &mockMenuService{
			DeleteSrvcFunc: func(ctx context.Context, menuID uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{}
			},
		}
		ctrl := menuController{service: mockSrv}
		req := newRequestWithParam(http.MethodDelete, "/api/menu/"+testUUID.String(), "menuID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.DeleteCtrl(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Menu deleted successfully") {
			t.Errorf("expected response to contain 'Menu deleted successfully', got %s", w.Body.String())
		}
	})
}
