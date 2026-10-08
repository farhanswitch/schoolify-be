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

type menuSortOrderQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func menuParentQueryValue(parentID uuid.UUID) any {
	if parentID == uuid.Nil {
		return nil
	}
	return parentID
}

func nextMenuSortOrder(ctx context.Context, queryer menuSortOrderQueryer, parentID uuid.UUID, exceptMenuID any) (int, error) {
	var next int
	err := queryer.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(sort_order), 0) + 1
		FROM menus
		WHERE parent_id IS NOT DISTINCT FROM $1
		  AND ($2::uuid IS NULL OR id <> $2);`, menuParentQueryValue(parentID), exceptMenuID).Scan(&next)
	return next, err
}

func menuSortOrderExists(ctx context.Context, queryer menuSortOrderQueryer, parentID uuid.UUID, sortOrder int, exceptMenuID any) (bool, error) {
	var exists bool
	err := queryer.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM menus
			WHERE parent_id IS NOT DISTINCT FROM $1
			  AND sort_order = $2
			  AND ($3::uuid IS NULL OR id <> $3)
		);`, menuParentQueryValue(parentID), sortOrder, exceptMenuID).Scan(&exists)
	return exists, err
}

func lockMenuSortOrderWrites(ctx context.Context, tx *sql.Tx) error {
	// Menu administration is low-volume; serializing these short transactions
	// keeps sibling sort-order recommendations and uniqueness checks race-free.
	_, err := tx.ExecContext(ctx, "LOCK TABLE menus IN SHARE ROW EXCLUSIVE MODE;")
	return err
}

