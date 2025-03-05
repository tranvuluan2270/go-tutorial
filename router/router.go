package router

import (
	"go-tutorial/handlers"
	"go-tutorial/middleware"
	"go-tutorial/models"
	"net/http"

	"github.com/gorilla/mux"
)

func SetupRoutes(h *handlers.Handler) *mux.Router {
	router := mux.NewRouter()

	// Public routes (no authentication required)
	router.HandleFunc("/signup", h.SignUp).Methods("POST")
	router.HandleFunc("/login", h.Login).Methods("POST")

	// Protected routes that require authentication
	protected := router.PathPrefix("").Subrouter()
	protected.Use(middleware.AuthMiddleware())

	// User management routes
	userRoutes := protected.PathPrefix("/user").Subrouter()
	userRoutes.Handle("/{id}",
		middleware.RequirePermission(models.PermissionReadUser)(
			http.HandlerFunc(h.GetUserDetails))).Methods("GET")
	userRoutes.Handle("/{id}",
		middleware.RequirePermission(models.PermissionUpdateUser)(
			http.HandlerFunc(h.UpdateUser))).Methods("PUT")
	userRoutes.Handle("/{id}",
		middleware.RequirePermission(models.PermissionDeleteUser)(
			http.HandlerFunc(h.DeleteUser))).Methods("DELETE")

	// Users list route (sub-admin and above)
	protected.Handle("/users",
		middleware.RequirePermission(models.PermissionListUsers)(
			http.HandlerFunc(h.GetUsers))).Methods("GET")

	// Role management routes (master-admin only)
	roleRoutes := protected.PathPrefix("/roles").Subrouter()
	roleRoutes.Handle("",
		middleware.RequirePermission(models.PermissionListRoles)(
			http.HandlerFunc(h.ListRoles))).Methods("GET")
	roleRoutes.Handle("",
		middleware.RequirePermission(models.PermissionAssignRole)(
			http.HandlerFunc(h.AssignRole))).Methods("POST")

	return router
}
