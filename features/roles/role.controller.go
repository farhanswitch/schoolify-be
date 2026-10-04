package roles

import (
	"context"
	"encoding/json"
	"net/http"
	models "schoolify_be/models"
	utilities "schoolify_be/utilities"

	repoPostgres "schoolify_be/repositories/postgresql"
	errUtility "schoolify_be/utilities/errors"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

type roleServiceInterface interface {
	GetAllRolesSrvc(ctx context.Context) ([]*models.RoleResponse, errUtility.CustomError)
	CreateRoleSrvc(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError
	UpdateRoleSrvc(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError
	GetByIDSrvc(ctx context.Context, roleID uuid.UUID) (models.RoleResponse, errUtility.CustomError)
	DeleteSrvc(ctx context.Context, roleID uuid.UUID) errUtility.CustomError
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
func (c roleController) GetByIDCtrl(w http.ResponseWriter, r *http.Request) {
	reqId := utilities.GetReqID(r.Context())
	path := r.URL.Path
	roleID := chi.URLParam(r, "roleID")
	utilities.Log(reqId, utilities.DEBUG, path, "roleController.GetByIDCtrl", map[string]string{"roleID": roleID}, "incoming request", nil)
	if roleID == "" {
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  map[string]string{"roleID": "roleID path parameter is required"},
		})
		return
	}
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors": map[string][]string{
				"roleID": []string{"Invalid roleID"},
			},
		})
		return
	}
	data, errObj := c.service.GetByIDSrvc(r.Context(), roleUUID)
	if errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)

		logLevel := utilities.ERROR
		if statusCode < 500 {
			logLevel = utilities.WARN
		}
		utilities.Log(reqId, logLevel, path, "roleController.GetByIDCtrl", map[string]string{"roleID": roleID}, "service execution failed", errObj)

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
				"message": "Failed to get role",
				"errors": map[string]string{
					errKey: errObj.MessageToSend,
				},
			})
			return
		}
	}
	utilities.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Role retrieved successfully",
		"data":    data,
	})

}
func (c roleController) DeleteCtrl(w http.ResponseWriter, r *http.Request) {
	reqId := utilities.GetReqID(r.Context())
	path := r.URL.Path
	roleID := chi.URLParam(r, "roleID")
	utilities.Log(reqId, utilities.DEBUG, path, "roleController.DeleteCtrl", map[string]string{"roleID": roleID}, "incoming request", nil)
	if roleID == "" {
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  map[string]string{"roleID": "roleID path parameter is required"},
		})
		return
	}
	roleUUID, err := uuid.Parse(roleID)
	if err != nil {
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors": map[string][]string{
				"roleID": []string{"Invalid roleID"},
			},
		})
		return
	}
	errObj := c.service.DeleteSrvc(r.Context(), roleUUID)
	if errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)

		logLevel := utilities.ERROR
		if statusCode < 500 {
			logLevel = utilities.WARN
		}
		utilities.Log(reqId, logLevel, path, "roleController.DeleteCtrl", map[string]string{"roleID": roleID}, "service execution failed", errObj)

		if statusCode == 500 {
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to delete role",
				"errors":  "Internal Server Error",
			})
			return
		} else {
			errKey := errObj.Property
			if errKey == "" {
				errKey = "error"
			}
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to delete role",
				"errors": map[string]string{
					errKey: errObj.MessageToSend,
				},
			})
			return
		}
	}
	utilities.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Role deleted successfully",
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
				"data": []string{"Invalid data"},
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
func (c roleController) updateCtrl(w http.ResponseWriter, r *http.Request) {
	reqId := utilities.GetReqID(r.Context())
	path := r.URL.Path

	var reqBody models.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utilities.Log(reqId, utilities.ERROR, path, "roleController.updateCtrl", nil, "json decode failed", err)
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors": map[string][]string{
				"data": []string{"Invalid data"},
			},
		})
		return
	}
	utilities.Log(reqId, utilities.DEBUG, path, "roleController.updateCtrl", reqBody, "incoming request", nil)
	if err := validate.Struct(reqBody); err != nil {
		errMap := errUtility.ParseError(err)
		utilities.Log(reqId, utilities.WARN, path, "roleController.updateCtrl", reqBody, "validation failed", errMap)
		utilities.WriteJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"message": "Validation failed",
			"errors":  errMap,
		})
		return
	}

	if errObj := c.service.UpdateRoleSrvc(r.Context(), reqBody); errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)

		logLevel := utilities.ERROR
		if statusCode < 500 {
			logLevel = utilities.WARN
		}
		utilities.Log(reqId, logLevel, path, "roleController.updateCtrl", reqBody, "service execution failed", errObj)

		if statusCode == 500 {
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to update role",
				"errors":  "Internal Server Error",
			})
			return
		} else {
			errKey := errObj.Property
			if errKey == "" {
				errKey = "error"
			}
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to update role",
				"errors": map[string]string{
					errKey: errObj.MessageToSend,
				},
			})
			return
		}
	}

	utilities.Log(reqId, utilities.DEBUG, path, "roleController.updateCtrl", nil, "success", nil)
	utilities.WriteJSON(w, http.StatusCreated, map[string]string{
		"message": "Role updated successfully",
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
