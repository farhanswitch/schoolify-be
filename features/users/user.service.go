package users

import (
	models "schoolify_be/models"
	repoPostgres "schoolify_be/repositories/postgresql"
	utilities "schoolify_be/utilities"
	custom_errors "schoolify_be/utilities/errors"
	errUtility "schoolify_be/utilities/errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type userService struct {
	repo repoPostgres.UserPostgresRepo
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
func factoryUserService(repo repoPostgres.UserPostgresRepo) userService {
	if service == (userService{}) {
		service = userService{repo: repo}
	}
	return service
}
