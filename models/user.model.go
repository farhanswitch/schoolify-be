package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Status       bool      `json:"isActive"`
	ID           uuid.UUID `json:"userID"`
	RoleID       uuid.UUID `json:"roleID"`
	Email        string    `json:"email"`
	Username     string    `json:"username"`
	FirstName    string    `json:"firstName"`
	LastName     string    `json:"lastName"`
	Password     string    `json:"-"`
	CreatedAt    time.Time `json:"-"`
	UpdatedAt    time.Time `json:"-"`
	StrCreatedAt string    `json:"createdAt"`
	StrUpdatedAt string    `json:"updatedAt"`
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
	Username string `json:"username" validate:"required,min=6,max=15,alphanum"`
	Password string `json:"password" validate:"required,min=3,max=30"`
}

type UserLoginData struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Password string    `json:"password"`
	RoleID   uuid.UUID `json:"roleID"`
	RoleName string    `json:"roleName"`
}
