package apis

import (
	"context"
	"net/http"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"

	"github.com/google/uuid"
)

type apiPostgresRepoInterface interface {
	GetAll(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError)
	Create(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError
	Update(ctx context.Context, apiID uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError
}

type apiService struct {
	repo apiPostgresRepoInterface
}

var service apiService

func (s apiService) GetAllSrvc(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
	return s.repo.GetAll(ctx, param)
}

func (s apiService) CreateApiSrvc(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError {
	if param.Id == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return errUtility.CustomError{
				Code:          http.StatusInternalServerError,
				Message:       err.Error(),
				MessageToSend: "Failed to create API",
				Function:      "apiService.CreateApiSrvc",
			}
		}
		param.Id = id
	}

	return s.repo.Create(ctx, param)
}

func (s apiService) UpdateApiSrvc(ctx context.Context, apiID uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError {
	return s.repo.Update(ctx, apiID, param)
}
func factoryApiService(repo apiPostgresRepoInterface) apiService {
	if service.repo == nil {
		service = apiService{repo: repo}
	}
	return service
}
