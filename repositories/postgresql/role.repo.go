package postgresql

import (
	"context"
	"database/sql"
	"fmt"
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
				Property:      "parentID",
				Function:      "RolePostgresRepo.Create",
			}
		}

		if strings.Contains(errStr, "unique constraint") {
			return errUtility.CustomError{
				Code:          http.StatusConflict,
				Message:       errStr,
				MessageToSend: "Role name already exists",
				Property:      "name",
				Function:      "RolePostgresRepo.Create",
			}
		}

		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       errStr,
			MessageToSend: "Failed create role",
			Function:      "RolePostgresRepo.Create",
		}
	}
	return errUtility.CustomError{}
}
func (r RolePostgresRepo) Update(ctx context.Context, param models.UpdateRoleRequest) errUtility.CustomError {
	db := connections.DbPostgres()

	var parentID any = nil
	if param.ParentID != uuid.Nil {
		parentID = param.ParentID
	}

	query := "UPDATE roles SET name=$1, description=$2, parent_id=$3 WHERE id=$4;"
	res, err := db.ExecContext(ctx, query, param.Name, param.Description, parentID, param.ID)
	if err != nil {
		errStr := err.Error()

		if strings.Contains(errStr, "roles_parent_id_fkey") || strings.Contains(errStr, "violates foreign key constraint") {
			return errUtility.CustomError{
				Code:          http.StatusBadRequest,
				Message:       errStr,
				MessageToSend: "Parent role with the specified ID does not exist",
				Property:      "parentID",
				Function:      "RolePostgresRepo.Update",
			}
		}

		if strings.Contains(errStr, "unique constraint") {
			return errUtility.CustomError{
				Code:          http.StatusConflict,
				Message:       errStr,
				MessageToSend: "Role name already exists",
				Property:      "name",
				Function:      "RolePostgresRepo.Update",
			}
		}

		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       errStr,
			MessageToSend: "Failed update role",
			Function:      "RolePostgresRepo.Update",
		}
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed update role",
			Function:      "RolePostgresRepo.Update",
		}
	} else {
		if rowsAffected == 0 {
			return errUtility.CustomError{
				Code:          http.StatusNotFound,
				MessageToSend: "Role not found",
				Function:      "RolePostgresRepo.Update",
				Property:      "id",
			}
		}
	}
	return errUtility.CustomError{}
}
func (r RolePostgresRepo) GetByID(ctx context.Context, roleID uuid.UUID) (models.RoleResponse, errUtility.CustomError) {
	db := connections.DbPostgres()
	var data models.RoleResponse
	var parentId, description, updatedAt sql.NullString
	query := "SELECT id, parent_id, name, description, created_at, updated_at FROM roles WHERE id=$1;"
	err := db.QueryRowContext(ctx, query, roleID).Scan(&data.ID, &parentId, &data.Name, &description, &data.CreatedAt, &updatedAt)

	if err != nil {
		return models.RoleResponse{}, errUtility.CustomError{
			Code:          http.StatusNotFound,
			Message:       err.Error(),
			MessageToSend: "Role not found",
			Function:      "RolePostgresRepo.GetByID",
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
	return data, errUtility.CustomError{}
}
func (r RolePostgresRepo) Delete(ctx context.Context, roleID uuid.UUID) errUtility.CustomError {
	db := connections.DbPostgres()

	var childID uuid.UUID
	err := db.QueryRowContext(ctx, "SELECT id FROM roles WHERE parent_id = $1 LIMIT 1;", roleID).Scan(&childID)
	if err == nil {
		return errUtility.CustomError{
			Code:          http.StatusBadRequest,
			Message:       "cannot delete role that has child roles",
			MessageToSend: "Cannot delete role because it has child roles",
			Property:      "id",
			Function:      "RolePostgresRepo.Delete",
		}
	} else if err != sql.ErrNoRows {
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed delete role",
			Function:      "RolePostgresRepo.Delete",
		}
	}

	query := "DELETE FROM roles WHERE id = $1;"
	res, err := db.ExecContext(ctx, query, roleID)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "roles_parent_id_fkey") || strings.Contains(errStr, "violates foreign key constraint") {
			return errUtility.CustomError{
				Code:          http.StatusBadRequest,
				Message:       errStr,
				MessageToSend: "Cannot delete role because it has child roles",
				Property:      "id",
				Function:      "RolePostgresRepo.Delete",
			}
		}

		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       errStr,
			MessageToSend: "Failed delete role",
			Function:      "RolePostgresRepo.Delete",
		}
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed delete role",
			Function:      "RolePostgresRepo.Delete",
		}
	} else {
		if rowsAffected == 0 {
			return errUtility.CustomError{
				Code:          http.StatusNotFound,
				MessageToSend: "Role not found",
				Function:      "RolePostgresRepo.Delete",
				Property:      "id",
			}
		}
	}
	return errUtility.CustomError{}
}
func (r RolePostgresRepo) AssignMenu(ctx context.Context, roleID uuid.UUID, param models.AssignMenuRequest) errUtility.CustomError {
	db := connections.DbPostgres()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed to assign menu",
			Function:      "RolePostgresRepo.AssignMenu",
		}
	}
	defer tx.Rollback()

	// 1. Check if role exists
	var existingRoleID uuid.UUID
	err = tx.QueryRowContext(ctx, "SELECT id FROM roles WHERE id = $1;", roleID).Scan(&existingRoleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return errUtility.CustomError{
				Code:          http.StatusNotFound,
				Message:       "role not found",
				MessageToSend: "Role not found",
				Property:      "roleID",
				Function:      "RolePostgresRepo.AssignMenu",
			}
		}
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed to assign menu",
			Function:      "RolePostgresRepo.AssignMenu",
		}
	}

	// 2. Validate and check all menu IDs
	uniqueMenuIDs := make([]uuid.UUID, 0, len(param.MenuIds))
	seen := make(map[uuid.UUID]bool)
	for _, menuID := range param.MenuIds {
		if menuID == uuid.Nil {
			return errUtility.CustomError{
				Code:          http.StatusBadRequest,
				Message:       "invalid menu id",
				MessageToSend: "Invalid menu ID",
				Property:      "listID",
				Function:      "RolePostgresRepo.AssignMenu",
			}
		}
		if !seen[menuID] {
			seen[menuID] = true
			uniqueMenuIDs = append(uniqueMenuIDs, menuID)
		}
	}

	for _, menuID := range uniqueMenuIDs {
		var existingMenuID uuid.UUID
		err = tx.QueryRowContext(ctx, "SELECT id FROM menus WHERE id = $1;", menuID).Scan(&existingMenuID)
		if err != nil {
			if err == sql.ErrNoRows {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					Message:       "menu not found",
					MessageToSend: fmt.Sprintf("Menu with ID %s does not exist", menuID.String()),
					Property:      "listID",
					Function:      "RolePostgresRepo.AssignMenu",
				}
			}
			return errUtility.CustomError{
				Code:          http.StatusInternalServerError,
				Message:       err.Error(),
				MessageToSend: "Failed to assign menu",
				Function:      "RolePostgresRepo.AssignMenu",
			}
		}
	}

	// 3. Delete existing association
	_, err = tx.ExecContext(ctx, "DELETE FROM role_menu WHERE role_id = $1;", roleID)
	if err != nil {
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed to assign menu",
			Function:      "RolePostgresRepo.AssignMenu",
		}
	}

	// 4. Insert new associations
	for _, menuID := range uniqueMenuIDs {
		id, err := uuid.NewV7()
		if err != nil {
			return errUtility.CustomError{
				Code:          http.StatusInternalServerError,
				Message:       err.Error(),
				MessageToSend: "Failed to assign menu",
				Function:      "RolePostgresRepo.AssignMenu",
			}
		}
		query := "INSERT INTO role_menu (id, role_id, menu_id) VALUES ($1, $2, $3) ON CONFLICT (role_id, menu_id) DO UPDATE SET menu_id = EXCLUDED.menu_id;"

		_, err = tx.ExecContext(ctx, query, id, roleID, menuID)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "violates foreign key constraint") {
				return errUtility.CustomError{
					Code:          http.StatusBadRequest,
					Message:       errStr,
					MessageToSend: "Menu with the specified ID does not exist",
					Property:      "listID",
					Function:      "RolePostgresRepo.AssignMenu",
				}
			}
			return errUtility.CustomError{
				Code:          http.StatusInternalServerError,
				Message:       errStr,
				MessageToSend: "Failed to assign menu",
				Function:      "RolePostgresRepo.AssignMenu",
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed to assign menu",
			Function:      "RolePostgresRepo.AssignMenu",
		}
	}
	return errUtility.CustomError{}
}

