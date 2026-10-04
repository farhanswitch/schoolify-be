package postgresql

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	connections "schoolify_be/connections"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"

	"github.com/google/uuid"
)

type MenuPostgresRepo struct{}

var menuRepo MenuPostgresRepo

func (r MenuPostgresRepo) GetAll(ctx context.Context) ([]models.MenuResponse, errUtility.CustomError) {
	db := connections.DbPostgres()
	query := "SELECT id, parent_id, name, description, created_at, updated_at FROM menus"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []models.MenuResponse{}, errUtility.CustomError{}
		} else {
			return []models.MenuResponse{}, errUtility.CustomError{
				Code:     http.StatusInternalServerError,
				Message:  err.Error(),
				Function: "MenuPostgresRepo.GetAll",
			}
		}
	}
	defer rows.Close()

	var list []models.MenuResponse
	for rows.Next() {
		var data models.MenuResponse
		var parentId, description, updatedAt sql.NullString

		err = rows.Scan(&data.Id, &parentId, &data.Name, &description, &data.CreatedAt, &updatedAt)
		if err != nil {
			return []models.MenuResponse{}, errUtility.CustomError{
				Code:     http.StatusInternalServerError,
				Message:  err.Error(),
				Function: "MenuPostgresRepo.GetAll",
			}
		}

		if parentId.Valid && parentId.String != "" {
			pId := parentId.String
			data.ParentId = &pId
		}
		if description.Valid {
			data.Description = description.String
		}
		if updatedAt.Valid {
			data.UpdatedAt = updatedAt.String
		}

		list = append(list, data)
	}

	if err := rows.Err(); err != nil {
		return []models.MenuResponse{}, errUtility.CustomError{
			Code:     http.StatusInternalServerError,
			Message:  err.Error(),
			Function: "MenuPostgresRepo.GetAll",
		}
	}

	return list, errUtility.CustomError{}
}

func (r MenuPostgresRepo) Create(ctx context.Context, param models.CreateMenuRequest) errUtility.CustomError {
	db := connections.DbPostgres()

	var parentID any = nil
	if param.ParentID != uuid.Nil {
		parentID = param.ParentID
	}

	query := "INSERT INTO menus(id, name, description, parent_id) VALUES($1, $2, $3, $4);"
	_, err := db.ExecContext(ctx, query, param.ID, param.Name, param.Description, parentID)
	if err != nil {
		errStr := err.Error()

		if strings.Contains(errStr, "menus_parent_id_fkey") || strings.Contains(errStr, "violates foreign key constraint") {
			return errUtility.CustomError{
				Code:          http.StatusBadRequest,
				Message:       errStr,
				MessageToSend: "Parent menu with the specified ID does not exist",
				Property:      "parentID",
				Function:      "MenuPostgresRepo.Create",
			}
		}

		if strings.Contains(errStr, "unique constraint") {
			return errUtility.CustomError{
				Code:          http.StatusConflict,
				Message:       errStr,
				MessageToSend: "Menu name already exists",
				Property:      "name",
				Function:      "MenuPostgresRepo.Create",
			}
		}

		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       errStr,
			MessageToSend: "Failed create menu",
			Function:      "MenuPostgresRepo.Create",
		}
	}
	return errUtility.CustomError{}
}

func (r MenuPostgresRepo) GetByID(ctx context.Context, menuID uuid.UUID) (models.MenuResponse, errUtility.CustomError) {
	db := connections.DbPostgres()
	var data models.MenuResponse
	var parentId, description, updatedAt sql.NullString
	query := "SELECT id, parent_id, name, description, created_at, updated_at FROM menus WHERE id=$1;"
	err := db.QueryRowContext(ctx, query, menuID).Scan(&data.Id, &parentId, &data.Name, &description, &data.CreatedAt, &updatedAt)

	if err != nil {
		return models.MenuResponse{}, errUtility.CustomError{
			Code:          http.StatusNotFound,
			Message:       err.Error(),
			MessageToSend: "Menu not found",
			Function:      "MenuPostgresRepo.GetByID",
			Property:      "id",
		}
	}
	if parentId.Valid && parentId.String != "" {
		pId := parentId.String
		data.ParentId = &pId
	}
	if description.Valid {
		data.Description = description.String
	}
	if updatedAt.Valid {
		data.UpdatedAt = updatedAt.String
	}
	return data, errUtility.CustomError{}
}

func (r MenuPostgresRepo) Update(ctx context.Context, param models.UpdateMenuRequest) errUtility.CustomError {
	db := connections.DbPostgres()

	var parentID any = nil
	if param.ParentID != uuid.Nil {
		parentID = param.ParentID
	}

	query := "UPDATE menus SET name=$1, description=$2, parent_id=$3 WHERE id=$4;"
	res, err := db.ExecContext(ctx, query, param.Name, param.Description, parentID, param.ID)
	if err != nil {
		errStr := err.Error()

		if strings.Contains(errStr, "menus_parent_id_fkey") || strings.Contains(errStr, "violates foreign key constraint") {
			return errUtility.CustomError{
				Code:          http.StatusBadRequest,
				Message:       errStr,
				MessageToSend: "Parent menu with the specified ID does not exist",
				Property:      "parentID",
				Function:      "MenuPostgresRepo.Update",
			}
		}

		if strings.Contains(errStr, "unique constraint") {
			return errUtility.CustomError{
				Code:          http.StatusConflict,
				Message:       errStr,
				MessageToSend: "Menu name already exists",
				Property:      "name",
				Function:      "MenuPostgresRepo.Update",
			}
		}

		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       errStr,
			MessageToSend: "Failed update menu",
			Function:      "MenuPostgresRepo.Update",
		}
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed update menu",
			Function:      "MenuPostgresRepo.Update",
		}
	} else {
		if rowsAffected == 0 {
			return errUtility.CustomError{
				Code:          http.StatusNotFound,
				MessageToSend: "Menu not found",
				Function:      "MenuPostgresRepo.Update",
				Property:      "id",
			}
		}
	}
	return errUtility.CustomError{}
}

func FactoryMenuRepo() MenuPostgresRepo {
	if menuRepo == (MenuPostgresRepo{}) {
		menuRepo = MenuPostgresRepo{}
	}
	return menuRepo
}
