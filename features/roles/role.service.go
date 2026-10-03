package roles

import (
	"context"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"

	"github.com/google/uuid"
)

type rolePostgresRepoInterface interface {
	GetAll(ctx context.Context) ([]models.RoleResponse, errUtility.CustomError)
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
		nodes[flatRoles[i].Id] = &models.RoleResponse{
			Id:          flatRoles[i].Id,
			ParentId:    flatRoles[i].ParentId,
			Name:        flatRoles[i].Name,
			Description: flatRoles[i].Description,
			CreatedAt:   flatRoles[i].CreatedAt,
			UpdatedAt:   flatRoles[i].UpdatedAt,
			Children:    []*models.RoleResponse{},
		}
	}

	rootNodes := make([]*models.RoleResponse, 0)
	for i := range flatRoles {
		node := nodes[flatRoles[i].Id]
		if node.ParentId == nil || *node.ParentId == uuid.Nil {
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

func (s roleService) GetAllRolesSrvc(ctx context.Context) ([]*models.RoleResponse, errUtility.CustomError) {
	flatList, customErr := s.repo.GetAll(ctx)
	if customErr.Code > 0 {
		return []*models.RoleResponse{}, customErr
	}
	return buildRoleTree(flatList), errUtility.CustomError{}
}

func factoryRoleService(repo rolePostgresRepoInterface) roleService {
	if service.repo == nil {
		service = roleService{repo: repo}
	}
	return service
}
