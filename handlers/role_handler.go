package handlers

import (
	"encoding/json"
	"go-tutorial/middleware"
	"go-tutorial/models"
	"go-tutorial/utils"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	// Return all available roles and their permissions
	roles := make(map[string][]middleware.Permission)
	for role, permissions := range middleware.RolePermissions {
		roles[role] = permissions
	}
	h.ResponseHdlr.Success(w, "Roles retrieved successfully", roles)
}

func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	// Get claims from context
	claims, ok := r.Context().Value("claims").(jwt.MapClaims)
	if !ok {
		h.ErrorHdlr.HandleUnauthorized(w, "Invalid token claims")
		return
	}

	// Get user role from claims
	userRole, ok := claims["role"].(string)
	if !ok {
		h.ErrorHdlr.HandleUnauthorized(w, "Invalid token claims")
		return
	}

	// Only master_admin can modify roles
	if userRole != "master_admin" {
		h.ErrorHdlr.HandleForbidden(w, "Only master admin can modify roles")
		return
	}

	var req struct {
		UserID string `json:"user_id" validate:"required"`
		Role   string `json:"role" validate:"required"`
	}

	// Decode request body
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.ErrorHdlr.HandleBadRequest(w, "Invalid request body")
		return
	}

	// Validate required fields
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		var validationErrors []utils.ErrorDetail
		for _, err := range err.(validator.ValidationErrors) {
			validationErrors = append(validationErrors, utils.ErrorDetail{
				Field:   err.Field(),
				Message: "This field is required",
			})
		}
		h.ErrorHdlr.HandleValidationError(w, validationErrors)
		return
	}

	// Validate role value
	validRoles := []string{"master_admin", "sub_admin", "user"}
	isValidRole := false
	for _, validRole := range validRoles {
		if req.Role == validRole {
			isValidRole = true
			break
		}
	}

	if !isValidRole {
		h.ErrorHdlr.HandleValidationError(w, []utils.ErrorDetail{
			{
				Field:   "role",
				Message: "Role must be one of: master_admin, sub_admin, user",
			},
		})
		return
	}

	// Convert user ID to ObjectID
	objID, err := primitive.ObjectIDFromHex(req.UserID)
	if err != nil {
		h.ErrorHdlr.HandleValidationError(w, []utils.ErrorDetail{
			{
				Field:   "user_id",
				Message: "Invalid user ID format",
			},
		})
		return
	}

	// Check if user exists before updating
	var existingUser models.UserDetails
	err = h.DB.Database(h.Database).Collection("users").
		FindOne(r.Context(), bson.M{"_id": objID}).
		Decode(&existingUser)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			h.ErrorHdlr.HandleNotFound(w, "User not found")
		} else {
			h.ErrorHdlr.HandleInternalError(w, "Error checking user existence")
		}
		return
	}

	// Update user's role in database
	result, err := h.DB.Database(h.Database).Collection("users").UpdateOne(
		r.Context(),
		bson.M{"_id": objID},
		bson.M{"$set": bson.M{"role": req.Role}},
	)

	if err != nil {
		h.ErrorHdlr.HandleInternalError(w, "Error updating user role")
		return
	}

	if result.MatchedCount == 0 {
		h.ErrorHdlr.HandleNotFound(w, "User not found")
		return
	}

	// Return success with updated user details
	updatedUser := models.UserResponse{
		ID:    existingUser.ID,
		Name:  existingUser.Name,
		Email: existingUser.Email,
		Role:  req.Role,
	}

	h.ResponseHdlr.Success(w, "User role updated successfully", updatedUser)
}
