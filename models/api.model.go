package models

import (
	"github.com/google/uuid"
)

type CreateApiRequest struct {
	Id          uuid.UUID `json:"id" validate:"omitempty,lte=40"`
	Name        string    `json:"name" validate:"required,gte=3,lte=50"`
	Description string    `json:"description" validate:"required,gte=3,lte=100"`
	Path        string    `json:"path" validate:"required,lte=255"`
	Method      string    `json:"method" validate:"required,oneof=GET POST PUT DELETE PATCH,lte=10"`
}

type UpdateApiRequest struct {
	Name        string `json:"name" validate:"required,gte=3,lte=50"`
	Description string `json:"description" validate:"required,gte=3,lte=100"`
	Path        string `json:"path" validate:"required,lte=255"`
	Method      string `json:"method" validate:"required,oneof=GET POST PUT DELETE PATCH,lte=10"`
}

type GetAllApiRequest struct {
	SortField string `json:"sortField"`
	SortOrder string `json:"sortOrder" validate:"oneof=ASC DESC asc desc"`
	Search    string `json:"search"`
	Page      int    `json:"paginationPage" validate:"required,min=1"`
	Rows      int    `json:"paginationRows" validate:"required,min=1"`
}

type ApiResponse struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Path        string    `json:"path"`
	Method      string    `json:"method"`
	CreatedAt   string    `json:"createdAt"`
	UpdatedAt   string    `json:"updatedAt"`
}
