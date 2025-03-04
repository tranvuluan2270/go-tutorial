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

	// Protected routes that require authentication
	protected := router.PathPrefix("").Subrouter()
	protected.Use(middleware.AuthMiddleware)

	// Routes accessible to all users
	protected.HandleFunc("/user/{id}", h.GetUserDetails).Methods("GET")

	// Admin-only routes
	admin := protected.PathPrefix("/admin").Subrouter()
	admin.Use(middleware.RoleMiddleware("admin"))
	admin.HandleFunc("/users", h.GetUsers).Methods("GET")
	admin.HandleFunc("/user/{id}", h.UpdateUser).Methods("PUT")
	admin.HandleFunc("/user/{id}", h.DeleteUser).Methods("DELETE")

	return router
}
