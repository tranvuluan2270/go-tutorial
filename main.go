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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const uri = "mongodb://localhost:27017/"

var client *mongo.Client
var err error

// User table struct
type User struct {
	ID      string `json:"id" bson:"_id"`
	Name    string `json:"name" bson:"name"`
	Email   string `json:"email" bson:"email"`
	Age     int    `json:"age" bson:"age"`
	Address string `json:"address" bson:"address"`
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
	router.HandleFunc("/", helloHandler).Methods("GET")         // GET request
	router.HandleFunc("/users", getUsersHandler).Methods("GET") // GET request

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
	cursor, err := usersCollection.Find(context.TODO(), bson.M{}, findOptions)
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
