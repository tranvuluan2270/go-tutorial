package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
)

// User table struct
// type User struct {
// 	Id      primitive.ObjectID `json:"id" bson:"_id"`
// 	Name    string             `json:"name" bson:"name"`
// 	Email   string             `json:"email" bson:"email"`
// 	Age     int                `json:"age" bson:"age"`
// 	Address string             `json:"address" bson:"address"`
// }

func main() {

	// default router (net/http)
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/hello", helloHandler)

	// goriila/mux router
	router := mux.NewRouter()
	router.HandleFunc("/", rootHandler).Methods("GET")       // GET request
	router.HandleFunc("/hello", helloHandler).Methods("GET") // GET request

	// start the server on port 8080
	http.ListenAndServe(":8080", router)
	// nil parameter is used to use the default router (net/http)
	// router parameter is used to use the gorilla/mux router

}
func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Welcome to the homepage!")
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, world!")
}
