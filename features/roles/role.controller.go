package roles

import (
	"context"
	"net/http"
	models "schoolify_be/models"
	utilities "schoolify_be/utilities"

	repoPostgres "schoolify_be/repositories/postgresql"
	errUtility "schoolify_be/utilities/errors"
)

type roleServiceInterface interface {
	GetAllRolesSrvc(ctx context.Context) ([]*models.RoleResponse, errUtility.CustomError)
}

type roleController struct {
	service roleServiceInterface
}

func (c roleController) getAllRolesCtrl(w http.ResponseWriter, r *http.Request) {
	reqID := utilities.GetReqID(r.Context())
	path := r.URL.Path

	utilities.Log(reqID, utilities.DEBUG, path, "roleController.getAll", nil, "Incoming Request", nil)

	tree, errObj := c.service.GetAllRolesSrvc(r.Context())
	if errObj.Code > 0 {
		utilities.Log(reqID, utilities.ERROR, path, "getAllCtrl", nil, "service execution failed", errObj)
		utilities.WriteJSON(w, http.StatusInternalServerError, map[string]string{"errors": errObj.MessageToSend})
		return
	}
	utilities.Log(reqID, utilities.DEBUG, path, "getAllCtrl", nil, "success", nil)
	if tree == nil {
		tree = []*models.RoleResponse{}
	}
	utilities.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Roles retrieved successfully",
		"data":    tree,
	})
}

var controller roleController

func factoryRoleController(repo repoPostgres.RolePostgresRepo) roleController {
	if controller.service == nil {
		controller = roleController{
			service: factoryRoleService(repo),
		}
	}
	return controller
}
