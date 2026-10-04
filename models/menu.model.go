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

type MenuResponse struct {
	Children    []*MenuResponse `json:"children"`
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
	ParentID    *uuid.UUID      `json:"parentID"`
}
