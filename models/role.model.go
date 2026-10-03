package models

import (
	"github.com/google/uuid"
)

type CreateRoleRequest struct {
	ID          uuid.UUID `json:"id" validate:"omitempty,lte=40"`
	ParentID    uuid.UUID `json:"parentID" validate:"omitempty,lte=40"` // Can be empty if root role
	Name        string    `json:"name" validate:"required,gte=3,lte=30"`
	Description string    `json:"description" validate:"required,gte=3,lte=100"`
}

type AssignMenuRequest struct {
	MenuIds uuid.UUIDs `json:"menuIds" validate:"required,dive,required"`
}

type RoleResponse struct {
	Children    []*RoleResponse `json:"children"`
	ID          uuid.UUID       `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
	ParentID    *uuid.UUID      `json:"parentID"`
}

type MenuResponse struct {
	Id          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	ParentId    *uuid.UUID `json:"parentId"`
}

type ValidRoleAndMenuResponse struct {
	Roles []RoleResponse `json:"roles"`
	Menus []MenuResponse `json:"menus"`
}
