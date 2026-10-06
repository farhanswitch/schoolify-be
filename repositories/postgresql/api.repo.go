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
	"sync"

	"github.com/google/uuid"
)

type ApiPostgresRepo struct{}

var apiRepo ApiPostgresRepo

func (r ApiPostgresRepo) GetAll(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
	db := connections.DbPostgres()

	// Whitelist SortField to prevent SQL injection
	sortField := "created_at"
	switch strings.ToLower(strings.TrimSpace(param.SortField)) {
	case "id":
		sortField = "id"
	case "name":
		sortField = "name"
	case "description":
		sortField = "description"
	case "path":
		sortField = "path"
	case "method":
		sortField = "method"
	case "created_at", "createdat":
		sortField = "created_at"
	case "updated_at", "updatedat":
		sortField = "updated_at"
	}

	// Whitelist SortOrder
	sortOrder := "ASC"
	if strings.ToUpper(strings.TrimSpace(param.SortOrder)) == "DESC" {
		sortOrder = "DESC"
	}

	// Pagination limit and offset calculation
	limit := param.Rows
	if limit <= 0 {
		limit = 10
	}
	page := param.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	var (
		listQuery  string
		countQuery string
		listArgs   []any
		countArgs  []any
	)

	search := strings.TrimSpace(param.Search)
	if search != "" {
		searchPattern := "%" + search + "%"
		listQuery = fmt.Sprintf("SELECT id, name, description, path, method, created_at, updated_at FROM apis WHERE name ILIKE $1 ORDER BY %s %s LIMIT $2 OFFSET $3;", sortField, sortOrder)
		listArgs = []any{searchPattern, limit, offset}

		countQuery = "SELECT COUNT(*) FROM apis WHERE name ILIKE $1;"
		countArgs = []any{searchPattern}
	} else {
		listQuery = fmt.Sprintf("SELECT id, name, description, path, method, created_at, updated_at FROM apis ORDER BY %s %s LIMIT $1 OFFSET $2;", sortField, sortOrder)
		listArgs = []any{limit, offset}

		countQuery = "SELECT COUNT(*) FROM apis;"
		countArgs = []any{}
	}

	var (
		listData []models.ApiResponse
		total    int
		errList  errUtility.CustomError
		errCount errUtility.CustomError
		wg       sync.WaitGroup
	)

	wg.Add(2)

	// Goroutine 1: Get list of APIs
	go func() {
		defer wg.Done()

		rows, err := db.QueryContext(ctx, listQuery, listArgs...)
		if err != nil {
			if err == sql.ErrNoRows {
				listData = []models.ApiResponse{}
				return
			}
			errList = errUtility.CustomError{
				Code:     http.StatusInternalServerError,
				Message:  err.Error(),
				Function: "ApiPostgresRepo.GetAll",
			}
			return
		}
		defer rows.Close()

		for rows.Next() {
			var data models.ApiResponse
			var description, createdAt, updatedAt sql.NullString

			err = rows.Scan(
				&data.Id,
				&data.Name,
				&description,
				&data.Path,
				&data.Method,
				&createdAt,
				&updatedAt,
			)
			if err != nil {
				errList = errUtility.CustomError{
					Code:     http.StatusInternalServerError,
					Message:  err.Error(),
					Function: "ApiPostgresRepo.GetAll",
				}
				return
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

			listData = append(listData, data)
		}

		if err = rows.Err(); err != nil {
			errList = errUtility.CustomError{
				Code:     http.StatusInternalServerError,
				Message:  err.Error(),
				Function: "ApiPostgresRepo.GetAll",
			}
			return
		}
	}()

	// Goroutine 2: Get count of APIs
	go func() {
		defer wg.Done()

		err := db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total)
		if err != nil {
			errCount = errUtility.CustomError{
				Code:     http.StatusInternalServerError,
				Message:  err.Error(),
				Function: "ApiPostgresRepo.GetAll",
			}
			return
		}
	}()

	wg.Wait()

	if errCount.Code != 0 {
		return nil, 0, errCount
	}

	if errList.Code != 0 {
		return nil, 0, errList
	}

	if listData == nil {
		listData = []models.ApiResponse{}
	}

	return listData, total, errUtility.CustomError{}
}

