package handlers

import (
	"context"
	"encoding/json"
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
	DB       *mongo.Client
	Database string
	Router   *mux.Router
}

// Response represents the structure of the response
type Response struct {
	Status  int         `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Access the users collection from the database
	usersCollection := h.DB.Database(h.Database).Collection("users")

	// Get the page and limit query parameters for pagination
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

	// Initialize the find options
	var findOptions *options.FindOptions

	// Check if the page and limit query parameters are provided
	if pageStr != "" && limitStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil || page < 1 {
			page = 1
			// If there is no page query parameter, the default value is 1
		}
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			limit = 2
			// If there is no limit query parameter, the default value is 2
		}
		// Calculate the number of documents to skip
		skip := (page - 1) * limit

		// Initialize findOptions with limit and skip
		findOptions = options.Find().
			SetLimit(int64(limit)).
			SetSkip(int64(skip))
	}

	// Get all users from the users collection
	cursor, err := usersCollection.Find(context.TODO(), bson.M{}, findOptions)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	// Create a slice to hold the users
	var users []models.BaseUser

	// Decode the cursor into the users slice
	if err := cursor.All(context.TODO(), &users); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with the users as JSON
	respondWithJSON(w, http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: "Users fetched successfully",
		Data:    users,
	})
}

func (h *Handler) GetUserDetails(w http.ResponseWriter, r *http.Request) {
	// To get the user details, we need to get the user ID from the URL parameters and show more information about the user

	// Access the users collection from the database
	usersCollection := h.DB.Database(h.Database).Collection("users")

	// Get the user ID from the URL parameters
	// this id parameter is the value of the {id} path variable in the URL (a string)
	vars := mux.Vars(r)
	id := vars["id"]

	// Convert the id string to an ObjectId to match the _id field in MongoDB (ObjectId)
	objID, err := primitive.ObjectIDFromHex(id)

	// Check if the ID is valid, if ID valid then find the user with the given ID, if not return an error and exit
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Create a UserDetails struct to hold the user details
	var user models.UserDetails

	// Find the user with the given ID
	err = usersCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user)

	// Check if the user was found
	if err != nil {
		// If the user was not found, return a 404 error
		if err == mongo.ErrNoDocuments {
			http.Error(w, "User not found", http.StatusNotFound)
		} else {
			// If there is an error, return a 500 error
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Respond with the users as JSON
	respondWithJSON(w, http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: "User details fetched successfully",
		Data:    user,
	})

}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {

	// Get user ID from URL
	vars := mux.Vars(r)
	id := vars["id"]

	// Convert the id string to an ObjectId to match the _id field in MongoDB (ObjectId)
	objID, err := primitive.ObjectIDFromHex(id)

	if err != nil {
		respondWithJSON(w, http.StatusBadRequest, Response{
			Status:  http.StatusBadRequest,
			Message: "Invalid user ID",
		})
		return
	}

	// Parse request body
	var updateReq models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		respondWithJSON(w, http.StatusBadRequest, Response{
			Status:  http.StatusBadRequest,
			Message: "Invalid request body",
		})
		return
	}

	// Validate request
	validate := validator.New()
	if err := validate.Struct(updateReq); err != nil {
		respondWithJSON(w, http.StatusBadRequest, Response{
			Status:  http.StatusBadRequest,
			Message: "Validation failed",
		})
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
			respondWithJSON(w, http.StatusInternalServerError, Response{
				Status:  http.StatusInternalServerError,
				Message: "Error processing password",
			})
			return
		}
		update["password"] = string(hashedPassword)
	}

	// Check if there are fields to update
	if len(update) == 0 {
		respondWithJSON(w, http.StatusBadRequest, Response{
			Status:  http.StatusBadRequest,
			Message: "No fields to update provided",
		})
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
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Status:  http.StatusInternalServerError,
			Message: "Error updating user",
		})
		return
	}

	// Check if user was found and updated
	if result.MatchedCount == 0 {
		respondWithJSON(w, http.StatusNotFound, Response{
			Status:  http.StatusNotFound,
			Message: "User not found",
		})
		return
	}

	//Get the updated user
	updatedUser := models.UserDetails{}
	err = usersCollection.FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&updatedUser)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Status:  http.StatusInternalServerError,
			Message: "Error getting updated user",
		})
		return
	}
	respondWithJSON(w, http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: "User updated successfully",
		Data:    updatedUser,
	})
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Access the users collection from the database
	usersCollection := h.DB.Database(h.Database).Collection("users")

	// Get the user ID from the URL parameters
	vars := mux.Vars(r)
	id := vars["id"]

	// Convert the id string to an ObjectId to match the _id field in MongoDB (ObjectId)
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Delete the user from the users collection
	result, err := usersCollection.DeleteOne(context.TODO(), bson.M{"_id": objID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Check if a user was actually deleted
	if result.DeletedCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Return a success message
	respondWithJSON(w, http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: "User successfully deleted",
	})
}

func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	// Parse and validate the request body
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithJSON(w, http.StatusBadRequest, Response{
			Status:  http.StatusBadRequest,
			Message: "Invalid request body",
		})
		return
	}

	// Set default role if not provided
	if req.Role == "" {
		req.Role = "user" // Default role
	}

	// Validate the request
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		respondWithJSON(w, http.StatusBadRequest, Response{
			Status:  http.StatusBadRequest,
			Message: "Validation failed",
			Data:    err.Error(),
		})
		return
	}

	// Check if user already exists
	usersCollection := h.DB.Database(h.Database).Collection("users")
	var existingUser models.UserDetails
	err := usersCollection.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&existingUser)
	if err == nil {
		respondWithJSON(w, http.StatusConflict, Response{
			Status:  http.StatusConflict,
			Message: "User with this email already exists",
		})
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Status:  http.StatusInternalServerError,
			Message: "Error processing request",
		})
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
	_, err = usersCollection.InsertOne(context.TODO(), newUser)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Status:  http.StatusInternalServerError,
			Message: "Error creating user",
		})
		return
	}

	// Remove password from response
	newUser.Password = ""

	// Return success response
	respondWithJSON(w, http.StatusCreated, Response{
		Status:  http.StatusCreated,
		Message: "User created successfully",
		Data:    newUser,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {

	// Parse and validate the request body
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithJSON(w, http.StatusBadRequest, Response{
			Status:  http.StatusBadRequest,
			Message: "Invalid request body",
		})
		return
	}

	// Validate the request
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		respondWithJSON(w, http.StatusBadRequest, Response{
			Status:  http.StatusBadRequest,
			Message: "Validation failed",
			Data:    err.Error(),
		})
		return
	}

	// Find user by email
	usersCollection := h.DB.Database(h.Database).Collection("users")
	var user models.UserDetails
	err := usersCollection.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			respondWithJSON(w, http.StatusUnauthorized, Response{
				Status:  http.StatusUnauthorized,
				Message: "Invalid email or password",
			})
			return
		}
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Status:  http.StatusInternalServerError,
			Message: "Error finding user",
		})
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password))
	if err != nil {
		respondWithJSON(w, http.StatusUnauthorized, Response{
			Status:  http.StatusUnauthorized,
			Message: "Invalid email or password",
		})
		return
	}

	// Generate JWT token
	token, err := GenerateJWT(user.ID.Hex(), user.Role)
	if err != nil {
		respondWithJSON(w, http.StatusInternalServerError, Response{
			Status:  http.StatusInternalServerError,
			Message: "Error generating token",
		})
		return
	}

	// Remove password from response
	user.Password = ""

	// Return token and user info
	respondWithJSON(w, http.StatusOK, Response{
		Status:  http.StatusOK,
		Message: "Login successful",
		Data: models.LoginResponse{
			Token: token,
			User: models.UserResponse{
				ID:    user.ID,
				Name:  user.Name,
				Email: user.Email,
			},
		},
	})
}

// GenerateJWT generates a new JWT token for a user
func GenerateJWT(userID string, role string) (string, error) {
	// Create the Claims
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Token expires in 24 hours
	}

	// Create token with claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Generate encoded token using the secret signing key
	return token.SignedString([]byte("your-secret-key")) // Replace with your secret key
}

// Helper function for JSON responses
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
