package postgresql

import (
	"context"
	"database/sql"
	"net/http"

	connections "schoolify_be/connections"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"
	"strings"

	"github.com/google/uuid"
)

type RolePostgresRepo struct{}

var roleRepo RolePostgresRepo

func (r RolePostgresRepo) GetAll(ctx context.Context) ([]models.RoleResponse, errUtility.CustomError) {
	db := connections.DbPostgres()
	query := "SELECT id, parent_id, name, description, created_at, updated_at FROM roles"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []models.RoleResponse{}, errUtility.CustomError{}
		} else {
			return []models.RoleResponse{}, errUtility.CustomError{
				Code:     http.StatusInternalServerError,
				Message:  err.Error(),
				Function: "RolePostgresRepo.GetAll",
			}
		}
	}
	defer rows.Close()

	var list []models.RoleResponse
	for rows.Next() {
		var data models.RoleResponse
		var parentId, description, updatedAt sql.NullString

		err = rows.Scan(&data.ID, &parentId, &data.Name, &description, &data.CreatedAt, &updatedAt)
		if err != nil {
			return []models.RoleResponse{}, errUtility.CustomError{
				Code:     http.StatusInternalServerError,
				Message:  err.Error(),
				Function: "RolePostgresRepo.GetAll",
			}
		}

		if parentId.Valid && parentId.String != "" {
			if parsedUUID, err := uuid.Parse(parentId.String); err == nil {
				data.ParentID = &parsedUUID
			}
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
		return []models.RoleResponse{}, errUtility.CustomError{
			Code:     http.StatusInternalServerError,
			Message:  err.Error(),
			Function: "RolePostgresRepo.GetAll",
		}
	}

	return list, errUtility.CustomError{}
}
func (r RolePostgresRepo) Create(ctx context.Context, param models.CreateRoleRequest) errUtility.CustomError {
	db := connections.DbPostgres()

	var parentID any = nil
	if param.ParentID != uuid.Nil {
		parentID = param.ParentID
	}

	query := "INSERT INTO roles(id, name, description, parent_id) VALUES($1, $2, $3, $4);"
	_, err := db.ExecContext(ctx, query, param.ID, param.Name, param.Description, parentID)
	if err != nil {
		errStr := err.Error()

		if strings.Contains(errStr, "roles_parent_id_fkey") || strings.Contains(errStr, "violates foreign key constraint") {
			return errUtility.CustomError{
				Code:          http.StatusBadRequest,
				Message:       errStr,
				MessageToSend: "Parent role with the specified ID does not exist",
			}
		}

		if strings.Contains(errStr, "unique constraint") {
			return errUtility.CustomError{
				Code:          http.StatusConflict,
				Message:       errStr,
				MessageToSend: "Role name already exists",
			}
		}

		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       errStr,
			MessageToSend: "Failed create role",
		}
	}
	return errUtility.CustomError{}
}
func FactoryRoleRepo() RolePostgresRepo {
	if roleRepo == (RolePostgresRepo{}) {
		roleRepo = RolePostgresRepo{}
	}
	return roleRepo
}
