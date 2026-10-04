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
	CreateFunc func(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError
}

func (m *mockMenuRepo) Create(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, param)
	}
	return errUtility.CustomError{}
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
