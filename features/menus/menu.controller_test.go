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

	"github.com/google/uuid"
)

type mockMenuService struct {
	CreateMenuSrvcFunc func(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError
}

func (m *mockMenuService) CreateMenuSrvc(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
	if m.CreateMenuSrvcFunc != nil {
		return m.CreateMenuSrvcFunc(ctx, param)
	}
	return errUtility.CustomError{}
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
