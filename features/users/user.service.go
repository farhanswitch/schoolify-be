package users

import (
	"context"
	"fmt"
	"schoolify_be/configs"
	models "schoolify_be/models"
	utilities "schoolify_be/utilities"
	custom_errors "schoolify_be/utilities/errors"
	errUtility "schoolify_be/utilities/errors"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/square/go-jose.v2/jwt"
)

type userPostgresRepoInterface interface {
	GetUserByUsername(param models.UserLoginRequest) (models.User, errUtility.CustomError)
	RegisterUser(user models.UserRegisterRequest) (bool, errUtility.CustomError)
	UserLogin(user models.UserLoginRequest) (models.UserLoginData, errUtility.CustomError)
}

type redisUtilityInterface interface {
	SaveValue(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	DeleteValue(ctx context.Context, key string) error
	GetValue(ctx context.Context, key string) (string, error)
}

type userService struct {
	repo  userPostgresRepoInterface
	redis redisUtilityInterface
}

var service userService

func (s userService) GetUserDataByUsernameSrvc(param models.UserLoginRequest) (models.User, errUtility.CustomError) {

	user, customErr := s.repo.GetUserByUsername(param)
	utilities.Log(utilities.INFO, "/api/auth/login", "GetUserDataByUsernameSrvc", param.Username, "Log Data", user)
	return user, customErr
}
func (s userService) RegisterUserSrvc(param models.UserRegisterRequest) (bool, errUtility.CustomError) {
	strID, err := uuid.NewV7()
	if err != nil {
		utilities.Log(utilities.ERROR, "/api/auth/register", "RegisterUserSrvc", param, "Error generating userID: "+err.Error(), err)
		return true, errUtility.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Internal Server Error",
		}
	}
	strRoleID, err := uuid.NewV7()
	if err != nil {
		utilities.Log(utilities.ERROR, "/api/auth/register", "RegisterUserSrvc", param, "Error generating userRoleID: "+err.Error(), err)
		return true, errUtility.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Internal Server Error",
		}
	}
	param.ID = strID
	param.RoleID = strRoleID

	if string(param.ConfirmPassword) != string(param.Password) {

		return true, errUtility.CustomError{
			Code:          400,
			Message:       "Password and Confirm Password must match!",
			MessageToSend: "Password and Confirm Password must match!",
		}
	}
	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(param.Password), bcrypt.DefaultCost)
	if err != nil {
		utilities.Log(utilities.ERROR, "/api/auth/register", "RegisterUserSrvc", param, "Error hashing password: "+err.Error(), err)
		return true, custom_errors.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Internal Server Error",
		}
	}
	param.Password = string(hashedPassword)
	return s.repo.RegisterUser(param)

}

func (s userService) UserLogin(param models.UserLoginRequest) (map[string]any, errUtility.CustomError) {
	loginData, errObj := s.repo.UserLogin(param)
	if errObj.Code > 0 {
		return map[string]any{}, errObj
	}
	err := bcrypt.CompareHashAndPassword([]byte(loginData.Password), []byte(param.Password))
	if err != nil {
		utilities.Log(utilities.ERROR, "/api/auth/login", "UserLoginSrvc", param, "Error bcrypt compare: "+err.Error(), err)
		return map[string]any{}, custom_errors.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Invalid username or password",
		}
	}
	claims := map[string]any{
		"username": loginData.Username,
		"id":       loginData.ID,
		"roleID":   loginData.RoleID,
		"roleName": loginData.RoleName,
		"Issuer":   "Schoolify",
		"Expiry":   jwt.NewNumericDate(time.Now().Add(time.Duration(configs.GetConfig().Service.SessionTime) * time.Second)),
		"IssuedAt": jwt.NewNumericDate(time.Now()),
	}
	strToken, err := utilities.JWEEncryptAES(claims, []byte(configs.GetConfig().Service.EncryptKey))
	if err != nil {
		return map[string]any{}, custom_errors.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Invalid username or password",
		}
	}
	redisUtil := s.redis
	if redisUtil == nil {
		redisUtil = utilities.RedisInstance()
	}

	err = redisUtil.DeleteValue(context.Background(), fmt.Sprintf("TOKEN_%s_1", loginData.Username))
	if err != nil {
		return map[string]any{}, custom_errors.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Invalid username or password"}
	}

	err = redisUtil.DeleteValue(context.Background(), fmt.Sprintf("TOKEN_%s_2", loginData.Username))
	if err != nil {
		return map[string]any{}, custom_errors.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Invalid username or password"}
	}
	err = redisUtil.SaveValue(context.Background(), fmt.Sprintf("TOKEN_%s_1", loginData.Username), strToken, time.Duration(configs.GetConfig().Service.SessionTime)*time.Second)
	if err != nil {
		return map[string]any{}, custom_errors.CustomError{
			Code:          500,
			Message:       err.Error(),
			MessageToSend: "Invalid username or password"}
	}
	claims["token"] = strToken

	return claims, errObj

}
func factoryUserService(repo userPostgresRepoInterface) userService {
	if service.repo == nil {
		service = userService{repo: repo, redis: utilities.RedisInstance()}
	}
	return service
}