func (r ApiPostgresRepo) GetList(ctx context.Context, param models.GetAllApiRequest) ([]models.ApiResponse, int, errUtility.CustomError) {
	return r.GetAll(ctx, param)
}

func (r ApiPostgresRepo) Create(ctx context.Context, param models.CreateApiRequest) errUtility.CustomError {
	db := connections.DbPostgres()

	query := "INSERT INTO apis (id, name, description, path, method) VALUES ($1, $2, $3, $4, $5);"
	_, err := db.ExecContext(ctx, query, param.Id, param.Name, param.Description, param.Path, param.Method)
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "unique constraint") {
			return errUtility.CustomError{
				Code:          http.StatusConflict,
				Message:       errStr,
				MessageToSend: "API already exists",
				Function:      "ApiPostgresRepo.Create",
			}
		}
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       errStr,
			MessageToSend: "Failed to create API",
			Function:      "ApiPostgresRepo.Create",
		}
	}
	return errUtility.CustomError{}
}

func (r ApiPostgresRepo) Update(ctx context.Context, apiID uuid.UUID, param models.UpdateApiRequest) errUtility.CustomError {
	db := connections.DbPostgres()

	query := "UPDATE apis SET name = $1, description = $2, path = $3, method = $4, updated_at = CURRENT_TIMESTAMP WHERE id = $5;"
	result, err := db.ExecContext(ctx, query, param.Name, param.Description, param.Path, param.Method, apiID)
	if err != nil {
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed to update API",
			Function:      "ApiPostgresRepo.Update",
		}
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed to update API",
			Function:      "ApiPostgresRepo.Update",
		}
	}
	if rowsAffected == 0 {
		return errUtility.CustomError{
			Code:          http.StatusNotFound,
			MessageToSend: "API not found",
			Property:      "apiID",
			Function:      "ApiPostgresRepo.Update",
		}
	}

	return errUtility.CustomError{}
}

func (r ApiPostgresRepo) GetByID(ctx context.Context, apiID uuid.UUID) (models.ApiResponse, errUtility.CustomError) {
	db := connections.DbPostgres()

	var data models.ApiResponse
	var description, createdAt, updatedAt sql.NullString
	query := "SELECT id, name, description, path, method, created_at, updated_at FROM apis WHERE id = $1;"
	err := db.QueryRowContext(ctx, query, apiID).Scan(
		&data.Id,
		&data.Name,
		&description,
		&data.Path,
		&data.Method,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.ApiResponse{}, errUtility.CustomError{
				Code:          http.StatusNotFound,
				Message:       err.Error(),
				MessageToSend: "API not found",
				Function:      "ApiPostgresRepo.GetByID",
				Property:      "id",
			}
		}
		return models.ApiResponse{}, errUtility.CustomError{
			Code:     http.StatusInternalServerError,
			Message:  err.Error(),
			Function: "ApiPostgresRepo.GetByID",
		}
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

	return data, errUtility.CustomError{}
}

func (r ApiPostgresRepo) Delete(ctx context.Context, apiID uuid.UUID) errUtility.CustomError {
	db := connections.DbPostgres()

	query := "DELETE FROM apis WHERE id = $1;"
	res, err := db.ExecContext(ctx, query, apiID)
	if err != nil {
		return errUtility.CustomError{
			Code:          http.StatusInternalServerError,
			Message:       err.Error(),
			MessageToSend: "Failed to delete API",
			Function:      "ApiPostgresRepo.Delete",
		}
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errUtility.CustomError{
			Code:     http.StatusInternalServerError,
			Message:  err.Error(),
			Function: "ApiPostgresRepo.Delete",
		}
	}
	if rowsAffected == 0 {
		return errUtility.CustomError{
			Code:          http.StatusNotFound,
			MessageToSend: "API not found",
			Function:      "ApiPostgresRepo.Delete",
			Property:      "id",
		}
	}

	return errUtility.CustomError{}
}

func FactoryApiRepo() ApiPostgresRepo {
	if apiRepo == (ApiPostgresRepo{}) {
		apiRepo = ApiPostgresRepo{}
	}
	return apiRepo
}
