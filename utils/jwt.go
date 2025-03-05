package utils

import (
	"time"
	"github.com/golang-jwt/jwt/v5"
)

// GenerateJWT generates a new JWT token for a user
func GenerateJWT(userID string, role string) (string, error) {
	// Create claims
	claims := jwt.MapClaims{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(time.Hour * 24).Unix(),
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	tokenString, err := token.SignedString([]byte("your-secret-key"))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}