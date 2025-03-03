package middleware

import (
    "context"
    "net/http"
    "strings"

    "github.com/golang-jwt/jwt/v5"
    "go.mongodb.org/mongo-driver/bson/primitive"
)

// AuthMiddleware verifies the JWT token in the Authorization header
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Get the Authorization header
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Authorization header is required", http.StatusUnauthorized)
            return
        }

        // Check if the Authorization header has the correct format
        bearerToken := strings.Split(authHeader, " ")
        if len(bearerToken) != 2 || strings.ToLower(bearerToken[0]) != "bearer" {
            http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
            return
        }

        // Parse and validate the JWT token
        token, err := jwt.Parse(bearerToken[1], func(token *jwt.Token) (interface{}, error) {
            // Validate the signing method
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, jwt.ErrSignatureInvalid
            }
            return []byte("your-secret-key"), nil // Replace with your secret key
        })

        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        // Check if the token is valid
        if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
            // Add user ID to request context
            userID, err := primitive.ObjectIDFromHex(claims["user_id"].(string))
            if err != nil {
                http.Error(w, "Invalid user ID in token", http.StatusUnauthorized)
                return
            }
            ctx := context.WithValue(r.Context(), "user_id", userID)
            next.ServeHTTP(w, r.WithContext(ctx))
        } else {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }
    })
}