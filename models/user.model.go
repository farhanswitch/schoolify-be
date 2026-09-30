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
type UserRegisterRequest struct {
	ID              uuid.UUID `json:"userID"`
	RoleID          uuid.UUID `json:"roleID"`
	Email           string    `json:"email" validate:"required,email"`
	Username        string    `json:"username" validate:"required,min=6,max=15,alphanum"`
	FirstName       string    `json:"firstName" validate:"required,min=3,max=30"`
	LastName        string    `json:"lastName" validate:"min=0,max=30"`
	Password        string    `json:"password" validate:"required,password"`
	ConfirmPassword string    `json:"confirmPassword" validate:"required"`
}

type UserLoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}
