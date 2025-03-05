package models

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
)

// RolePermissions maps roles to their permissions
var RolePermissions = map[string][]Permission{
	"master_admin": {
		PermissionListUsers,  // view all users
		PermissionReadUser,   // view user details
		PermissionUpdateUser, // update user details
		PermissionDeleteUser, // delete user
		PermissionListRoles,  // view all roles
		PermissionAssignRole, // assign role
	},
	"sub_admin": {
		PermissionListUsers,
		PermissionReadUser,
		PermissionUpdateUser,
		PermissionListRoles,
	},
	"user": {
		PermissionReadUser,
		PermissionUpdateUser,
	},
}
