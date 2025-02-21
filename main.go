package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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
	router.HandleFunc("/", helloHandler).Methods("GET")            // GET request
	router.HandleFunc("/users", getAllUsersHandler).Methods("GET") // GET request

	// Start the server on port 8080
	// nil parameter is used to use the default router (net/http)
	// router parameter is used to use the gorilla/mux router
	fmt.Println("Server running at http://localhost:80")
	log.Fatal(http.ListenAndServe(":80", router))

}
func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello World!")
}

func getAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Access the users collection from the database
	usersCollection := client.Database("test-db").Collection("users")

	// Get all users from the users collection
	cursor, err := usersCollection.Find(context.TODO(), bson.M{})
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
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
