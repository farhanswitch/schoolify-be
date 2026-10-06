package menus

import (
	"context"
	"encoding/json"
	"net/http"

	models "schoolify_be/models"
	repoPostgres "schoolify_be/repositories/postgresql"
	utilities "schoolify_be/utilities"
	errUtility "schoolify_be/utilities/errors"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

type menuServiceInterface interface {
	GetAllMenusSrvc(ctx context.Context) ([]*models.MenuResponse, errUtility.CustomError)
	CreateMenuSrvc(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError
	GetByIDSrvc(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError)
	UpdateMenuSrvc(ctx context.Context, param models.UpdateMenuRequest) errUtility.CustomError
	AssignApiSrvc(ctx context.Context, menuID uuid.UUID, param models.AssignApiRequest) errUtility.CustomError
	DeleteSrvc(ctx context.Context, menuID uuid.UUID) errUtility.CustomError
}

type menuController struct {
	service menuServiceInterface
}

func (c menuController) assignApiCtrl(w http.ResponseWriter, r *http.Request) {
	reqID := utilities.GetReqID(r.Context())
	path := r.URL.Path
	menuIDStr := chi.URLParam(r, "menuID")
	if menuIDStr == "" {
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  map[string][]string{"menuID": {"menuID path parameter is required"}},
		})
		return
	}

	menuID, err := uuid.Parse(menuIDStr)
	if err != nil {
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  map[string][]string{"menuID": {"Invalid menuID"}},
		})
		return
	}

	var reqBody models.AssignApiRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utilities.Log(reqID, utilities.ERROR, path, "menuController.assignApiCtrl", nil, "json decode failed", err)
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  map[string][]string{"data": {"Invalid data"}},
		})
		return
	}
	if err := validate.Struct(reqBody); err != nil {
		errMap := errUtility.ParseError(err)
		utilities.Log(reqID, utilities.WARN, path, "menuController.assignApiCtrl", reqBody, "validation failed", errMap)
		utilities.WriteJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"message": "Validation failed",
			"errors":  errMap,
		})
		return
	}

	if errObj := c.service.AssignApiSrvc(r.Context(), menuID, reqBody); errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)
		logLevel := utilities.ERROR
		if statusCode < http.StatusInternalServerError {
			logLevel = utilities.WARN
		}
		utilities.Log(reqID, logLevel, path, "menuController.assignApiCtrl", reqBody, "service execution failed", errObj)
		if statusCode == http.StatusInternalServerError {
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to assign API",
				"errors":  "Internal Server Error",
			})
			return
		}
		errKey := errObj.Property
		if errKey == "" {
			errKey = "error"
		}
		utilities.WriteJSON(w, statusCode, map[string]interface{}{
			"message": "Failed to assign API",
			"errors":  map[string]string{errKey: errObj.MessageToSend},
		})
		return
	}

	utilities.Log(reqID, utilities.DEBUG, path, "menuController.assignApiCtrl", nil, "success", nil)
	utilities.WriteJSON(w, http.StatusCreated, map[string]string{"message": "List API assigned successfully"})
}

func (c menuController) GetByIDCtrl(w http.ResponseWriter, r *http.Request) {
	reqId := utilities.GetReqID(r.Context())
	path := r.URL.Path
	menuID := chi.URLParam(r, "menuID")
	utilities.Log(reqId, utilities.DEBUG, path, "menuController.GetByIDCtrl", map[string]string{"menuID": menuID}, "incoming request", nil)
	if menuID == "" {
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  map[string]string{"menuID": "menuID path parameter is required"},
		})
		return
	}
	menuUUID, err := uuid.Parse(menuID)
	if err != nil {
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors": map[string][]string{
				"menuID": []string{"Invalid menuID"},
			},
		})
		return
	}
	data, errObj := c.service.GetByIDSrvc(r.Context(), menuUUID)
	if errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)

		logLevel := utilities.ERROR
		if statusCode < 500 {
			logLevel = utilities.WARN
		}
		utilities.Log(reqId, logLevel, path, "menuController.GetByIDCtrl", map[string]string{"menuID": menuID}, "service execution failed", errObj)

		if statusCode == 500 {
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to get menu",
				"errors":  "Internal Server Error",
			})
			return
		} else {
			errKey := errObj.Property
			if errKey == "" {
				errKey = "error"
			}
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to get menu",
				"errors": map[string]string{
					errKey: errObj.MessageToSend,
				},
			})
			return
		}
	}
	utilities.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Menu retrieved successfully",
		"data":    data,
	})
}

func (c menuController) getAllMenusCtrl(w http.ResponseWriter, r *http.Request) {
	reqID := utilities.GetReqID(r.Context())
	path := r.URL.Path

	utilities.Log(reqID, utilities.DEBUG, path, "menuController.getAll", nil, "Incoming Request", nil)

	tree, errObj := c.service.GetAllMenusSrvc(r.Context())
	if errObj.Code > 0 {
		utilities.Log(reqID, utilities.ERROR, path, "getAllCtrl", nil, "service execution failed", errObj)
		utilities.WriteJSON(w, http.StatusInternalServerError, map[string]string{"errors": errObj.MessageToSend})
		return
	}
	utilities.Log(reqID, utilities.DEBUG, path, "getAllCtrl", nil, "success", nil)
	if tree == nil {
		tree = []*models.MenuResponse{}
	}
	utilities.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Menu list",
		"data":    tree,
	})
}

