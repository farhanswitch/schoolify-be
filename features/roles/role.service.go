package roles

import (
	"context"
	"net/http"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"

	"github.com/google/uuid"
)

type rolePostgresRepoInterface interface {
	GetAll(ctx context.Context) ([]models.RoleResponse, errUtility.CustomError)
	Create(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError
}

type roleService struct {
	repo rolePostgresRepoInterface
}

var service roleService

// buildRoleTree converts a flat slice of roles into a hierarchical parent-child tree.
// Complexity is O(N) using map-based child aggregation.
func buildRoleTree(flatRoles []models.RoleResponse) []*models.RoleResponse {
	nodes := make(map[uuid.UUID]*models.RoleResponse, len(flatRoles))
	for i := range flatRoles {
		nodes[flatRoles[i].ID] = &models.RoleResponse{
			ID:          flatRoles[i].ID,
			ParentID:    flatRoles[i].ParentID,
			Name:        flatRoles[i].Name,
			Description: flatRoles[i].Description,
			CreatedAt:   flatRoles[i].CreatedAt,
			UpdatedAt:   flatRoles[i].UpdatedAt,
			Children:    []*models.RoleResponse{},
		}
	}

	rootNodes := make([]*models.RoleResponse, 0)
	for i := range flatRoles {
		node := nodes[flatRoles[i].ID]
		if node.ParentID == nil || *node.ParentID == uuid.Nil {
			rootNodes = append(rootNodes, node)
		} else {
			if parent, exists := nodes[*node.ParentID]; exists {
				parent.Children = append(parent.Children, node)
			} else {
				// Parent not found in flat list, fallback to treating as root
				rootNodes = append(rootNodes, node)
			}
		}
	}
	return rootNodes
}

func (s roleService) GetAllRolesSrvc(ctx context.Context) ([]*models.RoleResponse, errUtility.CustomError) {
	flatList, customErr := s.repo.GetAll(ctx)
	if customErr.Code > 0 {
		return []*models.RoleResponse{}, customErr
	}
	return buildRoleTree(flatList), errUtility.CustomError{}
}
func (s roleService) CreateRoleSrvc(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError {
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

func factoryRoleService(repo rolePostgresRepoInterface) roleService {
	if service.repo == nil {
		service = roleService{repo: repo}
	}
	return service
}
