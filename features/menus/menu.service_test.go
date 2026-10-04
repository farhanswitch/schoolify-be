package menus

import (
	"context"
	"net/http"
	"testing"

	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"

	"github.com/google/uuid"
)

type mockMenuRepo struct {
	GetAllFunc  func(ctx context.Context) ([]models.MenuResponse, errUtility.CustomError)
	CreateFunc  func(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError
	GetByIDFunc func(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError)
}

func (m *mockMenuRepo) GetAll(ctx context.Context) ([]models.MenuResponse, errUtility.CustomError) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(ctx)
	}
	return []models.MenuResponse{}, errUtility.CustomError{}
}

func (m *mockMenuRepo) Create(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, param)
	}
	return errUtility.CustomError{}
}

func (m *mockMenuRepo) GetByID(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, menuID)
	}
	return models.MenuResponse{}, errUtility.CustomError{}
}

func TestBuildMenuTree(t *testing.T) {
	t.Run("should return empty slice when input is empty", func(t *testing.T) {
		result := buildMenuTree([]models.MenuResponse{})
		if len(result) != 0 {
			t.Errorf("expected empty slice, got length %d", len(result))
		}
	})

	t.Run("should correctly build parent-child hierarchy", func(t *testing.T) {
		rootID1 := uuid.New().String()
		childID1 := uuid.New().String()
		grandchildID1 := uuid.New().String()
		rootID2 := uuid.New().String()

		flatMenus := []models.MenuResponse{
			{
				Id:       rootID1,
				Name:     "Settings",
				ParentId: nil,
			},
			{
				Id:       childID1,
				Name:     "User Management",
				ParentId: &rootID1,
			},
			{
				Id:       grandchildID1,
				Name:     "Permissions",
				ParentId: &childID1,
			},
			{
				Id:       rootID2,
				Name:     "Dashboard",
				ParentId: nil,
			},
		}

		tree := buildMenuTree(flatMenus)

		if len(tree) != 2 {
			t.Fatalf("expected 2 root nodes, got %d", len(tree))
		}

		// Temukan root Settings dan root Dashboard
		var settingsNode, dashboardNode *models.MenuResponse
		for _, node := range tree {
			if node.Id == rootID1 {
				settingsNode = node
			} else if node.Id == rootID2 {
				dashboardNode = node
			}
		}

		if settingsNode == nil {
			t.Fatalf("expected settings root node to be found")
		}
		if dashboardNode == nil {
			t.Fatalf("expected dashboard root node to be found")
		}

		if len(settingsNode.Children) != 1 {
			t.Fatalf("expected 1 child for settings, got %d", len(settingsNode.Children))
		}
		if settingsNode.Children[0].Id != childID1 {
			t.Errorf("expected child ID %s, got %s", childID1, settingsNode.Children[0].Id)
		}

		child := settingsNode.Children[0]
		if len(child.Children) != 1 {
			t.Fatalf("expected 1 grandchild, got %d", len(child.Children))
		}
		if child.Children[0].Id != grandchildID1 {
			t.Errorf("expected grandchild ID %s, got %s", grandchildID1, child.Children[0].Id)
		}

		if len(dashboardNode.Children) != 0 {
			t.Errorf("expected 0 children for dashboard, got %d", len(dashboardNode.Children))
		}
	})

	t.Run("should treat node with empty string parentId as root node", func(t *testing.T) {
		rootID := uuid.New().String()
		emptyParent := ""

		flatMenus := []models.MenuResponse{
			{
				Id:       rootID,
				Name:     "Root With Empty ParentId",
				ParentId: &emptyParent,
			},
		}

		tree := buildMenuTree(flatMenus)
		if len(tree) != 1 {
			t.Fatalf("expected 1 root node, got %d", len(tree))
		}
		if tree[0].Id != rootID {
			t.Errorf("expected root ID %s, got %s", rootID, tree[0].Id)
		}
	})

	t.Run("should fallback to root when parent is not found in flat list", func(t *testing.T) {
		missingParentID := uuid.New().String()
		menuID := uuid.New().String()

		flatMenus := []models.MenuResponse{
			{
				Id:       menuID,
				Name:     "Orphaned Menu",
				ParentId: &missingParentID,
			},
		}

		tree := buildMenuTree(flatMenus)
		if len(tree) != 1 {
			t.Fatalf("expected orphaned node to be placed at root, got %d roots", len(tree))
		}
		if tree[0].Id != menuID {
			t.Errorf("expected node ID %s, got %s", menuID, tree[0].Id)
		}
	})
}

