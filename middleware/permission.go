package middleware

import (
	"go-tutorial/models"
	"go-tutorial/utils"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

// HasPermission checks if a role has a specific permission
func HasPermission(role string, requiredPermission models.Permission) bool {

	permissions, exists := models.RolePermissions[role]

	if !exists {
		return false
	}

	for _, permission := range permissions {
		if permission == requiredPermission {
			return true
		}
	}
	return false
}

// RequirePermission middleware checks if the user has the required permission
func RequirePermission(requiredPermission models.Permission) mux.MiddlewareFunc {
	errorHandler := utils.NewErrorHandler()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get claims from context first
			claims, ok := r.Context().Value("claims").(jwt.MapClaims)
			if !ok {
				errorHandler.HandleUnauthorized(w, "Invalid token claims")
				return
			}

			// Get user role from claims
			userRole, ok := claims["role"].(string)
			if !ok {
				errorHandler.HandleUnauthorized(w, "Invalid token claims")
				return
			}

			// Check if user has the required permission
			if !HasPermission(userRole, requiredPermission) {
				errorHandler.HandleForbidden(w, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
