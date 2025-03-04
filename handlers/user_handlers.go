package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/crypto/bcrypt"

	"go-tutorial/models"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Handler struct contains the database client, database name, and router
type Handler struct {
	DB           *mongo.Client
	Database     string
	Router       *mux.Router
	ResponseHdlr *ResponseHandler
	ErrorHdlr    *ErrorHandler
}

// NewHandler creates a new handler with all dependencies
func NewHandler(db *mongo.Client, database string) *Handler {
	return &Handler{
		DB:           db,
		Database:     database,
		ResponseHdlr: NewResponseHandler(),
		ErrorHdlr:    NewErrorHandler(),
	}
}

func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Access the users collection from the database
	usersCollection := h.DB.Database(h.Database).Collection("users")

	// Get page and limit query parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	// Set default values if not provided
	page := 1
	limit := 5

	// Convert page and limit to integers if provided
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Get total count
	total, err := usersCollection.CountDocuments(context.TODO(), bson.M{})
	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error counting users")
		return
	}

	// Calculate skip
	skip := (page - 1) * limit

	// Create find options
	findOptions := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(skip))

	// Find users
	cursor, err := usersCollection.Find(context.TODO(), bson.M{}, findOptions)

	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error fetching users")
		return
	}
	defer cursor.Close(context.TODO())

	//
	var users []models.BaseUser
	if err := cursor.All(context.TODO(), &users); err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error processing users data")
		return
	}

	// Return a success response
	h.ResponseHdlr.Paginated(w, "Users fetched successfully", users, page, limit, int(total))
}

func (h *Handler) GetUserDetails(w http.ResponseWriter, r *http.Request) {
	// To get the user details, we need to get the user ID from the URL parameters and show more information about the user

	// Get the user ID from the URL parameters
	// this id parameter is the value of the {id} path variable in the URL (a string)
	vars := mux.Vars(r)
	id := vars["id"]

	// Convert the id string to an ObjectId to match the _id field in MongoDB (ObjectId)
	objID, err := primitive.ObjectIDFromHex(id)
	// Check if the ID is valid, if ID valid then find the user with the given ID, if not return an error and exit
	if err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid user ID")
		return
	}

	// Get user from database
	var user models.UserDetails
	err = h.DB.Database(h.Database).Collection("users").
		FindOne(context.TODO(), bson.M{"_id": objID}).
		Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// If the user was not found, return a 404 error
			h.ErrorHdlr.HandleNotFound(w, "User not found")
		} else {
			// If there is an error, return a 500 error
			h.ErrorHdlr.HandleInternalError(w, "Error fetching user details")
		}
		return
	}

	// Return a success response
	h.ResponseHdlr.Success(w, "User details fetched successfully", user)

}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	id := vars["id"]

	// Convert the id string to an ObjectId to match the _id field in MongoDB (ObjectId)
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		// If the user ID is invalid, return a 400 error
		h.ErrorHdlr.HandleBadRequest(w, "Invalid user ID format")
		return
	}

	// Parse request body
	var updateReq models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		// If the request body is invalid, return a 400 error
		h.ErrorHdlr.HandleBadRequest(w, "Invalid request body")
		return
	}

	// Validate request
	validate := validator.New()
	if err := validate.Struct(updateReq); err != nil {
		// If the request is invalid, return a 400 error
		h.ErrorHdlr.HandleBadRequest(w, "Invalid request")
		return
	}

	// Build update document
	update := bson.M{}
	if updateReq.Name != "" {
		update["name"] = updateReq.Name
	}
	if updateReq.Email != "" {
		update["email"] = updateReq.Email
	}
	if updateReq.Gender != "" {
		update["gender"] = updateReq.Gender
	}
	if updateReq.Age != 0 {
		update["age"] = updateReq.Age
	}
	if updateReq.Address != "" {
		update["address"] = updateReq.Address
	}
	if updateReq.Phone != "" {
		update["phone"] = updateReq.Phone
	}
	if updateReq.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updateReq.Password), bcrypt.DefaultCost)
		if err != nil {
			// If there is an error, return a 500 error
			h.ErrorHdlr.HandleInternalError(w, "Error processing request")
			return
		}
		update["password"] = string(hashedPassword)
	}

	// Check if there are fields to update
	if len(update) == 0 {
		// If there are no fields to update, return a 400 error
		h.ErrorHdlr.HandleBadRequest(w, "No fields to update")
		return
	}

	// Update user in database
	usersCollection := h.DB.Database(h.Database).Collection("users")
	result, err := usersCollection.UpdateOne(
		context.TODO(),
		bson.M{"_id": objID},
		bson.M{"$set": update},
	)
	if err != nil {
		// If there is an error, return a 500 error
		h.ErrorHdlr.HandleInternalError(w, "Error updating user")
		return
	}

	// Check if user was found and updated
	if result.MatchedCount == 0 {
		// If the user was not found, return a 404 error
		h.ErrorHdlr.HandleNotFound(w, "User not found")
		return
	}

	//Get the updated user
	updatedUser := models.UserDetails{}
	err = usersCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&updatedUser)
	if err != nil {
		// If there is an error, return a 500 error
		h.ErrorHdlr.HandleInternalError(w, "Error getting updated user")
		return
	}
	// Return a success response
	h.ResponseHdlr.Success(w, "User updated successfully", updatedUser)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Get user ID from URL
	vars := mux.Vars(r)
	id := vars["id"]

	// Convert the id string to an ObjectId to match the _id field in MongoDB (ObjectId)
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid user ID")
		return
	}

	// Delete user from database
	result, err := h.DB.Database(h.Database).Collection("users").
		DeleteOne(context.TODO(), bson.M{"_id": objID})

	if err != nil {
		// If there is an error, return a 500 error
		h.ErrorHdlr.HandleInternalError(w, "Error deleting user")
		return
	}

	if result.DeletedCount == 0 {
		// If the user was not found, return a 404 error
		h.ErrorHdlr.HandleNotFound(w, "User not found")
		return
	}

	// Return a success response
	h.ResponseHdlr.Success(w, "User successfully deleted", nil)
}

