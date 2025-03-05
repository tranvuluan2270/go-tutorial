package middleware

import (
	"context"
	"go-tutorial/utils"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

// AuthMiddleware verifies the JWT token and adds claims to the request context
func AuthMiddleware() mux.MiddlewareFunc {
	errorHandler := utils.NewErrorHandler()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				errorHandler.HandleUnauthorized(w, "Missing authorization header")
				return
			}

			// Extract token from "Bearer <token>"
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				errorHandler.HandleUnauthorized(w, "Invalid authorization format")
				return
			}

			// Parse and validate token
			claims := jwt.MapClaims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte("your-secret-key"), nil
			})

			if err != nil || !token.Valid {
				errorHandler.HandleUnauthorized(w, "Invalid token")
				return
			}

			// Add claims to request context
			ctx := context.WithValue(r.Context(), "claims", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
