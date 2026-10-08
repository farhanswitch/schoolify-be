package users

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	models "schoolify_be/models"
	repoPostgres "schoolify_be/repositories/postgresql"
	utilities "schoolify_be/utilities"
	"unicode"

	errUtility "schoolify_be/utilities/errors"

	"github.com/go-playground/validator/v10"
)

type userServiceInterface interface {
	GetUserDataByUsernameSrvc(param models.UserLoginRequest) (models.User, errUtility.CustomError)
	RegisterUserSrvc(param models.UserRegisterRequest) (bool, errUtility.CustomError)
	UserLogin(param models.UserLoginRequest) (map[string]any, errUtility.CustomError)
	LogoutSrvc(ctx context.Context, username string) errUtility.CustomError
	GetAccessMenusSrvc(ctx context.Context, userID string) ([]*models.UserAccessMenuResponse, errUtility.CustomError)
}

func (c userController) GetAccessMenusCtrl(w http.ResponseWriter, r *http.Request) {
	userID := utilities.GetUserID(r.Context())
	if userID == "" {
		utilities.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	data, errObj := c.service.GetAccessMenusSrvc(r.Context(), userID)
	if errObj.Code > 0 {
		errObj.Compile()
		statusCode := int(errObj.Code)
		if statusCode < http.StatusBadRequest {
			statusCode = http.StatusInternalServerError
		}
		utilities.WriteJSON(w, statusCode, map[string]interface{}{
			"message": "Failed to get access menus",
			"errors":  errObj.MessageToSend,
		})
		return
	}

	if data == nil {
		data = []*models.UserAccessMenuResponse{}
	}
	utilities.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Access menus retrieved successfully",
		"data":    data,
	})
}

func (c userController) LogoutCtrl(w http.ResponseWriter, r *http.Request) {
	username := utilities.GetUsername(r.Context())
	if username == "" {
		utilities.WriteJSON(w, http.StatusUnauthorized, map[string]string{"message": "Unauthorized"})
		return
	}

	if errObj := c.service.LogoutSrvc(r.Context(), username); errObj.Code > 0 {
		utilities.Log(utilities.ERROR, r.URL.Path, "LogoutCtrl", username, "Logout failed: "+errObj.Message, errObj)
		utilities.WriteJSON(w, http.StatusInternalServerError, map[string]string{"message": "Failed to logout"})
		return
	}

	utilities.WriteJSON(w, http.StatusOK, map[string]string{"message": "Logout successful"})
}

type userController struct {
	service userServiceInterface
}

var controller userController

func passwordValidation(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	var hasUpper, hasNumber, hasSpecial bool

	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsNumber(ch):
			hasNumber = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasSpecial = true
		}
	}

	return hasUpper && hasNumber && hasSpecial
}
func (c userController) RegisterUserCtrl(w http.ResponseWriter, r *http.Request) {
	var param models.UserRegisterRequest
	w.Header().Set("Content-Type", "application/json")
	err := json.NewDecoder(r.Body).Decode(&param)
	if err != nil {
		utilities.Log(utilities.WARN, r.URL.Path, "RegisterUserCtrl", r.Body, "Invalid JSON format: "+err.Error(), nil)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"message":"Invalid Data"}`)
		return
	}
	val := validator.New()
	val.RegisterValidation("password", passwordValidation)
	err = val.Struct(param)
	if err != nil {
		objError := errUtility.ParseError(err)
		strError, _ := json.Marshal(objError)
		// Warning log for validation error
		utilities.Log(utilities.WARN, r.URL.Path, "RegisterUserCtrl", param, "Validation error: "+string(strError), objError)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"errors":%s}`, strError)
		return
	}
	isErr, errObj := c.service.RegisterUserSrvc(param)
	if isErr {
		utilities.Log(utilities.ERROR, r.URL.Path, "UserLoginCtrl", param, "Service error: "+errObj.Message, errObj)
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"message":"%s"}`, errObj.MessageToSend)
		return
	}
	w.WriteHeader(http.StatusCreated)
	fmt.Fprintf(w, `{"message":"User created successfully"}`)
}
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
	userData, errObj := a.service.UserLogin(param)
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
	if controller.service == nil {
		controller = userController{
			service: factoryUserService(repo),
		}
	}
	return controller
}