func (r MenuPostgresRepo) GetAll(ctx context.Context) ([]models.MenuResponse, errUtility.CustomError) {
	db := connections.DbPostgres()
	query := "SELECT id, parent_id, name, description, sort_order, created_at, updated_at FROM menus ORDER BY parent_id NULLS FIRST, sort_order ASC, created_at ASC"
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

		err = rows.Scan(&data.Id, &parentId, &data.Name, &description, &data.SortOrder, &data.CreatedAt, &updatedAt)
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

func (r MenuPostgresRepo) Create(ctx context.Context, param models.CreateMenuRequest) (models.MenuSortOrderInfo, errUtility.CustomError) {
	db := connections.DbPostgres()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed create menu", Function: "MenuPostgresRepo.Create"}
	}
	defer tx.Rollback()

	if err = lockMenuSortOrderWrites(ctx, tx); err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed create menu", Function: "MenuPostgresRepo.Create"}
	}

	suggestedSortOrder, err := nextMenuSortOrder(ctx, tx, param.ParentID, nil)
	if err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed create menu", Function: "MenuPostgresRepo.Create"}
	}
	sortOrder := suggestedSortOrder
	if param.SortOrder != nil {
		sortOrder = *param.SortOrder
	}
	if sortOrder < 1 {
		return models.MenuSortOrderInfo{SuggestedSortOrder: suggestedSortOrder}, errUtility.CustomError{Code: http.StatusBadRequest, Message: "sortOrder must be greater than zero", MessageToSend: "Sort order must be greater than zero", Property: "sortOrder", Function: "MenuPostgresRepo.Create"}
	}

	used, err := menuSortOrderExists(ctx, tx, param.ParentID, sortOrder, nil)
	if err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed create menu", Function: "MenuPostgresRepo.Create"}
	}
	if used {
		message := fmt.Sprintf("Sort order %d is already in use; suggested sort order is %d", sortOrder, suggestedSortOrder)
		return models.MenuSortOrderInfo{SuggestedSortOrder: suggestedSortOrder}, errUtility.CustomError{Code: http.StatusConflict, Message: message, MessageToSend: message, Property: "sortOrder", Function: "MenuPostgresRepo.Create"}
	}

	query := "INSERT INTO menus(id, name, description, parent_id, sort_order) VALUES($1, $2, $3, $4, $5);"
	_, err = tx.ExecContext(ctx, query, param.ID, param.Name, param.Description, menuParentQueryValue(param.ParentID), sortOrder)
	if err != nil {
		errStr := err.Error()

		if strings.Contains(errStr, "menus_parent_id_fkey") || strings.Contains(errStr, "violates foreign key constraint") {
			return models.MenuSortOrderInfo{}, errUtility.CustomError{
				Code:          http.StatusBadRequest,
				Message:       errStr,
				MessageToSend: "Parent menu with the specified ID does not exist",
				Property:      "parentID",
				Function:      "MenuPostgresRepo.Create",
			}
		}

		if strings.Contains(errStr, "unique constraint") {
			return models.MenuSortOrderInfo{}, errUtility.CustomError{
				Code:          http.StatusConflict,
				Message:       errStr,
				MessageToSend: "Menu name already exists",
				Property:      "name",
				Function:      "MenuPostgresRepo.Create",
			}
		}

		return models.MenuSortOrderInfo{}, errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       errStr,
			MessageToSend: "Failed create menu",
			Function:      "MenuPostgresRepo.Create",
		}
	}

	suggestedSortOrder, err = nextMenuSortOrder(ctx, tx, param.ParentID, nil)
	if err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed create menu", Function: "MenuPostgresRepo.Create"}
	}
	if err = tx.Commit(); err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed create menu", Function: "MenuPostgresRepo.Create"}
	}
	return models.MenuSortOrderInfo{SortOrder: sortOrder, SuggestedSortOrder: suggestedSortOrder}, errUtility.CustomError{}
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
		ORDER BY status DESC, a.created_at ASC;`
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
	query := "SELECT id, parent_id, name, description, sort_order, created_at, updated_at FROM menus WHERE id=$1;"
	err := db.QueryRowContext(ctx, query, menuID).Scan(&data.Id, &parentId, &data.Name, &description, &data.SortOrder, &data.CreatedAt, &updatedAt)

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

func (r MenuPostgresRepo) Update(ctx context.Context, param models.UpdateMenuRequest) (models.MenuSortOrderInfo, errUtility.CustomError) {
	db := connections.DbPostgres()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed update menu", Function: "MenuPostgresRepo.Update"}
	}
	defer tx.Rollback()

	if err = lockMenuSortOrderWrites(ctx, tx); err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed update menu", Function: "MenuPostgresRepo.Update"}
	}

	var currentParentID sql.NullString
	var currentSortOrder int
	err = tx.QueryRowContext(ctx, "SELECT parent_id, sort_order FROM menus WHERE id=$1 FOR UPDATE;", param.ID).Scan(&currentParentID, &currentSortOrder)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusNotFound, Message: err.Error(), MessageToSend: "Menu not found", Function: "MenuPostgresRepo.Update", Property: "id"}
		}
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed update menu", Function: "MenuPostgresRepo.Update"}
	}

	var existingParentID uuid.UUID
	if currentParentID.Valid {
		existingParentID, err = uuid.Parse(currentParentID.String)
		if err != nil {
			return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed update menu", Function: "MenuPostgresRepo.Update"}
		}
	}

	suggestedSortOrder, err := nextMenuSortOrder(ctx, tx, param.ParentID, nil)
	if err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed update menu", Function: "MenuPostgresRepo.Update"}
	}
	sortOrder := currentSortOrder
	if existingParentID != param.ParentID {
		sortOrder = suggestedSortOrder
	}
	if param.SortOrder != nil {
		sortOrder = *param.SortOrder
	}
	if sortOrder < 1 {
		return models.MenuSortOrderInfo{SuggestedSortOrder: suggestedSortOrder}, errUtility.CustomError{Code: http.StatusBadRequest, Message: "sortOrder must be greater than zero", MessageToSend: "Sort order must be greater than zero", Property: "sortOrder", Function: "MenuPostgresRepo.Update"}
	}

	used, err := menuSortOrderExists(ctx, tx, param.ParentID, sortOrder, param.ID)
	if err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed update menu", Function: "MenuPostgresRepo.Update"}
	}
	if used {
		message := fmt.Sprintf("Sort order %d is already in use; suggested sort order is %d", sortOrder, suggestedSortOrder)
		return models.MenuSortOrderInfo{SuggestedSortOrder: suggestedSortOrder}, errUtility.CustomError{Code: http.StatusConflict, Message: message, MessageToSend: message, Property: "sortOrder", Function: "MenuPostgresRepo.Update"}
	}

	query := "UPDATE menus SET name=$1, description=$2, parent_id=$3, sort_order=$4 WHERE id=$5;"
	res, err := tx.ExecContext(ctx, query, param.Name, param.Description, menuParentQueryValue(param.ParentID), sortOrder, param.ID)
	if err != nil {
		errStr := err.Error()

		if strings.Contains(errStr, "menus_parent_id_fkey") || strings.Contains(errStr, "violates foreign key constraint") {
			return models.MenuSortOrderInfo{}, errUtility.CustomError{
				Code:          http.StatusBadRequest,
				Message:       errStr,
				MessageToSend: "Parent menu with the specified ID does not exist",
				Property:      "parentID",
				Function:      "MenuPostgresRepo.Update",
			}
		}

		if strings.Contains(errStr, "unique constraint") {
			return models.MenuSortOrderInfo{}, errUtility.CustomError{
				Code:          http.StatusConflict,
				Message:       errStr,
				MessageToSend: "Menu name already exists",
				Property:      "name",
				Function:      "MenuPostgresRepo.Update",
			}
		}

		return models.MenuSortOrderInfo{}, errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       errStr,
			MessageToSend: "Failed update menu",
			Function:      "MenuPostgresRepo.Update",
		}
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed update menu",
			Function:      "MenuPostgresRepo.Update",
		}
	} else {
		if rowsAffected == 0 {
			return models.MenuSortOrderInfo{}, errUtility.CustomError{
				Code:          http.StatusNotFound,
				MessageToSend: "Menu not found",
				Function:      "MenuPostgresRepo.Update",
				Property:      "id",
			}
		}
	}

	suggestedSortOrder, err = nextMenuSortOrder(ctx, tx, param.ParentID, nil)
	if err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed update menu", Function: "MenuPostgresRepo.Update"}
	}
	if err = tx.Commit(); err != nil {
		return models.MenuSortOrderInfo{}, errUtility.CustomError{Code: http.StatusInternalServerError, Message: err.Error(), MessageToSend: "Failed update menu", Function: "MenuPostgresRepo.Update"}
	}
	return models.MenuSortOrderInfo{SortOrder: sortOrder, SuggestedSortOrder: suggestedSortOrder}, errUtility.CustomError{}
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