func TestGetAllMenusSrvc(t *testing.T) {
	ctx := context.Background()

	t.Run("should return error when repository fails", func(t *testing.T) {
		mockRepo := &mockMenuRepo{
			GetAllFunc: func(ctx context.Context) ([]models.MenuResponse, errUtility.CustomError) {
				return []models.MenuResponse{}, errUtility.CustomError{
					Code:     http.StatusInternalServerError,
					Message:  "database error",
					Function: "MenuPostgresRepo.GetAll",
				}
			},
		}

		srv := menuService{repo: mockRepo}
		result, customErr := srv.GetAllMenusSrvc(ctx)

		if customErr.Code != http.StatusInternalServerError {
			t.Errorf("expected error code %d, got %d", http.StatusInternalServerError, customErr.Code)
		}
		if len(result) != 0 {
			t.Errorf("expected empty result on error, got %d items", len(result))
		}
	})

	t.Run("should return menu tree on success", func(t *testing.T) {
		rootID := uuid.New().String()
		childID := uuid.New().String()

		mockRepo := &mockMenuRepo{
			GetAllFunc: func(ctx context.Context) ([]models.MenuResponse, errUtility.CustomError) {
				return []models.MenuResponse{
					{
						Id:       rootID,
						Name:     "Dashboard",
						ParentId: nil,
					},
					{
						Id:       childID,
						Name:     "Analytics",
						ParentId: &rootID,
					},
				}, errUtility.CustomError{}
			},
		}

		srv := menuService{repo: mockRepo}
		result, customErr := srv.GetAllMenusSrvc(ctx)

		if customErr.Code != 0 {
			t.Fatalf("expected code 0, got %d", customErr.Code)
		}
		if len(result) != 1 {
			t.Fatalf("expected 1 root menu, got %d", len(result))
		}
		if len(result[0].Children) != 1 {
			t.Fatalf("expected 1 child menu, got %d", len(result[0].Children))
		}
		if result[0].Children[0].Id != childID {
			t.Errorf("expected child ID %s, got %s", childID, result[0].Children[0].Id)
		}
	})
}

func TestCreateMenuSrvc(t *testing.T) {
	ctx := context.Background()
	req := models.CreateMenuRequest{
		Name:        "Dashboard",
		Description: "Main dashboard menu",
	}

	t.Run("should return error when repository fails", func(t *testing.T) {
		mockRepo := &mockMenuRepo{
			CreateFunc: func(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:     http.StatusInternalServerError,
					Message:  "database error",
					Function: "MenuPostgresRepo.Create",
				}
			},
		}

		srv := menuService{repo: mockRepo}
		errObj := srv.CreateMenuSrvc(ctx, req)

		if errObj.Code != http.StatusInternalServerError {
			t.Errorf("expected error code %d, got %d", http.StatusInternalServerError, errObj.Code)
		}
	})

	t.Run("should return error when parent menu does not exist (foreign key constraint)", func(t *testing.T) {
		nonExistentParentID := uuid.New()
		reqWithParent := models.CreateMenuRequest{
			ParentID:    nonExistentParentID,
			Name:        "User Management",
			Description: "Sub menu under settings",
		}
		mockRepo := &mockMenuRepo{
			CreateFunc: func(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					Message:       "pq: insert or update on table \"menus\" violates foreign key constraint \"menus_parent_id_fkey\"",
					MessageToSend: "Parent menu with the specified ID does not exist",
					Property:      "parentID",
				}
			},
		}

		srv := menuService{repo: mockRepo}
		errObj := srv.CreateMenuSrvc(ctx, reqWithParent)

		if errObj.Code != http.StatusBadRequest {
			t.Errorf("expected error code %d, got %d", http.StatusBadRequest, errObj.Code)
		}
		if errObj.MessageToSend != "Parent menu with the specified ID does not exist" {
			t.Errorf("expected message 'Parent menu with the specified ID does not exist', got '%s'", errObj.MessageToSend)
		}
	})

	t.Run("should generate UUIDv7 and return success when repository succeeds", func(t *testing.T) {
		var capturedParam models.CreateMenuRequest
		mockRepo := &mockMenuRepo{
			CreateFunc: func(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
				capturedParam = param
				return errUtility.CustomError{}
			},
		}

		srv := menuService{repo: mockRepo}
		errObj := srv.CreateMenuSrvc(ctx, req)

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

func TestGetByIDSrvc(t *testing.T) {
	ctx := context.Background()
	menuID := uuid.New()

	t.Run("should return error when repository fails / menu not found", func(t *testing.T) {
		mockRepo := &mockMenuRepo{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (models.MenuResponse, errUtility.CustomError) {
				return models.MenuResponse{}, errUtility.CustomError{
					Code:          http.StatusNotFound,
					Message:       "sql: no rows in result set",
					MessageToSend: "Menu not found",
					Function:      "MenuPostgresRepo.GetByID",
				}
			},
		}

		srv := menuService{repo: mockRepo}
		result, errObj := srv.GetByIDSrvc(ctx, menuID)

		if errObj.Code != http.StatusNotFound {
			t.Errorf("expected error code %d, got %d", http.StatusNotFound, errObj.Code)
		}
		if errObj.MessageToSend != "Menu not found" {
			t.Errorf("expected message 'Menu not found', got '%s'", errObj.MessageToSend)
		}
		if result.Id != "" {
			t.Errorf("expected empty MenuResponse, got ID %s", result.Id)
		}
	})

	t.Run("should return menu when repository succeeds", func(t *testing.T) {
		expectedMenu := models.MenuResponse{
			Id:          menuID.String(),
			Name:        "Dashboard",
			Description: "Dashboard menu",
			CreatedAt:   "2026-10-04 13:00:00",
		}
		mockRepo := &mockMenuRepo{
			GetByIDFunc: func(ctx context.Context, id uuid.UUID) (models.MenuResponse, errUtility.CustomError) {
				return expectedMenu, errUtility.CustomError{}
			},
		}

		srv := menuService{repo: mockRepo}
		result, errObj := srv.GetByIDSrvc(ctx, menuID)

		if errObj.Code != 0 {
			t.Fatalf("expected error code 0, got %d", errObj.Code)
		}
		if result.Id != menuID.String() {
			t.Errorf("expected menu ID %s, got %s", menuID.String(), result.Id)
		}
		if result.Name != expectedMenu.Name {
			t.Errorf("expected menu name %s, got %s", expectedMenu.Name, result.Name)
		}
	})
}

