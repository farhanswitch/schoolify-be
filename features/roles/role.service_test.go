package roles

import (
	"context"
	"net/http"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"
	"testing"

	"github.com/google/uuid"
)

type mockRoleRepo struct {
	GetAllFunc  func(ctx context.Context) ([]models.RoleResponse, errUtility.CustomError)
	CreateFunc  func(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError
	UpdateFunc  func(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError
	GetByIDFunc func(ctx context.Context, roleID uuid.UUID) (models.RoleResponse, errUtility.CustomError)
	DeleteFunc     func(ctx context.Context, roleID uuid.UUID) errUtility.CustomError
	AssignMenuFunc func(ctx context.Context, roleID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError
}

func (m *mockRoleRepo) GetAll(ctx context.Context) ([]models.RoleResponse, errUtility.CustomError) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(ctx)
	}
	return []models.RoleResponse{}, errUtility.CustomError{}
}

func (m *mockRoleRepo) Create(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, param)
	}
	return errUtility.CustomError{}
}

func (m *mockRoleRepo) Update(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, param)
	}
	return errUtility.CustomError{}
}

func (m *mockRoleRepo) GetByID(ctx context.Context, roleID uuid.UUID) (models.RoleResponse, errUtility.CustomError) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, roleID)
	}
	return models.RoleResponse{}, errUtility.CustomError{}
}

func (m *mockRoleRepo) Delete(ctx context.Context, roleID uuid.UUID) errUtility.CustomError {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, roleID)
	}
	return errUtility.CustomError{}
}

