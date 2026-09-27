package postgresql

import (
	"database/sql"
	"net/http"
	connections "schoolify_be/connections"
	models "schoolify_be/models"
	utility "schoolify_be/utilities"
	errUtility "schoolify_be/utilities/errors"
	"strings"
)

type UserPostgresRepo struct{}

var userRepo UserPostgresRepo

func (UserPostgresRepo) GetUserByUsername(param models.UserLoginRequest) (models.User, errUtility.CustomError) {
	var user models.User
	var nullableUpdatedAt sql.NullTime
	err := connections.DbPostgres().QueryRow("SELECT id, role_id, username, email, first_name, last_name, password, status, created, updated FROM public.users WHERE username=$1;", param.Username).Scan(&user.ID, &user.RoleID, &user.Username, &user.Email, &user.FirstName, &user.LastName, &user.Password, &user.Status, &nullableUpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, errUtility.CustomError{
				Code:          http.StatusNotFound,
				Message:       "User not found",
				MessageToSend: "User not found",
				Function:      "UserPostgresRepo.GetUserByUsername",
			}
		} else if strings.Contains(err.Error(), "invalid input syntax for type uuid") {
			return models.User{}, errUtility.CustomError{
				Code:          http.StatusNotFound,
				Message:       "Invalid UUID",
				MessageToSend: "User not found",
				Function:      "UserPostgresRepo.GetUserByUsername",
			}
		}

		return models.User{}, errUtility.CustomError{
			Code:     http.StatusInternalServerError,
			Message:  err.Error(),
			Function: "UserPostgresRepo.GetUserByUsername",
		}
	}
	user.StrCreated = utility.FormatTimeMillis(user.Created)
	if nullableUpdatedAt.Valid {
		user.Updated = nullableUpdatedAt.Time
		user.StrUpdated = utility.FormatTimeMillis(nullableUpdatedAt.Time)
	} else {
		user.Updated = user.Created
		user.StrUpdated = utility.FormatTimeMillis(user.Created)
	}
	return user, errUtility.CustomError{}
}

func FactoryUserRepo() UserPostgresRepo {
	if userRepo == (UserPostgresRepo{}) {
		userRepo = UserPostgresRepo{}
	}
	return userRepo
}
