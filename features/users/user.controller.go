package users

import (
	"encoding/json"
	"fmt"
	"net/http"
	models "schoolify_be/models"
	repoPostgres "schoolify_be/repositories/postgresql"
	utilities "schoolify_be/utilities"
	errUtility "schoolify_be/utilities/errors"

	"github.com/go-playground/validator/v10"
)

type userController struct {
	service userService
}

var controller userController

func (a userController) UserLoginCtrl(w http.ResponseWriter, r *http.Request) {
	var param models.UserLoginRequest
	w.Header().Set("Content-Type", "application/json")

	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		utilities.Log(utilities.WARN, r.URL.Path, "UserLoginCtrl", r.Body, "Invalid JSON format: "+err.Error(), nil)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"message":"Invalid Data"}`)
		return
	}

	val := validator.New()
	err = val.Struct(param)
	if err != nil {
		objError := errUtility.ParseError(err)
		strError, _ := json.Marshal(objError)
		// Warning log for validation error
		utilities.Log(utilities.WARN, r.URL.Path, "UserLoginCtrl", param, "Validation error: "+string(strError), objError)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"errors":%s}`, strError)
		return
	}
	userData, errObj := a.service.GetUserDataByUsernameSrvc(param)
	if errObj.Code > 200 {
		errObj.Compile()
		// Error log for service error
		utilities.Log(utilities.ERROR, r.URL.Path, "UserLoginCtrl", param, "Service error: "+errObj.Message, errObj)
		w.WriteHeader(int(errObj.Code))
		fmt.Fprintf(w, `{"message":"%s"}`, errObj.MessageToSend)
		return
	}
	strRes, err := json.Marshal(map[string]any{
		"data":    userData,
		"message": "Success login",
	})
	if err != nil {
		utilities.Log(utilities.ERROR, r.URL.Path, "UserLoginCtrl", nil, err.Error(), nil)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"message":"Internal Server Error"}`)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(strRes)

}

func factoryUserController(repo repoPostgres.UserPostgresRepo) userController {
	if controller == (userController{}) {
		controller = userController{
			service: factoryUserService(repo),
		}
	}
	return controller
}
