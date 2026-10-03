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
	GetAllFunc func(ctx context.Context) ([]models.RoleResponse, errUtility.CustomError)
}

func (m *mockRoleRepo) GetAll(ctx context.Context) ([]models.RoleResponse, errUtility.CustomError) {
	if m.GetAllFunc != nil {
		return m.GetAllFunc(ctx)
	}
	return []models.RoleResponse{}, errUtility.CustomError{}
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
				Id:       rootID1,
				Name:     "Super Admin",
				ParentId: nil,
			},
			{
				Id:       childID1,
				Name:     "Admin School",
				ParentId: &rootID1,
			},
			{
				Id:       grandchildID1,
				Name:     "Teacher",
				ParentId: &childID1,
			},
			{
				Id:       rootID2,
				Name:     "Guest",
				ParentId: nil,
			},
		}

		tree := buildRoleTree(flatRoles)

		if len(tree) != 2 {
			t.Fatalf("expected 2 root nodes, got %d", len(tree))
		}

		// First root
		if tree[0].Id != rootID1 {
			t.Errorf("expected first root ID %s, got %s", rootID1, tree[0].Id)
		}
		if len(tree[0].Children) != 1 {
			t.Fatalf("expected 1 child for root 1, got %d", len(tree[0].Children))
		}
		if tree[0].Children[0].Id != childID1 {
			t.Errorf("expected child ID %s, got %s", childID1, tree[0].Children[0].Id)
		}

		// Grandchild
		child := tree[0].Children[0]
		if len(child.Children) != 1 {
			t.Fatalf("expected 1 child for child 1, got %d", len(child.Children))
		}
		if child.Children[0].Id != grandchildID1 {
			t.Errorf("expected grandchild ID %s, got %s", grandchildID1, child.Children[0].Id)
		}

		// Second root
		if tree[1].Id != rootID2 {
			t.Errorf("expected second root ID %s, got %s", rootID2, tree[1].Id)
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
				Id:       rootID,
				Name:     "Root With Nil ParentId",
				ParentId: &nilUUID,
			},
		}

		tree := buildRoleTree(flatRoles)
		if len(tree) != 1 {
			t.Fatalf("expected 1 root node, got %d", len(tree))
		}
		if tree[0].Id != rootID {
			t.Errorf("expected root ID %s, got %s", rootID, tree[0].Id)
		}
	})

	t.Run("should fallback to root when parent is not found in flat list", func(t *testing.T) {
		missingParentID := uuid.New()
		roleID := uuid.New()

		flatRoles := []models.RoleResponse{
			{
				Id:       roleID,
				Name:     "Orphaned Role",
				ParentId: &missingParentID,
			},
		}

		tree := buildRoleTree(flatRoles)
		if len(tree) != 1 {
			t.Fatalf("expected orphaned node to be placed at root, got %d roots", len(tree))
		}
		if tree[0].Id != roleID {
			t.Errorf("expected node ID %s, got %s", roleID, tree[0].Id)
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
						Id:       rootID,
						Name:     "Admin",
						ParentId: nil,
					},
					{
						Id:       childID,
						Name:     "Staff",
						ParentId: &rootID,
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
		if result[0].Children[0].Id != childID {
			t.Errorf("expected child ID %s, got %s", childID, result[0].Children[0].Id)
		}
	})
}
