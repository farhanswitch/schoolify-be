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
	GetAllMenusSrvcFunc func(ctx context.Context) ([]*models.MenuResponse, errUtility.CustomError)
	CreateMenuSrvcFunc  func(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError
	GetByIDSrvcFunc     func(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError)
	UpdateMenuSrvcFunc  func(ctx context.Context, param models.UpdateMenuRequest) errUtility.CustomError
	DeleteSrvcFunc      func(ctx context.Context, menuID uuid.UUID) errUtility.CustomError
}

func (m *mockMenuService) GetAllMenusSrvc(ctx context.Context) ([]*models.MenuResponse, errUtility.CustomError) {
	if m.GetAllMenusSrvcFunc != nil {
		return m.GetAllMenusSrvcFunc(ctx)
	}
	return []*models.MenuResponse{}, errUtility.CustomError{}
}

func (m *mockMenuService) CreateMenuSrvc(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
	if m.CreateMenuSrvcFunc != nil {
		return m.CreateMenuSrvcFunc(ctx, param)
	}
	return errUtility.CustomError{}
}

func (m *mockMenuService) GetByIDSrvc(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError) {
	if m.GetByIDSrvcFunc != nil {
		return m.GetByIDSrvcFunc(ctx, menuID)
	}
	return models.MenuResponse{}, errUtility.CustomError{}
}

func (m *mockMenuService) UpdateMenuSrvc(ctx context.Context, param models.UpdateMenuRequest) errUtility.CustomError {
	if m.UpdateMenuSrvcFunc != nil {
		return m.UpdateMenuSrvcFunc(ctx, param)
	}
	return errUtility.CustomError{}
}

func (m *mockMenuService) DeleteSrvc(ctx context.Context, menuID uuid.UUID) errUtility.CustomError {
	if m.DeleteSrvcFunc != nil {
		return m.DeleteSrvcFunc(ctx, menuID)
	}
	return errUtility.CustomError{}
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
			CreateMenuSrvcFunc: func(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{}
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
			UpdateMenuSrvcFunc: func(ctx context.Context, param models.UpdateMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{}
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



