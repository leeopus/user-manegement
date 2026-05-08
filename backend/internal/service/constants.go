package service

const (
	// User status
	StatusActive   = "active"
	StatusDisabled = "disabled"
	StatusInactive = "inactive"
	StatusBanned   = "banned"
	StatusDeleted  = "deleted"

	// System role codes
	RoleAdmin = "admin"
	RoleUser  = "user"

	// PostgreSQL constraint names (must match DB schema)
	ConstraintUsersUsernameKey = "idx_users_username"
	ConstraintUsersEmailKey    = "idx_users_email"

	// Permission codes (resource:action format)
	PermUserRead       = "users:read"
	PermUserWrite      = "users:write"
	PermUserDelete     = "users:delete"
	PermRoleManage     = "roles:manage"
	PermPermissionManage = "permissions:manage"
	PermOAuthManage    = "oauth:manage"
	PermAuditRead      = "audit:read"
)
