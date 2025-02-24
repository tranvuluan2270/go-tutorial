package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"go-tutorial/models"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Handler struct {
	DB       *mongo.Client
	Database string
	Router   *mux.Router
}

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
	var users []models.User

	// Decode the cursor into the users slice
	if err := cursor.All(context.TODO(), &users); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with the users as JSON
	respondWithJSON(w, http.StatusOK, users)
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
	respondWithJSON(w, http.StatusOK, user)

}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	// Access the users collection from the database
	usersCollection := h.DB.Database(h.Database).Collection("users")

	// Create a new UserDetails struct to hold the incoming data
	var newUser models.UserDetails

	// Decode the incoming JSON request body into the newUser struct
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate a new ObjectId for the user
	newUser.ID = primitive.NewObjectID()

	// Insert the new user into the users collection
	_, err = usersCollection.InsertOne(context.TODO(), newUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the created user as JSON
	respondWithJSON(w, http.StatusOK, newUser)

}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Access the users collection from the database
	usersCollection := h.DB.Database(h.Database).Collection("users")

	// Get the user ID from the URL parameters
	vars := mux.Vars(r)
	id := vars["id"]

	// Convert the id string to an ObjectId to match the _id field in MongoDB (ObjectId)
	objID, err := primitive.ObjectIDFromHex(id)

	// Check if the ID is valid, if ID valid then find the user with the given ID, if not return an error and exit
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// Create a new UserDetails struct to hold the updated data
	var updatedUser models.UserDetails

	// Decode the incoming JSON request body into the updatedUser struct
	err = json.NewDecoder(r.Body).Decode(&updatedUser)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure the ID in the URL matches the ID in the request body
	updatedUser.ID = objID

	// Update the user in the users collection
	result, err := usersCollection.ReplaceOne(context.TODO(), bson.M{"_id": objID}, updatedUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if result.MatchedCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Return the updated user as JSON
	respondWithJSON(w, http.StatusOK, updatedUser)

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
	respondWithJSON(w, http.StatusOK, map[string]string{"message": "User successfully deleted"})
}

// Helper function for JSON responses
func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}
