package models

import (
	"github.com/google/uuid"
)

type CreateMenuRequest struct {
	ID          uuid.UUID `json:"id" validate:"omitempty,lte=40"`
	ParentID    uuid.UUID `json:"parentID" validate:"omitempty,lte=40"`
	Name        string    `json:"name" validate:"required,gte=3,lte=50"`
	Description string    `json:"description" validate:"required,gte=3,lte=100"`
}

type UpdateMenuRequest struct {
	ID          uuid.UUID `json:"id" validate:"required,lte=40"`
	ParentID    uuid.UUID `json:"parentID" validate:"omitempty,lte=40"`
	Name        string    `json:"name" validate:"required,gte=3,lte=50"`
	Description string    `json:"description" validate:"required,gte=3,lte=100"`
}

type MenuResponse struct {
	Id          string          `json:"id"`
	ParentId    *string         `json:"parentId"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
	Children    []*MenuResponse `json:"children"`
}
