package models

import "go.mongodb.org/mongo-driver/bson/primitive"

// User struct with ID, Name, and Email fields
type User struct {
	ID    primitive.ObjectID `json:"id" bson:"_id"`
	Name  string             `json:"name" bson:"name"`
	Email string             `json:"email" bson:"email"`
}

// UserDetails struct with additional fields
type UserDetails struct {
	User    `bson:",inline"` // Include and map the fields from the User struct
	Gender  string           `json:"gender" bson:"gender"`
	Age     int              `json:"age" bson:"age"`
	Address string           `json:"address" bson:"address"`
	Phone   string           `json:"phone" bson:"phone"`
}
