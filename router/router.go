package router

import (
	"go-tutorial/handlers"
	"go-tutorial/middleware"
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
		middleware.RequirePermission(middleware.PermissionReadUser)(
			http.HandlerFunc(h.GetUserDetails))).Methods("GET")
	userRoutes.Handle("/{id}",
		middleware.RequirePermission(middleware.PermissionUpdateUser)(
			http.HandlerFunc(h.UpdateUser))).Methods("PUT")
	userRoutes.Handle("/{id}",
		middleware.RequirePermission(middleware.PermissionDeleteUser)(
			http.HandlerFunc(h.DeleteUser))).Methods("DELETE")

	// Users list route (sub-admin and above)
	protected.Handle("/users",
		middleware.RequirePermission(middleware.PermissionListUsers)(
			http.HandlerFunc(h.GetUsers))).Methods("GET")

	// Role management routes (master-admin only)
	roleRoutes := protected.PathPrefix("/roles").Subrouter()
	roleRoutes.Handle("",
		middleware.RequirePermission(middleware.PermissionListRoles)(
			http.HandlerFunc(h.ListRoles))).Methods("GET")
	roleRoutes.Handle("",
		middleware.RequirePermission(middleware.PermissionAssignRole)(
			http.HandlerFunc(h.AssignRole))).Methods("POST")

	// Product routes
	productRoutes := protected.PathPrefix("/product").Subrouter()
	productRoutes.Handle("",
		middleware.RequirePermission(middleware.PermissionListProducts)(
			http.HandlerFunc(h.GetProducts))).Methods("GET")
	productRoutes.Handle("/{id}",
		middleware.RequirePermission(middleware.PermissionReadProduct)(
			http.HandlerFunc(h.GetProductDetails))).Methods("GET")
	productRoutes.Handle("",
		middleware.RequirePermission(middleware.PermissionCreateProduct)(
			http.HandlerFunc(h.CreateProduct))).Methods("POST")
	productRoutes.Handle("/{id}",
		middleware.RequirePermission(middleware.PermissionUpdateProduct)(
			http.HandlerFunc(h.UpdateProduct))).Methods("PUT")
	productRoutes.Handle("/{id}",
		middleware.RequirePermission(middleware.PermissionDeleteProduct)(
			http.HandlerFunc(h.DeleteProduct))).Methods("DELETE")

	return router
}
