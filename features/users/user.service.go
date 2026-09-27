package users

import (
	models "schoolify_be/models"
	repoPostgres "schoolify_be/repositories/postgresql"
	utilities "schoolify_be/utilities"
	errUtility "schoolify_be/utilities/errors"
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
func factoryUserService(repo repoPostgres.UserPostgresRepo) userService {
	if service == (userService{}) {
		service = userService{repo: repo}
	}
	return service
}
