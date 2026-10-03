package roles

import (
	"context"
	"encoding/json"
	"net/http"
	models "schoolify_be/models"
	utilities "schoolify_be/utilities"

	repoPostgres "schoolify_be/repositories/postgresql"
	errUtility "schoolify_be/utilities/errors"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

type roleServiceInterface interface {
	GetAllRolesSrvc(ctx context.Context) ([]*models.RoleResponse, errUtility.CustomError)
	CreateRoleSrvc(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError
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
func (c roleController) createCtrl(w http.ResponseWriter, r *http.Request) {
	reqId := utilities.GetReqID(r.Context())
	path := r.URL.Path

	var reqBody models.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utilities.Log(reqId, utilities.ERROR, path, "roleController.createCtrl", nil, "json decode failed", err)
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors": map[string][]string{
				"payload": []string{"Invalid payload"},
			},
		})
		return
	}
	utilities.Log(reqId, utilities.DEBUG, path, "createCtrl", reqBody, "incoming request", nil)
	if err := validate.Struct(reqBody); err != nil {
		errMap := errUtility.ParseError(err)
		utilities.Log(reqId, utilities.WARN, path, "createCtrl", reqBody, "validation failed", errMap)
		utilities.WriteJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"message": "Validation failed",
			"errors":  errMap,
		})
		return
	}

	if errObj := c.service.CreateRoleSrvc(r.Context(), reqBody); errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)

		logLevel := utilities.ERROR
		if statusCode < 500 {
			logLevel = utilities.WARN
		}
		utilities.Log(reqId, logLevel, path, "createCtrl", reqBody, "service execution failed", errObj)

		if statusCode == 500 {
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to create role",
				"errors":  "Internal Server Error",
			})
			return
		} else {
			errKey := errObj.Property
			if errKey == "" {
				errKey = "error"
			}
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to create role",
				"errors": map[string]string{
					errKey: errObj.MessageToSend,
				},
			})
			return
		}
	}

	utilities.Log(reqId, utilities.DEBUG, path, "createCtrl", nil, "success", nil)
	utilities.WriteJSON(w, http.StatusCreated, map[string]string{
		"message": "Role created successfully",
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
