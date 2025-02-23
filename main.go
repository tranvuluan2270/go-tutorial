package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const uri = "mongodb://localhost:27017/"

var client *mongo.Client
var err error

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

func main() {
	// Initialize MongoDB client
	clientOptions := options.Client().ApplyURI(uri)
	client, err = mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(context.TODO())

	// Check the connection
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Connected to MongoDB!")

	// Initialize Gorilla Mux router
	router := mux.NewRouter()
	router.HandleFunc("/", helloHandler).Methods("GET") // GET request
	router.HandleFunc("/users", getUsersHandler).Methods("GET")
	router.HandleFunc("/users/{id}", getUserDetailsHandler).Methods("GET")
	router.HandleFunc("/users/create", createUserHandler).Methods("POST")        // POST request
	router.HandleFunc("/users/update/{id}", updateUserHandler).Methods("PUT")    // PUT request
	router.HandleFunc("/users/delete/{id}", deleteUserHandler).Methods("DELETE") // DELETE request

	// Start the server on port 80
	// nil parameter is used to use the default router (net/http)
	// router parameter is used to use the gorilla/mux router
	fmt.Println("Server running at http://localhost:80")
	log.Fatal(http.ListenAndServe(":80", router))

}
func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello World!")
}

func getUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Access the users collection from the database
	usersCollection := client.Database("test-db").Collection("users")

	// Get the page and limit query parameters for pagination
	pageStr := r.URL.Query().Get("page")
	limitStr := r.URL.Query().Get("limit")

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
		findOptions = options.Find()
		findOptions.SetLimit(int64(limit))
		findOptions.SetSkip(int64(skip))

	}

	// Get all users from the users collection
	cursor, err := usersCollection.Find(
		context.TODO(),
		bson.M{},
		findOptions,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	var users []User

	for cursor.Next(context.TODO()) {
		var user User
		err := cursor.Decode(&user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		users = append(users, user)
	}
	if err := cursor.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Return the users as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)

}

func getUserDetailsHandler(w http.ResponseWriter, r *http.Request) {
	// To get the user details, we need to get the user ID from the URL parameters and show more information about the user

	// Access the users collection from the database
	usersCollection := client.Database("test-db").Collection("users")

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

	// Find the user with the given ID
	var user UserDetails

	err = usersCollection.FindOne(
		context.TODO(),
		bson.M{"_id": objID},
	).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {

			http.Error(w, "User not found", http.StatusNotFound)
		} else {

			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return the user as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)

}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	// Access the users collection from the database
	usersCollection := client.Database("test-db").Collection("users")

	// Create a new UserDetails struct to hold the incoming data
	var newUser UserDetails

	// Decode the incoming JSON request body into the newUser struct
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate a new ObjectId for the user
	newUser.ID = primitive.NewObjectID()

	// Insert the new user into the users collection
	_, err = usersCollection.InsertOne(
		context.TODO(),
		newUser,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the created user as JSON
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(newUser)

}

func updateUserHandler(w http.ResponseWriter, r *http.Request) {
	// Access the users collection from the database
	usersCollection := client.Database("test-db").Collection("users")

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
	var updatedUser UserDetails

	// Decode the incoming JSON request body into the updatedUser struct
	err = json.NewDecoder(r.Body).Decode(&updatedUser)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Ensure the ID in the URL matches the ID in the request body
	updatedUser.ID = objID

	// Update the user in the users collection
	result, err := usersCollection.ReplaceOne(
		context.TODO(),
		bson.M{"_id": objID},
		updatedUser,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if result.MatchedCount == 0 {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// Return the updated user as JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedUser)

}

func deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	// Access the users collection from the database
	usersCollection := client.Database("test-db").Collection("users")

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
	result, err := usersCollection.DeleteOne(
		context.TODO(),
		bson.M{"_id": objID},
	)
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
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "User successfully deleted",
	})
}
