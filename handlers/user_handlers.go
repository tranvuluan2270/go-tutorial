package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"

	"golang.org/x/crypto/bcrypt"

	"go-tutorial/models"
	"go-tutorial/utils"

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
	ResponseHdlr *utils.ResponseHandler
	ErrorHdlr    *utils.ErrorHandler
}

// NewHandler creates a new handler with all dependencies
func NewHandler(db *mongo.Client, database string) *Handler {
	return &Handler{
		DB:           db,
		Database:     database,
		ResponseHdlr: utils.NewResponseHandler(),
		ErrorHdlr:    utils.NewErrorHandler(),
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
	var users []models.User
	if err := cursor.All(context.TODO(), &users); err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error processing users data")
		return
	}

	// Return a success response
	h.ResponseHdlr.Paginated(w, "Users fetched successfully", users, page, limit, int(total))
}

func (h *Handler) GetUserDetails(w http.ResponseWriter, r *http.Request) {
	// Get claims from context
	claims, ok := r.Context().Value("claims").(jwt.MapClaims)
	if !ok {
		h.ErrorHdlr.HandleUnauthorized(w, "Invalid token claims")
		return
	}

	// Get user ID and role from claims
	currentUserID := claims["user_id"].(string)
	userRole := claims["role"].(string)

	// Get requested user ID from URL
	vars := mux.Vars(r)
	requestedUserID := vars["id"]

	// Convert requested ID to ObjectID
	objID, err := primitive.ObjectIDFromHex(requestedUserID)
	if err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid user ID")
		return
	}

	// Check permissions
	if userRole == "user" && currentUserID != requestedUserID {
		h.ErrorHdlr.HandleForbidden(w, "Access denied")
		return
	}

	// Get user from database
	var user models.UserDetails
	err = h.DB.Database(h.Database).Collection("users").
		FindOne(context.TODO(), bson.M{"_id": objID}).
		Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			h.ErrorHdlr.HandleNotFound(w, "User not found")
			return
		}
		h.ErrorHdlr.HandleInternalError(w, "Error fetching user details")
		return
	}

	// Return response
	h.ResponseHdlr.Success(w, "User details fetched successfully", user)

}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Get claims from context
	claims, ok := r.Context().Value("claims").(jwt.MapClaims)
	if !ok {
		h.ErrorHdlr.HandleUnauthorized(w, "Invalid token claims")
		return
	}

	// Get current user ID and role from claims
	currentUserID := claims["user_id"].(string)
	userRole := claims["role"].(string)

	// Get user ID from URL
	vars := mux.Vars(r)
	requestedUserID := vars["id"]

	// Check permissions - users can only update their own profile
	if userRole == "user" && currentUserID != requestedUserID {
		h.ErrorHdlr.HandleForbidden(w, "You can only update your own profile")
		return
	}

	// Convert the id string to an ObjectId to match the _id field in MongoDB (ObjectId)
	objID, err := primitive.ObjectIDFromHex(requestedUserID)
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
		var validationErrors []utils.ErrorDetail
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, utils.ErrorDetail{
				Field:   err.Field(),
				Message: utils.FormatValidationError(err),
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
		User: models.User{
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
	// Parse request
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid request body")
		return
	}

	// Find user
	var user models.UserDetails
	err := h.DB.Database(h.Database).Collection("users").
		FindOne(context.TODO(), bson.M{"email": req.Email}).
		Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			h.ErrorHdlr.HandleUnauthorized(w, "Invalid email or password")
			return
		}
		h.ErrorHdlr.HandleInternalError(w, "Error finding user")
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		h.ErrorHdlr.HandleUnauthorized(w, "Invalid email or password")
		return
	}

	// Generate token
	token, err := utils.GenerateJWT(user.ID.Hex(), user.Role)
	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error generating token")
		return
	}

	// Create response
	response := models.LoginResponse{
		Token: token,
		User: models.UserResponse{
			ID:    user.ID,
			Name:  user.Name,
			Email: user.Email,
			Role:  user.Role,
		},
	}

	h.ResponseHdlr.Success(w, "Login successful", response)
}
