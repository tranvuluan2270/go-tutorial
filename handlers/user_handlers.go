package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

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

	"go-tutorial/cache"
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
	ctx := r.Context()

	// Get pagination parameters
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")
	page := 1
	limit := 10

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

	// Get basic filter parameters
	role := r.URL.Query().Get("role")          // Filter by role
	searchQuery := r.URL.Query().Get("search") // Search in name and email
	sortBy := r.URL.Query().Get("sort")        // Possible values: name_asc, name_desc, email_asc, email_desc

	// Create cache key
	cacheKey := fmt.Sprintf("users:p%d:l%d:role%s:q%s:sort%s",
		page, limit, role, searchQuery, sortBy)

	// Try to get from cache
	var cachedData struct {
		Users []models.UserResponse `json:"users"`
		Total int64                 `json:"total"`
	}

	err := cache.GetCache(ctx, cacheKey, &cachedData)
	if err == nil {
		w.Header().Set("X-Cache", "HIT")
		h.ResponseHdlr.Paginated(w, "Users fetched from cache", cachedData.Users, page, limit, int(cachedData.Total))
		return
	}

	w.Header().Set("X-Cache", "MISS")

	// Build filter query
	filterQuery := bson.M{}

	// Add role filter if provided
	if role != "" {
		filterQuery["role"] = role
	}

	// Add search filter if provided (search in name and email)
	if searchQuery != "" {
		filterQuery["$or"] = []bson.M{
			{"name": bson.M{"$regex": searchQuery, "$options": "i"}},
			{"email": bson.M{"$regex": searchQuery, "$options": "i"}},
		}
	}

	// Get total count with filters
	usersCollection := h.DB.Database(h.Database).Collection("users")
	total, err := usersCollection.CountDocuments(ctx, filterQuery)
	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error counting users")
		return
	}

	// Calculate skip for pagination
	skip := (page - 1) * limit

	// Build sort options
	sortOptions := bson.D{}
	switch sortBy {
	case "name_asc":
		sortOptions = append(sortOptions, bson.E{Key: "name", Value: 1})
	case "name_desc":
		sortOptions = append(sortOptions, bson.E{Key: "name", Value: -1})
	case "email_asc":
		sortOptions = append(sortOptions, bson.E{Key: "email", Value: 1})
	case "email_desc":
		sortOptions = append(sortOptions, bson.E{Key: "email", Value: -1})
	default:
		// Default sorting by name ascending
		sortOptions = append(sortOptions, bson.E{Key: "name", Value: 1})
	}

	// Find users with filters and sort
	opts := options.Find().
		SetLimit(int64(limit)).
		SetSkip(int64(skip)).
		SetSort(sortOptions)

	cursor, err := usersCollection.Find(ctx, filterQuery, opts)
	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error fetching users")
		return
	}
	defer cursor.Close(ctx)

	var users []models.UserResponse
	if err := cursor.All(ctx, &users); err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error processing users data")
		return
	}

	// Store in cache
	dataToCache := struct {
		Users []models.UserResponse `json:"users"`
		Total int64                 `json:"total"`
	}{
		Users: users,
		Total: total,
	}

	if err := cache.SetCache(ctx, cacheKey, dataToCache, 5*time.Minute); err != nil {
		log.Printf("Failed to cache users list: %v", err)
	}

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

	// Try to get user from cache first
	var user models.UserDetails
	ctx := r.Context()
	err := cache.GetCache(ctx, requestedUserID, &user)
	if err == nil {
		// Cache hit
		w.Header().Set("X-Cache", "HIT")
		h.ResponseHdlr.Success(w, "User details fetched from cache", user)
		return
	}

	// Cache miss
	w.Header().Set("X-Cache", "MISS")

	// If not in cache, get from database
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

	err = h.DB.Database(h.Database).Collection("users").
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			h.ErrorHdlr.HandleNotFound(w, "User not found")
			return
		}
		h.ErrorHdlr.HandleInternalError(w, "Error fetching user details")
		return
	}

	// Store in cache for future requests (cache for 30 minutes)
	go func() {
		if err := cache.SetCache(context.Background(), requestedUserID, user, 30*time.Minute); err != nil {
			log.Printf("Failed to cache user data: %v", err)
		}
	}()

	h.ResponseHdlr.Success(w, "User details fetched successfully", user)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	userID := vars["id"]

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid user ID format")
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid request body")
		return
	}

	// Validate request
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid request")
		return
	}

	// Build update document
	update := bson.M{}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.Email != "" {
		update["email"] = req.Email
	}
	if req.Gender != "" {
		update["gender"] = req.Gender
	}
	if req.Age != 0 {
		update["age"] = req.Age
	}
	if req.Address != "" {
		update["address"] = req.Address
	}
	if req.Phone != "" {
		update["phone"] = req.Phone
	}
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			h.ErrorHdlr.HandleInternalError(w, "Error processing request")
			return
		}
		update["password"] = string(hashedPassword)
	}

	if len(update) == 0 {
		h.ErrorHdlr.HandleBadRequest(w, "No fields to update")
		return
	}

	// Update user in database
	result, err := h.DB.Database(h.Database).Collection("users").
		UpdateOne(ctx, bson.M{"_id": objID}, bson.M{"$set": update})

	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error updating user")
		return
	}

	if result.MatchedCount == 0 {
		h.ErrorHdlr.HandleNotFound(w, "User not found")
		return
	}

	// Invalidate cache
	// 1. Delete specific user cache
	detailCacheKey := fmt.Sprintf(cache.UserDetailPattern, userID)
	if err := cache.DeleteCache(ctx, detailCacheKey); err != nil {
		log.Printf("Failed to invalidate user detail cache: %v", err)
	}

	// 2. Delete all user list caches
	if err := cache.DeleteByPattern(ctx, cache.UserListPattern); err != nil {
		log.Printf("Failed to invalidate user list cache: %v", err)
	}

	// Get updated user
	var updatedUser models.UserDetails
	err = h.DB.Database(h.Database).Collection("users").
		FindOne(ctx, bson.M{"_id": objID}).
		Decode(&updatedUser)

	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error getting updated user")
		return
	}

	h.ResponseHdlr.Success(w, "User updated successfully", updatedUser)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	userID := vars["id"]

	objID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid user ID")
		return
	}

	result, err := h.DB.Database(h.Database).Collection("users").
		DeleteOne(ctx, bson.M{"_id": objID})

	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error deleting user")
		return
	}

	if result.DeletedCount == 0 {
		h.ErrorHdlr.HandleNotFound(w, "User not found")
		return
	}

	// Invalidate cache
	// 1. Delete specific user cache
	detailCacheKey := fmt.Sprintf(cache.UserDetailPattern, userID)
	if err := cache.DeleteCache(ctx, detailCacheKey); err != nil {
		log.Printf("Failed to invalidate user detail cache: %v", err)
	}

	// 2. Delete all user list caches
	if err := cache.DeleteByPattern(ctx, cache.UserListPattern); err != nil {
		log.Printf("Failed to invalidate user list cache: %v", err)
	}

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