func (c menuController) createCtrl(w http.ResponseWriter, r *http.Request) {
	reqId := utilities.GetReqID(r.Context())
	path := r.URL.Path

	var reqBody models.CreateMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utilities.Log(reqId, utilities.ERROR, path, "menuController.createCtrl", nil, "json decode failed", err)
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors": map[string][]string{
				"data": []string{"Invalid data"},
			},
		})
		return
	}
	utilities.Log(reqId, utilities.DEBUG, path, "menuController.createCtrl", reqBody, "incoming request", nil)
	if err := validate.Struct(reqBody); err != nil {
		errMap := errUtility.ParseError(err)
		utilities.Log(reqId, utilities.WARN, path, "menuController.createCtrl", reqBody, "validation failed", errMap)
		utilities.WriteJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"message": "Validation failed",
			"errors":  errMap,
		})
		return
	}

	if errObj := c.service.CreateMenuSrvc(r.Context(), reqBody); errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)

		logLevel := utilities.ERROR
		if statusCode < 500 {
			logLevel = utilities.WARN
		}
		utilities.Log(reqId, logLevel, path, "menuController.createCtrl", reqBody, "service execution failed", errObj)

		if statusCode == 500 {
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to create menu",
				"errors":  "Internal Server Error",
			})
			return
		} else {
			errKey := errObj.Property
			if errKey == "" {
				errKey = "error"
			}
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to create menu",
				"errors": map[string]string{
					errKey: errObj.MessageToSend,
				},
			})
			return
		}
	}

	utilities.Log(reqId, utilities.DEBUG, path, "menuController.createCtrl", nil, "success", nil)
	utilities.WriteJSON(w, http.StatusCreated, map[string]string{
		"message": "Menu created successfully",
	})
}

func (c menuController) updateCtrl(w http.ResponseWriter, r *http.Request) {
	reqId := utilities.GetReqID(r.Context())
	path := r.URL.Path

	var reqBody models.UpdateMenuRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utilities.Log(reqId, utilities.ERROR, path, "menuController.updateCtrl", nil, "json decode failed", err)
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors": map[string][]string{
				"data": []string{"Invalid data"},
			},
		})
		return
	}
	utilities.Log(reqId, utilities.DEBUG, path, "menuController.updateCtrl", reqBody, "incoming request", nil)
	if err := validate.Struct(reqBody); err != nil {
		errMap := errUtility.ParseError(err)
		utilities.Log(reqId, utilities.WARN, path, "menuController.updateCtrl", reqBody, "validation failed", errMap)
		utilities.WriteJSON(w, http.StatusUnprocessableEntity, map[string]interface{}{
			"message": "Validation failed",
			"errors":  errMap,
		})
		return
	}

	if errObj := c.service.UpdateMenuSrvc(r.Context(), reqBody); errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)

		logLevel := utilities.ERROR
		if statusCode < 500 {
			logLevel = utilities.WARN
		}
		utilities.Log(reqId, logLevel, path, "menuController.updateCtrl", reqBody, "service execution failed", errObj)

		if statusCode == 500 {
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to update menu",
				"errors":  "Internal Server Error",
			})
			return
		} else {
			errKey := errObj.Property
			if errKey == "" {
				errKey = "error"
			}
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to update menu",
				"errors": map[string]string{
					errKey: errObj.MessageToSend,
				},
			})
			return
		}
	}

	utilities.Log(reqId, utilities.DEBUG, path, "menuController.updateCtrl", nil, "success", nil)
	utilities.WriteJSON(w, http.StatusCreated, map[string]string{
		"message": "Menu updated successfully",
	})
}

func (c menuController) DeleteCtrl(w http.ResponseWriter, r *http.Request) {
	reqId := utilities.GetReqID(r.Context())
	path := r.URL.Path
	menuID := chi.URLParam(r, "menuID")
	utilities.Log(reqId, utilities.DEBUG, path, "menuController.DeleteCtrl", map[string]string{"menuID": menuID}, "incoming request", nil)
	if menuID == "" {
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors":  map[string]string{"menuID": "menuID path parameter is required"},
		})
		return
	}
	menuUUID, err := uuid.Parse(menuID)
	if err != nil {
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"message": "Validation failed",
			"errors": map[string][]string{
				"menuID": []string{"Invalid menuID"},
			},
		})
		return
	}
	errObj := c.service.DeleteSrvc(r.Context(), menuUUID)
	if errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)

		logLevel := utilities.ERROR
		if statusCode < 500 {
			logLevel = utilities.WARN
		}
		utilities.Log(reqId, logLevel, path, "menuController.DeleteCtrl", map[string]string{"menuID": menuID}, "service execution failed", errObj)

		if statusCode == 500 {
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to delete menu",
				"errors":  "Internal Server Error",
			})
			return
		} else {
			errKey := errObj.Property
			if errKey == "" {
				errKey = "error"
			}
			utilities.WriteJSON(w, statusCode, map[string]interface{}{
				"message": "Failed to delete menu",
				"errors": map[string]string{
					errKey: errObj.MessageToSend,
				},
			})
			return
		}
	}
	utilities.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Menu deleted successfully",
	})
}

var controller menuController

func factoryMenuController(repo repoPostgres.MenuPostgresRepo) menuController {
	if controller.service == nil {
		controller = menuController{
			service: factoryMenuService(repo),
		}
	}
	return controller
}
