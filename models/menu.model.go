package models

import (
	"github.com/google/uuid"
)

type CreateMenuRequest struct {
	ID          uuid.UUID `json:"id" validate:"omitempty,lte=40"`
	ParentID    uuid.UUID `json:"parentID" validate:"omitempty,lte=40"`
	Name        string    `json:"name" validate:"required,gte=3,lte=50"`
	Description string    `json:"description" validate:"required,gte=3,lte=100"`
	SortOrder   *int      `json:"sortOrder,omitempty" validate:"omitempty,gte=1"`
}

type UpdateMenuRequest struct {
	ID          uuid.UUID `json:"id" validate:"required,lte=40"`
	ParentID    uuid.UUID `json:"parentID" validate:"omitempty,lte=40"`
	Name        string    `json:"name" validate:"required,gte=3,lte=50"`
	Description string    `json:"description" validate:"required,gte=3,lte=100"`
	SortOrder   *int      `json:"sortOrder,omitempty" validate:"omitempty,gte=1"`
}

type MenuSortOrderInfo struct {
	SortOrder          int `json:"sortOrder"`
	SuggestedSortOrder int `json:"suggestedSortOrder"`
}

type AssignApiRequest struct {
	ApiIDs uuid.UUIDs `json:"listID" validate:"required,dive,required"`
}

type MenuApiResponse struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Path        string    `json:"path"`
	Method      string    `json:"method"`
	CreatedAt   string    `json:"createdAt"`
	UpdatedAt   string    `json:"updatedAt"`
	Status      int       `json:"status"`
}

// UserAccessMenuResponse represents one menu available to the authenticated
// user. Every item and endpoint in this response is assigned to that user's
// current role.
type UserAccessMenuResponse struct {
	ID          string                    `json:"id"`
	ParentID    *string                   `json:"parentId,omitempty"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	SortOrder   int                       `json:"sortOrder"`
	Endpoints   []MenuApiResponse         `json:"endpoints"`
	Children    []*UserAccessMenuResponse `json:"children"`
}

type MenuResponse struct {
	Id          string          `json:"id"`
	ParentId    *string         `json:"parentId"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	SortOrder   int             `json:"sortOrder"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
	Children    []*MenuResponse `json:"children"`
}
