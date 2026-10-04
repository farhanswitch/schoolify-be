package menus

import (
	"context"
	"encoding/json"
	"net/http"

	models "schoolify_be/models"
	repoPostgres "schoolify_be/repositories/postgresql"
	utilities "schoolify_be/utilities"
	errUtility "schoolify_be/utilities/errors"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
}

type menuServiceInterface interface {
	CreateMenuSrvc(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError
}

type menuController struct {
	service menuServiceInterface
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
		utilities.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
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

var controller menuController

func factoryMenuController(repo repoPostgres.MenuPostgresRepo) menuController {
	if controller.service == nil {
		controller = menuController{
			service: factoryMenuService(repo),
		}
	}
	return controller
}
