package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// BaseUser contains the basic user fields
type User struct {
	ID       primitive.ObjectID `json:"id" bson:"_id"`
	Name     string             `json:"name" bson:"name"`
	Email    string             `json:"email" bson:"email"`
	Password string             `json:"-" bson:"password"`
	Role     string             `json:"role" bson:"role"`
}

// UserDetails contains all user information
type UserDetails struct {
	User    `bson:",inline"`
	Gender  string `json:"gender,omitempty" bson:"gender"`
	Age     int    `json:"age,omitempty" bson:"age" validate:"omitempty,gte=0,lte=150"`
	Address string `json:"address,omitempty" bson:"address"`
	Phone   string `json:"phone,omitempty" bson:"phone"`
}

// CreateUserRequest is used for user creation/signup requests
type CreateUserRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
	Role     string `json:"role" validate:"required,oneof=master_admin sub_admin user"`
	Gender   string `json:"gender,omitempty"`
	Age      int    `json:"age,omitempty" validate:"omitempty,gte=0,lte=150"`
	Address  string `json:"address,omitempty"`
	Phone    string `json:"phone,omitempty"`
}

// UpdateUserRequest is used for user update requests
type UpdateUserRequest struct {
	Name     string `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Email    string `json:"email,omitempty" validate:"omitempty,email"`
	Password string `json:"password,omitempty" validate:"omitempty,min=6"`
	Role     string `json:"role,omitempty" validate:"omitempty,oneof=admin user"`
	Gender   string `json:"gender,omitempty" validate:"omitempty,oneof=male female other"`
	Age      int    `json:"age,omitempty" validate:"omitempty,gte=0,lte=150"`
	Address  string `json:"address,omitempty"`
	Phone    string `json:"phone,omitempty" validate:"omitempty,e164"`
}

// UserResponse is used for sending user data in responses (without password)
type UserResponse struct {
	ID    primitive.ObjectID `json:"id" bson:"_id"`
	Name  string             `json:"name" bson:"name"`
	Email string             `json:"email" bson:"email"`
	Role  string             `json:"role" bson:"role"`
}

// LoginRequest is used for login requests
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse is used for login responses (user response with token but without password)
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}
