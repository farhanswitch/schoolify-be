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
	err := connections.DbPostgres().QueryRow("SELECT id, role_id, username, email, first_name, last_name, password, status, created_at, updated_at FROM public.users WHERE username=$1;", param.Username).Scan(&user.ID, &user.RoleID, &user.Username, &user.Email, &user.FirstName, &user.LastName, &user.Password, &user.Status, &user.CreatedAt, &nullableUpdatedAt)
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
	user.StrCreatedAt = utility.FormatTimeMillis(user.CreatedAt)
	if nullableUpdatedAt.Valid {
		user.UpdatedAt = nullableUpdatedAt.Time
		user.StrUpdatedAt = utility.FormatTimeMillis(nullableUpdatedAt.Time)
	} else {
		user.UpdatedAt = user.CreatedAt
		user.StrUpdatedAt = utility.FormatTimeMillis(user.CreatedAt)
	}
	return user, errUtility.CustomError{}
}
func (UserPostgresRepo) RegisterUser(user models.UserRegisterRequest) (bool, errUtility.CustomError) {
	_, err := connections.DbPostgres().Exec("INSERT INTO users(id, role_id, username, email, first_name, last_name, password) VALUES($1, $2, $3, $4, $5, $6, $7)", user.ID, user.RoleID, user.Username, user.Email, user.FirstName, user.LastName, user.Password)
	if err != nil {
		var customErr errUtility.CustomError

		if strings.Contains(err.Error(), "unique") {
			if strings.Contains(err.Error(), "username") {
				customErr = errUtility.CustomError{
					Code:          http.StatusUnprocessableEntity,
					Message:       err.Error(),
					MessageToSend: "The username has already been taken.",
				}
			} else if strings.Contains(err.Error(), "email") {
				customErr = errUtility.CustomError{
					Code:          http.StatusUnprocessableEntity,
					Message:       err.Error(),
					MessageToSend: "This email address is already registered. Please log in.",
				}
			} else {
				customErr = errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       err.Error(),
					MessageToSend: "Duplicate data.",
				}
			}
		} else {
			customErr = errUtility.CustomError{
				Code:          http.StatusInternalServerError,
				Message:       err.Error(),
				MessageToSend: "Internal server error.",
			}
		}
		return true, customErr
	}
	return false, errUtility.CustomError{}
}
func (UserPostgresRepo) UserLogin(user models.UserLoginRequest) (models.UserLoginData, errUtility.CustomError) {
	result, err := connections.DbPostgres().Query("SELECT u.id, u.username, u.password, r.id AS roleID, r.name AS roleName FROM users AS u JOIN roles AS r ON u.role_id = r.id WHERE u.username = $1 LIMIT 1;", user.Username)
	if err != nil {
		return models.UserLoginData{}, errUtility.CustomError{
			Code:          http.StatusBadRequest,
			Message:       err.Error(),
			MessageToSend: "Invalid username or password",
		}
	}

	if result.Next() {
		var data models.UserLoginData
		err := result.Scan(&data.ID, &data.Username, &data.Password, &data.RoleID, &data.RoleName)
		if err != nil {
			return models.UserLoginData{}, errUtility.CustomError{
				Code:          http.StatusBadRequest,
				Message:       err.Error(),
				MessageToSend: "Invalid username or password",
			}
		}
		return data, errUtility.CustomError{}
	}
	return models.UserLoginData{}, errUtility.CustomError{
		Code:          http.StatusBadRequest,
		Message:       "User not found",
		MessageToSend: "Invalid username or password",
	}
}
func FactoryUserRepo() UserPostgresRepo {
	if userRepo == (UserPostgresRepo{}) {
		userRepo = UserPostgresRepo{}
	}
	return userRepo
}