func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	// Parse and validate the request body
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If the request body is invalid, return a 400 error
		h.ErrorHdlr.HandleBadRequest(w, "Invalid request body")
		return
	}

	// Set default role if not provided
	if req.Role == "" {
		req.Role = "user" // Default role
	}

	// Validate the request
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var validationErrors []ErrorDetail
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, ErrorDetail{
				Field:   err.Field(),
				Message: formatValidationError(err),
			})
		}
		h.ErrorHdlr.HandleValidationError(w, validationErrors)
		return
	}

	// Check if user already exists
	var existingUser models.UserDetails
	err := h.DB.Database(h.Database).Collection("users").
		FindOne(context.TODO(), bson.M{"email": req.Email}).
		Decode(&existingUser)
	if err == nil {
		// If the user already exists, return a 400 error
		h.ErrorHdlr.HandleBadRequest(w, "User with this email already exists")
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		// If there is an error, return a 500 error
		h.ErrorHdlr.HandleInternalError(w, "Error processing request")
		return
	}

	// Create new user
	// Admin roles should be assigned manually or through a separate admin creation process
	newUser := models.UserDetails{
		BaseUser: models.BaseUser{
			ID:       primitive.NewObjectID(),
			Name:     req.Name,
			Email:    req.Email,
			Password: string(hashedPassword),
			Role:     req.Role, // Default role
		},
		Gender:  req.Gender,
		Age:     req.Age,
		Address: req.Address,
		Phone:   req.Phone,
	}

	// Insert the user into the database
	_, err = h.DB.Database(h.Database).Collection("users").
		InsertOne(context.TODO(), newUser)

	if err != nil {
		// If there is an error, return a 500 error
		h.ErrorHdlr.HandleInternalError(w, "Error creating user")
		return
	}
	// Return a success response
	h.ResponseHdlr.Created(w, "User created successfully", newUser)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	// Parse and validate the request body
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// If the request body is invalid, return a 400 error
		h.ErrorHdlr.HandleBadRequest(w, "Invalid request body")
		return
	}

	// Validate the request
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var validationErrors []ErrorDetail
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, ErrorDetail{
				Field:   err.Field(),
				Message: formatValidationError(err),
			})
		}
		h.ErrorHdlr.HandleValidationError(w, validationErrors)
		return
	}

	// Find user by email
	var user models.UserDetails
	err := h.DB.Database(h.Database).Collection("users").
		FindOne(context.TODO(), bson.M{"email": req.Email}).
		Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// If the user was not found, return a 401 error
			h.ErrorHdlr.HandleUnauthorized(w, "Invalid email or password")
			return
		}
		// If there is an error, return a 500 error
		h.ErrorHdlr.HandleInternalError(w, "Error finding user")
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		// If the password is incorrect, return a 401 error
		h.ErrorHdlr.HandleUnauthorized(w, "Invalid email or password")
		return
	}

	// Generate JWT token
	token, err := h.generateJWT(user.ID.Hex(), user.Role)
	if err != nil {
		// If there is an error, return a 500 error
		h.ErrorHdlr.HandleInternalError(w, "Error generating token")
		return
	}

	// Create response
	loginResponse := models.LoginResponse{
		Token: token,
		User: models.UserResponse{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
		},
	}
	// Return a success response
	h.ResponseHdlr.Success(w, "Login successful", loginResponse)
}

// Helper function to generate JWT
func (h *Handler) generateJWT(userID string, role string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}
	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// Generate encoded token using the secret signing key
	return token.SignedString([]byte("your-secret-key"))
}

// Helper function to format validation errors
func formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "email":
		return "Invalid email format"
	case "min":
		return fmt.Sprintf("Minimum length is %s", err.Param())
	case "max":
		return fmt.Sprintf("Maximum length is %s", err.Param())
	case "oneof":
		return fmt.Sprintf("Must be one of: %s", err.Param())
	default:
		return fmt.Sprintf("Validation failed on %s", err.Tag())
	}
}
