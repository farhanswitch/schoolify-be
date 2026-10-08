package postgresql

import (
	"context"
	"database/sql"
	"net/http"
	connections "schoolify_be/connections"
	models "schoolify_be/models"
	utility "schoolify_be/utilities"
	errUtility "schoolify_be/utilities/errors"
	"strings"

	"github.com/google/uuid"
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
	result, err := connections.DbPostgres().Query("SELECT u.id, u.username, u.password, r.id AS roleID, r.name AS roleName, COALESCE(u.first_name, ''), COALESCE(u.last_name, '') FROM users AS u JOIN roles AS r ON u.role_id = r.id WHERE u.username = $1 LIMIT 1;", user.Username)
	if err != nil {
		return models.UserLoginData{}, errUtility.CustomError{
			Code:          http.StatusBadRequest,
			Message:       err.Error(),
			MessageToSend: "Invalid username or password",
		}
	}

	if result.Next() {
		var data models.UserLoginData
		err := result.Scan(&data.ID, &data.Username, &data.Password, &data.RoleID, &data.RoleName, &data.FirstName, &data.LastName)
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

// GetAccessMenus returns the flat, assigned menu set for a user's current
// role. Tree construction and leaf endpoint aggregation are kept in the
// service so the repository stays responsible only for data retrieval.
func (UserPostgresRepo) GetAccessMenus(ctx context.Context, userID string) ([]models.UserAccessMenuResponse, errUtility.CustomError) {
	const query = `
		SELECT
			m.id, m.parent_id, m.name, m.description, m.sort_order,
			a.id, a.name, a.description, a.path, a.method, a.created_at, a.updated_at
		FROM users u
		JOIN role_menu rm ON rm.role_id = u.role_id
		JOIN menus m ON m.id = rm.menu_id
		LEFT JOIN menu_api ma ON ma.menu_id = m.id
		LEFT JOIN apis a ON a.id = ma.api_id
		WHERE u.id = $1
		ORDER BY m.parent_id NULLS FIRST, m.sort_order ASC, m.created_at ASC, a.created_at ASC;`

	rows, err := connections.DbPostgres().QueryContext(ctx, query, userID)
	if err != nil {
		return nil, errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed to get access menus",
			Function:      "UserPostgresRepo.GetAccessMenus",
		}
	}
	defer rows.Close()

	menusByID := make(map[string]*models.UserAccessMenuResponse)
	menuOrder := make([]string, 0)
	endpointIDsByMenu := make(map[string]map[uuid.UUID]struct{})

	for rows.Next() {
		var (
			menuID, menuName                                                                                                  string
			parentID, menuDescription                                                                                         sql.NullString
			sortOrder                                                                                                         int
			endpointID, endpointName, endpointDescription, endpointPath, endpointMethod, endpointCreatedAt, endpointUpdatedAt sql.NullString
		)
		if err := rows.Scan(
			&menuID, &parentID, &menuName, &menuDescription, &sortOrder,
			&endpointID, &endpointName, &endpointDescription, &endpointPath, &endpointMethod, &endpointCreatedAt, &endpointUpdatedAt,
		); err != nil {
			return nil, errUtility.CustomError{
				Code:          http.StatusInternalServerError,
				Message:       err.Error(),
				MessageToSend: "Failed to get access menus",
				Function:      "UserPostgresRepo.GetAccessMenus",
			}
		}

		menu, exists := menusByID[menuID]
		if !exists {
			menu = &models.UserAccessMenuResponse{
				ID:        menuID,
				Name:      menuName,
				SortOrder: sortOrder,
				Endpoints: []models.MenuApiResponse{},
				Children:  []*models.UserAccessMenuResponse{},
			}
			if parentID.Valid && parentID.String != "" {
				parent := parentID.String
				menu.ParentID = &parent
			}
			if menuDescription.Valid {
				menu.Description = menuDescription.String
			}
			menusByID[menuID] = menu
			menuOrder = append(menuOrder, menuID)
			endpointIDsByMenu[menuID] = make(map[uuid.UUID]struct{})
		}

		if endpointID.Valid {
			endpointUUID, parseErr := uuid.Parse(endpointID.String)
			if parseErr != nil {
				return nil, errUtility.CustomError{
					Code:          http.StatusInternalServerError,
					Message:       parseErr.Error(),
					MessageToSend: "Failed to get access menus",
					Function:      "UserPostgresRepo.GetAccessMenus",
				}
			}
			if _, duplicate := endpointIDsByMenu[menuID][endpointUUID]; !duplicate {
				endpointIDsByMenu[menuID][endpointUUID] = struct{}{}
				menu.Endpoints = append(menu.Endpoints, models.MenuApiResponse{
					Id:          endpointUUID,
					Name:        endpointName.String,
					Description: endpointDescription.String,
					Path:        endpointPath.String,
					Method:      endpointMethod.String,
					CreatedAt:   endpointCreatedAt.String,
					UpdatedAt:   endpointUpdatedAt.String,
					Status:      1,
				})
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed to get access menus",
			Function:      "UserPostgresRepo.GetAccessMenus",
		}
	}

	flatMenus := make([]models.UserAccessMenuResponse, 0, len(menuOrder))
	for _, menuID := range menuOrder {
		flatMenus = append(flatMenus, *menusByID[menuID])
	}
	return flatMenus, errUtility.CustomError{}
}

func FactoryUserRepo() UserPostgresRepo {
	if userRepo == (UserPostgresRepo{}) {
		userRepo = UserPostgresRepo{}
	}
	return userRepo
}
