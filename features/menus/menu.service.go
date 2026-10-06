package menus

import (
	"context"
	"net/http"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"

	"github.com/google/uuid"
)

type menuPostgresRepoInterface interface {
	GetAll(ctx context.Context) ([]models.MenuResponse, errUtility.CustomError)
	Create(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError
	Update(ctx context.Context, param models.UpdateMenuRequest) errUtility.CustomError
	AssignApi(ctx context.Context, menuID uuid.UUID, param models.AssignApiRequest) errUtility.CustomError
	GetAssignedEndpoints(ctx context.Context, menuID uuid.UUID) ([]models.MenuApiResponse, errUtility.CustomError)
	GetByID(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError)
	Delete(ctx context.Context, menuID uuid.UUID) errUtility.CustomError
}

type menuService struct {
	repo menuPostgresRepoInterface
}

var service menuService

type menuResponse = models.MenuResponse

// buildMenuTree converts a flat slice of menus into a hierarchical parent-child tree.
// Complexity is O(N) using a map-based child aggregation.
func buildMenuTree(flatMenus []menuResponse) []*menuResponse {
	nodes := make(map[string]*menuResponse)
	for i := range flatMenus {
		nodes[flatMenus[i].Id] = &menuResponse{
			Id:          flatMenus[i].Id,
			ParentId:    flatMenus[i].ParentId,
			Name:        flatMenus[i].Name,
			Description: flatMenus[i].Description,
			CreatedAt:   flatMenus[i].CreatedAt,
			UpdatedAt:   flatMenus[i].UpdatedAt,
			Children:    []*menuResponse{},
		}
	}

	var rootNodes []*menuResponse
	for _, node := range nodes {
		if node.ParentId == nil || *node.ParentId == "" {
			rootNodes = append(rootNodes, node)
		} else {
			if parent, exists := nodes[*node.ParentId]; exists {
				parent.Children = append(parent.Children, node)
			} else {
				// Parent not found in flat list, fallback to treating as root
				rootNodes = append(rootNodes, node)
			}
		}
	}
	return rootNodes
}

func (s menuService) GetAllMenusSrvc(ctx context.Context) ([]*menuResponse, errUtility.CustomError) {
	flatList, customErr := s.repo.GetAll(ctx)
	if customErr.Code > 0 {
		return []*menuResponse{}, customErr
	}
	return buildMenuTree(flatList), errUtility.CustomError{}
}

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

func (s menuService) GetByIDSrvc(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError) {
	return s.repo.GetByID(ctx, menuID)
}

func (s menuService) UpdateMenuSrvc(ctx context.Context, param models.UpdateMenuRequest) errUtility.CustomError {
	return s.repo.Update(ctx, param)
}

func (s menuService) AssignApiSrvc(ctx context.Context, menuID uuid.UUID, param models.AssignApiRequest) errUtility.CustomError {
	return s.repo.AssignApi(ctx, menuID, param)
}

func (s menuService) GetAssignedEndpointsSrvc(ctx context.Context, menuID uuid.UUID) ([]models.MenuApiResponse, errUtility.CustomError) {
	return s.repo.GetAssignedEndpoints(ctx, menuID)
}

func (s menuService) DeleteSrvc(ctx context.Context, menuID uuid.UUID) errUtility.CustomError {
	return s.repo.Delete(ctx, menuID)
}

func factoryMenuService(repo menuPostgresRepoInterface) menuService {
	if service.repo == nil {
		service = menuService{repo: repo}
	}
	return service
}