func (m *mockRoleRepo) AssignMenu(ctx context.Context, roleID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError {
	if m.AssignMenuFunc != nil {
		return m.AssignMenuFunc(ctx, roleID, param)
	}
	return errUtility.CustomError{}
}


func TestBuildRoleTree(t *testing.T) {
	t.Run("should return empty slice when input is empty", func(t *testing.T) {
		result := buildRoleTree([]models.RoleResponse{})
		if len(result) != 0 {
			t.Errorf("expected empty slice, got length %d", len(result))
		}
	})

	t.Run("should correctly build parent-child hierarchy", func(t *testing.T) {
		rootID1 := uuid.New()
		childID1 := uuid.New()
		grandchildID1 := uuid.New()
		rootID2 := uuid.New()

		flatRoles := []models.RoleResponse{
			{
				ID:       rootID1,
				Name:     "Super Admin",
				ParentID: nil,
			},
			{
				ID:       childID1,
				Name:     "Admin School",
				ParentID: &rootID1,
			},
			{
				ID:       grandchildID1,
				Name:     "Teacher",
				ParentID: &childID1,
			},
			{
				ID:       rootID2,
				Name:     "Guest",
				ParentID: nil,
			},
		}

		tree := buildRoleTree(flatRoles)

		if len(tree) != 2 {
			t.Fatalf("expected 2 root nodes, got %d", len(tree))
		}

		// First root
		if tree[0].ID != rootID1 {
			t.Errorf("expected first root ID %s, got %s", rootID1, tree[0].ID)
		}
		if len(tree[0].Children) != 1 {
			t.Fatalf("expected 1 child for root 1, got %d", len(tree[0].Children))
		}
		if tree[0].Children[0].ID != childID1 {
			t.Errorf("expected child ID %s, got %s", childID1, tree[0].Children[0].ID)
		}

		// Grandchild
		child := tree[0].Children[0]
		if len(child.Children) != 1 {
			t.Fatalf("expected 1 child for child 1, got %d", len(child.Children))
		}
		if child.Children[0].ID != grandchildID1 {
			t.Errorf("expected grandchild ID %s, got %s", grandchildID1, child.Children[0].ID)
		}

		// Second root
		if tree[1].ID != rootID2 {
			t.Errorf("expected second root ID %s, got %s", rootID2, tree[1].ID)
		}
		if len(tree[1].Children) != 0 {
			t.Errorf("expected 0 children for root 2, got %d", len(tree[1].Children))
		}
	})

	t.Run("should treat node with uuid.Nil parent as root node", func(t *testing.T) {
		rootID := uuid.New()
		nilUUID := uuid.Nil

		flatRoles := []models.RoleResponse{
			{
				ID:       rootID,
				Name:     "Root With Nil ParentId",
				ParentID: &nilUUID,
			},
		}

		tree := buildRoleTree(flatRoles)
		if len(tree) != 1 {
			t.Fatalf("expected 1 root node, got %d", len(tree))
		}
		if tree[0].ID != rootID {
			t.Errorf("expected root ID %s, got %s", rootID, tree[0].ID)
		}
	})

	t.Run("should fallback to root when parent is not found in flat list", func(t *testing.T) {
		missingParentID := uuid.New()
		roleID := uuid.New()

		flatRoles := []models.RoleResponse{
			{
				ID:       roleID,
				Name:     "Orphaned Role",
				ParentID: &missingParentID,
			},
		}

		tree := buildRoleTree(flatRoles)
		if len(tree) != 1 {
			t.Fatalf("expected orphaned node to be placed at root, got %d roots", len(tree))
		}
		if tree[0].ID != roleID {
			t.Errorf("expected node ID %s, got %s", roleID, tree[0].ID)
		}
	})
}

func TestGetAllRolesSrvc(t *testing.T) {
	ctx := context.Background()

	t.Run("should return error when repository fails", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			GetAllFunc: func(ctx context.Context) ([]models.RoleResponse, errUtility.CustomError) {
				return []models.RoleResponse{}, errUtility.CustomError{
					Code:     http.StatusInternalServerError,
					Message:  "database error",
					Function: "RolePostgresRepo.GetAll",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		result, customErr := srv.GetAllRolesSrvc(ctx)

		if customErr.Code != http.StatusInternalServerError {
			t.Errorf("expected error code %d, got %d", http.StatusInternalServerError, customErr.Code)
		}
		if len(result) != 0 {
			t.Errorf("expected empty result on error, got %d items", len(result))
		}
	})

	t.Run("should return role tree on success", func(t *testing.T) {
		rootID := uuid.New()
		childID := uuid.New()

		mockRepo := &mockRoleRepo{
			GetAllFunc: func(ctx context.Context) ([]models.RoleResponse, errUtility.CustomError) {
				return []models.RoleResponse{
					{
						ID:       rootID,
						Name:     "Admin",
						ParentID: nil,
					},
					{
						ID:       childID,
						Name:     "Staff",
						ParentID: &rootID,
					},
				}, errUtility.CustomError{}
			},
		}

		srv := roleService{repo: mockRepo}
		result, customErr := srv.GetAllRolesSrvc(ctx)

		if customErr.Code != 0 {
			t.Fatalf("expected code 0, got %d", customErr.Code)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 root role, got %d", len(result))
		}
		if len(result[0].Children) != 1 {
			t.Fatalf("expected 1 child role, got %d", len(result[0].Children))
		}
		if result[0].Children[0].ID != childID {
			t.Errorf("expected child ID %s, got %s", childID, result[0].Children[0].ID)
		}
	})
}

func TestCreateRoleSrvc(t *testing.T) {
	ctx := context.Background()
	req := models.CreateRoleRequest{
		Name:        "Teacher",
		Description: "Teacher role",
	}

	t.Run("should return error when repository fails", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			CreateFunc: func(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:     http.StatusInternalServerError,
					Message:  "database error",
					Function: "RolePostgresRepo.Create",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.CreateRoleSrvc(ctx, req)

		if errObj.Code != http.StatusInternalServerError {
			t.Errorf("expected error code %d, got %d", http.StatusInternalServerError, errObj.Code)
		}
	})

	t.Run("should return error when parent role does not exist (foreign key constraint)", func(t *testing.T) {
		nonExistentParentID := uuid.New()
		reqWithParent := models.CreateRoleRequest{
			ParentID:    nonExistentParentID,
			Name:        "Assistant Teacher",
			Description: "Assistant teacher role",
		}
		mockRepo := &mockRoleRepo{
			CreateFunc: func(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					Message:       "pq: insert or update on table \"roles\" violates foreign key constraint \"roles_parent_id_fkey\"",
					MessageToSend: "Parent role with the specified ID does not exist",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.CreateRoleSrvc(ctx, reqWithParent)

		if errObj.Code != http.StatusBadRequest {
			t.Errorf("expected error code %d, got %d", http.StatusBadRequest, errObj.Code)
		}
		if errObj.MessageToSend != "Parent role with the specified ID does not exist" {
			t.Errorf("expected message 'Parent role with the specified ID does not exist', got '%s'", errObj.MessageToSend)
		}
	})

	t.Run("should generate UUIDv7 and return success when repository succeeds", func(t *testing.T) {
		var capturedParam models.CreateRoleRequest
		mockRepo := &mockRoleRepo{
			CreateFunc: func(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError {
				capturedParam = param
				return errUtility.CustomError{}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.CreateRoleSrvc(ctx, req)

		if errObj.Code != 0 {
			t.Fatalf("expected error code 0, got %d", errObj.Code)
		}
		if capturedParam.ID == uuid.Nil {
			t.Errorf("expected generated ID to not be uuid.Nil")
		}
		if capturedParam.ID.Version() != 7 {
			t.Errorf("expected generated ID to be UUIDv7, got version %d", capturedParam.ID.Version())
		}
		if capturedParam.Name != req.Name {
			t.Errorf("expected name %s, got %s", req.Name, capturedParam.Name)
		}
	})
}

func TestUpdateRoleSrvc(t *testing.T) {
	ctx := context.Background()
	req := models.UpdateRoleRequest{
		ID:          uuid.New(),
		Name:        "Senior Teacher",
		Description: "Updated teacher role",
	}

	t.Run("should return error when repository fails", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			UpdateFunc: func(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "database error",
					MessageToSend: "Failed update role",
					Function:      "RolePostgresRepo.Update",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.UpdateRoleSrvc(ctx, req)

		if errObj.Code != http.StatusInternalServerError {
			t.Errorf("expected error code %d, got %d", http.StatusInternalServerError, errObj.Code)
		}
		if errObj.MessageToSend != "Failed update role" {
			t.Errorf("expected message 'Failed update role', got '%s'", errObj.MessageToSend)
		}
	})

	t.Run("should return error when parent role does not exist (foreign key constraint)", func(t *testing.T) {
		reqWithParent := models.UpdateRoleRequest{
			ID:          req.ID,
			ParentID:    uuid.New(),
			Name:        "Assistant Teacher",
			Description: "Assistant teacher role",
		}
		mockRepo := &mockRoleRepo{
			UpdateFunc: func(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					Message:       "pq: insert or update on table \"roles\" violates foreign key constraint \"roles_parent_id_fkey\"",
					MessageToSend: "Parent role with the specified ID does not exist",
					Property:      "parentID",
					Function:      "RolePostgresRepo.Update",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.UpdateRoleSrvc(ctx, reqWithParent)

		if errObj.Code != http.StatusBadRequest {
			t.Errorf("expected error code %d, got %d", http.StatusBadRequest, errObj.Code)
		}
		if errObj.Property != "parentID" {
			t.Errorf("expected property 'parentID', got '%s'", errObj.Property)
		}
	})

	t.Run("should return error when role name already exists (unique constraint)", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			UpdateFunc: func(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusConflict,
					Message:       "unique constraint",
					MessageToSend: "Role name already exists",
					Property:      "name",
					Function:      "RolePostgresRepo.Update",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.UpdateRoleSrvc(ctx, req)

		if errObj.Code != http.StatusConflict {
			t.Errorf("expected error code %d, got %d", http.StatusConflict, errObj.Code)
		}
		if errObj.Property != "name" {
			t.Errorf("expected property 'name', got '%s'", errObj.Property)
		}
	})

	t.Run("should return error when role to update is not found (404)", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			UpdateFunc: func(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusNotFound,
					MessageToSend: "Role not found",
					Function:      "RolePostgresRepo.Update",
					Property:      "id",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.UpdateRoleSrvc(ctx, req)

		if errObj.Code != http.StatusNotFound {
			t.Errorf("expected error code %d, got %d", http.StatusNotFound, errObj.Code)
		}
		if errObj.MessageToSend != "Role not found" {
			t.Errorf("expected message 'Role not found', got '%s'", errObj.MessageToSend)
		}
		if errObj.Property != "id" {
			t.Errorf("expected property 'id', got '%s'", errObj.Property)
		}
	})

	t.Run("should return success when repository succeeds", func(t *testing.T) {
		var capturedParam models.UpdateRoleRequest
		mockRepo := &mockRoleRepo{
			UpdateFunc: func(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
				capturedParam = param
				return errUtility.CustomError{}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.UpdateRoleSrvc(ctx, req)

		if errObj.Code != 0 {
			t.Fatalf("expected error code 0, got %d", errObj.Code)
		}
		if capturedParam.ID != req.ID {
			t.Errorf("expected ID %s, got %s", req.ID, capturedParam.ID)
		}
		if capturedParam.Name != req.Name {
			t.Errorf("expected name %s, got %s", req.Name, capturedParam.Name)
		}
	})
}

func TestGetByIDSrvc(t *testing.T) {
	ctx := context.Background()
	roleID := uuid.New()

	t.Run("should return error when repository fails / role not found", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (models.RoleResponse, errUtility.CustomError) {
				return models.RoleResponse{}, errUtility.CustomError{
					Code:          http.StatusNotFound,
					Message:       "sql: no rows in result set",
					MessageToSend: "Role not found",
					Function:      "RolePostgresRepo.GetByID",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		result, errObj := srv.GetByIDSrvc(ctx, roleID)

		if errObj.Code != http.StatusNotFound {
			t.Errorf("expected error code %d, got %d", http.StatusNotFound, errObj.Code)
		}
		if errObj.MessageToSend != "Role not found" {
			t.Errorf("expected message 'Role not found', got '%s'", errObj.MessageToSend)
		}
		if result.ID != uuid.Nil {
			t.Errorf("expected empty RoleResponse, got ID %s", result.ID)
		}
	})

	t.Run("should return role when repository succeeds", func(t *testing.T) {
		expectedRole := models.RoleResponse{
			ID:          roleID,
			Name:        "Admin",
			Description: "Administrator role",
			CreatedAt:   "2026-10-04 12:00:00",
		}
		mockRepo := &mockRoleRepo{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (models.RoleResponse, errUtility.CustomError) {
				return expectedRole, errUtility.CustomError{}
			},
		}

		srv := roleService{repo: mockRepo}
		result, errObj := srv.GetByIDSrvc(ctx, roleID)

		if errObj.Code != 0 {
			t.Fatalf("expected error code 0, got %d", errObj.Code)
		}
		if result.ID != roleID {
			t.Errorf("expected role ID %s, got %s", roleID, result.ID)
		}
		if result.Name != expectedRole.Name {
			t.Errorf("expected role name %s, got %s", expectedRole.Name, result.Name)
		}
	})
}

func TestDeleteSrvc(t *testing.T) {
	ctx := context.Background()
	roleID := uuid.New()

	t.Run("should return error when repository fails", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			DeleteFunc: func(ctx context.Context, id uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "database error",
					MessageToSend: "Failed delete role",
					Function:      "RolePostgresRepo.Delete",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.DeleteSrvc(ctx, roleID)

		if errObj.Code != http.StatusInternalServerError {
			t.Errorf("expected error code %d, got %d", http.StatusInternalServerError, errObj.Code)
		}
		if errObj.MessageToSend != "Failed delete role" {
			t.Errorf("expected message 'Failed delete role', got '%s'", errObj.MessageToSend)
		}
	})

	t.Run("should return error when role has child roles", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			DeleteFunc: func(ctx context.Context, id uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					Message:       "cannot delete role that has child roles",
					MessageToSend: "Cannot delete role because it has child roles",
					Property:      "id",
					Function:      "RolePostgresRepo.Delete",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.DeleteSrvc(ctx, roleID)

		if errObj.Code != http.StatusBadRequest {
			t.Errorf("expected error code %d, got %d", http.StatusBadRequest, errObj.Code)
		}
		if errObj.MessageToSend != "Cannot delete role because it has child roles" {
			t.Errorf("expected message 'Cannot delete role because it has child roles', got '%s'", errObj.MessageToSend)
		}
		if errObj.Property != "id" {
			t.Errorf("expected property 'id', got '%s'", errObj.Property)
		}
	})

	t.Run("should return error when role to delete is not found", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			DeleteFunc: func(ctx context.Context, id uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusNotFound,
					MessageToSend: "Role not found",
					Property:      "id",
					Function:      "RolePostgresRepo.Delete",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.DeleteSrvc(ctx, roleID)

		if errObj.Code != http.StatusNotFound {
			t.Errorf("expected error code %d, got %d", http.StatusNotFound, errObj.Code)
		}
		if errObj.MessageToSend != "Role not found" {
			t.Errorf("expected message 'Role not found', got '%s'", errObj.MessageToSend)
		}
	})

	t.Run("should return success when repository succeeds", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			DeleteFunc: func(ctx context.Context, id uuid.UUID) errUtility.CustomError {
				return errUtility.CustomError{}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.DeleteSrvc(ctx, roleID)

		if errObj.Code != 0 {
			t.Fatalf("expected error code 0, got %d", errObj.Code)
		}
	})
}

func TestAssignMenuSrvc(t *testing.T) {
	ctx := context.Background()
	roleID := uuid.New()
	menuID := uuid.New()
	req := models.AssignMenuRequest{
		MenuIds: []uuid.UUID{menuID},
	}

	t.Run("should return error when repository fails", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			AssignMenuFunc: func(ctx context.Context, rID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       "internal database failure",
					MessageToSend: "Failed to assign menu",
					Function:      "RolePostgresRepo.AssignMenu",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.AssignMenuSrvc(ctx, roleID, req)

		if errObj.Code != http.StatusInternalServerError {
			t.Errorf("expected error code %d, got %d", http.StatusInternalServerError, errObj.Code)
		}
		if errObj.MessageToSend != "Failed to assign menu" {
			t.Errorf("expected message 'Failed to assign menu', got '%s'", errObj.MessageToSend)
		}
	})

	t.Run("should return 404 when role does not exist", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			AssignMenuFunc: func(ctx context.Context, rID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusNotFound,
					MessageToSend: "Role not found",
					Property:      "roleID",
					Function:      "RolePostgresRepo.AssignMenu",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.AssignMenuSrvc(ctx, roleID, req)

		if errObj.Code != http.StatusNotFound {
			t.Errorf("expected error code %d, got %d", http.StatusNotFound, errObj.Code)
		}
		if errObj.MessageToSend != "Role not found" {
			t.Errorf("expected message 'Role not found', got '%s'", errObj.MessageToSend)
		}
		if errObj.Property != "roleID" {
			t.Errorf("expected property 'roleID', got '%s'", errObj.Property)
		}
	})

	t.Run("should return 400 when menu ID does not exist in database", func(t *testing.T) {
		mockRepo := &mockRoleRepo{
			AssignMenuFunc: func(ctx context.Context, rID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					MessageToSend: "Menu with ID " + menuID.String() + " does not exist",
					Property:      "listID",
					Function:      "RolePostgresRepo.AssignMenu",
				}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.AssignMenuSrvc(ctx, roleID, req)

		if errObj.Code != http.StatusBadRequest {
			t.Errorf("expected error code %d, got %d", http.StatusBadRequest, errObj.Code)
		}
		if errObj.Property != "listID" {
			t.Errorf("expected property 'listID', got '%s'", errObj.Property)
		}
	})

	t.Run("should return success when repository succeeds", func(t *testing.T) {
		var capturedRoleID uuid.UUID
		var capturedParam models.AssignMenuRequest
		mockRepo := &mockRoleRepo{
			AssignMenuFunc: func(ctx context.Context, rID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError {
				capturedRoleID = rID
				capturedParam = param
				return errUtility.CustomError{}
			},
		}

		srv := roleService{repo: mockRepo}
		errObj := srv.AssignMenuSrvc(ctx, roleID, req)

		if errObj.Code != 0 {
			t.Fatalf("expected error code 0, got %d", errObj.Code)
		}
		if capturedRoleID != roleID {
			t.Errorf("expected roleID %s, got %s", roleID, capturedRoleID)
		}
		if len(capturedParam.MenuIds) != 1 || capturedParam.MenuIds[0] != menuID {
			t.Errorf("expected param with menuID %s, got %+v", menuID, capturedParam)
		}
	})
}




