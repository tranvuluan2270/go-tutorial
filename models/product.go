package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// Product represents the basic product structure
type Product struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	Name        string             `json:"name" bson:"name"`
	Description string             `json:"description" bson:"description"`
	Price       float64            `json:"price" bson:"price"`
	Category    string             `json:"category" bson:"category"`
	Stock       int                `json:"stock" bson:"stock"`
}

// CreateProductRequest is used for product creation requests
type CreateProductRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=100"`
	Description string  `json:"description" validate:"required,min=10,max=1000"`
	Price       float64 `json:"price" validate:"required,gt=0"`
	Category    string  `json:"category" validate:"required"`
	Stock       int     `json:"stock" validate:"required,gte=0"`
}

// UpdateProductRequest is used for product update requests
type UpdateProductRequest struct {
	Name        string  `json:"name,omitempty" validate:"omitempty,min=2,max=100"`
	Description string  `json:"description,omitempty" validate:"omitempty,min=10,max=1000"`
	Price       float64 `json:"price,omitempty" validate:"omitempty,gt=0"`
	Category    string  `json:"category,omitempty"`
	Stock       int     `json:"stock,omitempty" validate:"omitempty,gte=0"`
}
