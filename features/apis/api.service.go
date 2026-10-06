package apis

import (
	"context"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"
)

type apiPostgresRepoInterface interface {
	GetAll(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError)
}

type apiService struct {
	repo apiPostgresRepoInterface
}

var service apiService

func (s apiService) GetAllSrvc(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
	return s.repo.GetAll(ctx, param)
}
func factoryApiService(repo apiPostgresRepoInterface) apiService {
	if service.repo == nil {
		service = apiService{repo: repo}
	}
	return service
}
