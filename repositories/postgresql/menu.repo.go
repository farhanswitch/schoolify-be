package postgresql

import (
	"context"
	"net/http"
	"strings"

	connections "schoolify_be/connections"
	models "schoolify_be/models"
	errUtility "schoolify_be/utilities/errors"

	"github.com/google/uuid"
)

type MenuPostgresRepo struct{}

var menuRepo MenuPostgresRepo

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

func FactoryMenuRepo() MenuPostgresRepo {
	if menuRepo == (MenuPostgresRepo{}) {
		menuRepo = MenuPostgresRepo{}
	}
	return menuRepo
}
