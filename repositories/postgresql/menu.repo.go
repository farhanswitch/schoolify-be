package postgresql

import (
	"context"
	"database/sql"
	"fmt"
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

func (r MenuPostgresRepo) AssignApi(ctx context.Context, menuID uuid.UUID, param models.AssignApiRequest) errUtility.CustomError {
	db := connections.DbPostgres()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed to assign API", Function: "MenuPostgresRepo.AssignApi"}
	}
	defer tx.Rollback()

	var existingMenuID uuid.UUID
	if err = tx.QueryRowContext(ctx, "SELECT id FROM menus WHERE id = $1;", menuID).Scan(&existingMenuID); err != nil {
		if err == sql.ErrNoRows {
			return errUtility.CustomError{Code: http.StatusNotFound, Message: "menu not found", MessageToSend: "Menu not found", Property: "menuID", Function: "MenuPostgresRepo.AssignApi"}
		}
		return errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed to assign API", Function: "MenuPostgresRepo.AssignApi"}
	}

	uniqueAPIIDs := make([]uuid.UUID, 0, len(param.ApiIDs))
	seen := make(map[uuid.UUID]bool)
	for _, apiID := range param.ApiIDs {
		if apiID == uuid.Nil {
			return errUtility.CustomError{Code: http.StatusBadRequest, Message: "invalid api id", MessageToSend: "Invalid API ID", Property: "listID", Function: "MenuPostgresRepo.AssignApi"}
		}
		if !seen[apiID] {
			seen[apiID] = true
			uniqueAPIIDs = append(uniqueAPIIDs, apiID)
		}
	}

	for _, apiID := range uniqueAPIIDs {
		var existingAPIID uuid.UUID
		if err = tx.QueryRowContext(ctx, "SELECT id FROM apis WHERE id = $1;", apiID).Scan(&existingAPIID); err != nil {
			if err == sql.ErrNoRows {
				return errUtility.CustomError{Code: http.StatusBadRequest, Message: "api not found", MessageToSend: fmt.Sprintf("API with ID %s does not exist", apiID.String()), Property: "listID", Function: "MenuPostgresRepo.AssignApi"}
			}
			return errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed to assign API", Function: "MenuPostgresRepo.AssignApi"}
		}
	}

	if _, err = tx.ExecContext(ctx, "DELETE FROM menu_api WHERE menu_id = $1;", menuID); err != nil {
		return errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed to assign API", Function: "MenuPostgresRepo.AssignApi"}
	}

	for _, apiID := range uniqueAPIIDs {
		id, uuidErr := uuid.NewV7()
		if uuidErr != nil {
			return errUtility.CustomError{Code: http.StatusInternalServerError, Message: uuidErr.Error(), MessageToSend: "Failed to assign API", Function: "MenuPostgresRepo.AssignApi"}
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO menu_api (id, menu_id, api_id) VALUES ($1, $2, $3) ON CONFLICT (menu_id, api_id) DO NOTHING;", id, menuID, apiID); err != nil {
			return errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed to assign API", Function: "MenuPostgresRepo.AssignApi"}
		}
	}

	if err = tx.Commit(); err != nil {
		return errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed to assign API", Function: "MenuPostgresRepo.AssignApi"}
	}
	return errUtility.CustomError{}
}

func (r MenuPostgresRepo) GetAssignedEndpoints(ctx context.Context, menuID uuid.UUID) ([]models.MenuApiResponse, errUtility.CustomError) {
	db := connections.DbPostgres()
	var existingMenuID uuid.UUID
	if err := db.QueryRowContext(ctx, "SELECT id FROM menus WHERE id = $1;", menuID).Scan(&existingMenuID); err != nil {
		if err == sql.ErrNoRows {
			return nil, errUtility.CustomError{Code: http.StatusNotFound, Message: "menu not found", MessageToSend: "Menu not found", Property: "menuID", Function: "MenuPostgresRepo.GetAssignedEndpoints"}
		}
		return nil, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed to get assigned endpoints", Function: "MenuPostgresRepo.GetAssignedEndpoints"}
	}

	query := `
		SELECT a.id, a.name, a.description, a.path, a.method, a.created_at, a.updated_at,
			CASE WHEN ma.menu_id IS NOT NULL THEN 1 ELSE 0 END AS status
		FROM apis a
		LEFT JOIN menu_api ma ON ma.api_id = a.id AND ma.menu_id = $1
		ORDER BY a.created_at ASC;`
	rows, err := db.QueryContext(ctx, query, menuID)
	if err != nil {
		return nil, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed to get assigned endpoints", Function: "MenuPostgresRepo.GetAssignedEndpoints"}
	}
	defer rows.Close()

	data := []models.MenuApiResponse{}
	for rows.Next() {
		var item models.MenuApiResponse
		var description, createdAt, updatedAt sql.NullString
		if err := rows.Scan(&item.Id, &item.Name, &description, &item.Path, &item.Method, &createdAt, &updatedAt, &item.Status); err != nil {
			return nil, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed to get assigned endpoints", Function: "MenuPostgresRepo.GetAssignedEndpoints"}
		}
		if description.Valid {
			item.Description = description.String
		}
		if createdAt.Valid {
			item.CreatedAt = createdAt.String
		}
		if updatedAt.Valid {
			item.UpdatedAt = updatedAt.String
		}
		data = append(data, item)
	}
	if err := rows.Err(); err != nil {
		return nil, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed to get assigned endpoints", Function: "MenuPostgresRepo.GetAssignedEndpoints"}
	}
	return data, errUtility.CustomError{}
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

func (r MenuPostgresRepo) Delete(ctx context.Context, menuID uuid.UUID) errUtility.CustomError {
	db := connections.DbPostgres()

	var childID uuid.UUID
	err := db.QueryRowContext(ctx, "SELECT id FROM menus WHERE parent_id = $1 LIMIT 1;", menuID).Scan(&childID)
	if err == nil {
		return errUtility.CustomError{
			Code:          http.StatusBadRequest,
			Message:       "cannot delete menu that has child menus",
			MessageToSend: "Cannot delete menu because it has child menus",
			Property:      "id",
			Function:      "MenuPostgresRepo.Delete",
		}
	} else if err != sql.ErrNoRows {
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed delete menu",
			Function:      "MenuPostgresRepo.Delete",
		}
	}

	query := "DELETE FROM menus WHERE id = $1;"
	res, err := db.ExecContext(ctx, query, menuID)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "menus_parent_id_fkey") || strings.Contains(errStr, "violates foreign key constraint") {
			return errUtility.CustomError{
				Code:          http.StatusBadRequest,
				Message:       errStr,
				MessageToSend: "Cannot delete menu because it has child menus",
				Property:      "id",
				Function:      "MenuPostgresRepo.Delete",
			}
		}

		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       errStr,
			MessageToSend: "Failed delete menu",
			Function:      "MenuPostgresRepo.Delete",
		}
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed delete menu",
			Function:      "MenuPostgresRepo.Delete",
		}
	} else {
		if rowsAffected == 0 {
			return errUtility.CustomError{
				Code:          http.StatusNotFound,
				MessageToSend: "Menu not found",
				Function:      "MenuPostgresRepo.Delete",
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
