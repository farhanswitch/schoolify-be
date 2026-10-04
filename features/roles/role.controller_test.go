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

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type mockRoleService struct {
	GetAllRolesSrvcFunc func(ctx context.Context) ([]*models.RoleResponse, errUtility.CustomError)
	CreateRoleSrvcFunc  func(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError
	UpdateRoleSrvcFunc  func(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError
	GetByIDSrvcFunc     func(ctx context.Context, roleID uuid.UUID) (models.RoleResponse, errUtility.CustomError)
	DeleteSrvcFunc      func(ctx context.Context, roleID uuid.UUID) errUtility.CustomError
	AssignMenuSrvcFunc   func(ctx context.Context, roleID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError
	GetAssignedMenusSrvcFunc func(ctx context.Context, roleID uuid.UUID) ([]*models.RoleMenuResponse, errUtility.CustomError)
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

func (m *mockRoleService) UpdateRoleSrvc(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
	if m.UpdateRoleSrvcFunc != nil {
		return m.UpdateRoleSrvcFunc(ctx, param)
	}
	return errUtility.CustomError{}
}

func (m *mockRoleService) GetByIDSrvc(ctx context.Context, roleID uuid.UUID) (models.RoleResponse, errUtility.CustomError) {
	if m.GetByIDSrvcFunc != nil {
		return m.GetByIDSrvcFunc(ctx, roleID)
	}
	return models.RoleResponse{}, errUtility.CustomError{}
}

func (m *mockRoleService) DeleteSrvc(ctx context.Context, roleID uuid.UUID) errUtility.CustomError {
	if m.DeleteSrvcFunc != nil {
		return m.DeleteSrvcFunc(ctx, roleID)
	}
	return errUtility.CustomError{}
}

func (m *mockRoleService) AssignMenuSrvc(ctx context.Context, roleID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError {
	if m.AssignMenuSrvcFunc != nil {
		return m.AssignMenuSrvcFunc(ctx, roleID, param)
	}
	return errUtility.CustomError{}
}

func (m *mockRoleService) GetAssignedMenusSrvc(ctx context.Context, roleID uuid.UUID) ([]*models.RoleMenuResponse, errUtility.CustomError) {
	if m.GetAssignedMenusSrvcFunc != nil {
		return m.GetAssignedMenusSrvcFunc(ctx, roleID)
	}
	return []*models.RoleMenuResponse{}, errUtility.CustomError{}
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
		if !strings.Contains(w.Body.String(), "Invalid data") {
			t.Errorf("expected response to contain 'Invalid data', got %s", w.Body.String())
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
					Property:      "parentID",
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
					Property:      "name",
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

func TestUpdateCtrl(t *testing.T) {
	defaultRoleID := uuid.New()
	defaultPayload := map[string]interface{}{
		"id":          defaultRoleID.String(),
		"name":        "Staff",
		"description": "School staff member",
	}

	buildJSONBody := func(payload interface{}) *bytes.Buffer {
		data, _ := json.Marshal(payload)
		return bytes.NewBuffer(data)
	}

	t.Run("should return 400 when JSON body is invalid", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		req := httptest.NewRequest(http.MethodPut, "/api/role", strings.NewReader("{invalid-json"))
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
		ctrl := roleController{service: &mockRoleService{}}
		req := httptest.NewRequest(http.MethodPut, "/api/role", buildJSONBody(map[string]interface{}{
			"name":        "Staff",
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
			"id":          defaultRoleID.String(),
			"parentID":    parentID,
			"name":        "Guest",
			"description": "Guest role",
		}
		mockSrv := &mockRoleService{
			UpdateRoleSrvcFunc: func(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					Message:       "pq: insert or update on table \"roles\" violates foreign key constraint \"roles_parent_id_fkey\"",
					MessageToSend: "Parent role with the specified ID does not exist",
					Property:      "parentID",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPut, "/api/role", buildJSONBody(payload))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

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
			UpdateRoleSrvcFunc: func(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusConflict,
					Message:       "unique constraint",
					MessageToSend: "Role name already exists",
					Property:      "name",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPut, "/api/role", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

		if w.Code != http.StatusConflict {
			t.Errorf("expected status 409, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Role name already exists") {
			t.Errorf("expected response to contain 'Role name already exists', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"name"`) {
			t.Errorf("expected response errors to specify 'name' field, got %s", w.Body.String())
		}
	})

	t.Run("should return 404 when role to update is not found", func(t *testing.T) {
		mockSrv := &mockRoleService{
			UpdateRoleSrvcFunc: func(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusNotFound,
					MessageToSend: "Role not found",
					Property:      "id",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPut, "/api/role", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Role not found") {
			t.Errorf("expected response to contain 'Role not found', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"id"`) {
			t.Errorf("expected response errors to specify 'id' field, got %s", w.Body.String())
		}
	})

	t.Run("should return 500 when service fails unexpectedly", func(t *testing.T) {
		mockSrv := &mockRoleService{
			UpdateRoleSrvcFunc: func(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "internal database failure",
					MessageToSend: "Failed update role",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPut, "/api/role", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Internal Server Error") {
			t.Errorf("expected response to contain 'Internal Server Error', got %s", w.Body.String())
		}
	})

	t.Run("should return 201 Created on successful role update", func(t *testing.T) {
		mockSrv := &mockRoleService{
			UpdateRoleSrvcFunc: func(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := httptest.NewRequest(http.MethodPut, "/api/role", buildJSONBody(defaultPayload))
		w := httptest.NewRecorder()

		ctrl.updateCtrl(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Role updated successfully") {
			t.Errorf("expected response to contain 'Role updated successfully', got %s", w.Body.String())
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

	t.Run("should return 400 when roleID is missing from path parameter", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		req := newRequestWithParam(http.MethodGet, "/api/role/", "", "")
		w := httptest.NewRecorder()

		ctrl.GetByIDCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "roleID path parameter is required") {
			t.Errorf("expected response to contain 'roleID path parameter is required', got %s", w.Body.String())
		}
	})

	t.Run("should return 400 when roleID is not a valid UUID", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		req := newRequestWithParam(http.MethodGet, "/api/role/not-a-uuid", "roleID", "not-a-uuid")
		w := httptest.NewRecorder()

		ctrl.GetByIDCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid roleID") {
			t.Errorf("expected response to contain 'Invalid roleID', got %s", w.Body.String())
		}
	})

	t.Run("should return 404 when role is not found", func(t *testing.T) {
		mockSrv := &mockRoleService{
			GetByIDSrvcFunc: func(ctx context.Context, roleID uuid.UUID) (models.RoleResponse, errUtility.CustomError) {
				return models.RoleResponse{}, errUtility.CustomError{
					Code:          http.StatusNotFound,
					MessageToSend: "Role not found",
					Property:      "id",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParam(http.MethodGet, "/api/role/"+testUUID.String(), "roleID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.GetByIDCtrl(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Role not found") {
			t.Errorf("expected response to contain 'Role not found', got %s", w.Body.String())
		}
	})

	t.Run("should return 500 when service fails unexpectedly", func(t *testing.T) {
		mockSrv := &mockRoleService{
			GetByIDSrvcFunc: func(ctx context.Context, roleID uuid.UUID) (models.RoleResponse, errUtility.CustomError) {
				return models.RoleResponse{}, errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "internal database failure",
					MessageToSend: "Failed get role",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParam(http.MethodGet, "/api/role/"+testUUID.String(), "roleID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.GetByIDCtrl(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Internal Server Error") {
			t.Errorf("expected response to contain 'Internal Server Error', got %s", w.Body.String())
		}
	})

	t.Run("should return 200 with role data on success", func(t *testing.T) {
		expectedRole := models.RoleResponse{
			ID:          testUUID,
			Name:        "Teacher",
			Description: "Teacher role",
			CreatedAt:   "2026-10-04 12:00:00",
		}
		mockSrv := &mockRoleService{
			GetByIDSrvcFunc: func(ctx context.Context, roleID uuid.UUID) (models.RoleResponse, errUtility.CustomError) {
				return expectedRole, errUtility.CustomError{}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParam(http.MethodGet, "/api/role/"+testUUID.String(), "roleID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.GetByIDCtrl(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Role retrieved successfully") {
			t.Errorf("expected response to contain 'Role retrieved successfully', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "Teacher") {
			t.Errorf("expected response to contain role name 'Teacher', got %s", w.Body.String())
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

	t.Run("should return 400 when roleID is missing from path parameter", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		req := newRequestWithParam(http.MethodDelete, "/api/role/", "", "")
		w := httptest.NewRecorder()

		ctrl.DeleteCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "roleID path parameter is required") {
			t.Errorf("expected response to contain 'roleID path parameter is required', got %s", w.Body.String())
		}
	})

	t.Run("should return 400 when roleID is not a valid UUID", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		req := newRequestWithParam(http.MethodDelete, "/api/role/not-a-uuid", "roleID", "not-a-uuid")
		w := httptest.NewRecorder()

		ctrl.DeleteCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid roleID") {
			t.Errorf("expected response to contain 'Invalid roleID', got %s", w.Body.String())
		}
	})

	t.Run("should return 400 when role has child roles and cannot be deleted", func(t *testing.T) {
		mockSrv := &mockRoleService{
			DeleteSrvcFunc: func(ctx context.Context, roleID uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					MessageToSend: "Cannot delete role because it has child roles",
					Property:      "id",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParam(http.MethodDelete, "/api/role/"+testUUID.String(), "roleID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.DeleteCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Cannot delete role because it has child roles") {
			t.Errorf("expected response to contain 'Cannot delete role because it has child roles', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"id"`) {
			t.Errorf("expected response errors to specify 'id' field, got %s", w.Body.String())
		}
	})

	t.Run("should return 404 when role to delete is not found", func(t *testing.T) {
		mockSrv := &mockRoleService{
			DeleteSrvcFunc: func(ctx context.Context, roleID uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusNotFound,
					MessageToSend: "Role not found",
					Property:      "id",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParam(http.MethodDelete, "/api/role/"+testUUID.String(), "roleID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.DeleteCtrl(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Role not found") {
			t.Errorf("expected response to contain 'Role not found', got %s", w.Body.String())
		}
	})

	t.Run("should return 500 when service fails unexpectedly", func(t *testing.T) {
		mockSrv := &mockRoleService{
			DeleteSrvcFunc: func(ctx context.Context, roleID uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "internal database failure",
					MessageToSend: "Failed delete role",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParam(http.MethodDelete, "/api/role/"+testUUID.String(), "roleID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.DeleteCtrl(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Internal Server Error") {
			t.Errorf("expected response to contain 'Internal Server Error', got %s", w.Body.String())
		}
	})

	t.Run("should return 200 on successful role deletion", func(t *testing.T) {
		mockSrv := &mockRoleService{
			DeleteSrvcFunc: func(ctx context.Context, roleID uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParam(http.MethodDelete, "/api/role/"+testUUID.String(), "roleID", testUUID.String())
		w := httptest.NewRecorder()

		ctrl.DeleteCtrl(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Role deleted successfully") {
			t.Errorf("expected response to contain 'Role deleted successfully', got %s", w.Body.String())
		}
	})
}

func TestAssignMenuCtrl(t *testing.T) {
	testRoleUUID := uuid.New()
	testMenuUUID := uuid.New()

	newRequestWithParamAndBody := func(method, target, paramKey, paramVal string, body []byte) *http.Request {
		req := httptest.NewRequest(method, target, bytes.NewBuffer(body))
		rctx := chi.NewRouteContext()
		if paramKey != "" {
			rctx.URLParams.Add(paramKey, paramVal)
		}
		return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	validBody, _ := json.Marshal(map[string]interface{}{
		"listID": []string{testMenuUUID.String()},
	})

	t.Run("should return 400 when roleID is missing from path parameter", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		req := newRequestWithParamAndBody(http.MethodPut, "/api/role/assign-menu/", "", "", validBody)
		w := httptest.NewRecorder()

		ctrl.assignMenuCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "roleID path parameter is required") {
			t.Errorf("expected response to contain 'roleID path parameter is required', got %s", w.Body.String())
		}
	})

	t.Run("should return 400 when roleID is not a valid UUID", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		req := newRequestWithParamAndBody(http.MethodPut, "/api/role/assign-menu/not-a-uuid", "roleID", "not-a-uuid", validBody)
		w := httptest.NewRecorder()

		ctrl.assignMenuCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid roleID") {
			t.Errorf("expected response to contain 'Invalid roleID', got %s", w.Body.String())
		}
	})

	t.Run("should return 400 when JSON body is invalid", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		req := newRequestWithParamAndBody(http.MethodPut, "/api/role/assign-menu/"+testRoleUUID.String(), "roleID", testRoleUUID.String(), []byte("invalid-json"))
		w := httptest.NewRecorder()

		ctrl.assignMenuCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid data") {
			t.Errorf("expected response to contain 'Invalid data', got %s", w.Body.String())
		}
	})

	t.Run("should return 422 when validation fails (missing listID)", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		emptyBody, _ := json.Marshal(map[string]interface{}{})
		req := newRequestWithParamAndBody(http.MethodPut, "/api/role/assign-menu/"+testRoleUUID.String(), "roleID", testRoleUUID.String(), emptyBody)
		w := httptest.NewRecorder()

		ctrl.assignMenuCtrl(w, req)

		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected status 422, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Validation failed") {
			t.Errorf("expected response to contain 'Validation failed', got %s", w.Body.String())
		}
	})

	t.Run("should return 404 when role does not exist in database", func(t *testing.T) {
		mockSrv := &mockRoleService{
			AssignMenuSrvcFunc: func(ctx context.Context, roleID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusNotFound,
					MessageToSend: "Role not found",
					Property:      "roleID",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParamAndBody(http.MethodPut, "/api/role/assign-menu/"+testRoleUUID.String(), "roleID", testRoleUUID.String(), validBody)
		w := httptest.NewRecorder()

		ctrl.assignMenuCtrl(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Role not found") {
			t.Errorf("expected response to contain 'Role not found', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"roleID"`) {
			t.Errorf("expected response errors to specify 'roleID' field, got %s", w.Body.String())
		}
	})

	t.Run("should return 400 when a menu ID does not exist in database", func(t *testing.T) {
		mockSrv := &mockRoleService{
			AssignMenuSrvcFunc: func(ctx context.Context, roleID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					MessageToSend: "Menu with ID " + testMenuUUID.String() + " does not exist",
					Property:      "listID",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParamAndBody(http.MethodPut, "/api/role/assign-menu/"+testRoleUUID.String(), "roleID", testRoleUUID.String(), validBody)
		w := httptest.NewRecorder()

		ctrl.assignMenuCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "does not exist") {
			t.Errorf("expected response to contain 'does not exist', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"listID"`) {
			t.Errorf("expected response errors to specify 'listID' field, got %s", w.Body.String())
		}
	})

	t.Run("should return 500 when service fails unexpectedly", func(t *testing.T) {
		mockSrv := &mockRoleService{
			AssignMenuSrvcFunc: func(ctx context.Context, roleID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "internal database failure",
					MessageToSend: "Failed to assign menu",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParamAndBody(http.MethodPut, "/api/role/assign-menu/"+testRoleUUID.String(), "roleID", testRoleUUID.String(), validBody)
		w := httptest.NewRecorder()

		ctrl.assignMenuCtrl(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Internal Server Error") {
			t.Errorf("expected response to contain 'Internal Server Error', got %s", w.Body.String())
		}
	})

	t.Run("should return 201 on successful menu assignment", func(t *testing.T) {
		mockSrv := &mockRoleService{
			AssignMenuSrvcFunc: func(ctx context.Context, roleID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParamAndBody(http.MethodPut, "/api/role/assign-menu/"+testRoleUUID.String(), "roleID", testRoleUUID.String(), validBody)
		w := httptest.NewRecorder()

		ctrl.assignMenuCtrl(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "List Menu assigned successfully") {
			t.Errorf("expected response to contain 'List Menu assigned successfully', got %s", w.Body.String())
		}
	})
}

func TestListAssignedMenuCtrl(t *testing.T) {
	testRoleUUID := uuid.New()

	newRequestWithParam := func(method, target, paramKey, paramVal string) *http.Request {
		req := httptest.NewRequest(method, target, nil)
		rctx := chi.NewRouteContext()
		if paramKey != "" {
			rctx.URLParams.Add(paramKey, paramVal)
		}
		return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	}

	t.Run("should return 400 when roleID is missing from path parameter", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		req := newRequestWithParam(http.MethodGet, "/api/role/list-assigned-menu/", "", "")
		w := httptest.NewRecorder()

		ctrl.listAssignedMenuCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "roleID path parameter is required") {
			t.Errorf("expected response to contain 'roleID path parameter is required', got %s", w.Body.String())
		}
	})

	t.Run("should return 400 when roleID is not a valid UUID", func(t *testing.T) {
		ctrl := roleController{service: &mockRoleService{}}
		req := newRequestWithParam(http.MethodGet, "/api/role/list-assigned-menu/not-a-uuid", "roleID", "not-a-uuid")
		w := httptest.NewRecorder()

		ctrl.listAssignedMenuCtrl(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid roleID") {
			t.Errorf("expected response to contain 'Invalid roleID', got %s", w.Body.String())
		}
	})

	t.Run("should return 404 when role does not exist in database", func(t *testing.T) {
		mockSrv := &mockRoleService{
			GetAssignedMenusSrvcFunc: func(ctx context.Context, roleID uuid.UUID) ([]*models.RoleMenuResponse, errUtility.CustomError) {
				return nil, errUtility.CustomError{
					Code:          http.StatusNotFound,
					MessageToSend: "Role not found",
					Property:      "roleID",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParam(http.MethodGet, "/api/role/list-assigned-menu/"+testRoleUUID.String(), "roleID", testRoleUUID.String())
		w := httptest.NewRecorder()

		ctrl.listAssignedMenuCtrl(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Role not found") {
			t.Errorf("expected response to contain 'Role not found', got %s", w.Body.String())
		}
	})

	t.Run("should return 500 when service fails unexpectedly", func(t *testing.T) {
		mockSrv := &mockRoleService{
			GetAssignedMenusSrvcFunc: func(ctx context.Context, roleID uuid.UUID) ([]*models.RoleMenuResponse, errUtility.CustomError) {
				return nil, errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "internal database failure",
					MessageToSend: "Failed to get assigned menus",
				}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParam(http.MethodGet, "/api/role/list-assigned-menu/"+testRoleUUID.String(), "roleID", testRoleUUID.String())
		w := httptest.NewRecorder()

		ctrl.listAssignedMenuCtrl(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Internal Server Error") {
			t.Errorf("expected response to contain 'Internal Server Error', got %s", w.Body.String())
		}
	})

	t.Run("should return 200 with assigned menus list on success", func(t *testing.T) {
		expectedMenus := []*models.RoleMenuResponse{
			{
				Id:          "menu-1",
				Name:        "Dashboard",
				Description: "Dashboard menu",
				Status:      1,
				Children:    []*models.RoleMenuResponse{},
			},
			{
				Id:          "menu-2",
				Name:        "Settings",
				Description: "Settings menu",
				Status:      0,
				Children:    []*models.RoleMenuResponse{},
			},
		}
		mockSrv := &mockRoleService{
			GetAssignedMenusSrvcFunc: func(ctx context.Context, roleID uuid.UUID) ([]*models.RoleMenuResponse, errUtility.CustomError) {
				return expectedMenus, errUtility.CustomError{}
			},
		}
		ctrl := roleController{service: mockSrv}
		req := newRequestWithParam(http.MethodGet, "/api/role/list-assigned-menu/"+testRoleUUID.String(), "roleID", testRoleUUID.String())
		w := httptest.NewRecorder()

		ctrl.listAssignedMenuCtrl(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Assigned menus retrieved successfully") {
			t.Errorf("expected response to contain 'Assigned menus retrieved successfully', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "Dashboard") {
			t.Errorf("expected response to contain 'Dashboard', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"status":1`) {
			t.Errorf("expected response to contain '\"status\":1', got %s", w.Body.String())
		}
		if !strings.Contains(w.Body.String(), `"status":0`) {
			t.Errorf("expected response to contain '\"status\":0', got %s", w.Body.String())
		}
	})
}





