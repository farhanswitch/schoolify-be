package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Status     bool      `json:"isActive"`
	ID         uuid.UUID `json:"userID"`
	RoleID     uuid.UUID `json:"roleID"`
	Email      string    `json:"email"`
	Username   string    `json:"username"`
	FirstName  string    `json:"firstName"`
	LastName   string    `json:"lastName"`
	Password   string    `json:"-"`
	Created    time.Time `json:"-"`
	Updated    time.Time `json:"-"`
	StrCreated string    `json:"created"`
	StrUpdated string    `json:"updated"`
}

type UserLoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}