func (r RolePostgresRepo) GetAssignedMenus(ctx context.Context, roleID uuid.UUID) ([]models.RoleMenuResponse, errUtility.CustomError) {
	db := connections.DbPostgres()

	// 1. Check if role exists
	var existingRoleID uuid.UUID
	err := db.QueryRowContext(ctx, "SELECT id FROM roles WHERE id = $1;", roleID).Scan(&existingRoleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errUtility.CustomError{
				Code:          http.StatusNotFound,
				Message:       "role not found",
				MessageToSend: "Role not found",
				Property:      "roleID",
				Function:      "RolePostgresRepo.GetAssignedMenus",
			}
		}
		return nil, errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed to get assigned menus",
			Function:      "RolePostgresRepo.GetAssignedMenus",
		}
	}

	// 2. Query all menus with status flag
	query := `
		SELECT 
			m.id, 
			m.parent_id, 
			m.name, 
			m.description, 
			m.sort_order,
			m.created_at, 
			m.updated_at,
			CASE WHEN rm.role_id IS NOT NULL THEN 1 ELSE 0 END AS status
		FROM menus m
		LEFT JOIN role_menu rm ON rm.menu_id = m.id AND rm.role_id = $1
		ORDER BY m.parent_id NULLS FIRST, m.sort_order ASC, m.created_at ASC;
	`
	rows, err := db.QueryContext(ctx, query, roleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return []models.RoleMenuResponse{}, errUtility.CustomError{}
		}
		return nil, errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed to get assigned menus",
			Function:      "RolePostgresRepo.GetAssignedMenus",
		}
	}
	defer rows.Close()

	var list []models.RoleMenuResponse
	for rows.Next() {
		var data models.RoleMenuResponse
		var parentId, description, createdAt, updatedAt sql.NullString

		err = rows.Scan(&data.Id, &parentId, &data.Name, &description, &data.SortOrder, &createdAt, &updatedAt, &data.Status)
		if err != nil {
			return nil, errUtility.CustomError{
				Code:          http.StatusInternalServerError,
				Message:       err.Error(),
				MessageToSend: "Failed to get assigned menus",
				Function:      "RolePostgresRepo.GetAssignedMenus",
			}
		}

		if parentId.Valid && parentId.String != "" {
			pId := parentId.String
			data.ParentId = &pId
		}
		if description.Valid {
			data.Description = description.String
		}
		if createdAt.Valid {
			data.CreatedAt = createdAt.String
		}
		if updatedAt.Valid {
			data.UpdatedAt = updatedAt.String
		}
		data.Children = []*models.RoleMenuResponse{}

		list = append(list, data)
	}

	if err = rows.Err(); err != nil {
		return nil, errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed to get assigned menus",
			Function:      "RolePostgresRepo.GetAssignedMenus",
		}
	}

	if list == nil {
		list = []models.RoleMenuResponse{}
	}

	return list, errUtility.CustomError{}
}

func FactoryRoleRepo() RolePostgresRepo {
	if roleRepo == (RolePostgresRepo{}) {
		roleRepo = RolePostgresRepo{}
	}
	return roleRepo
}
