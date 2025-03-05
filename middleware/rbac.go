package middleware

import (
	"go-tutorial/utils"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

func RoleMiddleware(allowedRoles ...string) mux.MiddlewareFunc {
	errorHandler := utils.NewErrorHandler()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get claims from context (set by AuthMiddleware)
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

			// Check if user role is in allowed roles
			roleAllowed := false
			for _, role := range allowedRoles {
				if role == userRole {
					roleAllowed = true
					break
				}
			}

			if !roleAllowed {
				errorHandler.HandleForbidden(w, "Insufficient role permissions")
				return
			}

			// Proceed to next handler
			next.ServeHTTP(w, r)
		})
	}
}
