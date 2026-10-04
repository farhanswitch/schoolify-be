package menus

import (
	"context"
	"net/http"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"

	"github.com/google/uuid"
)

type menuPostgresRepoInterface interface {
	Create(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError
}

type menuService struct {
	repo menuPostgresRepoInterface
}

var service menuService

func (s menuService) CreateMenuSrvc(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
	id, err := uuid.NewV7()
	if err != nil {
		return errUtility.CustomError{
			Code:    http.StatusInternalServerError,
			Message: "Failed generate UUID",
		}
	}
	param.ID = id
	return s.repo.Create(ctx, param)
}

func factoryMenuService(repo menuPostgresRepoInterface) menuService {
	if service.repo == nil {
		service = menuService{repo: repo}
	}
	return service
}
