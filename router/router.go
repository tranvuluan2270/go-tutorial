package router

import (
	"go-tutorial/handlers"
	"go-tutorial/middleware"

	"github.com/gorilla/mux"
)

func SetupRoutes(h *handlers.Handler) *mux.Router {
	router := mux.NewRouter()

	// Public routes
	router.HandleFunc("/signup", h.SignUp).Methods("POST")
	router.HandleFunc("/login", h.Login).Methods("POST")

	// Protected routes
	protected := router.PathPrefix("").Subrouter()
	protected.Use(middleware.AuthMiddleware)

	protected.HandleFunc("/users", h.GetUsers).Methods("GET")
	protected.HandleFunc("/user/{id}", h.GetUserDetails).Methods("GET")
	protected.HandleFunc("/user/{id}", h.UpdateUser).Methods("PUT")
	protected.HandleFunc("/user/{id}", h.DeleteUser).Methods("DELETE")

	return router
}
