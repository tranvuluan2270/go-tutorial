package middleware

import (
	"go-tutorial/utils"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
)

// Permission represents a single permission action
type Permission string

const (
	// User permissions
	PermissionListUsers  Permission = "list:users"
	PermissionReadUser   Permission = "read:user"
	PermissionUpdateUser Permission = "update:user"
	PermissionDeleteUser Permission = "delete:user"

	// Role permissions
	PermissionListRoles  Permission = "list:roles"
	PermissionAssignRole Permission = "assign:role"

	// Product permissions
	PermissionListProducts  Permission = "list:products"
	PermissionReadProduct   Permission = "read:product"
	PermissionCreateProduct Permission = "create:product"
	PermissionUpdateProduct Permission = "update:product"
	PermissionDeleteProduct Permission = "delete:product"
)

// RolePermissions maps roles to their permissions
var RolePermissions = map[string][]Permission{
	"master_admin": {
		// User permissions
		PermissionListUsers,
		PermissionReadUser,
		PermissionUpdateUser,
		PermissionDeleteUser,

		// Role permissions
		PermissionListRoles,
		PermissionAssignRole,

		// Product permissions
		PermissionListProducts,
		PermissionReadProduct,
		PermissionCreateProduct,
		PermissionUpdateProduct,
		PermissionDeleteProduct,
	},
	"sub_admin": {
		// User permissions
		PermissionListUsers,
		PermissionReadUser,
		PermissionUpdateUser,
		PermissionListRoles,

		// Product permissions
		PermissionListProducts,
		PermissionReadProduct,
		PermissionCreateProduct,
		PermissionUpdateProduct,
	},
	"user": {
		// User permissions
		PermissionReadUser,
		PermissionUpdateUser,

		// Product permissions
		PermissionListProducts,
		PermissionReadProduct,
	},
}

// HasPermission checks if a role has a specific permission
func HasPermission(role string, requiredPermission Permission) bool {
	permissions, exists := RolePermissions[role]
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
func RequirePermission(requiredPermission Permission) mux.MiddlewareFunc {
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
