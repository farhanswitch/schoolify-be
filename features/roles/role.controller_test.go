package roles

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type mockRoleService struct {
	GetAllRolesSrvcFunc func(ctx context.Context) ([]*models.RoleResponse, errUtility.CustomError)
	CreateRoleSrvcFunc  func(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError
}

func (m *mockRoleService) GetAllRolesSrvc(ctx context.Context) ([]*models.RoleResponse, errUtility.CustomError) {
	if m.GetAllRolesSrvcFunc != nil {
		return m.GetAllRolesSrvcFunc(ctx)
	}
	return []*models.RoleResponse{}, errUtility.CustomError{}
}

func (m *mockRoleService) CreateRoleSrvc(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError {
	if m.CreateRoleSrvcFunc != nil {
		return m.CreateRoleSrvcFunc(ctx, param)
	}
	return errUtility.CustomError{}
}

func TestGetAllRolesCtrl(t *testing.T) {
	t.Run("should return 500 when service fails", func(t *testing.T) {
		mockSrv := &mockRoleService{
			GetAllRolesSrvcFunc: func(ctx context.Context) ([]*models.RoleResponse, errUtility.CustomError) {
				return nil, errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "db failure",
					MessageToSend: "Failed get roles",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := httptest.NewRequest(http.MethodGet, "/api/role/list", nil)
		w := httptest.NewRecorder()

		ctrl.getAllRolesCtrl(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})

	t.Run("should return 200 with role tree on success", func(t *testing.T) {
		mockSrv := &mockRoleService{
			GetAllRolesSrvcFunc: func(ctx context.Context) ([]*models.RoleResponse, errUtility.CustomError) {
				return []*models.RoleResponse{
					{
						ID:   uuid.New(),
						Name: "Super Admin",
					},
				}, errUtility.CustomError{}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := httptest.NewRequest(http.MethodGet, "/api/role/list", nil)
		w := httptest.NewRecorder()

		ctrl.getAllRolesCtrl(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Roles retrieved successfully") {
			t.Errorf("expected response to contain 'Roles retrieved successfully', got %s", w.Body.String())
		}
	})
}

func TestCreateCtrl(t *testing.T) {
	defaultPayload := map[string]interface{}{
		"name":        "Staff",
		"description": "School staff member",
	}

	buildJSONBody := func(payload interface{}) *bytes.Buffer {
		data, _ := json.Marshal(payload)
		return bytes.NewBuffer(data)
	}

	t.Run("should return 400 when JSON body is invalid", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		req := httptest.NewRequest(http.MethodPost, "/api/role", strings.NewReader("{invalid-json"))
		w := httptest.NewRecorder()

		ctrl.createCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid payload") {
			t.Errorf("expected response to contain 'Invalid payload', got %s", w.Body.String())
		}
	})

	t.Run("should return 422 when validation fails (missing required name)", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		req := httptest.NewRequest(http.MethodPost, "/api/role", buildJSONBody(map[string]interface{}{
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
			"name":        "Guest",
			"description": "Guest role",
		}
		mockSrv := &mockRoleService{
			CreateRoleSrvcFunc: func(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					Message:       "pq: insert or update on table \"roles\" violates foreign key constraint \"roles_parent_id_fkey\"",
					MessageToSend: "Parent role with the specified ID does not exist",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPost, "/api/role", buildJSONBody(payload))
		w := httptest.NewRecorder()

		ctrl.createCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Parent role with the specified ID does not exist") {
			t.Errorf("expected response to contain 'Parent role with the specified ID does not exist', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"parentID"`) {
			t.Errorf("expected response errors to specify 'parentID' field, got %s", w.Body.String())
		}
	})

	t.Run("should return 409 when role name already exists", func(t *testing.T) {
		mockSrv := &mockRoleService{
			CreateRoleSrvcFunc: func(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusConflict,
					Message:       "unique constraint",
					MessageToSend: "Role name already exists",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPost, "/api/role", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.createCtrl(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected status 409, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Role name already exists") {
			t.Errorf("expected response to contain 'Role name already exists', got %s", w.Body.String())
		}
	})

	t.Run("should return 201 Created on successful role creation", func(t *testing.T) {
		mockSrv := &mockRoleService{
			CreateRoleSrvcFunc: func(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPost, "/api/role", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.createCtrl(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Role created successfully") {
			t.Errorf("expected response to contain 'Role created successfully', got %s", w.Body.String())
		}
	})
}
